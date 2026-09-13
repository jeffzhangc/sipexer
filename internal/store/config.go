// Package store provides persistent JSON storage for sipx phone profiles
// and dial history, plus config-directory resolution and atomic file writes.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// EnvConfigDir overrides the configuration directory.
const EnvConfigDir = "SIPEXER_CONFIG_DIR"

// DefaultHistoryCap is the default maximum number of history entries kept.
const DefaultHistoryCap = 100

// ConfigDir resolves the directory holding profiles.json and history.json.
// Precedence (design.md D4):
//
//  1. explicit --config-dir (pass as explicitDir)
//  2. $SIPEXER_CONFIG_DIR
//  3. $XDG_CONFIG_HOME/sipexer (Linux/macOS) or %APPDATA%\sipexer (Windows)
//  4. ~/.config/sipexer
func ConfigDir(explicitDir string) (string, error) {
	if explicitDir != "" {
		return explicitDir, nil
	}

	if v := os.Getenv(EnvConfigDir); v != "" {
		return v, nil
	}

	var base string
	if runtime.GOOS == "windows" {
		base = os.Getenv("APPDATA")
		if base == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("resolve config dir: %w", err)
			}
			base = home
		}
		return filepath.Join(base, "sipexer"), nil
	}

	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "sipexer"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(home, ".config", "sipexer"), nil
}

// writeFile atomic writes data to path: it writes to a temp file in the same
// directory and renames it into place. On Windows, rename over an existing
// destination can fail, in which case we fall back to remove+rename
// (design.md D4).
func writeFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op after successful rename

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, perm); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		if runtime.GOOS == "windows" {
			if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
				return fmt.Errorf("remove existing file (windows fallback): %w", rmErr)
			}
			if err := os.Rename(tmpName, path); err != nil {
				return fmt.Errorf("rename temp file (windows fallback): %w", err)
			}
			return nil
		}
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}