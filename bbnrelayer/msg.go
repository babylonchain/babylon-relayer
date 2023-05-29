package bbnrelayer

import (
	"context"

	"github.com/avast/retry-go/v4"
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// CreateMsgUpdateClient queries for the current client state on receiver,
// then queries for the latest and trusted headers on sender
// in order to build a MsgUpdateClient message that carries a `sender`
// header and is sent to the `sender` IBC client with the given clientID
// in the `receiver` chain.
// same as https://github.com/cosmos/relayer/blob/v2.3.1/relayer/client.go
// except for using the client ID from DB
func (r *Relayer) CreateMsgUpdateClient(
	ctx context.Context,
	sender, receiver *relayer.Chain,
	senderHeight, receiverHeight int64,
	clientID string,
) (provider.RelayerMessage, error) {
	var dstClientState ibcexported.ClientState
	if err := retry.Do(func() error {
		var err error
		dstClientState, err = receiver.ChainProvider.QueryClientState(ctx, receiverHeight, clientID)
		return err
	}, retry.Context(ctx), relayer.RtyAtt, relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
		r.logger.Info(
			"Failed to query client state when updating clients",
			zap.String("client_id", clientID),
			zap.Uint("attempt", n+1),
			zap.Uint("max_attempts", relayer.RtyAttNum),
			zap.Error(err),
		)
	})); err != nil {
		return nil, err
	}

	var srcHeader, dstTrustedHeader provider.IBCHeader

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return retry.Do(func() error {
			var err error
			srcHeader, err = sender.ChainProvider.QueryIBCHeader(egCtx, senderHeight)
			return err
		}, retry.Context(egCtx), relayer.RtyAtt, relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
			r.logger.Info(
				"Failed to query IBC header when building update client message",
				zap.String("client_id", clientID),
				zap.Uint("attempt", n+1),
				zap.Uint("max_attempts", relayer.RtyAttNum),
				zap.Error(err),
			)
		}))
	})
	eg.Go(func() error {
		return retry.Do(func() error {
			var err error
			dstTrustedHeader, err = sender.ChainProvider.QueryIBCHeader(egCtx, int64(dstClientState.GetLatestHeight().GetRevisionHeight())+1)
			return err
		}, retry.Context(egCtx), relayer.RtyAtt, relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
			r.logger.Info(
				"Failed to query IBC header when building update client message",
				zap.String("client_id", clientID),
				zap.Uint("attempt", n+1),
				zap.Uint("max_attempts", relayer.RtyAttNum),
				zap.Error(err),
			)
		}))
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	var updateHeader ibcexported.ClientMessage
	if err := retry.Do(func() error {
		var err error
		updateHeader, err = sender.ChainProvider.MsgUpdateClientHeader(srcHeader, dstClientState.GetLatestHeight().(clienttypes.Height), dstTrustedHeader)
		return err
	}, retry.Context(ctx), relayer.RtyAtt, relayer.RtyDel, relayer.RtyErr, retry.OnRetry(func(n uint, err error) {
		r.logger.Info(
			"Failed to build update client header",
			zap.String("client_id", clientID),
			zap.Uint("attempt", n+1),
			zap.Uint("max_attempts", relayer.RtyAttNum),
			zap.Error(err),
		)
	})); err != nil {
		return nil, err
	}

	// updates off-chain light client
	return receiver.ChainProvider.MsgUpdateClient(clientID, updateHeader)
}
