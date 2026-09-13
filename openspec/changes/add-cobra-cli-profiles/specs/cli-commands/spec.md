## ADDED Requirements

### Requirement: Binary separation
The project SHALL build two binaries: the original `sipexer` and a wrapped `sipx`. `sipexer`
SHALL remain functionally and behaviorally identical to its pre-change version.

#### Scenario: Both binaries build
- **WHEN** the project is built (e.g. `go build ./cmd/sipx` and `go build .` from the repo root)
- **THEN** both `sipexer` and `sipx` executables are produced

#### Scenario: Original binary unchanged
- **WHEN** the built `./sipexer` is run with a legacy flag set and target (`./sipexer udp:host:port`)
- **THEN** it behaves exactly as before this change: same flags, output, and exit codes

### Requirement: Engine discovery
`sipx` SHALL locate the `sipexer` engine binary using, in order of precedence: an explicit
`--engine <path>` flag; the `SIPEXER_BIN` environment variable; a file named `sipexer` in the
same directory as the `sipx` binary; and `PATH`.

#### Scenario: Engine found on PATH
- **WHEN** `sipx` is run with no `--engine` flag and `SIPEXER_BIN` unset, and `sipexer` is found
  next to `sipx` or on `PATH`
- **THEN** `sipx` uses that engine for its dials

#### Scenario: Explicit override
- **WHEN** `sipx --engine /opt/sipexer/bin/sipexer dial ...` is run
- **THEN** the specified engine path is used

#### Scenario: Missing engine error
- **WHEN** `sipx dial` is run and no engine binary can be discovered
- **THEN** `sipx` reports a clear error naming the discovery rule it tried, and exits non-zero
  without attempting a call

### Requirement: Subprocess execution
`sipx` SHALL execute the engine binary as a subprocess for SIP traffic, streaming the engine's
stdout/stderr to its own, and propagate the engine's exit code as `sipx`'s own exit code.

#### Scenario: Exit code forwarded
- **WHEN** `sipx dial work 1001` causes the engine subprocess to exit with code 0
- **THEN** `sipx` exits with code 0

#### Scenario: Engine failure propagated
- **WHEN** the engine exits non-zero (e.g. engine reports an error)
- **THEN** `sipx` exits with the same non-zero code and the user sees the engine's stderr

### Requirement: Raw flag passthrough
`sipx` SHALL allow the user to pass raw flags and destination arguments straight to the engine
after a `--` separator on `dial`, so every engine capability remains reachable from `sipx`.

#### Scenario: Passthrough after --
- **WHEN** a user runs `sipx dial work 1001 -- --verbosity 2`
- **THEN** `--verbosity 2` is appended verbatim to the engine's argument list

### Requirement: Doctor
`sipx` SHALL provide a `doctor` command that reports the discovered engine path, whether the
engine binary is executable, and its reported `--version` output.

#### Scenario: Report engine info
- **WHEN** a user runs `sipx doctor`
- **THEN** the engine path, executable status, and version are printed

### Requirement: Help and subcommand discoverability
`sipx` SHALL provide `--help` and `--version` on the root and every subcommand, listing available
subcommands and flags.

#### Scenario: Help shown
- **WHEN** a user runs `sipx --help` or `sipx dial --help`
- **THEN** usage text is printed and the process exits successfully