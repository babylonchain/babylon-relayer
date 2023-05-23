package bbnrelayer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	relaydebug "github.com/babylonchain/babylon-relayer/debug"
	relayercmd "github.com/cosmos/relayer/v2/cmd"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
)

// Relayer is a relayer that allows to relay multiple chains concurrently.
// It is made thread-safe to avoid account sequence mismatch errors in Cosmos SDK accounts.
type Relayer struct {
	homePath string
	cfg      *relayercmd.Config
	logger   *zap.Logger
	metrics  *relaydebug.PrometheusMetrics
}

func New(homePath string, cfg *relayercmd.Config, logger *zap.Logger, metrics *relaydebug.PrometheusMetrics) *Relayer {
	return &Relayer{
		homePath: homePath,
		cfg:      cfg,
		logger:   logger,
		metrics:  metrics,
	}
}

// UpdateClient updates the IBC light client on src chain that tracks dst chain
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/relayer/client.go#L17)
func (r *Relayer) UpdateClient(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	numRetries uint,
) error {
	// get client ID for the dst IBC light client on src chain in DB
	clientID, err := r.getClientID(dst.ChainID())
	if err != nil {
		r.logger.Error(
			"failed to get client ID for CZ light client",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.Error(err),
		)
		return err
	}

	// query the latest heights on src and dst
	var srch, dsth int64
	if err := retry.Do(func() error {
		var err error
		srch, dsth, err = relayer.QueryLatestHeights(ctx, src, dst)
		if err != nil {
			return fmt.Errorf("failed to query latest heights: %w", err)
		}
		return nil
	}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
		r.logger.Info(
			"Failed to query latest heights",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.Uint("attempt", n+1),
			zap.Uint("max_attempts", numRetries),
			zap.Error(err),
		)
	})); err != nil {
		return err
	}

	// generate MsgUpdateClient that carries dst header and is sent to src
	srcMsgUpdateClient, err := r.CreateMsgUpdateClient(ctx, dst, src, dsth, srch, clientID)
	if err != nil {
		return err
	}

	clients := &relayer.RelayMsgs{
		Src: []provider.RelayerMessage{srcMsgUpdateClient},
		Dst: []provider.RelayerMessage{}, // nothing will be sent to dst chain
	}

	// Send msgs to src chain in a thread-safe way
	var result relayer.SendMsgsResult
	krErr := r.accessKeyWithLock(func() {
		result = clients.Send(ctx, r.logger, relayer.AsRelayMsgSender(src), relayer.AsRelayMsgSender(dst), r.cfg.Global.Memo)
	})
	if krErr != nil {
		return err
	}
	if err := result.Error(); err != nil {
		if result.PartiallySent() {
			r.logger.Info(
				"Partial success when updating clients",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.Object("send_result", result),
			)
		}
		return err
	}

	r.logger.Info(
		"successfully updated the client",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client", clientID),
	)

	return nil
}

func (r *Relayer) KeepUpdatingClient(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	interval time.Duration,
	numRetries uint,
) error {
	// ensure the CZ chain light client exists on Babylon
	if err := r.createClientIfNotExist(ctx, src, dst, numRetries); err != nil {
		r.logger.Error(
			"failed to ensure CZ light client exists on Babylon. Stop relaying the chain",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.Error(err),
		)
		return err
	}

	r.logger.Info(
		"Keep updating client",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.Duration("interval", interval),
	)
	r.metrics.RelayedChainsCounter.WithLabelValues(src.ChainID(), dst.ChainID()).Inc()

	ticker := time.NewTicker(interval)
	for ; true; <-ticker.C {
		// Note that UpdateClient is a thread-safe function
		if err := r.UpdateClient(ctx, src, dst, numRetries); err != nil {
			r.logger.Error(
				"Failed to update client",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.Error(err),
			)
			r.metrics.FailedHeadersCounter.WithLabelValues(src.ChainID(), dst.ChainID()).Inc()

			// NOTE: the for loop continues here since it's possible that
			// the endpoint of dst chain is temporarily unavailable
			// TODO: distinguish unrecoverable errors
		} else {
			r.metrics.RelayedHeadersCounter.WithLabelValues(src.ChainID(), dst.ChainID()).Inc()
		}
	}
	return nil
}

func (r *Relayer) KeepUpdatingClients(
	ctx context.Context,
	wg *sync.WaitGroup,
	interval time.Duration,
	numRetries uint,
) {
	// get babylonChain object from config
	babylonChain, err := r.cfg.Chains.Get("babylon") // TODO: parameterise Babylon chain name
	if err != nil {
		r.logger.Error(
			"babylon not found in config",
			zap.Error(err),
		)
		// none of the chains can be relayed without Babylon
		return
	}

	// ensure that key in babylonChain chain exists
	if exists := babylonChain.ChainProvider.KeyExists(babylonChain.ChainProvider.Key()); !exists {
		r.logger.Error(
			"key not found on Babylon chain",
			zap.String("key", babylonChain.ChainProvider.Key()),
			zap.String("src_chain_id", babylonChain.ChainID()),
		)
		// non of the chains can be relayed without a keyring of Babylon
		return
	}

	r.logger.Info("Start relaying headers for the following chains", zap.Any("paths", r.cfg.Paths))

	// for each CZ (other than Babylon), start a KeepUpdatingClient go routine
	for chainName, chain := range r.cfg.Chains {
		if chainName == "babylon" {
			continue
		}

		// get CZ object from config
		czChain, err := r.cfg.Chains.Get(chain.ChainID())
		if err != nil {
			r.logger.Error(
				"CZ chain not found in config",
				zap.String("dst_chain_id", chain.ChainID()),
				zap.Error(err),
			)
			continue
		}

		// ensure the czChain light client exists, then start updating the czChain light client on babylonChain
		wg.Add(1)
		go func() {
			defer wg.Done()

			// keep updating the client
			if err := r.KeepUpdatingClient(ctx, babylonChain, czChain, interval, numRetries); err != nil {
				// NOTE: we don't panic here since the relayer should keep relaying other chains
				r.logger.Error(
					"failed to update CZ chain. Stop relaying the chain",
					zap.String("src_chain_id", babylonChain.ChainID()),
					zap.String("dst_chain_id", czChain.ChainID()),
					zap.Error(err),
				)
				r.metrics.FailedChainsCounter.WithLabelValues(babylonChain.ChainID(), czChain.ChainID()).Inc()
			}
		}()
	}
}
