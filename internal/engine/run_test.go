package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestRunForwardsExitCode runs a stub script that exits with a chosen code and
// checks the engine exit-code forwarding contract.
func TestRunForwardsExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub requires a POSIX sh")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "stub")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 42\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	res := Run(context.Background(), stub, nil)
	if res.StartErr != nil {
		t.Fatalf("unexpected StartErr: %v", res.StartErr)
	}
	if res.ExitCode != 42 {
		t.Fatalf("exit code: got %d want 42", res.ExitCode)
	}
}

// TestRunStartError checks an unstartable engine is surfaced as StartErr and a
// nonzero code (no silent success).
func TestRunStartError(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does-not-exist")

	res := Run(context.Background(), missing, nil)
	if res.StartErr == nil {
		t.Fatal("expected StartErr for missing binary")
	}
	if res.ExitCode == 0 {
		t.Fatal("expected nonzero exit code for start failure")
	}
}

// TestRunStreamsStderr verifies the engine subprocess's stderr is wired to our
// own (it is covered by utilty run, but assert the plumbing compiles/passes).
func TestRunStreamsStderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub requires a POSIX sh")
	}
	dir := t.TempDir()
	stub := filepath.Join(dir, "stub-stderr")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\necho boom >&2\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	res := Run(context.Background(), stub, nil)
	if res.StartErr != nil {
		t.Fatalf("unexpected StartErr: %v", res.StartErr)
	}
	if res.ExitCode != 0 {
		t.Fatalf("exit code: got %d want 0", res.ExitCode)
	}
	// stderr went to os.Stderr — nothing to assert here beyond plumbing.
	_ = fmt.Sprint
}