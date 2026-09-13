package dial

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/miconda/sipexer/internal/store"
)

// Resolved describes what a dial will send.
type Resolved struct {
	Target  store.Number // the (from-)number used
	To      string       // the number/URI dialed
	Server  string       // the service address used
	Profile string
}

// ResolveTarget takes the selected profile, the typed target argument (may be
// empty), and returns what a dial will use.
//
// Resolution order (design.md D2):
//  1. A @N history index → re-dial that entry's target/from-number.
//  2. A configured number alias → dial that number, record resolved target.
//  3. A history-entry alias → re-dial that entry (within the profile).
//  4. Otherwise the raw input is dialed verbatim.
//
// If targetArg is empty and history has entries, the caller is expected to
// offer the interactive picker (dial command handles that).
func ResolveTarget(p store.Profile, hist *store.History, targetArg string) (*Resolved, error) {
	if targetArg == "" {
		return nil, nil // signal "needs picker"
	}

	// 1. history index @N (most recent first).
	if strings.HasPrefix(targetArg, "@") {
		if idx, err := strconv.Atoi(strings.TrimPrefix(targetArg, "@")); err == nil {
			if e, ok := hist.Entry(idx); ok {
				return &Resolved{
					Target:  store.Number{User: e.FromUser},
					To:      e.Target,
					Server:  e.Server,
					Profile: e.Profile,
				}, nil
			}
			return nil, fmt.Errorf("history index %q out of range (have %d entries)", targetArg, hist.Len())
		}
		return nil, fmt.Errorf("invalid history index %q (use @N)", targetArg)
	}

	// 2. configured number alias in the profile.
	for _, num := range p.Numbers {
		if num.Alias == targetArg {
			return &Resolved{
				Target:  num,
				To:      num.String(),
				Server:  "",
				Profile: p.Name,
			}, nil
		}
	}

	// 3. history-entry alias (profile history first) — routed through the
	//    selected profile (spec: "the call is still routed through the
	//    selected profile").
	if e, ok := historyByAlias(hist, targetArg); ok {
		return &Resolved{
			Target:  store.Number{User: e.FromUser},
			To:      e.Target,
			Server:  e.Server,
			Profile: p.Name,
		}, nil
	}

	// 4. raw dial.
	return &Resolved{
		Target:  store.Number{},
		To:      targetArg,
		Server:  "",
		Profile: p.Name,
	}, nil
}

// historyByAlias finds an entry by its alias (most recent first).
func historyByAlias(hist *store.History, name string) (store.Entry, bool) {
	if hist == nil {
		return store.Entry{}, false
	}
	for _, e := range hist.Entries() {
		if e.Alias == name {
			return e, true
		}
	}
	return store.Entry{}, false
}