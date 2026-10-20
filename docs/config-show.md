# Config Show Command

The `config show` command displays the effective runtime configuration as read from flags, environment variables, and config files.

## Usage

```bash
# Human-readable output
glassbox config show

# JSON output with source annotations
glassbox config show --json
```

## Output

### Human-Readable Format

```
Active configuration:
{
  "rpc_url": "https://...",
  "network": "testnet",
  ...
}

Loaded from: /home/user/.glassbox/config.toml

Configuration sources:
  ✓ Environment variables (active)
  ✓ User config (active)
```

### JSON Format

```json
{
  "values": {
    "rpc_url": {
      "value": "https://test.example.com",
      "source": "environment"
    },
    "network": {
      "value": "testnet",
      "source": "file"
    }
  },
  "source_file": "/home/user/.glassbox/config.toml"
}
```

## Configuration Precedence

For a setting supplied by more than one source, the tested order is (highest
wins):

1. An explicit command-line value, when that command accepts a flag for the setting
2. Environment variables
3. Config files
4. Built-in defaults

The tests also verify that project-local values override home-directory values,
and that the XDG-style file overrides the legacy filename in the same
directory. Files are merged by key; finding a project file does not discard
settings from a lower-priority file. Environment variables override file
values. The CLI flag-over-environment case is tested for RPC tokens.

## Config File Locations

The loader reads these user and project files, merging later/higher-priority
values over earlier ones:

- `~/.Glassbox.toml` (home directory, legacy)
- `~/.glassbox/config.toml` (home directory, XDG-style)
- `.Glassbox.toml` (current directory, legacy)
- `.glassbox.toml` (current directory, XDG-style)

The loader also reads `/etc/Glassbox/config.toml`; current precedence tests do
not establish that file's position relative to the user and project files.

## Source Annotations

In JSON mode, each config value includes a `source` field:

| Source | Description |
|--------|-------------|
| `environment` | Set via `GLASSBOX_*` environment variable |
| `file` | Set in a config file |
| `default` | Built-in default value |

For a safe per-field inspection, use the explain report. Its tested redaction
replaces sensitive effective values with `[redacted]`; the selection below also
keeps the output focused on the value, source, and redaction status:

```bash
glassbox config show --explain --json \
  | jq '{active_file, fields: [.fields[] | {field, source, effective_value, redacted, default_applied}]}'
```

The environment bindings and their supported variable names are documented in
[Bindings and environment variables](bindings-environments.md).

## Migrating a Configuration File

`glassbox config migrate` upgrades a config file to the current schema version.
It accepts these flags:

| Flag | Description |
|------|-------------|
| `--path <file>` | Config file to migrate (defaults to the standard user config path) |
| `--dry-run` | Preview the migration without writing changes |
| `--backup` | Write a `.bak` copy of the original before migrating (default: true) |
| `--no-backup` | Skip the backup copy |

Migration changes local state, so preserve the original file before applying
anything. A safe sequence:

```bash
# 1. Back up the original file yourself
cp ~/.glassbox/config.toml ~/.glassbox/config.toml.bak

# 2. Inspect the current effective configuration and per-field sources
glassbox config show --explain

# 3. Preview the migration without writing
glassbox config migrate --dry-run

# 4. Apply the migration (writes a .bak copy by default)
glassbox config migrate

# 5. Review the result and validate
glassbox config show
```

Notes:

- Migration is idempotent. Running it again on an up-to-date file reports
  `Nothing to migrate` and leaves the file unchanged.
- Files written before schema versioning are treated as version 1.
- `--dry-run` is the only preview mechanism; it reports the changes that would
  be made without modifying the file.

## Troubleshooting Configuration

| Symptom | Likely source | Next action |
|---------|---------------|-------------|
| Invalid value rejected (e.g. `request_timeout` <= 0 or > 300, `max_trace_depth` out of range, invalid `network` or `log_level`) | A bad value in a config file or environment variable | Run `glassbox config show --explain` to locate the field and its source, then correct it |
| Unknown field warning | A key that is not part of the documented schema | Compare the file against the documented keys and remove or rename the entry |
| A setting does not take effect as expected | Precedence conflict (flag > env > file > default) | Run `glassbox config show --explain --json` to see the per-field `source` |
| `schema_version` newer than supported | Config written by a newer release | Run `glassbox config migrate` to upgrade the file |

See [Config Show Command](#config-show-command) above for inspection output and
[Bindings and environment variables](bindings-environments.md) for supported
variable names.

## Redacting Sensitive Values

`config show` already masks sensitive values: `rpc_token` and
`network_passphrase` are displayed as `[redacted]`. With `--explain`, each field
also carries a redaction status so you can tell which values were masked.

When sharing logs or diagnostics output, run the redaction script before
uploading:

```bash
scripts/redact-logs.sh <staging-dir>
```

Sensitive classes include RPC tokens, network passphrases, and any account
identifiers. Example of safe, synthetic output:

```json
{
  "rpc_url": { "value": "https://rpc.example.test", "source": "file" },
  "rpc_token": { "value": "[redacted]", "source": "environment" },
  "network_passphrase": { "value": "[redacted]", "source": "file" }
}
```

Never paste real tokens, passphrases, or account identifiers into issues or
chat; use placeholder values as above.