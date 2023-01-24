package bbnrelayer

import (
	"context"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
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
	memo string,
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
	}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr); err != nil {
		return err
	}
	// in case block at srch/dsth has not been committed yet
	// see https://github.com/tendermint/tendermint/issues/7641
	srch--
	dsth--

	// check whether the dst light client exists on src at the latest height
	// if exists, return directly
	if len(src.PathEnd.ClientID) != 0 {
		_, err := src.ChainProvider.QueryClientState(ctx, srch, src.ClientID())
		if err == nil {
			r.logger.Info(
				"the light client already exists. Skip creating the light client.",
				zap.String("src_chain_id", src.ChainID()),
				zap.String("dst_chain_id", dst.ChainID()),
				zap.String("dst_client_id", src.PathEnd.ClientID),
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
	}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr); err != nil {
		return err
	}

	// automatically get TrustingPeriod, which has to be smaller than UnbondingPeriod
	dstUnbondingPeriod, err := dst.ChainProvider.QueryUnbondingPeriod(ctx)
	if err != nil {
		return fmt.Errorf("failed to get UnbondingPeriod of chain %s: %w", dst.Chainid, err)
	}
	// 85% of unbonding period
	// TODO: parameterise percentage
	dstTrustingPeriod := dstUnbondingPeriod / 100 * trustingPeriodPercentage

	// create the client on src chain, where we use default values for some fields
	r.Lock()
	clientID, err := relayer.CreateClient(
		ctx,
		src,
		dst,
		srcUpdateHeader,
		dstUpdateHeader,
		allowUpdateAfterExpiry,
		allowUpdateAfterMisbehaviour,
		override,
		dstTrustingPeriod,
		memo,
	)
	if err != nil {
		return err
	}
	r.Unlock()

	src.PathEnd.ClientID = clientID

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
		}, retry.Context(ctx), retry.Attempts(numRetries), relayer.RtyDel, relayer.RtyErr); err != nil {
			return err
		}
		// in case block at srch/dsth has not been committed yet
		// see https://github.com/tendermint/tendermint/issues/7641
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
