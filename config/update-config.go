package config

import (
	"fmt"
	"os"
	"path"
	"time"

	relayercmd "github.com/cosmos/relayer/v2/cmd"
	"github.com/juju/fslock"
	"gopkg.in/yaml.v3"
)

// OverwriteConfig writes the given cfg back to the config gile w.r.t. the homePath
// the function is thread-safe
func OverwriteConfig(cfg *relayercmd.Config, homePath string) error {
	cfgPath := path.Join(homePath, "config", "config.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		return fmt.Errorf("failed to check existence of config file at %s: %w", cfgPath, err)
	}

	// use lock file to guard concurrent access to config.yaml
	lockFilePath := path.Join(homePath, "config", "config.lock")
	lock := fslock.New(lockFilePath)
	err := lock.LockWithTimeout(10 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to acquire config lock: %w", err)
	}

	// marshal the new config
	out, err := yaml.Marshal(cfg.Wrapped())
	if err != nil {
		return err
	}

	// Overwrite the config file
	if err := os.WriteFile(cfgPath, out, 0600); err != nil {
		return fmt.Errorf("failed to write config file at %s: %w", cfgPath, err)
	}

	// unlock and release access
	if err := lock.Unlock(); err != nil {
		return fmt.Errorf("error unlocking config file lock (%s), please manually delete", lockFilePath)
	}

	return nil
}
