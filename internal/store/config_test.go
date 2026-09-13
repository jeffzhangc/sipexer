package store

import (
	"testing"
	"os"
	"path/filepath"
)

func TestConfigDirPriority(t *testing.T) {
	t.Setenv(EnvConfigDir, "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	// explicit wins
	got, err := ConfigDir("/explicit/dir")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/explicit/dir" {
		t.Fatalf("explicit: got %q", got)
	}

	// env next
	t.Setenv(EnvConfigDir, "/env/dir")
	got, err = ConfigDir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/env/dir" {
		t.Fatalf("env: got %q", got)
	}

	// XDG falls back to $HOME/.config/sipexer
	t.Setenv(EnvConfigDir, "")
	t.Setenv("XDG_CONFIG_HOME", "")
	got, err = ConfigDir("")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "sipexer")
	if got != want {
		t.Fatalf("xdg/default: got %q want %q", got, want)
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "data.json")

	if err := writeFile(path, []byte(`{"a":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"a":1}` {
		t.Fatalf("content: got %q", string(data))
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := st.Mode().Perm(); perm != 0o600 {
		t.Fatalf("perm: got %o want 0600", perm)
	}

	// overwrite works and leaves no temp litter
	if err := writeFile(path, []byte(`{"a":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if len(e.Name()) >= 4 && e.Name()[:5] == ".tmp-" {
			t.Fatalf("leftover temp file: %s", e.Name())
		}
	}
}