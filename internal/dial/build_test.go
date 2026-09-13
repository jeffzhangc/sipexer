package dial

import (
	"testing"

	"github.com/miconda/sipexer/internal/store"
)

func iptr(v int) *int { return &v }
func bptr(v bool) *bool { return &v }

func TestBuildArgumentsBasic(t *testing.T) {
	p := store.Profile{
		Name: "work",
		Method: "INVITE",
		AuthUser: "alice",
		AuthPass: store.NewString("p"),
		Numbers: []store.Number{{User: "1001", Domain: "example.com"}},
		Servers: []store.Server{{Addr: "udp://pbx.example.com:5060"}},
		DialDefaults: store.DialDefaults{
			SessionWaitMs: iptr(5000),
			Verbosity:     iptr(2),
			ColorOutput:   bptr(true),
		},
	}
	from := store.Number{User: "1001", Domain: "example.com"}
	args := BuildArguments(p, from, "2002", "udp://pbx.example.com:5060", nil, nil)
	want := []string{
		"-i", // INVITE
		"-au", "alice",
		"-ap", "p",
		"-fuser", "1001",
		"-fdomain", "example.com",
		"-tuser", "2002",
		"-rn", "2002",
		"-vl", "2",
		"-sw", "5000",
		"-co",
		"udp:pbx.example.com:5060",
	}
	have := args
	if len(want) != len(have) {
		t.Fatalf("len: want %d got %d\nwant %v\n got %v", len(want), len(have), want, have)
	}
	for i := range want {
		if have[i] != want[i] {
			t.Fatalf("arg[%d] want %q got %q", i, want[i], have[i])
		}
	}
}

func TestBuildArgumentsUnsetNotEmitted(t *testing.T) {
	p := store.Profile{
		Name: "work", // no DialDefaults set → nothing emitted
		Numbers: []store.Number{{User: "1001"}},
		Servers: []store.Server{{Addr: "udp://x:5060"}},
	}
	args := BuildArguments(p, store.Number{User: "1001"}, "2002", "udp://x:5060", nil, nil)
	for _, a := range args {
		if a == "-sw" || a == "-co" {
			t.Fatalf("unset default emitted: %v", args)
		}
	}
	// method unset → no -i/-m etc.
	if contains(args, "-i") || contains(args, "-m") {
		t.Fatalf("unset method emitted: %v", args)
	}
	if len(args) == 0 || args[len(args)-1] != "udp:x:5060" {
		t.Fatalf("server positional missing (want normalized udp:x:5060): %v", args)
	}
}

func TestBuildArgumentsMethodMap(t *testing.T) {
	cases := map[string]string{
		"INVITE":   "-i",
		"REGISTER": "-r",
		"message":  "-m",
		"":         "",
	}
	for method, want := range cases {
		if got := MethodFlag(method); got != want {
			t.Fatalf("MethodFlag(%q): got %q want %q", method, got, want)
		}
	}
}

func TestBuildArgumentsToURI(t *testing.T) {
	p := store.Profile{Name: "x", Numbers: []store.Number{{User: "u"}}}
	// URI target → -to-uri verbatim, no -su.
	args := BuildArguments(p, store.Number{User: "u"}, "sip:2002@far.example.com", "udp://x:5060", nil, nil)
	if !contains(args, "-to-uri") {
		t.Fatalf("expected -to-uri: %v", args)
	}
	if contains(args, "-su") {
		t.Fatalf("URI path should not set -su: %v", args)
	}
}

func contains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}