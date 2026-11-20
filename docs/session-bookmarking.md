# Session Bookmarking

Saved debug sessions are stored in `~/.Glassbox/sessions.db` with a
`schema_version` field that tracks the on-disk format. Glassbox automatically
upgrades older sessions when they are loaded and rejects sessions created by
a newer binary with actionable upgrade guidance.

Run `glassbox session doctor` to scan all saved sessions for schema or integrity
problems before resuming work.

Saved debug sessions can be bookmarked with a human-readable name:

```bash
glassbox session save --name payroll-bug
glassbox session list
glassbox session load payroll-bug
```

`session load` is an alias for `session resume`. Lookups accept exact session
IDs, bookmark names, unique ID prefixes, and transaction hashes.

Bookmark names are stored with the saved session snapshot metadata and must be
unique.

When a session is saved, Glassbox validates the persisted snapshot before writing
it to the session store. This includes any audit-chain metadata
(`audit_hash`, `audit_signature`, `previous_session_hash`), so malformed or
incomplete chain state is rejected up front with actionable hints.
Names are limited to **128 characters**; longer names are rejected with
an actionable error.

### Validation

When saving or exporting a session, Glassbox validates the session data before
persisting it:

- **Transaction hash** must be present and exactly 64 characters
- **Network** must be one of: `testnet`, `mainnet`, or `futurenet`
- **Status** must be one of `active`, `saved`, `resumed`, `recovered`,
  `expired` (auto-set to `active` if empty; `session save` always sets `saved`)
- **Name** must not exceed 128 characters
- **Horizon URL** is auto-populated from the network if not provided

If validation fails, the error message lists every issue found with a
remediation hint so you know exactly what to fix. Use `glassbox session
save --name <name>` to apply a bookmark after fixing the reported issues.

## Golden examples

The examples below are taken from the implementation in
`internal/cmd/session.go`, `internal/cmd/resource_lookup.go`, and
`internal/session/store.go`. Placeholders in `<angle brackets>` vary per run;
everything else is literal output. Errors are written to stderr by
`cmd/glassbox/main.go` as `Error: <err>`, and the exit code comes from
`ExitCodeFor` in `internal/cmd/exitcode.go` (see
[stable-error-codes.md](stable-error-codes.md#exit-code-taxonomy)).

> **Active-session scope:** the "active session" that `session save` persists
> is held in process memory (`SetCurrentSession` / `GetCurrentSession` in
> `internal/cmd/session.go`). `glassbox debug <tx-hash>` prints
> `Run 'Glassbox session save' to persist this session.`, but a *separate*
> `glassbox session save` invocation starts with no active session and fails
> with `SIMULATION_LOGIC_ERROR` (see [Save without an active session](#save-without-an-active-session)).
> The successful save path is currently exercised in-process by
> `internal/cmd/session_contract_test.go`.

### Preview a save (works on any checkout)

`--plan` has no side effects and does not need an active session:

```console
$ glassbox session save --name payroll-bug --plan
Execution Plan — session save
Built at: <RFC3339 UTC timestamp>

File operations (1):
  WRITE  <home>/.Glassbox/sessions.db — Session store (SQLite)

Outputs (1):
  stdout   Confirmation with session ID

Notes:
  ! Session ID: (auto-generated)
  ! Session name: payroll-bug

This is a dry-run plan. No network calls, signing, simulator invocations,
or file writes will be performed.
$ echo $?
0
```

With `--id my-debug-session`, the note reads `Session ID: my-debug-session`.

### Successful save

```text
Session saved: <session-id>
  Name: payroll-bug
  Transaction: <64-char tx hash>
  Network: testnet
  Created: <RFC3339 timestamp>
```

The `Name:` line is printed only when `--name` was given. Exit code `0`.

### Successful resume

`session resume` and its alias `session load` resolve the argument in this
order: exact ID, bookmark name, unique ID prefix, unique transaction-hash
prefix.

```console
$ glassbox session load payroll-bug
Session resumed: <session-id>
  Name: payroll-bug
  Transaction: <64-char tx hash>
  Network: testnet
  Created: <RFC3339 timestamp>
  Last accessed: <RFC3339 timestamp>
```

Optional lines follow only when the saved data contains them:
`  Pinned endpoint: <url>` (after `Network:`), a `Transaction Envelope:` block
with `Size: <n> bytes`, and a `Simulation Results:` block with
`Status:`, and when present `Error:`, `Events:`, `Logs:`. Exit code `0`.

### Failure paths

| Scenario | stderr (first line) | Exit |
|---|---|---|
| Save without an active session | `Error: SIMULATION_LOGIC_ERROR: no active session to save. Run 'Glassbox debug <tx-hash>' first` | 1 |
| Save fails validation | `Error: VALIDATION_FAILED: failed to save session: session data failed validation (<n> issue(s)):` | 1 |
| `--name` longer than 128 characters | `Error: session name is too long (<n> characters, max 128)` | 3 |
| Resume with an unknown ID or name | `Error: session "<input>" not found: resource not found; did you mean <closest>?` | 3 |
| Resume a session that fails the integrity check | `Error: session <id> failed integrity validation (<n> issue(s))` | 3 |

The name-length and lookup errors are plain Go errors, not `ErstError`, so
they carry no stable code and fall through to exit code `3`. Automation should
not key on their text.

#### Save without an active session

```console
$ glassbox session save --name payroll-bug
Error: SIMULATION_LOGIC_ERROR: no active session to save. Run 'Glassbox debug <tx-hash>' first
$ echo $?
1
```

#### Save fails validation

Every issue is listed with the field name used by `session.ValidateIntegrity`
(for example `TxHash`, `Network`, `AuditSignature`):

```text
Error: VALIDATION_FAILED: failed to save session: session data failed validation (1 issue(s)):
  1. [Network] network "devnet" is not a recognized Stellar network
     Hint: Accepted values are: testnet, mainnet, futurenet.
```

#### Resume not found

```text
Error: session "payrol-bug" not found: resource not found; did you mean payroll-bug?
Hint: run 'glassbox session list' to see all available sessions
```

When no candidate is close enough, the message ends at `resource not found`.

#### Integrity check failure on resume

The issue list goes to stderr before the error line:

```text
Session integrity check FAILED for <session-id>:
  1. [<Field>] <description>
     Hint: <remediation>

This session cannot be resumed safely.
To remove it:  glassbox session delete <session-id>
To re-debug:   glassbox debug <tx-hash> --network <network>
Error: session <session-id> failed integrity validation (1 issue(s))
```

Run `glassbox session doctor` to scan every saved session for the same issues.

### Verifying these examples

```bash
# Unit tests for validation, integrity, schema upgrade, and name lookup
go test ./internal/session/

# CLI-level save/resume/doctor contract tests
go test ./internal/cmd/ -run 'Session|ContractCLI'
```
