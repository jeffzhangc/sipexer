package dial

import (
	"fmt"
	"strings"

	"github.com/miconda/sipexer/internal/store"
)

// MethodFlag maps a profile method name to the sipexer method flag (or "").
// e.g. "INVITE" -> "-i"; "REGISTER" -> "-r"; "" -> nothing.
func MethodFlag(method string) string {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "":
		return ""
	case "INVITE":
		return "-i"
	case "REGISTER":
		return "-r"
	case "OPTIONS":
		return "-o"
	case "MESSAGE":
		return "-m"
	case "SUBSCRIBE":
		return "-subscribe"
	case "NOTIFY":
		return "-notify"
	case "INFO":
		return "-info"
	case "PUBLISH":
		return "-publish"
	case "ACK":
		return "-ack"
	case "CANCEL":
		return "-cancel"
	case "PRACK":
		return "-prack"
	default:
		return "-method " + method
	}
}

// BuildArguments assembles the engine argv for a dial. `from` is the
// profile's registered (from-)number; `to` is the dialed number/URI.
//
// Order:
//  1. method flag (if the profile sets one)
//  2. auth flags (-au, -ap, -ha1)
//  3. from user/domain
//  4. to: bare number → -tuser/+ -sd -su (R-URI user = the number, via proxy);
//     URI → -to-uri verbatim
//  5. typed call defaults (only non-nil profile fields)
//  6. profile server as the final positional (engine target is proto:host:port)
func BuildArguments(p store.Profile, from store.Number, to, server string, extraDialFlags, afterDash []string) []string {
	args := []string{}

	if mf := MethodFlag(p.Method); mf != "" {
		args = append(args, mf)
	}
	if p.RegisterFirst {
		args = append(args, "-register-first")
	}
	if p.SetUser {
		args = append(args, "-su")
	}
	if p.ContactBuild {
		args = append(args, "-cb")
	}

	if p.AuthUser != "" {
		args = append(args, "-au", p.AuthUser)
	}
	switch {
	case p.HA1.Plain() != "":
		args = append(args, "-ha1", "-ap", p.HA1.Plain())
	case p.AuthPass.Plain() != "":
		args = append(args, "-ap", p.AuthPass.Plain())
	}

	if from.User != "" {
		args = append(args, "-fuser", from.User)
	}
	if from.Domain != "" {
		args = append(args, "-fdomain", from.Domain)
	}

	if to != "" {
		if isURILike(to) {
			args = append(args, "-to-uri", to)
		} else {
			// Bare number through a proxy: To-URI user = the callee and the
			// R-URI user = the callee too (the proxy resolves the route).
			args = append(args, "-tuser", to, "-rn", to)
		}
	}

	d := p.DialDefaults
	dict := []struct {
		p    *int
		flag string
	}{
		{d.Verbosity, "-vl"},
		{d.SessionWaitMs, "-sw"},
		{d.CallDurationMs, "-cd"},
		{d.RingTimeMs, "-rt"},
		{d.TimeoutMs, "-timeout"},
		{d.TimeoutConnectMs, "-timeout-connect"},
		{d.TimeoutWriteMs, "-timeout-write"},
	}
	for _, kv := range dict {
		if kv.p != nil {
			args = append(args, kv.flag, fmt.Sprintf("%d", *kv.p))
		}
	}
	boolOpts := []struct {
		p    *bool
		flag string
	}{
		{d.ColorOutput, "-co"},
		{d.ColorMessage, "-com"},
		{d.TLSInsecure, "-ti"},
	}
	for _, kv := range boolOpts {
		if kv.p != nil && *kv.p {
			args = append(args, kv.flag)
		}
	}
	if p.Expires != "" {
		args = append(args, "-ex", p.Expires)
	}
	if p.UserAgent != "" {
		args = append(args, "-ua", p.UserAgent)
	}
	if p.ContactURI != "" {
		args = append(args, "-cu", p.ContactURI)
	}
	if p.ContentType != "" {
		args = append(args, "-ct", p.ContentType)
	}
	if p.Body != "" {
		args = append(args, "-mb", p.Body)
	}
	for _, h := range p.ExtraHeaders {
		args = append(args, "-xh", h)
	}

	args = append(args, extraDialFlags...)
	args = append(args, normalizeServer(server))

	if len(afterDash) > 0 {
		args = append(args, "--")
		args = append(args, afterDash...)
	}
	return args
}

// normalizeServer converts a profile's "proto://host:port" into the engine's
// "proto:host:port" positional form (sipexer expects "udp:host:port", the
// "://" form only applies to ws/wss where it normalizes internally).
func normalizeServer(server string) string {
	for _, proto := range []string{"udp", "tcp", "tls", "sctp", "ws", "wss"} {
		prefix := proto + "://"
		if strings.HasPrefix(server, prefix) {
			return proto + ":" + strings.TrimPrefix(server, prefix)
		}
	}
	return server
}

// isURILike reports whether to looks like a SIP URI (contains '@' or ':' or a
// sip:/sips: prefix) rather than a bare number.
func isURILike(to string) bool {
	return strings.Contains(to, ":") || strings.Contains(to, "@")
}