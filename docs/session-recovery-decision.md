# Session Recovery Decision Points

Closes #1170.

`glassbox session doctor` and `glassbox session recover` solve different
problems. Always start with a read-only inspection command before running
anything that mutates persisted state.

---

## Quick reference: symptom to command

| Symptom | First command (read-only) | Repair command | Deletes state? |
|---------|--------------------------|----------------|---------------|
| Want a health overview of all sessions | `glassbox session doctor` | — | No |
| Session behaves unexpectedly after resume | `glassbox session doctor` | `glassbox session delete <id>` + re-debug | Yes |
| `SESSION_LOCK_HELD` error on save | `glassbox session lock:status <id>` | `glassbox session recover` | No (removes stale lock only) |
| `SESSION_WRITE_CONFLICT` error on save | `glassbox session resume <id>` | `glassbox session save --force` | Yes (overwrites conflicting revision) |
| Glassbox crashed mid-session | `glassbox session lock:status <id>` | `glassbox session recover` | No |
| Checkpoint found but session not in store | `glassbox session list` | `glassbox debug <tx-hash>` | No |
| Corrupt checkpoint file | `glassbox session recover` (self-clears) | `glassbox debug <tx-hash>` | Clears checkpoint only |
| Session schema too old to load | — | `glassbox debug <tx-hash> --network <net>` | Creates new session |
| Session schema too new (binary too old) | — | Upgrade Glassbox, then `glassbox session resume <id>` | No |
| Want to view session without migrating | `glassbox session resume <id> --read-only` | — | No |

---

## What each command does to persisted state

### Read-only commands — safe to run at any time

| Command | What it reads | What it never touches |
|---------|--------------|----------------------|
| `glassbox session list` | Session metadata | Session data, lock files |
| `glassbox session doctor` | All sessions (integrity check) | Session rows; removes only stale temp files and stale lock files from dead processes |
| `glassbox session lock:status <id>` | Lock file metadata | Lock file itself, session data |
| `glassbox session resume <id> --read-only` | Session data | No migration, no write |
| `glassbox session migrate --dry-run` | Migration plan | Database untouched |
| `glassbox session import --preview` | Archive + local store | Nothing written |
| `glassbox session provenance <id>` | Provenance timeline | Nothing |

### Commands that repair state

These commands are safe to run after confirming the relevant read-only check,
but they do write to disk:

| Command | What it modifies | Reversible? |
|---------|----------------|------------|
| `glassbox session recover` | Removes stale lock; updates session status to `recovered` | Lock removal is not undoable; status write can be overwritten |
| `glassbox session resume <id>` | Updates session status and `last_access_at` | Yes — just a metadata update |
| `glassbox session save` | Writes session data to the store | Overwritten on next save |
| `glassbox session import` | Adds or merges session in the store | Depends on `--on-conflict` policy |
| `glassbox session migrate` | Upgrades session schema version | Not automatically reversible |

### Commands that delete state — destructive, confirm before running

| Command | What it deletes | Recoverable? |
|---------|----------------|-------------|
| `glassbox session delete <id>` | Entire session row from the store | No — re-run `glassbox debug` to recreate |
| `glassbox session save --force` | Overwrites the current on-disk revision | No — previous revision is lost |
| `glassbox session gc` | Expired sessions beyond retention policy | No |

---

## Recommended sequence

```
1. glassbox session doctor           # Read-only: is anything degraded?
        │
        ├─ All healthy ─────────────► Done
        │
        └─ Degraded session found
                │
                ▼
        2. glassbox session list     # Read-only: which session? what ID?
                │
                ▼
        3. glassbox session resume <id> --read-only
                │                   # Read-only: can the session be inspected?
                ├─ Loads cleanly ──► Decide whether to keep or replace it
                │
                └─ Schema / integrity error
                        │
                        ▼
                4. glassbox session delete <id>   # Destructive — last resort
                   glassbox debug <tx-hash> --network <net>
```

Destructive actions (`delete`, `save --force`) are never the default first
step. Only reach for them after the read-only inspection commands confirm
that the session cannot be repaired.

---

## Related documentation

- [Session locking and concurrent-writer handling](session-locking.md)
- [Session lock troubleshooting](session-locking.md#troubleshooting-lock-symptoms)
- [Session export portability](session-portability.md)
- [Session doctor output examples](session-doctor-examples.md)
- [Session schema migration guide](schema/session-migration-guide.md)
