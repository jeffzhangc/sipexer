# Design: add-cobra-cli-profiles

See `proposal.md` for the motivation and `specs/` for the requirements this design satisfies.

## Context

`sipexer` is a single Go package (`package main`) driven by the standard `flag` package — one
process runs one scenario (`SIPExerRunScenario`, `sipexer.go:756 main()`) built from a large set
of global flags and a destination argument. There is no persistence layer beyond user-supplied
JSON field files; every run is a fresh composition of CLI args. go.mod is `github.com/miconda/sipexer`,
Go 1.20, currently depending only on `uuid`, `x/net/websocket`-era modules, plus `golang.org/x/sync`.

The new wrapped binary `sipx` must **not** modify the SIP engine. Per decision D0 the engine runs
untouched as a subprocess.

## Goals / Non-Goals

**Goals**
- Ship two binaries: untouched `sipexer` + new `sipx` (Cobra/pflag).
- `sipx` manages profiles (multi-number, multi-server), dial history with aliases, server lists,
  and a `doctor`/engine-discovery story.
- `sipx dial` prefers the selected profile's own history as candidates, can dial numbers, aliases,
  or history indices, and forwards raw flags to the engine after `--`.
- Legacy `sipexer` behavior: byte-for-byte preserved.

**Non-Goals**
- No change to `sipexer.go`/`sgsip`/the engine message template system.
- No encryption at rest — obfuscation only (see D5).
- No network-specific semantics beyond what the engine already does; history records the attempt,
  not wire-level success.
- No interactive SIP phone behavior; `sipx` is a request-crafting wrapper, pickers are CLI-level.

## Decisions

### D0: Two binaries; `sipx` executes `sipexer` as a subprocess
`cmd/sipx` builds `sipx`; the repo-root `sipexer` build is untouched. `sipx` runs the engine with
`exec.Command`, streams stdout/stderr, forwards the exit code, and resolves targets/aliases itself
before handing the resulting engine args to the subprocess. This means zero risk to the engine and
a clean separation of concerns: `sipx` owns state (profiles/history/config), the engine owns SIP.

Alternatives: (a) rewrite `sipexer.main` as importable engine — rejected, too invasive (D2 in the
earlier design); (b) in-process call and mimic flags — rejected for same risk; (c) exec was chosen.
Subprocess cost (a few ms) is irrelevant for SIP dials.

### D1: Profile model — one profile, multiple numbers, multiple servers, aliases, call defaults
`Profile`:

```
type Profile struct {
    Name     string
    AuthUser string
    AuthPass Secret   // obfuscated
    HA1      Secret   // obfuscated
    Numbers  []Number // each Number{ User, Domain, Alias }
    Servers  []Server // each Server{ Addr, Default bool }  (proto:host:port or proto://host:port)
    // D1b — typed defaults applied on every dial (only non-zero pointers are emitted):
    SessionWaitMs int        // maps to -sw
    CallDurationMs int       // maps to -cd
    RingTimeMs int           // maps to -rt
    TimeoutMs int            // maps to -timeout
    TimeoutConnectMs int     // maps to -timeout-connect
    TimeoutWriteMs int       // maps to -timeout-write
    Verbosity int            // maps to -vl
    ColorOutput bool         // maps to -co
    ColorMessage bool        // maps to -com
    Expires string           // maps to -ex
    UserAgent string         // maps to -ua
    ContactURI string        // maps to -cu
    ContentType string       // maps to -ct
    Body string              // maps to -mb
    TLSInsecure bool         // maps to -ti
    Method string            // maps to the -m/-invite/... method flag
    ExtraHeaders []string    // maps to -xh (name:body each)
    ExtraDialFlags []string        // raw engine args appended at end
    ExtraDialFlagsAfterDash []string // raw engine args appended after `--`
}
```

`Numbers` (registered from-numbers) and `Servers` (service addresses) are independent lists, so a
profile holds several registered numbers and can reach any server. The user selects profile, then
a server defaults to the marked default (or flag `--server` overrides); a from-number is chosen
(default from the profile, or by number-alias, or explicitly with `--from-user`).

### D1b: Typed call defaults are per-profile, unset ≠ zero
Default call params are stored as **typed fields** (not raw strings) for the commonly reused
sipexer options, so a single `sipx dial work 1001` carries `--sw`, `--cd`, `--co`, etc. as the
profile intends. Distinguishing "unset" from "zero" matters: an unset `SessionWaitMs` must not
emit `-sw 0` and clobber the engine's own default. Numeric opt-in fields therefore use pointer
`*int` (nil ⇒ unset) in the JSON model; booleans (`ColorOutput`, `ColorMessage`, `TLSInsecure`)
use `*bool` for the same reason (a missing `-co` is different from passing `-co=true`).

Everything else stays reachable: `ExtraDialFlags` (after the resolved args, before the target) and
`ExtraDialFlagsAfterDash` (after the engine's `--`) carry one-off engine flags that aren't worth
first-class modeling.

Precedence when assembling engine argv: **CLI flag on `dial` > profile typed/default field >
engine's own default** — an unset profile field is simply not emitted, so the engine default
wins. Rationale: keeps the profile a *baseline*, never a trap, for behavior the engine already
defaults sanely (`--sw` default is effectively the engine's session behavior).

### D2: Dial target resolution order
`dial` resolves its target argument, in order:
1. Parses as a SIP URI or a bare numeric string → dial it as typed.
2. Matches a `Numbers[].Alias` (number alias configured in the profile) → dial; records the
   resolved raw number in history.
3. Matches a history-entry alias (`HistoryAlias` field on an entry) → re-dial that entry's target
   and settings (kept within the selected profile).
4. Matches history-index syntax `@N` (most recent first) → re-dial that entry.
5. No target given → interactive picker over the profile's own history (most recent first) plus a
   manual number option. (Non-TTY with no target → error listing the targeted choices.)

### D3: Interactive picker (TTY only)
`dial` with a TTY and a missing target prints the selected profile's history (index, alias,
timestamp, target) as numbered/short choices plus options for a manual number and for each
configured number alias, and reads one line from stdin via `golang.org/x/term`. Non-TTY without a
target fails with a concise error naming what could be provided (`<number>`, `@<index>`,
`<number-alias>`, `<history-alias>`). Non-TTY with credentials prompt → error guidance instead of
hanging.

### D4: JSON store layout & atomic writes
Config root resolved in order: `--config-dir` flag → `$SIPEXER_CONFIG_DIR` → `$XDG_CONFIG_HOME/sipexer`
→ `~/.config/sipexer` (Windows: `%APPDATA%\sipexer`). Two files: `profiles.json`, `history.json`
each written atomically (temp + rename; Windows fallback remove+rename on rename error).

History JSON shape:
```
type Entry struct {
    When     string  // RFC3339
    Profile  string
    Server   string
    FromUser string
    Target   string
    Alias    string  // optional user label; resolved in profile-scope or global via HistoryAlias lookup
}
```
Store keeps entries oldest-first; the CLI lists most-recent-first (`@0` = newest).

### D5: Secret obfuscation (defense-in-depth, not security)
`AuthPass`, and `HA1` are stored XOR-obfuscated with a key derived from hostname+username salt, so
`cat profiles.json` shows gibberish rather than plaintext. Documented explicitly as not-encryption.
Users who prefer plaintext can rely on file permissions.

### D6: Engine discovery order
`--engine <path>` → `SIPEXER_BIN` env → `<dir-of-sipx>/sipexer` → `PATH`. `sipx doctor` reports the
resolved path, executability, and `sipexer --version` output. Dial fails with a precise message if
no engine is found.

### D7: Raw flag passthrough
On `dial`, everything after `--` is appended verbatim to the engine argv. `--server`/`--from-user`
and profile options are translated to engine flags by `sipx`; precedence: explicit CLI flags beat
profile `ExtraDialFlags`, which beat profile defaults. (Engine flags are shaped like `./sipexer
-m REGISTER -a user:pass -f <from> -t <to> <server>` following sipexer's own conventions.)

### D8: `history alias <index> <name>`
Mutates an entry's `Alias` (indexing most-recent-first = `@0`). Alias namespace: number aliases
(profile scope) and history aliases (profile or global scope) share the lookup in D2; conflicts
report an error instead of silent overwrite.

## Risks / Trade-offs

- [Risk] Subprocess keeps engine unmodified but cannot share state/exit detail → Mitigation:
  propagate exit code + stream stderr; if more depth is ever needed, revisit importing the engine
  behind a build tag.
- [Risk] Alias ambiguity (number alias vs history alias) → Mitigation: fixed resolution order D2;
  user can prefix `@` for history-index or write the raw number to force plain dial.
- [Risk] Obfuscation is not encryption → Mitigation: documented caveat; OS file permissions.
- [Risk] Windows rename semantics → Mitigation: atomic-write fallback (remove+rename).
- [Risk] `--` passthrough quoting edge cases in shell → Mitigation: document that `dial ... -- ...`
  passes args verbatim; passthrough args are never re-quoted by `sipx`.
- [Trade-off] History recorded on attempt, not on success: simple, spec-covered; wire-level
  filtering can later be a flag.

## Migration Plan

No migration needed. `sipx` is a new binary; both binaries build from the same repo, config dir is
created lazily on first store write. Rollback for users: stop using `sipx`. No engine changes.

## Open Questions

None that would change the specs or the task breakdown. Held as implementation-time choices:
exact default server/from-user selection when neither is marked (first configured wins), and
whether `history alias` scope applies profile-scoped vs global (implemented as: alias resolves
against the selected profile's history first, then global entries).