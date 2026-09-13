package store

import (
	"testing"
)

func iptr(v int) *int { return &v }

func TestProfilesCRUD(t *testing.T) {
	dir := t.TempDir()
	p, err := LoadProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}

	// add
	if err := p.Add(Profile{
		Name:     "work",
		AuthUser: "alice",
		AuthPass: NewString("s3cret"),
		Numbers:  []Number{{User: "1001", Domain: "example.com", Alias: "desk"}},
		Servers:  []Server{{Addr: "udp://pbx.example.com:5060", Default: true}},
		DialDefaults: DialDefaults{
			SessionWaitMs: iptr(5000),
		},
	}); err != nil {
		t.Fatal(err)
	}

	// duplicate rejected
	if err := p.Add(Profile{Name: "work"}); err == nil {
		t.Fatal("duplicate add should fail")
	}

	// list + get
	if _, ok := p.Get("work"); !ok {
		t.Fatal("Get(work) failed")
	}
	if len(p.List()) != 1 {
		t.Fatalf("List len = %d", len(p.List()))
	}

	// edit preserves other fields + secret round trip through reload
	pr, _ := p.Get("work")
	pr.Numbers = append(pr.Numbers, Number{User: "1002", Alias: "mob"})
	if err := p.Edit(pr); err != nil {
		t.Fatal(err)
	}

	// reload from disk, check secret + pointers + defaults survived
	p2, err := LoadProfiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := p2.Get("work")
	if !ok {
		t.Fatal("reload lost profile")
	}
	if got.AuthPass.Plain() != "s3cret" {
		t.Fatalf("secret lost in reload: %q", got.AuthPass.Plain())
	}
	if got.DialDefaults.SessionWaitMs == nil || *got.DialDefaults.SessionWaitMs != 5000 {
		t.Fatalf("session wait lost: %v", got.DialDefaults.SessionWaitMs)
	}
	// unset pointer stays nil after round trip
	if got.DialDefaults.Verbosity != nil {
		t.Fatalf("verbosity should be nil (unset), got %v", *got.DialDefaults.Verbosity)
	}
	if len(got.Numbers) != 2 {
		t.Fatalf("numbers len = %d", len(got.Numbers))
	}

	// remove
	if err := p2.Remove("work"); err != nil {
		t.Fatal(err)
	}
	if _, ok := p2.Get("work"); ok {
		t.Fatal("profile still present after remove")
	}
	if err := p2.Remove("nope"); err == nil {
		t.Fatal("removing missing profile should fail")
	}
}

func TestSelectServerAndNumber(t *testing.T) {
	dir := t.TempDir()
	p, _ := LoadProfiles(dir)

	pr := Profile{
		Name: "x",
		Servers: []Server{
			{Addr: "udp://a:5060"},
			{Addr: "tcp://b:5060", Default: true},
		},
		Numbers: []Number{
			{User: "1001", Alias: "one"},
			{User: "1002", Alias: "two"},
		},
	}

	// default server wins
	addr, err := p.SelectServer(pr)
	if err != nil {
		t.Fatal(err)
	}
	if addr != "tcp://b:5060" {
		t.Fatalf("server: got %q", addr)
	}

	// explicit from-user match by user
	n, err := p.SelectFromNumber(pr, "1002")
	if err != nil {
		t.Fatal(err)
	}
	if n.User != "1002" {
		t.Fatalf("from number: got %+v", n)
	}

	// alias exists
	if _, ok := p.ResolveNumberAlias(pr, "one"); !ok {
		t.Fatal("alias 'one' should resolve")
	}

	// server missing -> error
	if _, err := p.SelectServer(Profile{Name: "empty"}); err == nil {
		t.Fatal("empty server list should error")
	}
}