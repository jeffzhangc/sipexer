package store

import (
	"testing"
)

func TestHistoryAppendCapAndClear(t *testing.T) {
	dir := t.TempDir()

	// Cap of 3.
	h, err := LoadHistory(dir, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := h.Append(Entry{Target: "100" + string(rune('0'+i)), Profile: "work"}); err != nil {
			t.Fatal(err)
		}
	}
	if h.Len() != 3 {
		t.Fatalf("Len = %d want 3", h.Len())
	}
	// Newest first: after appending 1000..1004 with cap 3, kept are 1002,1003,1004.
	es := h.Entries()
	if es[0].Target != "1004" || es[2].Target != "1002" {
		t.Fatalf("recent-first order wrong: %+v", es)
	}

	// Persistence across reload.
	h2, err := LoadHistory(dir, 3)
	if err != nil {
		t.Fatal(err)
	}
	if h2.Len() != 3 {
		t.Fatalf("reload Len = %d want 3", h2.Len())
	}

	// Entry-by-index and SetAlias (most-recent-first).
	if e, ok := h2.Entry(0); !ok || e.Target != "1004" {
		t.Fatalf("Entry(0) = %+v ok=%v", e, ok)
	}
	if _, ok := h2.Entry(99); ok {
		t.Fatal("Entry(99) should be out of range")
	}
	if err := h2.SetAlias(1, "boss"); err != nil {
		t.Fatal(err)
	}
	if !h2.AliasResolves("boss") {
		t.Fatal("alias boss should resolve")
	}
	if e, ok := h2.Entry(1); !ok || e.Alias != "boss" {
		t.Fatalf("Entry(1) alias = %q", e.Alias)
	}

	// Clear.
	if err := h2.Clear(); err != nil {
		t.Fatal(err)
	}
	if h2.Len() != 0 {
		t.Fatalf("Len after clear = %d", h2.Len())
	}
}

func TestHistoryDefaultCap(t *testing.T) {
	dir := t.TempDir()
	h, err := LoadHistory(dir, 0) // default
	if err != nil {
		t.Fatal(err)
	}
	if h.cap != DefaultHistoryCap {
		t.Fatalf("cap = %d want %d", h.cap, DefaultHistoryCap)
	}
}