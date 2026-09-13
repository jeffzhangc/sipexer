# Add Cobra/pflag CLI wrapper with phone profiles and dial history

## Why

`sipexer` is a powerful SIP CLI, but today every invocation requires retyping the full set of
parameters (server, transport, From/To URIs, credentials, target number). For day-to-day outbound
telephony testing there is no way to persist the SIP phone configuration or to quickly re-dial a
number that was called before. This change adds a second, wrapped binary (`sipx`) built on
`cobra`/`pflag` that manages named phone configurations (each holding multiple registered numbers
and multiple service addresses), records outgoing-call history with aliases, and drives the
original `sipexer` as a subprocess — so a frequently used scenario can be re-run with one short
command while the original binary keeps working byte-for-byte.

## What Changes

- Build **two binaries**: the original `sipexer` (untouched) and a new wrapped `sipx` produced
  from a new `cmd/sipx` command tree built on `cobra` + `pflag`.
- `sipx` locates the `sipexer` binary at runtime (explicit `--engine` flag, `SIPEXER_BIN` env,
  same directory as `sipx`, then `PATH`) and executes dials as a subprocess, forwarding sipexer's
  exit code and streaming its stdout/stderr.
- Add a persistent **phone profile store**: each named profile holds **multiple registered
  numbers** (user/domain, with optional alias) and **multiple service addresses**
  (proto:host:port), plus auth credentials (username + password/HA1, stored obfuscated).
- Add a persistent **dial history store**: last N dialed targets (raw number/URI), with the
  profile, server and from-number used, an optional user-assigned alias, and a timestamp.
  Entries can be re-dialed by index or by alias; the dial command surfaces the selected profile's
  own history as the preferred candidates.
- `sipx dial [profile] [target]` — target may be a raw number/URI, a history alias, a history
  index, or a configured-number alias. With a TTY and sparse arguments, `dial` runs an interactive
  picker (profile → server → from-number → target from history suggestions). Raw sipexer flags can
  be passed through after `--`.
- Commands: `sipx phone add|edit|list|rm|show`, `sipx dial`, `sipx history list|clear|alias`,
  `sipx server add|rm|list`, and `sipx doctor` (reports discovered engine path/version).
- Existing `sipexer` invocation is completely unchanged. **No BREAKING** changes.
- New dependencies for `sipx` only: `github.com/spf13/cobra`, `github.com/spf13/pflag`,
  `golang.org/x/term`.

## Capabilities

### New Capabilities
- `cli-commands`: the `sipx` Cobra command surface that drives the unmodified `sipexer` binary via
  subprocess (engine discovery, exit-code forwarding, raw-flag passthrough, `doctor`).
- `phone-profiles`: the persistent store of named SIP phone configurations — each with multiple
  registered numbers and multiple service addresses — and the `phone` + `server` subcommands.
- `dial-history`: the persistent record of outgoing calls with aliases, re-dial by index/alias,
  per-profile suggestion, and `history` subcommands.

### Modified Capabilities
<!-- No existing openspec/specs exist yet; this change introduces the first specs. -->

## Impact

- New code: `cmd/sipx` (Cobra tree, dial orchestration, picker) and `internal/store` (profile,
  history, config-dir, secret obfuscation).
- Build: a build script/Makefile target producing both `sipexer` and `sipx`.
- `go.mod`/`go.sum`: adds `spf13/cobra`, `spf13/pflag`, `golang.org/x/term` (used by `sipx` only).
- `sipexer.go`, `sgsip`, and the SIP engine: **no changes**.
- Runtime: new config files under `~/.config/sipexer/` (`profiles.json`, `history.json`).
- Tests: unit tests for the store packages, target resolution, and engine discovery; a parity
  smoke test that `sipx dial` runs the engine subprocess.