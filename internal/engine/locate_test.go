package engine

import (
	"testing"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// TestLocatePriority checks the precedence: engineFlag > env > same-dir > PATH.
// We exercise flag and env rules directly, and the same-dir rule by placing a
// bogus executable next to a simulated sipx argv[0] (via os.Executable is not
// overridable, so we test the PATH fallback through a PATH-scoped file).
func TestLocateFlagAndEnv(t *testing.T) {
	dir := t.TempDir()
	bogus := filepath.Join(dir, "bogus-engine")
	if err := os.WriteFile(bogus, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(EnvEngine, "")

	// 1st precedence: explicit flag beats env.
	t.Setenv(EnvEngine, "its-not-this")
	got, err := Locate(bogus)
	if err != nil {
		t.Fatalf("Locate(flag) failed: %v", err)
	}
	if got != bogus {
		t.Fatalf("flag rule: got %q want %q", got, bogus)
	}

	// env rule when no flag given.
	got, err = Locate("")
	if err != nil {
		t.Fatalf("Locate(env) failed: %v", err)
	}
	if got != "its-not-this" {
		t.Fatalf("env rule: got %q want %q", got, "its-not-this")
	}

	// missing engine error lists the tried rules.
	if _, err := Locate("/definitely/missing"); err == nil {
		t.Fatal("expected error for missing --engine path")
	} else if !strings.Contains(err.Error(), "--engine") {
		t.Fatalf("unhelpful error: %v", err)
	}
}

func TestLocateSameDirOrPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("exec.LookPath extension handling on Windows differs")
	}
	t.Setenv(EnvEngine, "")
	dir := t.TempDir()
	// A fake engine named sipexer placed on PATH (dir) — makes LookPath succeed.
	fake := filepath.Join(dir, "sipexer")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	got, err := Locate("")
	if err != nil {
		t.Fatalf("Locate(PATH) failed: %v", err)
	}
	if got != fake {
		t.Fatalf("PATH rule: got %q want %q", got, fake)
	}
}