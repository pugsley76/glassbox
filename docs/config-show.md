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
3. Config files (loaded in the order documented in [Config File Locations](#config-file-locations))
4. Built-in defaults

The precedence tests verify that:
- Project-local values override home-directory values
- XDG-style filenames override legacy filenames in the same directory
- Environment variables override file values
- CLI flags override environment variables (tested for RPC tokens)

Files are merged by key; finding a higher-priority file does not discard
settings from lower-priority files. Only keys present in the higher-priority
file override those same keys in lower-priority files.

## Config File Locations

The loader reads config files in ascending priority order (later files override
earlier ones). The complete precedence order from lowest to highest priority is:

1. `/etc/Glassbox/config.toml` — system-wide defaults (lowest priority)
2. `~/.Glassbox.toml` — home directory, legacy filename
3. `~/.glassbox/config.toml` — home directory, XDG-style
4. `.Glassbox.toml` — current directory, legacy filename
5. `.glassbox.toml` — current directory, XDG-style (highest file priority)

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
[Environment Variables](environment-variables.md).