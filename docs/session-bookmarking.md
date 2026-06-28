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
unique. Names are limited to **128 characters**; longer names are rejected with
an actionable error.

### Validation

When saving or exporting a session, Glassbox validates the session data before
persisting it:

- **Transaction hash** must be present
- **Network** must be one of: `testnet`, `mainnet`, or `futurenet`
- **Status** must be a recognized value (auto-set to `active` if empty)
- **Name** must not exceed 128 characters
- **Horizon URL** is auto-populated from the network if not provided

If validation fails, the error message lists every issue found with a
remediation hint so you know exactly what to fix. Use `glassbox session
save --name <name>` to apply a bookmark after fixing the reported issues.
