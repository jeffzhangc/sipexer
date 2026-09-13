package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/miconda/sipexer/internal/dial"
	"github.com/miconda/sipexer/internal/store"
)

// isTerminal reports whether stdin is a terminal (interactive picker allowed).
func isTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// pickTarget prints the profile's recent dial history as numbered choices and
// a manual-number option, then reads one line and resolves it the same way a
// target argument would. Returns the Resolved dial for the pick.
func pickTarget(app *app, pr store.Profile) (*dial.Resolved, error) {
	fmt.Printf("profile %s — recent calls (most recent first):\n", pr.Name)
	entries := app.history.Entries()
	if len(entries) == 0 {
		fmt.Println("  (no history yet)")
	}
	for i, e := range entries {
		alias := ""
		if e.Alias != "" {
			alias = " [" + e.Alias + "]"
		}
		when := e.When
		if len(when) > 10 {
			when = when[0:10]
		}
		fmt.Printf("  @%-2d  %s  %s  %s%s\n", i, when, e.Server, e.Target, alias)
	}
	// Number aliases of the profile.
	if len(pr.Numbers) > 0 {
		fmt.Println("numbers:")
		for _, n := range pr.Numbers {
			alias := ""
			if n.Alias != "" {
				alias = " (alias " + n.Alias + ")"
			}
			fmt.Printf("  %-20s%s\n", n.String(), alias)
		}
	}

	fmt.Println("Enter: @<index>, a number alias, a raw number, or 'q' to cancel.")
	fmt.Print("> ")
	sc := bufio.NewScanner(os.Stdin)
	if !sc.Scan() {
		return nil, fmt.Errorf("no input")
	}
	line := strings.TrimSpace(sc.Text())
	if line == "" || line == "q" || line == "Q" || line == "quit" {
		return nil, fmt.Errorf("dial cancelled")
	}

	// Try number alias in this profile first.
	for _, n := range pr.Numbers {
		if n.Alias == line {
			return &dial.Resolved{Target: n, To: n.String(), Profile: pr.Name}, nil
		}
	}

	// History index.
	if strings.HasPrefix(line, "@") {
		i, err := strconv.Atoi(strings.TrimPrefix(line, "@"))
		if err != nil || i < 0 || i >= len(entries) {
			return nil, fmt.Errorf("invalid history index %q", line)
		}
		e := entries[i]
		return &dial.Resolved{Target: store.Number{User: e.FromUser}, To: e.Target, Server: e.Server, Profile: e.Profile}, nil
	}

	// Raw number/URI.
	return &dial.Resolved{Target: store.Number{}, To: line, Profile: pr.Name}, nil
}