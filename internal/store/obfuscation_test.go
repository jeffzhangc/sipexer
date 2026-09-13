package store

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSecretRoundTrip(t *testing.T) {
	s := NewString("hunter2:HA1hex")
	j, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(j), "hunter2") {
		t.Fatalf("plaintext leaked in JSON: %s", j)
	}

	var back String
	if err := json.Unmarshal(j, &back); err != nil {
		t.Fatal(err)
	}
	if back.Plain() != "hunter2:HA1hex" {
		t.Fatalf("round trip: got %q", back.Plain())
	}
}

func TestSecretRedact(t *testing.T) {
	s := NewString("secret")
	if s.Redact() == "" || s.Redact() == "secret" {
		t.Fatalf("Redact should be a placeholder, got %q", s.Redact())
	}
	// Redact must not leak through JSON either (listings use Redact).
	var empty String
	if empty.Redact() != "" {
		t.Fatalf("empty secret should redact to empty, got %q", empty.Redact())
	}
}