## ADDED Requirements

### Requirement: History recording
Each dial made through `sipx` SHALL be recorded in the history store with: the target number/URI
(or the resolved raw number when an alias was used), the profile used, the service address used,
the from-number used, and the timestamp. At most ONE entry SHALL be kept per
(profile, from-number, target) triple: re-dialing a known triple updates the existing entry
(refreshing timestamp and server, preserving any assigned alias) and moves it to the most-recent
position, rather than appending a duplicate. The store SHALL be capped at a configurable maximum
number of entries (default 100), evicting the oldest first.

#### Scenario: Dial recorded
- **WHEN** a user runs `sipx dial work 1001`
- **THEN** an entry with target `1001`, profile `work`, the chosen server, and a timestamp is
  appended

#### Scenario: Re-dial updates in place
- **WHEN** a user dials the same triple (profile, from-number, target) again
- **THEN** the store still contains a single matching entry, now the most recent, with the alias
  preserved

#### Scenario: Cap enforced
- **WHEN** history has the configured maximum entries and a new dial occurs
- **THEN** the oldest entry is removed and the new one is appended

### Requirement: Alias assignment
`sipx` SHALL support assigning a user-chosen alias to a history entry (by its index), so it can be
re-dialed by alias or shown nicely in listings.

#### Scenario: Assign alias
- **WHEN** a user runs `sipx history alias 3 boss`
- **THEN** the third (most recent-first) entry is labeled `boss` and `sipx history` shows that
  alias
- **AND** subsequent dials using `boss` target that entry

#### Scenario: Alias collision
- **WHEN** a user assigns an alias that already exists among history levels or the profile's
  number aliases
- **THEN** `sipx` reports the conflict and does not overwrite silently

### Requirement: History listing
`sipx history` SHALL list entries most recent first, showing index, timestamp, profile, server,
from-number, target, and a display of the aliases that resolve to the entry. `sipx history clear`
SHALL empty the store with confirmation.

#### Scenario: List history
- **WHEN** a user runs `sipx history`
- **THEN** entries are printed most recent first with the required fields

#### Scenario: Clear history
- **WHEN** a user runs `sipx history clear`
- **THEN** the store is emptied and the CLI confirms

### Requirement: Per-profile history suggestion
When the user selects a profile and has not given a target, `sipx dial` SHALL list that profile's
own history entries as the suggested candidates for reuse (most recent first), so the user can
pick one.

#### Scenario: Suggestions shown
- **WHEN** a user runs `sipx dial work` with no target and profile `work` has history
- **THEN** `sipx` prints the profile's recent history and prompts for a choice or a raw number

### Requirement: History persistence
History SHALL persist across invocations in a JSON file under the same config directory as phone
profiles.

#### Scenario: Persists across restart
- **WHEN** a user dials, exits, and runs `sipx history` again
- **THEN** the entry is still listed