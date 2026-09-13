package store

import (
	"testing"
)

func TestHistoryDedupSameTarget(t *testing.T) {
	dir := t.TempDir()
	h, _ := LoadHistory(dir, 10)

	_ = h.Append(Entry{Target: "1001", Profile: "work", Server: "udp://a:5060", FromUser: "100"})
	_ = h.Append(Entry{Target: "1002", Profile: "work", Server: "udp://a:5060", FromUser: "100"})
	// Re-dial 1001: should update + move to newest, not duplicate.
	_ = h.Append(Entry{Target: "1001", Profile: "work", Server: "tcp://b:5060", FromUser: "100"})

	if h.Len() != 2 {
		t.Fatalf("Len = %d want 2 (dedup failed)", h.Len())
	}
	es := h.Entries()
	if es[0].Target != "1001" || es[0].Server != "tcp://b:5060" {
		t.Fatalf("re-dialed entry should be newest with updated server: %+v", es[0])
	}
	if es[1].Target != "1002" {
		t.Fatalf("order wrong: %+v", es[1])
	}

	// Alias survives the dedup update.
	_ = h.SetAlias(0, "boss") // 1001
	_ = h.Append(Entry{Target: "1001", Profile: "work", Server: "udp://c:5060", FromUser: "100"})
	if !h.AliasResolves("boss") {
		t.Fatal("alias lost after dedup update")
	}
	if e, ok := h.Entry(0); !ok || e.Alias != "boss" || e.Server != "udp://c:5060" {
		t.Fatalf("updated entry wrong: %+v", e)
	}

	// Different profile or from-number is a distinct entry.
	_ = h.Append(Entry{Target: "1001", Profile: "home", Server: "udp://a:5060", FromUser: "100"})
	_ = h.Append(Entry{Target: "1001", Profile: "work", Server: "udp://a:5060", FromUser: "200"})
	if h.Len() != 4 {
		t.Fatalf("Len = %d want 4", h.Len())
	}

	// Persisted dedup survives reload.
	h2, _ := LoadHistory(dir, 10)
	if h2.Len() != 4 {
		t.Fatalf("reload Len = %d want 4", h2.Len())
	}
}

func TestDefaultProfileSelection(t *testing.T) {
	dir := t.TempDir()
	p, _ := LoadProfiles(dir)

	// single profile → implicit default
	_ = p.Add(Profile{Name: "solo"})
	if def, ok := p.DefaultProfile(); !ok || def.Name != "solo" {
		t.Fatalf("single-profile implicit default failed: %+v ok=%v", def, ok)
	}
	// not explicitly marked
	if def, _ := p.Get("solo"); def.Default {
		t.Fatal("solo should not be explicitly marked")
	}

	// second profile → no default until set
	_ = p.Add(Profile{Name: "zebra"})
	if def, ok := p.DefaultProfile(); ok {
		t.Fatalf("two unmarked profiles should have no default, got %q", def.Name)
	}

	if err := p.SetDefault("zebra"); err != nil {
		t.Fatal(err)
	}
	def, ok := p.DefaultProfile()
	if !ok || def.Name != "zebra" || !def.Default {
		t.Fatalf("SetDefault failed: %+v ok=%v", def, ok)
	}
	// persistence + other flag cleared
	p2, _ := LoadProfiles(dir)
	d2, _ := p2.Get("zebra")
	s2, _ := p2.Get("solo")
	if !d2.Default || s2.Default {
		t.Fatalf("defaults after reload: zebra=%v solo=%v", d2.Default, s2.Default)
	}

	// switching default clears previous
	if err := p2.SetDefault("solo"); err != nil {
		t.Fatal(err)
	}
	d3, _ := p2.Get("zebra")
	if d3.Default {
		t.Fatal("zebra should lose default when solo is set")
	}

	// removing default promotes another
	if err := p2.Remove("solo"); err != nil {
		t.Fatal(err)
	}
	d4, ok := p2.DefaultProfile()
	if !ok || d4.Name != "zebra" {
		t.Fatalf("removal should promote remaining profile, got %+v ok=%v", d4, ok)
	}

	// SetDefault unknown name errors
	if err := p2.SetDefault("nope"); err == nil {
		t.Fatal("SetDefault(nope) should fail")
	}
}