package bbnrelayer

import (
	"context"
	"sync"

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

// UpdateClient updates the IBC light client on src chain that tracks dst chain given the configured path
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/relayer/client.go#L17)
func (r *Relayer) UpdateClient(
	ctx context.Context,
	src *relayer.Chain,
	dst *relayer.Chain,
	memo string,
) error {
	r.Lock()
	defer r.Unlock()

	srch, dsth, err := relayer.QueryLatestHeights(ctx, src, dst)
	if err != nil {
		return err
	}

	var srcMsgUpdateClient provider.RelayerMessage
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		srcMsgUpdateClient, err = relayer.MsgUpdateClient(egCtx, dst, src, dsth, srch)
		return err
	})

	if err = eg.Wait(); err != nil {
		return err
	}

	clients := &relayer.RelayMsgs{
		Src: []provider.RelayerMessage{srcMsgUpdateClient},
		Dst: []provider.RelayerMessage{}, // nothing will be sent to dst chain
	}

	// Send msgs to src chain
	result := clients.Send(ctx, r.logger, relayer.AsRelayMsgSender(src), relayer.AsRelayMsgSender(dst), memo)
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
		"Clients updated",
		zap.String("src_chain_id", src.ChainID()),
		zap.String("src_client", src.PathEnd.ClientID),
		zap.String("dst_chain_id", dst.ChainID()),
		zap.String("dst_client", dst.PathEnd.ClientID),
	)

	return nil
}
