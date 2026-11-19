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

| Command | Flags | Envelope |
|---------|--------|----------|
| `debug` | `--json`, `--format json` | Yes |
| `trace` | `--output-json <file>`, `--format json` | Yes (writes to file) |
| `export` | `--format json` | Yes |
| `audit:sign` | `--json` | Yes (legacy output without the flag) |
| `audit:verify` | `--json` | Yes |
| `protocol:diagnose` | `--json`, `--format json` | Yes |
| `generate-bindings` | `--json`, `--format json` | Yes |
| `check-bindings` | `--json` | No (direct JSON) |
| `config show` | `--json` | No (direct JSON) |
| `bench` | `--json` | No (direct JSON) |
| `version` | `--json` | No (flat JSON only) |

**Envelope usage:** Most commands wrap their output in the schema envelope shown above. The following commands emit JSON directly without the envelope:
- `version --json` — flat JSON object with version metadata
- `config show --json` — direct JSON with configuration values
- `check-bindings --json` — direct JSON validation report
- `bench --json` — direct JSON benchmark results

**Format validation:** `--format` is validated at startup for all commands that accept it. Unsupported values (e.g. `--format yaml`) are rejected immediately with an explicit error listing the accepted values.

## Example

```bash
# debug with JSON output
glassbox debug --json <txhash>

# export snapshot as JSON envelope
glassbox export --snapshot ./state.snap.json --format json

# trace export as JSON (writes to file)
glassbox trace --export trace.json --format json execution.json

# config show as direct JSON (no envelope)
glassbox config show --json
```
