## ADDED Requirements

### Requirement: Number alias resolution
`sipx` SHALL resolve a dial target that matches a configured registered-number alias, a history
entry alias, or the history-index syntax (`@N`, most recent first). Resolution order SHALL be:
number/URI (as typed, accepted as-is when it parses as a SIP URI or numeric), alias of a
registered number in the selected profile, history entry alias, then history index.

#### Scenario: Dial by number alias
- **WHEN** a user runs `sipx dial work boss` and the profile `work` has a registered number whose
  alias is `boss`
- **THEN** the call is addressed to that number (resolved via the alias)

#### Scenario: History alias preferred
- **WHEN** a user runs `sipx dial work boss` and `boss` matches an entry alias in the selected
  profile's history
- **THEN** the history entry's target and recorded settings are used
- **AND** the call is still routed through the selected profile

#### Scenario: Dial by history index
- **WHEN** a user runs `sipx dial work @0`
- **THEN** the most recent history entry of profile `work` is re-dialed

#### Scenario: Raw number accepted
- **WHEN** a user runs `sipx dial work 1001` and `1001` is not an alias or history reference
- **THEN** the call goes to number `1001`

### Requirement: Profile call defaults
A profile SHALL store a set of default call parameters applied to every dial through the profile.
This SHALL include a curated, typed set — session wait (`--sw`), call and ring durations (`--cd`,
`--rt`), timeouts (`--timeout`, `--timeout-connect`, `--timeout-write`), verbosity (`--vl`), color
output (`--co`, `--com`), expires (`--ex`), user agent (`--ua`), contact URI (`--cu`), content
type (`--ct`), extra headers (`--xh`), TLS insecure (`--ti`), message body (`--mb`), and method —
plus a free-form list of additional raw engine flag strings (`extra-dial-flags`). An unset option
SHALL be distinguishable from its zero value and SHALL NOT be passed to the engine, so the
engine's own default applies.

#### Scenario: Typed option stored and applied
- **WHEN** a user runs `sipx phone add work --sw 5000 --co --method INVITE`
- **THEN** profile `work` stores sessionwait=5000, color-output=true, method=INVITE, and a
  subsequent `sipx dial work 1001` passes `-sw 5000`, `-co`, `-i` (or equivalent) to the engine

#### Scenario: Unset option not transmitted
- **WHEN** a profile has no sessionwait stored
- **THEN** `sipx dial` does not pass any `-sw` flag, so the engine applies its own default

#### Scenario: Free-form extra flags
- **WHEN** a user runs `sipx phone add work --extra-dial-flag "--timer-t1 500" --extra-dial-flag "-fv expires:3600"`
- **THEN** both raw flag strings are appended to the engine argv on every `sipx dial work`

#### Scenario: CLI override beats stored default
- **WHEN** a profile stores `--sw 5000` and the user runs `sipx dial work 1001 --sw 2000`
- **THEN** the engine receives `-sw 2000`, not `-sw 5000`

### Requirement: Default (current) profile
The store SHALL support designating one profile as the default (the "current" profile) via
`sipx phone default [name]` (also settable to show the current one). A bare `sipx dial <target>`
with no profile argument SHALL dial through the default profile. When exactly one profile exists
it is the implicit default; when several exist and none is marked, `sipx dial <target>` SHALL fail
naming the `default` command. Removing the default profile SHALL promote another (first by name).
`sipx phone list` SHALL mark the default profile.

#### Scenario: Set and use default
- **WHEN** a user runs `sipx phone default work` and then `sipx dial 1001`
- **THEN** the call is placed through profile `work`

#### Scenario: Ambiguity without default
- **WHEN** two profiles exist, none marked default, and the user runs `sipx dial 1001`
- **THEN** `sipx` fails, instructing the user to run `sipx phone default <name>`

#### Scenario: Single profile implicit
- **WHEN** exactly one profile exists and no default was ever set
- **THEN** `sipx dial 1001` uses that profile

### Requirement: Profile model and persistence
A profile SHALL hold a name, multiple registered numbers (user[@domain], each optionally
aliased), multiple service addresses (one markable default), and optional auth credentials.
Profile data SHALL persist in JSON under the config directory; `phone add` SHALL reject a
duplicate name, `phone edit` SHALL preserve fields it does not change, and listings SHALL never
print a secret.

#### Scenario: Multiple numbers and servers
- **WHEN** a profile is created with two numbers and two servers
- **THEN** `sipx phone show` lists both numbers (with aliases) and both servers (default marked)

#### Scenario: Secret not shown
- **WHEN** a user runs `sipx phone list` or `show`
- **THEN** no password or HA1 value is displayed