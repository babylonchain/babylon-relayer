package cmd

import (
	"fmt"
	"strings"
	"sync"

	"github.com/babylonchain/babylon-relayer/bbnrelayer"
	"github.com/babylonchain/babylon-relayer/config"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/spf13/cobra"
)

// updateClientCmd is the command for updating a CZ light client in Babylon
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/tx.go#L232)
func updateClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-client babylon_chain_name cz_chain_name path_name",
		Short: "update IBC client on babylon_chain_name that tracks cz_chain_name with a configured path",
		Long: `Update IBC client on babylon_chain_name that tracks cz_chain_name with a configured path.
Clients are updated by querying headers from cz_chain_name and then sending the
corresponding update-client message to babylon_chain_name.`,
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s update-client babylon osmosis demo-path`, AppName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, babylonChain, czChain, err := getLoggerAndPathEnds(cmd, args)
			if err != nil {
				return err
			}

			// ensure that key in babylonChain chain exist
			if exists := babylonChain.ChainProvider.KeyExists(babylonChain.ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on babylonChain chain %s", babylonChain.ChainProvider.Key(), babylonChain.ChainID())
			}

			memo, err := cmd.Flags().GetString("memo")
			if err != nil {
				return err
			}
			numRetries, err := cmd.Flags().GetUint("retry")
			if err != nil {
				return err
			}

			relayer := bbnrelayer.New(logger)

			return relayer.UpdateClient(cmd.Context(), babylonChain, czChain, memo, numRetries)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")
	cmd.Flags().Uint("retry", relayer.RtyAttNum, "number of retry attempts for requests")

	return cmd
}

func keepUpdatingClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keep-update-client babylon_chain_name cz_chain_name path_name",
		Short: "keep updating IBC client on babylon_chain_name that tracks cz_chain_name with a configured path",
		Long: `Keep updating IBC client on babylon_chain_name that tracks cz_chain_name with a configured path.
Clients are updated by querying headers from cz_chain_name and then sending the
corresponding update-client message to babylon_chain_name.`,
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s keep-update-client babylon osmosis demo-path`, AppName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, babylonChain, czChain, err := getLoggerAndPathEnds(cmd, args)
			if err != nil {
				return err
			}

			// ensure that key in babylonChain chain exists
			if exists := babylonChain.ChainProvider.KeyExists(babylonChain.ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on babylonChain chain %s", babylonChain.ChainProvider.Key(), babylonChain.ChainID())
			}

			// retrieve necessary flags
			memo, err := cmd.Flags().GetString("memo")
			if err != nil {
				return err
			}
			interval, err := cmd.Flags().GetDuration("interval")
			if err != nil {
				return err
			}
			numRetries, err := cmd.Flags().GetUint("retry")
			if err != nil {
				return err
			}

			relayer := bbnrelayer.New(logger)

			return relayer.KeepUpdatingClient(cmd.Context(), babylonChain, czChain, memo, interval, numRetries)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")
	cmd.Flags().String("interval", "10m", "the interval between two update-client attempts")
	cmd.Flags().Uint("retry", relayer.RtyAttNum, "number of retry attempts for requests")

	return cmd
}

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
			memo, err := cmd.Flags().GetString("memo")
			if err != nil {
				return err
			}
			interval, err := cmd.Flags().GetDuration("interval")
			if err != nil {
				return err
			}
			numRetries, err := cmd.Flags().GetUint("retry")
			if err != nil {
				return err
			}

			// we want the program to exit only after all go routines have finished
			var wg sync.WaitGroup

			// start the relayer for all paths in cfg.Paths
			relayer := bbnrelayer.New(logger)
			relayer.KeepUpdatingClients(cmd.Context(), &wg, cfg.Paths, cfg.Chains, memo, interval, numRetries)

			// Note that this function is executed inside `root.go`'s `Execute()` function,
			// which keeps the program to be alive until being interrupted.
			// Here we just need to keep the main thread to be alive all the time.
			wg.Wait()

			return nil
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")
	cmd.Flags().String("interval", "10m", "the interval between two update-client attempts")
	cmd.Flags().Uint("retry", relayer.RtyAttNum, "number of retry attempts for requests")

	return cmd
}
