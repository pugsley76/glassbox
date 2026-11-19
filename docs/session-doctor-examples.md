# Session Doctor Output Examples

Closes #1173.

`glassbox session doctor` is a **read-only** diagnostic command. It scans every
persisted session in `~/.Glassbox/sessions.db` and reports integrity or schema
problems without modifying any session data. The only side-effects are the
removal of stale temp files and stale advisory lock files left by previously
crashed processes; neither action touches live session records.

---

## Healthy output

When every session passes its integrity check the command exits 0 and prints a
one-line summary:

```
Session store: 3 session(s) — all healthy.
```

No further action is required.

---

## Empty store

When no sessions have been saved yet:

```
Session store: no sessions found.
```

---

## Unhealthy output — missing transaction hash

The excerpt below is derived from the fixture used in
`TestRunStoreDiagnostics_DegradedSession_AppearsInReports` (see
`internal/session/store_diagnostics_test.go`). Local paths and generated IDs
are replaced with fixed placeholders.

```
Session store: 2 session(s), 1 healthy, 1 degraded — run 'glassbox session list' for details.

Degraded sessions (1):

Session sess-0a1b2c3d:
  1. [TxHash] transaction hash is missing or empty
     Hint: re-run 'glassbox debug <tx-hash> --network testnet' to recreate this session

1 session(s) failed diagnostics
```

**What this means:** the session row in the store has no `tx_hash`. The data
cannot be trusted and cannot be resumed safely.

**Recommended next step (read-only first):**

```bash
# Confirm which session is affected — read-only
glassbox session list
```

**Repair (deletes state — irreversible):**

```bash
glassbox session delete sess-0a1b2c3d
glassbox debug <tx-hash> --network testnet
```

---

## Unhealthy output — incomplete audit chain

The excerpt below corresponds to
`TestRunStoreDiagnostics_AuditChainIssue_AppearsInReports`.

```
Session store: 1 session(s), 0 healthy, 1 degraded — run 'glassbox session list' for details.

Degraded sessions (1):

Session sess-4d5e6f7a:
  1. [AuditSignature] audit_signature is absent but audit_hash and previous_session_hash are set
     Hint: the session audit chain is incomplete; delete and recreate the session to restore a valid chain

1 session(s) failed diagnostics
```

**What this means:** the session carries an `audit_hash` and a
`previous_session_hash` but no `audit_signature`. The chain cannot be verified
and the session is treated as degraded.

---

## Stale file cleanup (informational)

When stale temp or lock files are found the doctor prints extra lines after the
summary. These are automatic cleanup actions; they never touch live session
records:

```
Session store: 2 session(s) — all healthy.
Removed 1 stale temp/recovery-journal file(s) from ~/.Glassbox
Removed 2 stale advisory lock file(s) from ~/.Glassbox/locks/
```

---

## Action reference

| Output line | Read-only? | What it changes |
|-------------|-----------|-----------------|
| `all healthy` / `no sessions found` | Yes | Nothing |
| `N degraded` summary line | Yes | Nothing — just a count |
| `Removed N stale temp file(s)` | No | Deletes `~/.Glassbox/*.tmp` and similar recovery-journal files |
| `Removed N stale advisory lock file(s)` | No | Deletes `~/.Glassbox/locks/<id>.lock` for dead processes only |
| `glassbox session delete <id>` | No | Permanently removes the session row from the store |

The `doctor` command itself never deletes session rows. Deleting a degraded
session is a separate, explicit step that requires the user to run
`glassbox session delete`.
