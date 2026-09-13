// Package engine locates and executes the original sipexer binary.
//
// sipx never touches the SIP engine itself; it discovers the sipexer
// executable and runs it as a subprocess (see design.md D0/D6).
package engine

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// EnvEngine is the environment variable that overrides where sipexer is.
const EnvEngine = "SIPEXER_BIN"

// engineBinaryName is the fixed name of the original engine executable.
const engineBinaryName = "sipexer"

// Locate resolves the sipexer engine binary path using, in order of
// precedence:
//
//  1. explicit overrides: the non-empty candidates passed in (from a
//     --engine flag builder).
//  2. the SIPEXER_BIN environment variable.
//  3. a file named "sipexer" in the same directory as the sipx binary
//     (os.Executable()).
//  4. the PATH lookup via exec.LookPath.
//
// It returns an error describing which rules were tried when nothing is
// found, so the user knows what to set.
func Locate(engineFlag string) (string, error) {
	if engineFlag != "" {
		if st, err := os.Stat(engineFlag); err == nil && !st.IsDir() {
			return engineFlag, nil
		}
		return "", fmt.Errorf("engine not found at --engine path: %s", engineFlag)
	}

	if v := os.Getenv(EnvEngine); v != "" {
		return v, nil
	}

	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), engineBinaryName)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate, nil
		}
	}

	if p, err := exec.LookPath(engineBinaryName); err == nil {
		return p, nil
	}

	return "", errors.New("sipexer engine not found: tried SIPEXER_BIN env, same directory as sipx, and PATH")
}