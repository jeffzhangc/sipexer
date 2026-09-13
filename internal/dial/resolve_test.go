package dial

import (
	"testing"

	"github.com/miconda/sipexer/internal/store"
)

func TestResolveTargetRaw(t *testing.T) {
	p := store.Profile{Name: "work"}
	hist, _ := store.LoadHistory(t.TempDir(), 5)
	r, err := ResolveTarget(p, hist, "2002")
	if err != nil {
		t.Fatal(err)
	}
	_ = r
	if r == nil || r.To != "2002" || r.Profile != "work" {
		t.Fatalf("raw dial: %+v", r)
	}
}

func TestResolveTargetNumberAlias(t *testing.T) {
	p := store.Profile{
		Name:    "work",
		Numbers: []store.Number{{User: "1001", Domain: "example.com", Alias: "desk"}},
	}
	hist, _ := store.LoadHistory(t.TempDir(), 5)
	r, err := ResolveTarget(p, hist, "desk")
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || r.To != "1001@example.com" || r.Profile != "work" {
		t.Fatalf("alias dial: %+v", r)
	}
}

func TestResolveTargetHistoryAliasAndIndex(t *testing.T) {
	p := store.Profile{Name: "work"}
	hist, _ := store.LoadHistory(t.TempDir(), 5)
	_ = hist.Append(store.Entry{Target: "555111", Profile: "work", Alias: "old"})
	_ = hist.Append(store.Entry{Target: "555222", Profile: "work"})
	_ = hist.Append(store.Entry{Target: "555333", Profile: "work"})

	// history alias
	r, err := ResolveTarget(p, hist, "old")
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || r.To != "555111" {
		t.Fatalf("history alias dial: %+v", r)
	}
	if r.Profile != "work" {
		t.Fatalf("history alias should stay in selected profile, got %q", r.Profile)
	}

	// index @0 = newest (555333)
	r, err = ResolveTarget(p, hist, "@0")
	if err != nil {
		t.Fatal(err)
	}
	if r == nil || r.To != "555333" {
		t.Fatalf("@0 dial: %+v", r)
	}

	// out of range
	if _, err := ResolveTarget(p, hist, "@99"); err == nil {
		t.Fatal("expected out-of-range error")
	}
}

func TestResolveTargetEmptyNeedsPicker(t *testing.T) {
	p := store.Profile{Name: "work"}
	hist, _ := store.LoadHistory(t.TempDir(), 5)
	r, err := ResolveTarget(p, hist, "")
	if err != nil {
		t.Fatal(err)
	}
	if r != nil {
		t.Fatalf("empty target should be nil (picker): %+v", r)
	}
}