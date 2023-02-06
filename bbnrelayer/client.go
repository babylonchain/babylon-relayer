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
	sync.Mutex
	cfg     *relayercmd.Config
	logger  *zap.Logger
	metrics *relaydebug.PrometheusMetrics
}

func New(cfg *relayercmd.Config, logger *zap.Logger, metrics *relaydebug.PrometheusMetrics) *Relayer {
	return &Relayer{
		cfg:     cfg,
		logger:  logger,
		metrics: metrics,
	}
}

// UpdateClient updates the IBC light client on src chain that tracks dst chain given the configured path
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/relayer/client.go#L17)
func (r *Relayer) UpdateClient(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	numRetries uint,
) error {
	// query the latest heights on src and dst
	var srch, dsth int64
	if err := retry.Do(func() error {
		var err error
		srch, dsth, err = relayer.QueryLatestHeights(ctx, src, dst)
		if err != nil {
			return fmt.Errorf("failed to query latest heights: %w", err)
		}
		return nil
	}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr); err != nil {
		return err
	}

	// generate MsgUpdateClient that carries dst header to src
	srcMsgUpdateClient, err := relayer.MsgUpdateClient(ctx, dst, src, dsth, srch)
	if err != nil {
		return err
	}

	clients := &relayer.RelayMsgs{
		Src: []provider.RelayerMessage{srcMsgUpdateClient},
		Dst: []provider.RelayerMessage{}, // nothing will be sent to dst chain
	}

	// Send msgs to src chain in a thread-safe way
	r.Lock()
	result := clients.Send(ctx, r.logger, relayer.AsRelayMsgSender(src), relayer.AsRelayMsgSender(dst), r.cfg.Global.Memo)
	r.Unlock()
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
		zap.String("src_client", src.PathEnd.ClientID),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client", dst.PathEnd.ClientID),
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
	r.logger.Info(
		"Keep updating client",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("src_client", src.PathEnd.ClientID),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client", dst.PathEnd.ClientID),
		zap.Duration("interval", interval),
	)
	r.metrics.RelayedChainsCounter.WithLabelValues(src.ChainID(), dst.ChainID()).Inc()

	ticker := time.NewTicker(interval)
	for ; true; <-ticker.C {
		r.metrics.RelayedHeadersCounter.WithLabelValues(src.ChainID(), dst.ChainID()).Inc()

		// Note that UpdateClient is a thread-safe function
		if err := r.UpdateClient(ctx, src, dst, numRetries); err != nil {
			r.logger.Error(
				"Failed to update client",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("src_client", src.PathEnd.ClientID),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.String("dst_client", dst.PathEnd.ClientID),
				zap.Error(err),
			)
			r.metrics.FailedHeadersCounter.WithLabelValues(src.ChainID(), dst.ChainID()).Inc()

			// NOTE: the for loop continues here since it's possible that
			// the endpoint of dst chain is temporarily unavailable
			// TODO: distinguish unrecoverable errors
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
	r.logger.Info("Start relaying headers for the following chains", zap.Any("paths", r.cfg.Paths))

	// for each CZ, start a KeepUpdatingClient go routine
	for pathName, path := range r.cfg.Paths {
		// get babylonChain object from config
		babylonChain, err := r.cfg.Chains.Get(path.Src.ChainID)
		if err != nil {
			r.logger.Error(
				"babylon not found in config",
				zap.String("path", pathName),
				zap.String("src_chain_id", path.Src.ChainID),
				zap.Error(err),
			)
			// non of the chains can be relayed without Babylon
			return
		}
		// ensure that key in babylonChain chain exists
		if exists := babylonChain.ChainProvider.KeyExists(babylonChain.ChainProvider.Key()); !exists {
			r.logger.Error(
				"key not found on Babylon chain",
				zap.String("path", pathName),
				zap.String("key", babylonChain.ChainProvider.Key()),
				zap.String("src_chain_id", babylonChain.ChainID()),
			)
			// non of the chains can be relayed without a keyring of Babylon
			return
		}

		// get CZ object from config
		czChain, err := r.cfg.Chains.Get(path.Dst.ChainID)
		if err != nil {
			r.logger.Error(
				"CZ chain not found in config",
				zap.String("path", pathName),
				zap.String("dst_chain_id", path.Dst.ChainID),
				zap.Error(err),
			)
			continue
		}

		// copy the objects of two chains to prevent them from sharing the same PathEnd
		copiedBabylonChain := *babylonChain
		copiedCZChain := *czChain
		// set path end for two chains
		copiedBabylonChain.PathEnd = path.End(babylonChain.ChainID())
		copiedCZChain.PathEnd = path.End(czChain.ChainID())

		// ensure the czChain light client exists, then start updating the czChain light client on babylonChain
		wg.Add(1)
		go func() {
			defer wg.Done()
			// ensure the CZ chain light client exists on Babylon
			if err := r.createClientIfNotExist(ctx, &copiedBabylonChain, &copiedCZChain, numRetries); err != nil {
				r.logger.Error(
					"failed to ensure CZ light client exists on Babylon. Stop relaying the chain",
					zap.String("src_client_id", copiedBabylonChain.PathEnd.ClientID),
					zap.String("src_chain_id", copiedBabylonChain.ChainID()),
					zap.String("dst_client_id", copiedCZChain.PathEnd.ClientID),
					zap.String("dst_chain_id", copiedCZChain.ChainID()),
					zap.Error(err),
				)
				return
			}
			// keep updating the client
			if err := r.KeepUpdatingClient(ctx, &copiedBabylonChain, &copiedCZChain, interval, numRetries); err != nil {
				// NOTE: we don't panic here since the relayer should keep relaying other chains
				r.logger.Error(
					"failed to update CZ chain. Stop relaying the chain",
					zap.String("src_client_id", copiedBabylonChain.PathEnd.ClientID),
					zap.String("src_chain_id", copiedBabylonChain.ChainID()),
					zap.String("dst_client_id", copiedCZChain.PathEnd.ClientID),
					zap.String("dst_chain_id", copiedCZChain.ChainID()),
					zap.Error(err),
				)
				r.metrics.FailedChainsCounter.WithLabelValues(copiedBabylonChain.ChainID(), copiedCZChain.ChainID()).Inc()
			}
		}()
	}
}
