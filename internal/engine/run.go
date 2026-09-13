package engine

import (
	"context"
	"errors"
	"os"
	"os/exec"
)

// Result holds the outcome of a subprocess run.
type Result struct {
	// ExitCode is the process exit code. It is non-zero when the engine could
	// not be started at all, so callers can rely on it for forwarding.
	ExitCode int
	// StartErr is non-nil only when the engine binary could not be started
	// (missing binary, bad permissions).
	StartErr error
}

// Run starts the engine binary with args, wiring its stdout and stderr to the
// current process's stdout and stderr, and waits for it to finish. It returns
// a Result carrying the engine's exit code so sipx can propagate it verbatim
// (design.md D0/D7).
func Run(ctx context.Context, bin string, args []string) Result {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return Result{ExitCode: 1, StartErr: err}
	}

	err := cmd.Wait()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return Result{ExitCode: exitErr.ExitCode()}
	}
	if err != nil {
		return Result{ExitCode: 1}
	}
	return Result{ExitCode: 0}
}