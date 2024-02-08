package bbnrelayer

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"github.com/juju/fslock"
	"go.uber.org/zap"
)

const (
	// constants for creating a light client
	trustingPeriodPercentage     = 85 // TrustingPeriodPercentage * UnbondingPeriod = TrustingPeriod
	allowUpdateAfterExpiry       = true
	allowUpdateAfterMisbehaviour = true
	override                     = true
)

// createClientIfNotExist ensures that the dst light client exists on src chain
// if does not exist, the function will create a new dst light client on src chain
func (r *Relayer) createClientIfNotExist(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	numRetries uint,
) error {
	// query the latest heights on src and dst
	// retry here in case the CZ endpoint becomes unstable
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
	// in case block at srch/dsth has not been committed yet
	// see https://github.com/tendermint/tendermint/issues/7641
	srch--
	dsth--

	// check whether the dst light client exists on src at the latest height
	// if exists and queryable, return directly
	clientID, err := r.getClientID(dst.ChainID())
	if err != nil {
		return err
	}
	if len(clientID) > 0 {
		if _, err := src.ChainProvider.QueryClientState(ctx, srch, clientID); err == nil {
			r.logger.Info(
				"the light client already exists. Skip creating the light client.",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.String("dst_client_id", clientID),
			)
			return nil
		}
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
	}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
		r.logger.Info(
			"Failed to query update headers",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.Uint("attempt", n+1),
			zap.Uint("max_attempts", numRetries),
			zap.Error(err),
		)
	})); err != nil {
		return err
	}

	// `relayer.CreateClient` will access the PathEnd of src chain
	// since we don't require phase1 integration to set up paths,
	// we need to create empty PathEnd here to prevent nil pointer error
	if src.PathEnd == nil {
		src.PathEnd = &relayer.PathEnd{}
	}
	// create the client on src chain, where we use default values for some fields
	krErr := r.accessKeyWithLock(func() {
		clientID, err = relayer.CreateClient(
			ctx,
			src,
			dst,
			srcUpdateHeader,
			dstUpdateHeader,
			allowUpdateAfterExpiry,
			allowUpdateAfterMisbehaviour,
			override,
			0, // relayer will calculate the trusting period based on unbonding period if this is 0
			0, // relayer will query the unbonding period if this is 0
			r.cfg.Global.Memo,
		)
	})
	if krErr != nil {
		return krErr
	}
	if err != nil {
		return err
	}

	r.logger.Info(
		"successfully created the light client",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client_id", src.ClientID()),
	)

	// wait until client is queryable on chain
	if err := r.waitUntilQuerable(ctx, src, dst, numRetries); err != nil {
		return err
	}

	// the client is now created and queryable
	// writes the config with this client ID to DB
	if err := r.setClientID(dst.ChainID(), clientID); err != nil {
		return fmt.Errorf("error writing clientID %s for chain %s to DB: %w", clientID, dst.ChainID(), err)
	}

	r.logger.Info(
		"successfully inserted the light client ID to DB",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client_id", src.ClientID()),
	)

	return nil
}

// waitUntilQuerable asks the relayer to wait until the dst light client is queryable on src chain
func (r *Relayer) waitUntilQuerable(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	numRetries uint,
) error {
	ticker := time.NewTicker(time.Second * 5)

	for range ticker.C {
		// query the latest heights on src and dst
		// retry here in case the CZ endpoint becomes unstable
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
		// in case block at srch/dsth has not been committed yet
		// see https://github.com/tendermint/tendermint/issues/7641
		// TODO: remove below after bumping Tendermint to versions after https://github.com/tendermint/tendermint/pull/7642
		srch--
		dsth--

		if _, err := src.ChainProvider.QueryClientState(ctx, srch, src.ClientID()); err == nil {
			r.logger.Info(
				"the light client becomes committed on-chain, complete creating the light client",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.String("dst_client_id", src.ClientID()),
			)

			break
		}

		r.logger.Info(
			"the light client has not been committed on-chain yet, keep waiting",
			zap.String("src_chain_id", src.ChainID()),
			zap.String("dst_chain_id", dst.ChainID()),
			zap.String("dst_client_id", src.ClientID()),
		)
	}

	return nil
}

// accessKeyWithLock triggers a function that access key ring while acquiring
// the file system lock, in order to remain thread-safe when multiple concurrent
// relayers are running on the same machine and accessing the same keyring
func (r *Relayer) accessKeyWithLock(accessFunc func()) error {
	// use lock file to guard concurrent access to the keyring
	lockFilePath := path.Join(r.homePath, "keys", "keys.lock")
	lock := fslock.New(lockFilePath)
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire file system lock (%s): %w", lockFilePath, err)
	}

	// trigger function that access keyring
	accessFunc()

	// unlock and release access
	if err := lock.Unlock(); err != nil {
		return fmt.Errorf("error unlocking file system lock (%s), please manually delete", lockFilePath)
	}

	return nil
}
