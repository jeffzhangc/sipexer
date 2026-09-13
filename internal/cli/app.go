package cli

import (
	"fmt"

	"github.com/miconda/sipexer/internal/store"
)

// app bundles the two persistent stores plus the resolved config directory
// for a single sipx invocation.
type app struct {
	configDir string
	profiles  *store.Profiles
	history   *store.History
}

// loadApp builds the app from the current flags/env, loading both stores.
// history uses the default cap.
func loadApp() (*app, error) {
	dir, err := store.ConfigDir(configDirFlag)
	if err != nil {
		return nil, fmt.Errorf("config dir: %w", err)
	}
	profiles, err := store.LoadProfiles(dir)
	if err != nil {
		return nil, err
	}
	history, err := store.LoadHistory(dir, 0)
	if err != nil {
		return nil, err
	}
	return &app{configDir: dir, profiles: profiles, history: history}, nil
}

// configDirFlag is the --config-dir override (root persistent flag).
var configDirFlag string