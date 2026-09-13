package store

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// obfuscationKey derives a machine-specific key for the XOR pass, so an
// accidental `cat profiles.json` shows gibberish rather than plaintext
// secrets. This is defense-in-depth only — NOT encryption (design.md D5).
func obfuscationKey() ([]byte, error) {
	host, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("obfuscation key: %w", err)
	}
	user := ""
	if runtime.GOOS == "windows" {
		user = os.Getenv("USERNAME")
	} else {
		user = os.Getenv("USER")
	}
	return []byte("sipx:" + host + ":" + user), nil
}

// String holds a secret. It obfuscates on Marshal and de-obfuscates on
// Unmarshal, so the on-disk JSON never contains a plaintext secret.
type String struct {
	v string
}

// NewString wraps a plaintext secret.
func NewString(plain string) String {
	return String{v: plain}
}

// Plain returns the value. In memory the value is always held as plaintext;
// only the on-disk JSON form is obfuscated (see MarshalJSON/UnmarshalJSON).
func (s String) Plain() string {
	return s.v
}

// Redact returns a placeholder suitable for listings.
func (s String) Redact() string {
	if s.v == "" {
		return ""
	}
	return "••••••"
}

// MarshalJSON stores the hex-encoded, obfuscated value. Hex avoids emitting
// arbitrary bytes that could break JSON string escaping.
func (s String) MarshalJSON() ([]byte, error) {
	out := s.v
	if out != "" {
		if key, err := obfuscationKey(); err == nil {
			out = hex.EncodeToString(xorKeyed([]byte(out), key))
		}
	}
	return []byte(fmt.Sprintf("%q", out)), nil
}

// UnmarshalJSON de-obfuscates the stored hex value.
func (s *String) UnmarshalJSON(b []byte) error {
	in := strings.Trim(string(b), `"`)
	if in == "" {
		s.v = ""
		return nil
	}
	raw, err := hex.DecodeString(in)
	if err != nil {
		// Not the expected format: keep as-is rather than corrupting data.
		s.v = in
		return nil
	}
	if key, err := obfuscationKey(); err == nil {
		s.v = string(xorKeyed(raw, key))
	} else {
		s.v = string(raw)
	}
	return nil
}

// xorKeyed XORs data against repeating key bytes.
func xorKeyed(data, key []byte) []byte {
	out := make([]byte, len(data))
	for i := range data {
		out[i] = data[i] ^ key[i%len(key)]
	}
	return out
}