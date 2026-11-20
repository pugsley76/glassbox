# Session Export Portability

Closes #1171.

A session archive (`.gbx`) produced by `glassbox session share` is a ZIP file
containing `session.json`, `annotations.json`, and a signed `manifest.json`.
Not every field inside the archive travels between machines without
reconfiguration. This document identifies what is portable, what is
environment-specific, and how to handle each category when importing on a
different host.

---

## Portable fields

These fields are embedded in `session.json` and carry the same meaning on any
machine:

| Field | Notes |
|-------|-------|
| `id` | Stable session identifier; conflicts are resolved by `session import --on-conflict` |
| `tx_hash` | 64-char hex transaction identifier — fully portable |
| `network` | `testnet`, `mainnet`, or `futurenet` — portable |
| `status` | Lifecycle state (`saved`, `recovered`, etc.) — portable |
| `name` | Bookmark name; at most 128 characters — portable |
| `created_at` / `last_access_at` | UTC timestamps — portable |
| `envelope_xdr` / `result_xdr` / `result_meta_xdr` | Stellar XDR payloads — portable |
| `sim_request_json` / `sim_response_json` | Simulator I/O — portable |
| `audit_hash` / `audit_signature` / `previous_session_hash` | Audit chain — portable and verified on import |
| `schema_version` | Integer; see [schema compatibility](#schema-version-compatibility) |
| `provenance_json` | Append-only event log — portable; the importing host appends its own entry |
| `annotations` | Free-text notes stored in `annotations.json` — portable; `merge` policy combines them |
| `tags` | Short label array — portable |

---

## Environment-specific fields

These fields capture host-local configuration and may need reconfiguration after
a transfer:

| Field | Why it is local | What to do |
|-------|----------------|------------|
| `pinned_endpoint` | The RPC URL used on the exporting host | If the same endpoint is reachable, no action needed. Otherwise update with `glassbox session save --pin-endpoint <new-url>` after resuming |
| `env_fingerprint` | Captures OS, Go version, Stellar SDK version, and network passphrase at debug time | Informational only; not required for replay. Glassbox regenerates it on the first write after import |
| `horizon_url` | Auto-derived from `network` when not explicitly set | Automatically re-derived from `network` on import; no manual action needed unless you use a private Horizon |

---

## Transfer example

```bash
# On the exporting machine — share produces a .gbx archive
glassbox session share <session-id> --output payroll-bug.gbx

# Optionally preview conflicts before importing
glassbox session import payroll-bug.gbx --preview

# Import on the receiving machine (default: fail on conflict, non-destructive)
glassbox session import payroll-bug.gbx

# If the session ID already exists locally, rename it to keep both
glassbox session import payroll-bug.gbx --on-conflict rename

# Resume the imported session
glassbox session resume <session-id>
```

`--preview` is read-only and prints every conflicting field plus a portability
verdict without writing anything to disk.

---

## Redaction behavior

`glassbox session share` applies a redaction pass before packaging the archive.
By default the `full` profile is used, which removes or pseudonymizes:

- `env_metadata` (Horizon URL, endpoint) — removed
- `account_identifiers` — pseudonymized
- `contract_args` — removed if they contain secrets

Pass `--redact strict` to apply additional scrubbing, or `--redact none` to
export without redaction (not recommended for public sharing). The redaction
report is embedded in `manifest.json` so recipients can see exactly which
categories were sanitized.

See [`internal/session/redact.go`](../internal/session/redact.go) for the
complete field list.

---

## Schema version compatibility

The archive's `meta.json` records `schema_version` and `archive_version`.

| Scenario | Outcome |
|----------|---------|
| Importing host runs the same or newer Glassbox | Transparent automatic upgrade |
| Importing host runs an older Glassbox (archive schema too new) | Import rejected; upgrade Glassbox first |
| Session schema within the supported range | Auto-upgraded on first write |
| Session schema below `MinSupportedSchemaVersion` | Re-create with `glassbox debug <tx-hash>` |

Check compatibility before sharing with a team member on an older binary:

```bash
glassbox session import ./shared.gbx --preview
# Output includes: "Schema compatible: true"
```

For the full version history see
[docs/schema/session-migration-guide.md](schema/session-migration-guide.md).

---

## Session data directory and overrides

Closes #1302.

Imported sessions, and every other piece of local session state, live under one
data directory: `.Glassbox` in the user's home directory. The home directory
comes from Go's `os.UserHomeDir()`, which reads `$HOME` on Linux and macOS and
`%USERPROFILE%` on Windows.

| Path (relative to home) | Contents | Defined in |
|---|---|---|
| `.Glassbox/sessions.db` | Session store (SQLite, WAL mode) | `internal/session/store.go` (`DefaultDBPath`, `NewStore`) |
| `.Glassbox/active_session.json` | Crash-recovery checkpoint | `internal/session/checkpoint.go` |
| `.Glassbox/locks/<session-id>.lock`, `.lock.meta` | Advisory write locks | `internal/session/lock.go` |
| `.Glassbox/ui_state.db` | Viewer preferences | `internal/session/uistate.go` |
| `.Glassbox/viewer_state/` | Viewer sidecar files | `internal/session/viewer_state_file.go` |

The directory name is `.Glassbox` with a capital `G`. On case-sensitive file
systems, `~/.glassbox` is a different directory; that is where the XDG-style
config file `~/.glassbox/config.toml` lives (see
[`config show`](./config-show.md)). A code comment in `uistate.go` says
`~/.glassbox/ui_state.db`, but the code uses `.Glassbox`.

### Supported overrides

There is **no** `GLASSBOX_SESSION_DIR` environment variable and no global
`--session-dir` flag. These are the only ways to change where session state
lives:

| Mechanism | Scope | Notes |
|---|---|---|
| `HOME` (`USERPROFILE` on Windows) | All session state | Moves everything in the table above, along with home-relative config files. |
| `glassbox session gc --root <dir>` | `session gc` only | Opens `<dir>/sessions.db` instead of the default. |
| `GLASSBOX_VIEWER_STATE_DIR` | Viewer sidecars only | See [Viewer state persistence](./viewer-state-persistence.md#relocating-the-state-directory). |

### Example: isolated session store

Use a throwaway home directory to import and inspect a shared archive without
touching your real session store:

```bash
export HOME="$(mktemp -d)"
glassbox session import payroll-bug.gbx --preview
glassbox session import payroll-bug.gbx
ls "$HOME/.Glassbox"
# Expected: sessions.db (plus sessions.db-wal / sessions.db-shm while open)
```

Because `HOME` also moves the home-relative config files, set any values you
need, such as `GLASSBOX_RPC_URL` or `GLASSBOX_RPC_TOKEN`, as environment
variables in that shell.

### Example: `session gc --root`

`--root` must point at a directory whose final path element is exactly
`.Glassbox`, and it must not be the file-system root
(`ValidateGCRoot` in `internal/session/gc.go`):

```bash
# Success path: preview cleanup of another data directory
glassbox session gc --dry-run --root /srv/ci-home/.Glassbox

# Failure path: the final path element is not ".Glassbox"
glassbox session gc --dry-run --root /srv/ci-home
# Error: VALIDATION_FAILED: refusing to run cleanup against "/srv/ci-home" — expected a '.Glassbox' directory
#   Fix: pass the path to your Glassbox data directory, or omit --root to use the default
echo $?
# 1
```

`/srv/ci-home` is an example path. If the `.Glassbox` directory does not exist
yet, `session gc --root` creates it with an empty store.

---

## Related commands

| Command | Read-only? | Purpose |
|---------|-----------|---------|
| `glassbox session share` | No | Export a session as a `.gbx` archive |
| `glassbox session import --preview` | Yes | Preview conflicts without importing |
| `glassbox session import` | No | Import and persist to the local store |
| `glassbox session provenance <id>` | Yes | View the append-only provenance timeline |
| `glassbox session doctor` | Yes | Scan all local sessions for integrity issues |
