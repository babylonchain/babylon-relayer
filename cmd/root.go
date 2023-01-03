/*
Package cmd includes relayer commands
Copyright © 2020 Jack Zampolin jack.zampolin@gmail.com

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"

	relayercmd "github.com/cosmos/relayer/v2/cmd"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const AppName = "babylon-relayer"

// NewRootCmd returns the root command for relayer.
// If log is nil, a new zap.Logger is set on the app state
// based on the command line flags regarding logging.
func NewRootCmd(log *zap.Logger) *cobra.Command {
	// fetch the rootCmd of the official IBC relayer
	rootCmd := relayercmd.NewRootCmd(log)

	// override application metadata
	rootCmd.Use = AppName
	rootCmd.Short = "This application is used for relaying headers from Cosmos Zones to Babylon periodically."
	rootCmd.Long = strings.TrimSpace(fmt.Sprintf(`%s has:
	1. Configuration management for Chains and Paths
	2. Key management for managing multiple keys for multiple chains
	3. Query and transaction functionality for IBC
	4. Functionality for relaying headers from Cosmos Zones to Babylon periodically
 
	NOTE: Most of the commands have aliases that make typing them much quicker 
		  (i.e. '%s tx', '%s q', etc...)`, AppName, AppName, AppName))

	// add Babylon-specific commands
	rootCmd.AddCommand(
		lineBreakCommand(),
		updateClientCmd(),
		keepUpdatingClientCmd(),
		lineBreakCommand(),
	)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/root.go#L134)
func Execute() {
	cobra.EnableCommandSorting = false

	rootCmd := NewRootCmd(nil)
	rootCmd.SilenceUsage = true

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt) // Using signal.Notify, instead of signal.NotifyContext, in order to see details of signal.
	go func() {
		// Wait for interrupt signal.
		sig := <-sigCh

		// Cancel context on root command.
		// If the invoked command respects this quickly, the main goroutine will quit right away.
		cancel()

		// Short delay before printing the received signal message.
		// This should result in cleaner output from non-interactive commands that stop quickly.
		time.Sleep(250 * time.Millisecond)
		fmt.Fprintf(os.Stderr, "Received signal %v. Attempting clean shutdown. Send interrupt again to force hard shutdown.\n", sig)

		// Dump all goroutines on panic, not just the current one.
		debug.SetTraceback("all")

		// Block waiting for a second interrupt or a timeout.
		// The main goroutine ought to finish before either case is reached.
		// But if a case is reached, panic so that we get a non-zero exit and a dump of remaining goroutines.
		select {
		case <-time.After(time.Minute):
			panic(errors.New("rly did not shut down within one minute of interrupt"))
		case sig := <-sigCh:
			panic(fmt.Errorf("received signal %v; forcing quit", sig))
		}
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
