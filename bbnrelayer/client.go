package bbnrelayer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// Relayer is a relayer that allows to relay multiple chains concurrently.
// It is made thread-safe to avoid account sequence mismatch errors in Cosmos SDK accounts.
type Relayer struct {
	sync.Mutex
	logger *zap.Logger
}

func New(logger *zap.Logger) *Relayer {
	return &Relayer{
		logger: logger,
	}
}

func (r *Relayer) createClientIfNotExist(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	path *relayer.Path,
	memo string,
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

	// check whether the dst light client exists on src at the latest height
	// if err is nil, then the client exists, return directly
	if _, err := src.ChainProvider.QueryClientState(ctx, srch, dst.ClientID()); err == nil {
		r.logger.Info(
			"the light client already exists. Skip creating the light client.",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
		)
		return nil
	}

	// if the code reaches here, then it means the client does not exist
	// we need to create a new one
	r.logger.Info(
		"the light client does not exist. Creating a new light client.",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("dst_chain_id", dst.ChainID()),
	)

	// Query the light signed headers for src & dst at the heights srch & dsth
	var srcUpdateHeader, dstUpdateHeader provider.IBCHeader
	if err := retry.Do(func() error {
		var err error
		srcUpdateHeader, dstUpdateHeader, err = relayer.QueryIBCHeaders(ctx, src, dst, srch, dsth)
		if err != nil {
			return fmt.Errorf("failed to query update headers: %w", err)
		}
		return nil
	}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr); err != nil {
		return err
	}

	// create the client on src chain, where we use default values for some fields
	// TODO: allow custom TrustingPeriod
	r.Lock()
	if _, err := relayer.CreateClient(
		ctx,
		src,
		dst,
		srcUpdateHeader,
		dstUpdateHeader,
		true,  // allowUpdateAfterExpiry
		true,  // allowUpdateAfterMisbehaviour
		false, // override
		0,     // customClientTrustingPeriod
		memo,
	); err != nil {
		return err
	}
	r.Unlock()

	r.logger.Info(
		"successfully created the light client",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("dst_chain_id", dst.ChainID()),
	)

	return nil
}

// UpdateClient updates the IBC light client on src chain that tracks dst chain given the configured path
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/relayer/client.go#L17)
func (r *Relayer) UpdateClient(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	memo string,
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

	var srcMsgUpdateClient provider.RelayerMessage
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		srcMsgUpdateClient, err = relayer.MsgUpdateClient(egCtx, dst, src, dsth, srch)
		return err
	})

	if err := eg.Wait(); err != nil {
		return err
	}

	clients := &relayer.RelayMsgs{
		Src: []provider.RelayerMessage{srcMsgUpdateClient},
		Dst: []provider.RelayerMessage{}, // nothing will be sent to dst chain
	}

	// Send msgs to src chain in a thread-safe way
	r.Lock()
	result := clients.Send(ctx, r.logger, relayer.AsRelayMsgSender(src), relayer.AsRelayMsgSender(dst), memo)
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
	memo string,
	interval time.Duration,
	numRetries uint,
) error {
	ticker := time.NewTicker(interval)
	r.logger.Info(
		"Keep updating client",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("src_client", src.PathEnd.ClientID),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client", dst.PathEnd.ClientID),
		zap.Duration("interval", interval),
	)
	for ; true; <-ticker.C {
		// Note that UpdateClient is a thread-safe function
		if err := r.UpdateClient(ctx, src, dst, memo, numRetries); err != nil {
			return err
		}
	}
	return nil
}

func (r *Relayer) KeepUpdatingClients(
	ctx context.Context,
	wg *sync.WaitGroup,
	paths relayer.Paths,
	chains relayer.Chains,
	memo string,
	interval time.Duration,
	numRetries uint,
) {
	r.logger.Info("Start relaying headers for the following chains", zap.Any("paths", paths))

	// for each CZ, start a KeepUpdatingClient go routine
	for pathName, path := range paths {
		// get babylonChain object from config
		babylonChain, err := chains.Get(path.Src.ChainID)
		if err != nil {
			r.logger.Error(
				"babylon not found in config",
				zap.String("path", pathName),
				zap.String("chain_id", path.Src.ChainID),
				zap.Error(err),
			)
			continue
		}
		// ensure that key in babylonChain chain exists
		if exists := babylonChain.ChainProvider.KeyExists(babylonChain.ChainProvider.Key()); !exists {
			r.logger.Error(
				"key not found on Babylon chain, skipping this path",
				zap.String("path", pathName),
				zap.String("key", babylonChain.ChainProvider.Key()),
				zap.String("chain_id", babylonChain.ChainID()),
			)
			continue
		}

		// get CZ object from config
		czChain, err := chains.Get(path.Dst.ChainID)
		if err != nil {
			r.logger.Error(
				"CZ chain not found in config",
				zap.String("path", pathName),
				zap.String("chain_id", path.Dst.ChainID),
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

		// ensure the CZ chain light client exists on Babylon
		if err := r.createClientIfNotExist(ctx, &copiedBabylonChain, &copiedCZChain, path, memo, numRetries); err != nil {
			r.logger.Error(
				"failed to ensure CZ light client exsits on Babylon",
				zap.String("chain_id", copiedCZChain.ChainID()),
				zap.Error(err),
			)
			continue
		}

		// start updating the czChain light client on babylonChain
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := r.KeepUpdatingClient(ctx, &copiedBabylonChain, &copiedCZChain, memo, interval, numRetries); err != nil {
				// NOTE: we don't panic here since the relayer should keep relaying other chains
				r.logger.Error(
					"failed to update CZ chain",
					zap.String("chain_id", copiedCZChain.ChainID()),
					zap.Error(err),
				)
			}
		}()
	}
}
