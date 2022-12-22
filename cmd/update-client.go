package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/babylonchain/babylon-relayer/bbnrelayer"
	"github.com/babylonchain/babylon-relayer/config"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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

			return bbnrelayer.UpdateClient(cmd.Context(), logger, babylonChain, czChain, memo)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")

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
			intervalStr, err := cmd.Flags().GetString("interval")
			if err != nil {
				return err
			}
			interval, err := time.ParseDuration(intervalStr)
			if err != nil {
				return err
			}

			return bbnrelayer.KeepUpdatingClient(cmd.Context(), logger, babylonChain, czChain, memo, interval)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")
	cmd.Flags().String("interval", "10m", "the interval between two update-client attempts")

	return cmd
}

// getLoggerAndPathEnds is a helper function that retrieves the logger, babylonChain and czChain
// from the given cmd and args
func getLoggerAndPathEnds(cmd *cobra.Command, args []string) (*zap.Logger, *relayer.Chain, *relayer.Chain, error) {
	// load config
	homePath, err := cmd.Flags().GetString("home")
	if err != nil {
		return nil, nil, nil, err
	}
	cfg, err := config.LoadConfig(homePath, cmd)
	if err != nil {
		return nil, nil, nil, err
	}

	// construct logger
	logFormat, err := cmd.Flags().GetString("log-format")
	if err != nil {
		return nil, nil, nil, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return nil, nil, nil, err
	}
	logger, err := config.NewRootLogger(logFormat, debug)
	if err != nil {
		return nil, nil, nil, err
	}

	// retrieve babylonChain chain, czChain chain, and path between babylonChain and czChain
	babylonName := args[0]
	babylonChain, ok := cfg.Chains[babylonName]
	if !ok {
		return nil, nil, nil, fmt.Errorf("babylonChain %s not found in config. consider running `%s chains add %s`", babylonName, AppName, babylonName)
	}
	czName := args[1]
	czChain, ok := cfg.Chains[czName]
	if !ok {
		return nil, nil, nil, fmt.Errorf("czChain %s not found in config. consider running `%s chains add %s`", czName, AppName, czName)
	}
	path, err := cfg.Paths.Get(args[2])
	if err != nil {
		return nil, nil, nil, err
	}
	babylonChain.PathEnd = path.End(babylonChain.ChainID())
	czChain.PathEnd = path.End(czChain.ChainID())

	return logger, babylonChain, czChain, nil
}
