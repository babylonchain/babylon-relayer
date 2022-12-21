package cmd

import (
	"fmt"
	"strings"

	"github.com/babylonchain/babylon-relayer/bbnrelayer"
	"github.com/babylonchain/babylon-relayer/config"
	"github.com/spf13/cobra"
)

func updateClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-client src_chain_name dst_chain_name path_name",
		Short: "update IBC client on src_chain_name that tracks dst_chain_name with a configured path",
		Long: `Update IBC client on src_chain_name that tracks dst_chain_name with a configured path.
Clients are updated by querying headers from dst_chain_name and then sending the
corresponding update-client message to src_chain_name.`,
		Args:    withUsage(cobra.ExactArgs(3)),
		Example: strings.TrimSpace(fmt.Sprintf(`$ %s transact update-client babylon osmosis demo-path`, AppName)),
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

			// ensure that keys exist
			// TODO: only need to ensure source key exist?
			if exists := src.ChainProvider.KeyExists(src.ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on src chain %s", src.ChainProvider.Key(), src.ChainID())
			}
			if exists := dst.ChainProvider.KeyExists(dst.ChainProvider.Key()); !exists {
				return fmt.Errorf("key %s not found on dst chain %s", dst.ChainProvider.Key(), dst.ChainID())
			}

			memo, err := cmd.Flags().GetString("memo")
			if err != nil {
				return err
			}

			return bbnrelayer.UpdateClient(cmd.Context(), src, dst, memo)
		},
	}

	cmd.Flags().String("memo", "", "a memo to include in relayed packets")

	return cmd
}
