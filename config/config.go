package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"

	relayercmd "github.com/cosmos/relayer/v2/cmd"
	relayer "github.com/cosmos/relayer/v2/relayer"
	"github.com/cosmos/relayer/v2/relayer/provider"
	zaplogfmt "github.com/jsternberg/zap-logfmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// CONTRACT: this has to be consistent in the official relayer
// so that all commands inherited from the official relayer can find the config file
var DefaultHome = filepath.Join(os.Getenv("HOME"), ".relayer")

// LoadConfig loads the config file in the given path to a config struct
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/config.go#L544)
func LoadConfig(homePath string, cmd *cobra.Command) (*relayercmd.Config, error) {
	// get config path from home path
	cfgPath := path.Join(homePath, "config", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		return nil, fmt.Errorf("path %s does not point to a config file: %v", cfgPath, err)
	}

	// read the config file bytes
	file, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %v", cfgPath, err)
	}

	// unmarshall config file into the wrapper struct
	cfgWrapper := &relayercmd.ConfigInputWrapper{}
	err = yaml.Unmarshal(file, cfgWrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config file at %s: %v", cfgPath, err)
	}

	// verify that the channel filter rule is valid for every path in the config
	for _, p := range cfgWrapper.Paths {
		if err := p.ValidateChannelFilterRule(); err != nil {
			return nil, fmt.Errorf("error initializing the relayer config for path %s: %w", p.String(), err)
		}
	}

	// build the logger struct
	logFormat, err := cmd.Flags().GetString("log-format")
	if err != nil {
		return nil, err
	}
	debug, err := cmd.Flags().GetBool("debug")
	if err != nil {
		return nil, err
	}
	logger, err := NewRootLogger(logFormat, debug)
	if err != nil {
		return nil, err
	}

	// build configs for chains
	chains := make(relayer.Chains)
	for chainName, pcfg := range cfgWrapper.ProviderConfigs {
		prov, err := pcfg.Value.(provider.ProviderConfig).NewProvider(
			logger.With(zap.String("provider_type", pcfg.Type)),
			homePath,
			debug,
			chainName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build ChainProviders: %w", err)
		}

		chain := relayer.NewChain(logger, prov, debug)
		chains[chainName] = chain
	}

	// build the config struct
	config := &relayercmd.Config{
		Global: cfgWrapper.Global,
		Chains: chains,
		Paths:  cfgWrapper.Paths,
	}

	return config, nil
}

// NewRootLogger returns a new Zap logger
// (adapted from https://github.com/cosmos/relayer/blob/v2.1.2/cmd/root.go#L177)
func NewRootLogger(format string, debug bool) (*zap.Logger, error) {
	config := zap.NewProductionEncoderConfig()
	config.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format("2006-01-02T15:04:05.000000Z07:00"))
	}
	config.LevelKey = "lvl"

	var enc zapcore.Encoder
	switch format {
	case "json":
		enc = zapcore.NewJSONEncoder(config)
	case "console":
		enc = zapcore.NewConsoleEncoder(config)
	case "logfmt":
		enc = zaplogfmt.NewEncoder(config)
	case "auto":
		if term.IsTerminal(int(os.Stderr.Fd())) {
			// When a user runs relayer in the foreground, use easier to read output.
			enc = zapcore.NewConsoleEncoder(config)
		} else {
			// Otherwise, use consistent logfmt format for simplistic machine processing.
			enc = zaplogfmt.NewEncoder(config)
		}
	default:
		return nil, fmt.Errorf("unrecognized log format %q", format)
	}

	level := zap.InfoLevel
	if debug {
		level = zap.DebugLevel
	}
	return zap.New(zapcore.NewCore(
		enc,
		os.Stderr,
		level,
	)), nil
}
