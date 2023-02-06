package cmd

import (
	"fmt"

	"github.com/babylonchain/babylon-relayer/config"
	relayercmd "github.com/cosmos/relayer/v2/cmd"
	"github.com/cosmos/relayer/v2/relayer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// withUsage wraps a PositionalArgs to display usage only when the PositionalArgs
// variant is violated.
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/root.go#L229)
func withUsage(inner cobra.PositionalArgs) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := inner(cmd, args); err != nil {
			cmd.Root().SilenceUsage = false
			cmd.SilenceUsage = false
			return err
		}

		return nil
	}
}

// lineBreakCommand returns a new instance of the lineBreakCommand every time to avoid
// data races in concurrent tests exercising commands.
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/root.go#L223)
func lineBreakCommand() *cobra.Command {
	return &cobra.Command{Run: func(*cobra.Command, []string) {}}
}

// getLoggerAndPathEnds is a helper function that retrieves the logger, babylonChain and czChain
// from the given cmd and args
func getLoggerAndPathEnds(cmd *cobra.Command, cfg *relayercmd.Config, args []string) (*zap.Logger, *relayer.Chain, *relayer.Chain, error) {
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
