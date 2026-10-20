# Telemetry Metadata

Glassbox includes environment metadata in telemetry events to help diagnose usage patterns and troubleshoot issues.

## Environment Metadata

When telemetry is enabled (non-anonymized mode), the following metadata is included in command usage events:

| Field | Description | Example |
|-------|-------------|---------|
| `env.version` | CLI version | `"1.2.3"` or `"dev"` |
| `env.platform` | Operating system | `"linux"`, `"darwin"`, `"windows"` |
| `env.arch` | CPU architecture | `"amd64"`, `"arm64"` |
| `env.feature_flags` | Enabled feature flags | `["telemetry", "failover"]` |

## Privacy

Sensitive values are automatically excluded from telemetry:

- Configuration values containing tokens, passwords, or keys are redacted
- Transaction hashes are fingerprinted (hashed client-side) rather than transmitted raw
- File paths are reduced to basename only
- Long strings are truncated to 128 characters

## Anonymized Mode

When `--telemetry-anonymized` is enabled (or `GLASSBOX_TELEMETRY_ANONYMIZED=true`), environment metadata is not included in telemetry events. Only the command name is recorded.

## Enabling Telemetry

```bash
# Enable basic telemetry
glassbox --telemetry

# Enable with anonymization
glassbox --telemetry --telemetry-anonymized
```

Or via environment variables:

```bash
GLASSBOX_TELEMETRY=true
GLASSBOX_TELEMETRY_ANONYMIZED=false
```

## Verifying the Effective Telemetry State

After changing configuration or environment variables, you can check the effective
telemetry state without sending anything:

```bash
glassbox telemetry status
```

This command only reads local state (environment variables and the consent file).
It does not transmit any data.

The output reports the following fields:

| Field | Description |
|-------|-------------|
| `Source` | Where the effective setting came from (environment variable or consent file) |
| `Enabled` | Whether telemetry is currently enabled |
| `Updated` | When the consent record was last updated |

### Precedence

The effective level is resolved in this order (matching `ResolveConsentLevel`):

1. `GLASSBOX_TELEMETRY_LEVEL` environment variable (`anonymous`, `full`, or `disabled`)
2. `GLASSBOX_TELEMETRY` boolean environment variable
3. Consent file at `~/.Glassbox/telemetry_consent.json`
4. Default: disabled

### Example: verifying an override in a fresh shell

```bash
export GLASSBOX_TELEMETRY=true
glassbox telemetry status
```

`Source` reports the environment variable and `Enabled` reflects the override.
Unsetting the variable (`unset GLASSBOX_TELEMETRY`) falls back to the consent file.

## Sanitization Example

The following synthetic before/after payload illustrates how `RedactPayload` and
`SanitizeValue` transform event metadata. All hashes, paths, and contract IDs below
are fabricated.

Before:

```json
{
  "command.name": "contract invoke",
  "tx_hash": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2",
  "contract_id": "CAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
  "file_path": "/home/example/projects/demo/contracts/token.wasm",
  "command_arg": "--network testnet",
  "secret_token": "synthetic-secret-value",
  "note": "this is a long synthetic note that keeps going and going until it exceeds the truncation limit set by the sanitizer"
}
```

After:

```json
{
  "command.name": "contract invoke",
  "tx_hash": "[redacted]",
  "contract_id": "[redacted]",
  "file_path": "[redacted]",
  "command_arg": "[redacted]",
  "secret_token": "[redacted]",
  "note": "this is a long synthetic note that keeps going and going until it exceeds the truncation limit set by the sanitizer..."
}
```

Field-by-field:

| Field | Outcome | Notes |
|-------|---------|-------|
| `command.name` | Retained | Not matched by redaction patterns |
| `tx_hash` | Removed | Key matches redacted key patterns (`hash`, `tx`) |
| `contract_id` | Removed | Key matches redacted key patterns (`contract`) |
| `file_path` | Removed | Key matches redacted key patterns (`path`, `file`) |
| `command_arg` | Removed | Key matches redacted key patterns (`arg`) |
| `secret_token` | Removed | Key matches redacted key patterns (`secret`, `token`) |
| `note` | Transformed | Long strings are truncated at 128 characters with `...` |

Identifier values that are not under a redacted key are fingerprinted client-side as
`sha256:<25 hex chars>` (32 characters total) rather than transmitted raw. Paths that
are retained are reduced to basename only.

These behaviors apply only to fields covered by the tested redaction patterns; they
do not constitute a guarantee for arbitrary field names. See
[Telemetry Registry](telemetry-registry.md) for the tracked fields and the sampling
section above for how events are sampled.

## Disabling Telemetry

Telemetry is opt-in and can be disabled temporarily or persistently.

| Scope | Method | Effect |
|-------|--------|--------|
| Temporary (current shell) | `export GLASSBOX_TELEMETRY=false` | Overrides the consent file for this shell session |
| Temporary (current shell) | `export GLASSBOX_TELEMETRY_LEVEL=disabled` | Overrides all other settings for this shell session |
| Persistent | `glassbox telemetry disable` | Writes an opt-out record to `~/.Glassbox/telemetry_consent.json` |

Precedence, as established by `ResolveConsentLevel`:

1. `GLASSBOX_TELEMETRY_LEVEL` environment variable
2. `GLASSBOX_TELEMETRY` boolean environment variable
3. Consent file at `~/.Glassbox/telemetry_consent.json`
4. Default: disabled

To confirm the effective state after any change, run `glassbox telemetry status`.
This command only reads local state and does not transmit any data.

## Feature Flags

The following feature flags are currently tracked:

- `telemetry` - When telemetry is enabled
- `crash_reporting` - When crash reporting is enabled
- `failover` - When multi-RPC failover is configured
- `multi_rpc` - When multiple Soroban RPC URLs are configured