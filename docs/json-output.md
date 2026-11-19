# JSON CLI Output

Several Glassbox commands support machine-readable JSON for automation.

## Envelope format

When `--json` is set (or `--format json`), output is wrapped as:

```json
{
  "schema_version": "1.0",
  "glassbox_version": "0.0.0-dev",
  "generated_at": "2026-06-01T12:00:00Z",
  "command": "debug",
  "data": { }
}
```

## Error envelope

Errors use the same envelope with the failure under `data.error`
(`code`, `severity`, `message`, and optional `remediation` and `context`).
See [stable-error-codes.md](stable-error-codes.md#error-payload-fields) for
field definitions and which `context` keys are safe for log aggregation, and
[json-output-automation.md](json-output-automation.md) for filtering on
`error.code`.

## Commands

| Command | Flags |
|---------|--------|
| `debug` | `--json`, `--format json` |
| `export` | `--format json` |
| `audit:sign` | `--json` (schema envelope; legacy output without the flag) |
| `protocol:diagnose` | `--json` |
| `generate-bindings` | `--json`, `--format json` |
| `check-bindings` | `--json` |
| `config show` | `--json` |
| `bench` | `--json` |

**Format validation:** `--format` is validated at startup for all commands that accept it. Unsupported values (e.g. `--format yaml`) are rejected immediately with an explicit error listing the accepted values.

## Example

```bash
# debug with JSON output
glassbox debug --wasm ./contract.wasm --json

# export snapshot as JSON envelope
glassbox export --snapshot ./state.snap.json --format json
```
