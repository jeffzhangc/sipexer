package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry is a single dial-history record. When is stored as RFC3339.
type Entry struct {
	When     string `json:"when"`
	Profile  string `json:"profile"`
	Server   string `json:"server,omitempty"`
	FromUser string `json:"from-user,omitempty"`
	Target   string `json:"target"`
	Alias    string `json:"alias,omitempty"`
}

// History is the persistent dial history store. Entries are kept oldest-first
// on disk; the CLI presents them most-recent-first (index 0 = newest).
type History struct {
	dir string
	cap int
	es  []Entry
}

// HistoryFile is the file name inside the config directory.
const HistoryFile = "history.json"

// LoadHistory reads the history store, applying cap (0 means the default).
func LoadHistory(configDir string, cap int) (*History, error) {
	if cap <= 0 {
		cap = DefaultHistoryCap
	}
	h := &History{dir: configDir, cap: cap}
	path := filepath.Join(configDir, HistoryFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return h, nil
		}
		return nil, fmt.Errorf("read history: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, &h.es); err != nil {
			return nil, fmt.Errorf("parse history: %w", err)
		}
	}
	h.dedup()
	h.trim()
	return h, nil
}

// dedup compacts legacy duplicates: keeps one entry per
// (profile, from-user, target), preferring the newest, and folds in any
// alias found on older copies.
func (h *History) dedup() {
	newest := map[string]Entry{}
	for _, e := range h.es {
		k := e.Profile + "\x00" + e.FromUser + "\x00" + e.Target
		old, seen := newest[k]
		if !seen {
			newest[k] = e
			continue
		}
		// newest wins overall; alias survives from either copy
		if e.Alias == "" {
			e.Alias = old.Alias
		}
		if e.When >= old.When {
			newest[k] = e
		}
	}
	if len(newest) == len(h.es) {
		return
	}
	out := make([]Entry, 0, len(newest))
	for _, e := range newest {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].When < out[j].When })
	h.es = out
}

// trim enforces the cap, keeping the newest entries.
func (h *History) trim() {
	if len(h.es) > h.cap {
		h.es = h.es[len(h.es)-h.cap:]
	}
}

// Append adds an entry with Now as the timestamp (most recent last), trims to
// the cap, and persists. De-duplication: at most one entry per
// (profile, from-number, target) — re-dialing an existing pair updates that
// entry (new timestamp/server, alias preserved) and moves it to the newest
// position instead of appending a duplicate.
func (h *History) Append(e Entry) error {
	if e.When == "" {
		e.When = time.Now().Format(time.RFC3339)
	}
	for i, old := range h.es {
		if old.Profile == e.Profile && old.FromUser == e.FromUser && old.Target == e.Target {
			// Update in place: keep the assigned alias, refresh the rest,
			// and move to the tail (newest).
			if e.Alias == "" {
				e.Alias = old.Alias
			}
			h.es = append(h.es[:i], h.es[i+1:]...)
			break
		}
	}
	h.es = append(h.es, e)
	h.trim()
	return h.save()
}

// save writes the store atomically.
func (h *History) save() error {
	if h.es == nil {
		h.es = []Entry{}
	}
	data, err := json.MarshalIndent(h.es, "", "  ")
	if err != nil {
		return fmt.Errorf("encode history: %w", err)
	}
	data = append(data, '\n')
	return writeFile(filepath.Join(h.dir, HistoryFile), data, 0o600)
}

// Clear empties and persists the store.
func (h *History) Clear() error {
	h.es = nil
	return h.save()
}

// Len returns the number of entries.
func (h *History) Len() int { return len(h.es) }

// Entries returns a copy sorted most-recent-first.
func (h *History) Entries() []Entry {
	out := make([]Entry, len(h.es))
	copy(out, h.es)
	// Reverse so index 0 = newest.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// Entry returns the entry at most-recent-first index i (0 = newest).
func (h *History) Entry(i int) (Entry, bool) {
	if i < 0 || i >= len(h.es) {
		return Entry{}, false
	}
	return h.Entries()[i], true
}

// SetAlias sets the alias of the entry at most-recent-first index i.
func (h *History) SetAlias(i int, name string) error {
	if i < 0 || i >= len(h.es) {
		return fmt.Errorf("history index %d out of range (0..%d)", i, len(h.es)-1)
	}
	// Index into the oldest-first backing slice.
	idx := len(h.es) - 1 - i
	h.es[idx].Alias = name
	return h.save()
}

// AliasResolves tells whether any entry currently has the given alias.
func (h *History) AliasResolves(name string) bool {
	for _, e := range h.es {
		if e.Alias == name {
			return true
		}
	}
	return false
}

// SortRecent provides deterministic recent-first order for callers needing
// control over the sort (used by the dial picker).
func SortRecent(entries []Entry) []Entry {
	out := make([]Entry, len(entries))
	copy(out, entries)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].When > out[j].When
	})
	return out
}