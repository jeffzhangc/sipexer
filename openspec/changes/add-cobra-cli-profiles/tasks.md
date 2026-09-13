# Tasks: add-cobra-cli-profiles

Implements the `sipx` wrapped binary (profiles, history, dial) on Cobra/pflag, driving the
untouched `sipexer` engine via subprocess. Specs:
`openspec/changes/add-cobra-cli-profiles/specs/{cli-commands,phone-profiles,dial-history}/spec.md`.
Design: `openspec/changes/add-cobra-cli-profiles/design.md` (D0–D8).

## 1. Dependencies & scaffolding

- [x] 1.1 Add `github.com/spf13/cobra`, `github.com/spf13/pflag`, `golang.org/x/term` to go.mod
  (go get) and verify `go build ./...` succeeds
- [x] 1.2 Create `cmd/sipx/main.go` and `internal/store` skeleton packages; verify
  `go vet ./...` is clean
- [x] 1.3 Add a Makefile or build script that builds both `./sipexer` and `./cmd/sipx`; verify
  both binaries are produced

## 2. Engine discovery & subprocess execution

- [x] 2.1 Implement `engine.Locate` (`--engine` flag → `SIPEXER_BIN` env → same-dir `sipexer` →
  `PATH`) in `internal/engine`; verify with a unit test covering each rule and the missing-engine
  error
- [x] 2.2 Implement `engine.Run(ctx, bin, args...)` that streams stdout/stderr and forwards the
  engine exit code; verify with a unit test running a stub binary that exits 42 and another that
  prints to stderr
- [x] 2.3 Implement `sipx doctor` (reports discovered path, executability, `sipexer --version`
  output); verify against a real built `sipexer` in a temp dir

## 3. Store packages (JSON persistence)

- [x] 3.1 Implement `internal/store/config.go` (config-dir resolution order D4, atomic write with
  remove+rename fallback); unit test with an overridden `--config-dir`/env
- [x] 3.2 Implement `internal/store/profiles.go` — `Load`, `Save`, `Add`, `Edit`, `Remove`,
  `Show` with duplicate-name rejection and default-server/from-user-selection logic (first
  configured wins when none marked); typed call-default fields (D1b: pointers for int/bool so
  unset ≠ zero) and the `ExtraDialFlags`/`ExtraDialFlagsAfterDash` lists; unit tests cover
  add/duplicate/remove/edit and that an unset field round-trips as unset (nil pointer preserved)
- [x] 3.3 Implement `internal/store/history.go` — `Load`, `Save`, `Append` (cap, default 100,
  configurable), `Clear`; unit tests cover cap-eviction and persistence-across-reload
- [x] 3.4 Implement `internal/store/obfuscation.go` (XOR with hostname+username-derived key,
  Secret wrapper, `RedactString`); round-trip test + redaction test

## 4. Command tree

- [x] 4.1 Root command: `sipx` with `--engine`, `--config-dir`, `--version`, `--help`, and
  subcommands `phone`, `server`, `dial`, `history`, `doctor`; verify `sipx --help` lists them and
  exit code 0
- [x] 4.2 `phone add|edit|list|rm|show` — add/edit accept `--number <alias=user@domain>`,
  `--server <proto://host:port>>`, credentials via flags or TTY prompt (non-TTY → error guidance),
  and typed call-default options (`--sw`, `--cd`, `--rt`, `--timeout-*`, `--vl`, `--co`, `--com`,
  `--ex`, `--ua`, `--cu`, `--ct`, `--mb`, `--ti`, `--method`, `--xh`, `--extra-dial-flag`);
  verify `sipx phone add`, `list`, `show`, `rm` against a temp `--config-dir`
- [x] 4.3 `server add|rm|list <profile>` with default-server marking; verify server list shows
  default mark and removals persist
- [x] 4.4 `history list|clear|alias <index> <name>` (most-recent-first indexing, `@0` = newest;
  alias-collision error); verify list ordering, clear, alias assignment
- [x] 4.5 `dial [profile] [target]` — target resolution order D2 (raw → number-alias →
  history-alias → `@index`); assemble engine argv from the profile's typed call defaults (emit
  only non-nil pointer fields) + `ExtraDialFlags` + per-dial CLI overrides + passthrus; verify each
  branch with a stubbed engine (fixed exit 0) recording argv, plus history append and cap behavior
  and that unset profile fields emit no flag
- [x] 4.6 `dial` interactive picker (TTY): print profile history as choices + manual number +
  number aliases, read one line, dial selection; non-TTY without target → error naming `@<index>`,
  `<alias>`, `<number>` choices; verify with a pty-based test or manual
- [x] 4.7 `dial ... -- <raw engine args>` passthrough appended verbatim after resolution; verify
  a passthrough arg reaches the stub engine argv

## 5. Integration & docs

- [x] 5.1 End-to-end smoke: temp `--config-dir`, `phone add` a profile, `server add`, `dial` a
  local loopback (or stub-engine) target, `history`, `history alias`, re-`dial`; verify all state
  persists across invocations
- [x] 5.2 Build script parity check: `make` (or script) produces `sipexer` and `sipx`; `go vet
  ./...` and `go test ./...` pass
- [x] 5.3 Update `README.md` with `sipx` usage, config-dir layout, obfuscation caveat, and engine
  discovery rules; verify README commands against the built binaries