# Report Command

`glassbox report` generates diagnostic reports from a completed execution trace.

---

## Synopsis

```sh
glassbox report --file <trace.json> [--format <fmt>] [--output <dir>]
```

`--file` is **required**. All other flags are optional.

---

## Flags

| Flag | Default | Description |
|---|---|---|
| `--file` | _(required)_ | Path to a JSON trace file produced by `glassbox debug --json` or `glassbox trace`. |
| `--format` | `html` | Output format: `text`, `json`, `html`, `pdf`, or `html,pdf`. Any other value is rejected early with the valid options listed. |
| `--output` | `.` | Output directory for the generated report(s). Must be a directory, not a file. |

---

## Validation

All inputs are validated before any I/O is performed:

| Condition | Error message |
|---|---|
| `--file` not set | `--file is required; provide the path to a JSON trace file` |
| `--format` is unknown | `invalid --format "…"; must be one of: html, pdf, html,pdf, json, text` |
| Trace file not found | `trace file not found: "…"` |
| Trace file is empty | `trace file "…" is empty` |
| Trace file is not valid JSON | Parse error with a hint to use `glassbox debug --json` |
| `--output` exists but is a file | `--output "…" exists but is not a directory` |
| `--output` directory does not exist | Directory is automatically created; error only if creation fails due to permissions |

---

## Output Directory Behavior

The `--output` directory is created automatically if it does not exist. Only the final path component is created; if the parent directory is missing, the command fails with a clear permissions or path error.

```sh
# This creates reports/ automatically if it doesn't exist
glassbox report --file trace.json --output reports/

# Nested directories are also created automatically
glassbox report --file trace.json --output build/artifacts/reports/
```

---

## Formats

### `text`

Plain-text diagnostic summary. Each issue is printed with severity, category, details, and a recommended action.

```sh
glassbox report --file trace.json --format text
```

### `json`

Machine-readable JSON array of `DiagnosticSummary` objects. Useful for CI pipelines and downstream tooling.

```sh
glassbox report --file trace.json --format json --output reports/
```

Each entry contains:

| Field | Description |
|---|---|
| `severity` | `critical`, `high`, `medium`, `low`, or `info` |
| `category` | `execution`, `wasm`, `authorization`, etc. |
| `summary` | Short issue title |
| `details` | Supporting context |
| `action` | Recommended next step |
| `step` | Step index (when available) |
| `contract_id` | Contract ID (when available) |

### `html` / `pdf` / `html,pdf`

Full execution report with summary, call stack, analytics, and risk assessment.

```sh
# HTML only
glassbox report --file trace.json --format html --output reports/

# PDF only
glassbox report --file trace.json --format pdf --output reports/

# Both in a single pass
glassbox report --file trace.json --format html,pdf --output reports/
```

---

## Examples

```sh
# Generate an HTML report (default)
glassbox report --file trace.json --output reports/

# Generate a plain-text report to stdout-friendly location
glassbox report --file trace.json --format text --output .

# Generate a JSON report for CI consumption
glassbox report --file trace.json --format json --output reports/

# Generate both HTML and PDF
glassbox report --file trace.json --format html,pdf --output reports/
```

---

## See Also

- [`glassbox debug`](./debug-command.md) — run a simulation and produce a trace with `--json`
- [`glassbox trace`](./debug-command.md) — stream execution trace events
