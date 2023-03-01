package cmd

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/babylonchain/babylon-relayer/bbnrelayer"
	"github.com/babylonchain/babylon-relayer/config"
	relaydebug "github.com/babylonchain/babylon-relayer/debug"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func keepUpdatingClientsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "keep-update-clients",
		Short:   "keep updating IBC client of a list of chains specified in config on Babylon",
		Long:    `Keep updating IBC client of a list of chains specified in config on Babylon.`,
		Args:    withUsage(cobra.ExactArgs(0)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s keep-update-clients`, AppName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			homePath, err := cmd.Flags().GetString("home")
			if err != nil {
				return err
			}
			cfg, err := config.LoadConfig(homePath, cmd)
			if err != nil {
				return err
			}

			// construct logger
			logFormat, err := cmd.Flags().GetString("log-format")
			if err != nil {
				return err
			}
			debug, err := cmd.Flags().GetBool("debug")
			if err != nil {
				return err
			}
			logger, err := config.NewRootLogger(logFormat, debug)
			if err != nil {
				return err
			}

			// retrieve necessary flags
			interval, err := cmd.Flags().GetDuration("interval")
			if err != nil {
				return err
			}
			// override retry in relayer config
			numRetries, err := cmd.Flags().GetUint("retry")
			if err != nil {
				return err
			}
			relayer.RtyAttNum = numRetries
			relayer.RtyAtt = retry.Attempts(relayer.RtyAttNum)
			relayer.RtyDel = retry.Delay(time.Second)

			// initialise prometheus registry
			metrics := relaydebug.NewPrometheusMetrics()

			// start debug server with prometheus metrics
			debugAddr, err := cmd.Flags().GetString("debug-addr")
			if err != nil {
				return err
			}
			ln, err := net.Listen("tcp", debugAddr)
			if err != nil {
				logger.Error("Failed to listen on debug address. If you have another relayer process open, use --debug-addr to pick a different address.")
				return fmt.Errorf("failed to listen on debug address %q: %w", debugAddr, err)
			}
			debugServerLogger := logger.With(zap.String("sys", "debughttp"))
			debugServerLogger.Info("Debug server listening", zap.String("addr", debugAddr))
			relaydebug.StartDebugServer(cmd.Context(), debugServerLogger, ln, metrics)

			// we want the program to exit only after all go routines have finished
			var wg sync.WaitGroup

			// start the relayer for all paths in cfg.Paths
			relayer := bbnrelayer.New(homePath, cfg, logger, metrics)
			relayer.KeepUpdatingClients(cmd.Context(), &wg, interval, numRetries)

			// Note that this function is executed inside `root.go`'s `Execute()` function,
			// which keeps the program to be alive until being interrupted.
			// Here we just need to keep the main thread to be alive all the time.
			wg.Wait()

			return nil
		},
	}

	cmd.Flags().Duration("interval", time.Minute*10, "the interval between two update-client attempts")
	cmd.Flags().Uint("retry", 5, "number of retry attempts for requests")
	cmd.Flags().String("debug-addr", "", "address for the debug server with Prometheus metrics")

	return cmd
}
