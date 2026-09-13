package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Number is a registered (from-)number under a profile. Each may carry an
// alias usable as a dial target.
type Number struct {
	User   string `json:"user"`
	Domain string `json:"domain,omitempty"`
	Alias  string `json:"alias,omitempty"`
}

// String renders the number as user@domain (or just user).
func (n Number) String() string {
	if n.Domain != "" {
		return n.User + "@" + n.Domain
	}
	return n.User
}

// Server is a service address under a profile: proto://host:port. Exactly one
// server may be marked Default; the first configured wins when none is.
type Server struct {
	Addr    string `json:"addr"`
	Default bool   `json:"default,omitempty"`
}

// Profile is a named SIP account usable for outbound dials. Call-default
// fields use pointers so that "unset" is distinguishable from zero and an
// unset option is not emitted to the engine (design.md D1b).
type Profile struct {
	Name     string   `json:"name"`
	Default  bool     `json:"default,omitempty"` // current/main profile used by bare 'dial'
	AuthUser string   `json:"auth-user,omitempty"`
	AuthPass String   `json:"auth-pass,omitempty"`
	HA1      String   `json:"ha1,omitempty"`
	Numbers  []Number `json:"numbers"`
	Servers  []Server `json:"servers"`

	// Typed call defaults. Nested struct so pointers are preserved even when
	// the number of optional fields grows; each nil pointer means "unset".
	DialDefaults DialDefaults `json:"dial-defaults,omitempty"`

	Method              string   `json:"method,omitempty"`
	RegisterFirst       bool     `json:"register-first,omitempty"`
	SetUser             bool     `json:"set-user,omitempty"`
	ContactBuild        bool     `json:"contact-build,omitempty"`
	Expires             string   `json:"expires,omitempty"`
	UserAgent           string   `json:"user-agent,omitempty"`
	ContactURI          string   `json:"contact-uri,omitempty"`
	ContentType         string   `json:"content-type,omitempty"`
	Body                string   `json:"body,omitempty"`
	ExtraHeaders        []string `json:"extra-headers,omitempty"`
	ExtraDialFlags      []string `json:"extra-dial-flags,omitempty"`
	ExtraFlagsAfterDash []string `json:"extra-flags-after-dash,omitempty"`
}

// DialDefaults holds typed, optional call parameters. Nil pointer means
// unset (engine default applies).
type DialDefaults struct {
	SessionWaitMs    *int `json:"session-wait-ms,omitempty"`
	CallDurationMs   *int `json:"call-duration-ms,omitempty"`
	RingTimeMs       *int `json:"ring-time-ms,omitempty"`
	TimeoutMs        *int `json:"timeout-ms,omitempty"`
	TimeoutConnectMs *int `json:"timeout-connect-ms,omitempty"`
	TimeoutWriteMs   *int `json:"timeout-write-ms,omitempty"`
	Verbosity        *int `json:"verbosity,omitempty"`
	ColorOutput      *bool `json:"color-output,omitempty"`
	ColorMessage     *bool `json:"color-message,omitempty"`
	TLSInsecure      *bool `json:"tls-insecure,omitempty"`
}

// Profiles is the in-memory phone profile store.
type Profiles struct {
	dir string
	set map[string]Profile
}

// ProfilesFile is the file name inside the config directory.
const ProfilesFile = "profiles.json"

// LoadProfiles reads the profile store from configDir, creating an empty
// store if the file does not exist yet.
func LoadProfiles(configDir string) (*Profiles, error) {
	p := &Profiles{dir: configDir, set: map[string]Profile{}}
	path := filepath.Join(configDir, ProfilesFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return p, nil
		}
		return nil, fmt.Errorf("read profiles: %w", err)
	}
	var list []Profile
	if len(data) > 0 {
		if err := json.Unmarshal(data, &list); err != nil {
			return nil, fmt.Errorf("parse profiles: %w", err)
		}
	}
	for _, pr := range list {
		p.set[pr.Name] = pr
	}
	return p, nil
}

// save writes the store back to disk atomically.
func (p *Profiles) save() error {
	list := make([]Profile, 0, len(p.set))
	for _, pr := range p.set {
		list = append(list, pr)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profiles: %w", err)
	}
	data = append(data, '\n')
	return writeFile(filepath.Join(p.dir, ProfilesFile), data, 0o600)
}

// Add inserts a new profile, rejecting a duplicate name.
func (p *Profiles) Add(pr Profile) error {
	if _, ok := p.set[pr.Name]; ok {
		return fmt.Errorf("profile %q already exists (use edit to change it)", pr.Name)
	}
	p.set[pr.Name] = pr
	return p.save()
}

// Edit overwrites an existing profile (the profile must exist).
func (p *Profiles) Edit(pr Profile) error {
	if _, ok := p.set[pr.Name]; !ok {
		return fmt.Errorf("profile %q does not exist (use add)", pr.Name)
	}
	p.set[pr.Name] = pr
	return p.save()
}

// Remove deletes a named profile; an unknown name is an error. If the removed
// profile was the default and others remain, the first (by name) is promoted.
func (p *Profiles) Remove(name string) error {
	pr, ok := p.set[name]
	if !ok {
		return fmt.Errorf("profile %q does not exist", name)
	}
	delete(p.set, name)
	if pr.Default && len(p.set) > 0 {
		// Promote the first remaining profile (by name) as the new default.
		first := p.List()[0]
		first.Default = true
		p.set[first.Name] = first
	}
	return p.save()
}

// Get returns a profile by name.
func (p *Profiles) Get(name string) (Profile, bool) {
	pr, ok := p.set[name]
	return pr, ok
}

// List returns all profiles sorted by name.
func (p *Profiles) List() []Profile {
	out := make([]Profile, 0, len(p.set))
	for _, pr := range p.set {
		out = append(out, pr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Names returns profile names (saved order).
func (p *Profiles) Names() []string {
	out := []string{}
	for _, pr := range p.List() {
		out = append(out, pr.Name)
	}
	return out
}

// DefaultProfile returns the profile marked Default (the "current" one used
// by a bare `sipx dial <target>`). When none is explicitly marked but exactly
// one profile exists, that one is used.
func (p *Profiles) DefaultProfile() (Profile, bool) {
	for _, pr := range p.set {
		if pr.Default {
			return pr, true
		}
	}
	if len(p.set) == 1 {
		for _, pr := range p.set {
			return pr, true
		}
	}
	return Profile{}, false
}

// SetDefault marks one profile as default, clearing the flag on all others,
// and persists.
func (p *Profiles) SetDefault(name string) error {
	pr, ok := p.set[name]
	if !ok {
		return fmt.Errorf("profile %q does not exist", name)
	}
	for n, other := range p.set {
		if other.Default && n != name {
			other.Default = false
			p.set[n] = other
		}
	}
	pr.Default = true
	p.set[name] = pr
	return p.save()
}

// SelectServer returns the dial server for a profile: the marked default one,
// else the first configured, else an error.
func (p *Profiles) SelectServer(pr Profile) (string, error) {
	for _, s := range pr.Servers {
		if s.Default {
			return s.Addr, nil
		}
	}
	if len(pr.Servers) > 0 {
		return pr.Servers[0].Addr, nil
	}
	return "", fmt.Errorf("profile %q has no servers; add one with 'sipx server add %s <addr>'", pr.Name, pr.Name)
}

// SelectFromNumber returns the from-number for a dial: an explicit user/alias
// match, else the first configured number.
func (p *Profiles) SelectFromNumber(pr Profile, explicit string) (Number, error) {
	if explicit != "" {
		for _, n := range pr.Numbers {
			if n.Alias == explicit || n.User == explicit {
				return n, nil
			}
		}
		return Number{}, fmt.Errorf("profile %q has no number matching %q", pr.Name, explicit)
	}
	if len(pr.Numbers) > 0 {
		return pr.Numbers[0], nil
	}
	return Number{}, fmt.Errorf("profile %q has no numbers; add one with 'sipx phone edit %s'", pr.Name, pr.Name)
}

// ResolveNumberAlias returns the number whose alias matches, if any.
func (p *Profiles) ResolveNumberAlias(pr Profile, alias string) (Number, bool) {
	for _, n := range pr.Numbers {
		if n.Alias == alias {
			return n, true
		}
	}
	return Number{}, false
}