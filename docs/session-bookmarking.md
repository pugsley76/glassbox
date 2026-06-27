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
