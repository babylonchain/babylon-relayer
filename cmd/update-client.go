package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/babylonchain/babylon-relayer/bbnrelayer"
	"github.com/babylonchain/babylon-relayer/config"
	"github.com/spf13/cobra"
)

// updateClientCmd is the command for updating a client
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/tx.go#L232)
func updateClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-client src_chain_name dst_chain_name path_name",
		Short: "update IBC client on src_chain_name that tracks dst_chain_name with a configured path",
		Long: `Update IBC client on src_chain_name that tracks dst_chain_name with a configured path.
Clients are updated by querying headers from dst_chain_name and then sending the
corresponding update-client message to src_chain_name.`,
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s update-client babylon osmosis demo-path`, AppName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			homePath, err := cmd.Flags().GetString("home")
			if err != nil {
				panic(err)
			}
			cfg, err := config.LoadConfig(homePath, cmd)
			if err != nil {
				panic(err)
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

			memo, err := cmd.Flags().GetString("memo")
			if err != nil {
				return err
			}

			// retrieve src chain, dst chain, and path between src and dst
			srcName := args[0]
			src, ok := cfg.Chains[srcName]
			if !ok {
				return fmt.Errorf("chain %s not found in config. consider running `%s chains add %s`", srcName, AppName, srcName)
			}
			dstName := args[1]
			dst, ok := cfg.Chains[dstName]
			if !ok {
				return fmt.Errorf("chain %s not found in config. consider running `%s chains add %s`", dstName, AppName, dstName)
			}
			path, err := cfg.Paths.Get(args[2])
			if err != nil {
				return err
			}
			src.PathEnd = path.End(src.ChainID())
			dst.PathEnd = path.End(dst.ChainID())

			// ensure that key in src chain exist
			if exists := src.ChainProvider.KeyExists(src.ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on src chain %s", src.ChainProvider.Key(), src.ChainID())
			}

			return bbnrelayer.UpdateClient(cmd.Context(), logger, src, dst, memo)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")

	return cmd
}

func keepUpdatingClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keep-update-client src_chain_name dst_chain_name path_name",
		Short: "keep updating IBC client on src_chain_name that tracks dst_chain_name with a configured path",
		Long: `Keep updating IBC client on src_chain_name that tracks dst_chain_name with a configured path.
Clients are updated by querying headers from dst_chain_name and then sending the
corresponding update-client message to src_chain_name.`,
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s keep-update-client babylon osmosis demo-path`, AppName)),
		RunE: func(cmd *cobra.Command, args []string) error {
			// load config
			homePath, err := cmd.Flags().GetString("home")
			if err != nil {
				panic(err)
			}
			cfg, err := config.LoadConfig(homePath, cmd)
			if err != nil {
				panic(err)
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
			intervalStr, err := cmd.Flags().GetString("interval")
			if err != nil {
				return err
			}
			interval, err := time.ParseDuration(intervalStr)
			if err != nil {
				return err
			}

			// retrieve src chain, dst chain, and path between src and dst
			srcName := args[0]
			src, ok := cfg.Chains[srcName]
			if !ok {
				return fmt.Errorf("chain %s not found in config. consider running `%s chains add %s`", srcName, AppName, srcName)
			}
			dstName := args[1]
			dst, ok := cfg.Chains[dstName]
			if !ok {
				return fmt.Errorf("chain %s not found in config. consider running `%s chains add %s`", dstName, AppName, dstName)
			}
			path, err := cfg.Paths.Get(args[2])
			if err != nil {
				return err
			}
			src.PathEnd = path.End(src.ChainID())
			dst.PathEnd = path.End(dst.ChainID())

			// ensure that key in src chain exists
			if exists := src.ChainProvider.KeyExists(src.ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on src chain %s", src.ChainProvider.Key(), src.ChainID())
			}

			return bbnrelayer.KeepUpdatingClient(cmd.Context(), logger, src, dst, memo, interval)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")
	cmd.Flags().String("interval", "10m", "the interval between two update-client attempts")

	return cmd
}
