# Trace Export Validation and Diagnostics

The `glassbox debug` and `glassbox trace` commands include comprehensive validation and diagnostic capabilities for trace export operations. This document describes the validation checks, error handling, and troubleshooting guidance for both commands.

---

## Overview

Trace export validation occurs at multiple stages:

1. **Pre-flight validation** — CLI flag validation before any simulation or file load
2. **Pre-export validation** — Trace data and configuration validation before export
3. **Format compatibility** — Format-specific checks for data compatibility
4. **Export execution** — File system and I/O validation during write

---

## Pre-flight Validation (CLI Flags)

The debug command validates trace-related flags in `PreRunE` before any network or simulator operations:

### `--trace-verbosity`

**Valid values:** `summary`, `normal`, `verbose`

**Validation:**
- Must be one of the three supported values (case-insensitive)
- Checked at parse time before any execution

**Error example:**
```
invalid --trace-verbosity "ultra" — must be one of: summary, normal, verbose
  Fix: use --trace-verbosity normal (default), summary (minimal), or verbose (detailed)
```

### `--format`

**Valid values:** `text`, `json`, `html`, `markdown` (or `md`)

**Validation:**
- Must be one of the supported export formats
- Checked before simulation begins

**Error example:**
```
invalid trace export format "yaml" — must be one of: text, json, html, markdown
  Fix: use --format html (interactive), json (machine-readable), markdown (shareable), or text (CLI output)
```

### `--trace-output`

**Validation:**
- Must be a file path, not a directory path (no trailing `/` or `\`)
- Cannot contain null bytes
- Path traversal sequences (`..`) trigger a security warning
- Parent directory must exist or be creatable

**Error examples:**
```
--trace-output "./traces/" looks like a directory path; provide a full file path
  Fix: specify a complete file path (e.g. ./traces/trace.html or ./output/trace.json)
  Example: glassbox debug --trace-output ./traces/debug-$(date +%Y%m%d).html <tx-hash>
```

```
--trace-output "../../../etc/passwd" contains directory traversal sequences (..)
  Fix: use absolute paths or relative paths without '..' for security
  Example: use './output/trace.html' instead of '../output/trace.html'
```

---

## Pre-export Validation

Before attempting to write a trace export, the system validates all export parameters using comprehensive validation functions: `ValidateTraceExportParams()` and `ValidateTraceFormatCompatibility()`.

### Trace Data Validation (ValidateTraceExportParams)

**Checks:**
- Trace object is not nil
- Trace contains at least one execution state
- Transaction hash is present and non-empty
- Start and end times are valid (not zero, end >= start)
- Event types are recognized (unrecognized types trigger warnings)

**Error example:**
```
trace has no execution states — empty trace cannot be exported
  Fix: verify that the trace was captured correctly and contains at least one step
  Tip: check that the traced transaction actually executed any code
```

Another example with time validation:

```
trace end time is before start time — invalid temporal ordering
  Fix: verify the trace timestamps were recorded correctly
  Start: 2026-01-02T15:04:05Z, End: 2026-01-02T15:04:00Z
```

### Format and Path Validation

**Checks:**
- Export format is not empty and is supported
- Output path is not empty
- Output path is not a directory
- Output path does not contain invalid characters

**Error example:**
```
export format is empty — must specify one of: html, markdown, json, text
  Fix: provide --format html (default), markdown, json, or text
```

### Export Options Validation

**Checks:**
- Comment count does not exceed 100
- Individual comment length does not exceed 10,000 characters
- Session metadata keys and values are valid strings

**Error example:**
```
too many comments (150) — maximum is 100 comments per trace export
  Fix: reduce the number of comments or split into multiple exports
```

---

## Format Compatibility Checks

Format compatibility validation is performed by `ValidateTraceFormatCompatibility()` to ensure trace data is suitable for the target export format. Each format has specific requirements and constraints.

### JSON Format

**Requirements:**
- All trace data must be JSON-serializable
- No circular references
- Step indices must be sequential and match array position

**Error example:**
```
trace step mismatch at position 5: expected step 5 but got 10 — trace may be corrupted
```

### HTML Format

**Compatibility constraints:**
- Traces with >50,000 steps may cause browser rendering to be slow or unresponsive
- Individual error messages >1MB will cause rendering issues

**Error example:**
```
trace has 60000 steps — too large for HTML export (browser may become unresponsive)
  Fix: use --format json for large traces or filter the trace verbosity
  Alternatively: use --trace-verbosity summary to reduce output size
```

### Markdown Format

**Compatibility constraints:**
- Traces with >10,000 steps produce very large markdown files (>1MB)
- Code fence markers (```) in error messages should be reviewed for formatting

### Text Format

**Requirements:**
- Most permissive format
- No special constraints

---

## Export Execution Errors

Errors that occur during file write operations include detailed remediation:

### Directory Creation Failure

```
failed to create trace export directory: permission denied
  Directory: /restricted/traces
  Fix: ensure you have write permissions to the parent directory
  Or choose a different output path with --trace-output
```

### File Write Failure

```
failed to write trace export file: no space left on device
  Path: /tmp/trace.html
  Fix: ensure you have write permissions and sufficient disk space
  Check: ls -la /tmp
```

### Template Rendering Failure

```
failed to generate HTML trace: template execution error: ...
  This may indicate invalid trace data or a template rendering error
  Check that all trace fields are properly populated
```

---

## Validation in Dry-Run Mode

When using `--dry-run`, trace output configuration is validated without executing the simulation:

```sh
glassbox debug --dry-run --trace-output ./invalid/ --network testnet <tx-hash>
```

**Output:**
```
Additional environment checks:
[FAIL] Trace output validation failed: --trace-output "./invalid/" looks like a directory path
       Fix: ensure trace output path is valid and format is correct
```

---

## Multiple Validation Errors

When multiple validation errors are detected, all failures are reported together so they can be fixed in a single pass:

```
3 trace input validation error(s):
  1. invalid --trace-verbosity "ultra" — must be one of: summary, normal, verbose
     Fix: use --trace-verbosity normal (default), summary (minimal), or verbose (detailed)
  2. invalid trace export format "yaml" — must be one of: text, json, html, markdown
     Fix: use --format html (interactive), json (machine-readable), markdown (shareable), or text (CLI output)
  3. --trace-output "./traces/" looks like a directory path; provide a full file path
     Fix: specify a complete file path (e.g. ./traces/trace.html or ./output/trace.json)
```

---

## Best Practices

### 1. Use Dry-Run for Validation

Always run with `--dry-run` first when setting up trace export in CI/CD:

```sh
glassbox debug --dry-run \
  --network testnet \
  --trace-output ./artifacts/trace.html \
  --format html \
  <tx-hash>
```

### 2. Choose the Right Format

- **HTML**: Interactive viewing in browsers, best for manual analysis
- **JSON**: Machine-readable, best for CI/CD and automated processing
- **Markdown**: Shareable in chat/issues, best for collaboration
- **Text**: Plain CLI output, best for simple logging

### 3. Organize Output Paths

Use dated directories for trace exports:

```sh
glassbox debug \
  --trace-output "./traces/$(date +%Y-%m-%d)/${TX_HASH}.html" \
  --format html \
  $TX_HASH
```

### 4. Validate Before Large Exports

For traces with many steps, validate format compatibility first:

```sh
# Check trace size first
glassbox debug --format json $TX_HASH | jq '.States | length'

# Use JSON for very large traces (>1000 steps)
if [ $STEPS -gt 1000 ]; then
  glassbox debug --format json --trace-output ./trace.json $TX_HASH
else
  glassbox debug --format html --trace-output ./trace.html $TX_HASH
fi
```

---

## Troubleshooting

### "Trace contains no steps"

**Cause:** The simulator did not produce any diagnostic events.

**Solutions:**
1. Verify the transaction hash is correct
2. Run `glassbox doctor` to check simulator compatibility
3. Check that the transaction actually executed on the network
4. Ensure the simulator binary is up-to-date

### "Trace step mismatch"

**Cause:** The trace data structure is corrupted.

**Solutions:**
1. Re-run the debug command to regenerate the trace
2. Check for filesystem corruption if using `--save-snapshots`
3. Verify the simulator version matches the CLI version

### "Very large arguments"

**Cause:** Contract arguments exceed browser rendering limits for HTML export.

**Solutions:**
1. Use JSON format instead: `--format json`
2. Filter the trace to specific event types
3. Use `--trace-verbosity summary` for less detail

### "Permission denied"

**Cause:** Insufficient write permissions to the output directory.

**Solutions:**
1. Choose a different output path with write permissions
2. Create the output directory manually with correct permissions
3. Check filesystem mount options (read-only mounts)

---

## `glassbox trace` Command Export Validation

The `glassbox trace` command validates its export-related flags in `PreRunE` before loading or processing any trace file.

### `--export-format`

**Valid values:** `html`, `markdown` (or `md`), `json`, `text`

**Validation:**
- Only checked when `--export` is also provided
- Must be one of the four supported values (case-insensitive)
- Empty/default value (`html`) is always accepted

**Error example:**
```
invalid --export-format "yaml" — must be one of: html, markdown, json, text
  Fix: use --export-format html (interactive), markdown (shareable), json (machine-readable), or text (plain)
```

### `--export`

**Validation:**
- Path must not end with `/` or `\` (directory path guard)
- Cannot be combined with `--print`
- Cannot be combined with `--export-markdown`

**Error example:**
```
--export "./traces/" looks like a directory path; provide a full file path
  Fix: specify a filename (e.g. --export ./traces/output.html)
  Example: glassbox trace --export ./traces/report.html execution.json
```

```
cannot specify both --export and --print
  Fix: use --export to write to a file, or --print to output to stdout — not both
```

### `--export-markdown`

**Validation:**
- Path must not end with `/` or `\`
- Cannot be combined with `--export`

**Error example:**
```
--export-markdown "./reports/" looks like a directory path; provide a full file path
  Fix: specify a filename (e.g. --export-markdown ./traces/report.md)
```

### `--output-json`

**Validation:**
- Path must not end with `/` or `\`
- Produces a deterministic JSON envelope with `schema_version`, `generated_at`, and a nested `trace` object

**Schema version:**
The output envelope always embeds the current `schema_version` (e.g. `"1.0"`). The version is defined by the `CurrentJSONSchemaVersion` constant in the trace package — it is never hardcoded at call sites, so all paths stay in sync automatically when the schema evolves.

**Loading files back:**
`glassbox trace <file>` can load files written by `--output-json`. The loader detects the `schema_version` string envelope and validates it before parsing:

```
unsupported schema_version "99.0" in trace file "trace.json"
  This binary supports schema versions: "1.0"
  Fix: re-export the trace with the current CLI version, or upgrade Glassbox
```

Files written by a slightly older minor version produce a deprecation warning but load successfully:

```
Warning: trace file "old-trace.json" uses schema_version "0.9"; current is "1.0"
  Consider re-exporting with the current CLI for full compatibility
```

### `--export-svg`

**Validation:**
- Path must not end with `/` or `\`
- The trace must contain diagnostic events (error if empty, with remediation)

**Error example:**
```
no diagnostic events found in trace — call graph cannot be generated
  Possible causes:
    - The trace was captured without diagnostic events
    - The transaction did not call any contracts
  Fix: re-run with a transaction that includes contract calls
  Tip: use --trace-verbosity verbose when capturing the trace for maximum detail
```

### `--trace-verbosity`

**Valid values:** `summary`, `normal`, `verbose`

**Validation:**
- Must be one of the three supported values

**Error example:**
```
invalid --trace-verbosity "extreme" — must be one of: summary, normal, verbose
  Fix: use --trace-verbosity normal (default), summary (minimal), or verbose (detailed)
```

### `--annotations`

**Validation:**
- File must exist on disk

**Error example:**
```
--annotations: file not found: "/path/to/annotations.json"
  Fix: provide a valid path to an annotations JSON file
```

### `--gas-model`

**Validation:**
- File must exist on disk

**Error example:**
```
--gas-model: file not found: "/path/to/gas.json"
  Fix: provide a valid path to a gas model JSON file
```

### `--meta`

**Validation:**
- Every value must be in `key=value` format
- Key must not be empty

**Error example:**
```
--meta value "no-equals-sign" is not in key=value format
  Fix: supply metadata as key=value pairs, e.g. --meta env=testnet --meta version=1.2
```

### Multiple Validation Errors (`glassbox trace`)

All failures are collected and reported together:

```
2 trace command validation error(s):
  1. invalid --export-format "yaml" — must be one of: html, markdown, json, text
     Fix: use --export-format html (interactive), markdown (shareable), json (machine-readable), or text (plain)
  2. --export "./traces/" looks like a directory path; provide a full file path
     Fix: specify a filename (e.g. --export ./traces/output.html)
```

### Trace File Not Found (`glassbox trace`)

When the trace file argument does not exist:

```
trace file not found: "execution.json"
  Fix: verify the path is correct and the file exists
  Tip: trace files are produced by 'glassbox debug --trace-output <file>'
```

### No Trace File Supplied

```
trace file is required
  Usage: glassbox trace <trace-file>
  Or:    glassbox trace --file <trace-file>
  Run 'glassbox trace --help' for all available options
```

---

## Schema Stability and Upgrades

The `--output-json` flag produces a structured envelope with an explicit `schema_version` field. This section explains how Glassbox handles schema versioning, loading older files, and detecting incompatible versions.

### Schema Version Format

Schema versions use `MAJOR.MINOR` notation (e.g. `"1.0"`):

- **MAJOR** changes indicate breaking structural changes to the envelope or field layouts. Files from a different major version cannot be loaded.
- **MINOR** changes add new optional fields. Files from older minor versions load with a deprecation warning. Files from newer minor versions that are explicitly listed in `SupportedJSONSchemaVersions` also load successfully.

The current schema version constant is `CurrentJSONSchemaVersion = "1.0"`.

### Two JSON Envelope Formats

Glassbox uses two different JSON shapes depending on the export path:

| Flag | Envelope shape | Version field |
|------|----------------|---------------|
| `--output-json` | `{"schema_version":"1.0","generated_at":"...","trace":{...}}` | String: `"1.0"` |
| `--export --format json` | `{"version":{"major":1,"minor":0,"patch":0},"trace":{...}}` | Semver object |

`glassbox trace` detects and handles both shapes automatically. The detection is done by probing for the `schema_version` string key (ExportJSON shape) versus the `version` object key (VersionedTrace shape).

### Unsupported Schema Version Error

```
unsupported schema_version "99.0" in trace file "trace.json"
  This binary supports schema versions: "1.0"
  Fix: re-export the trace with the current CLI version, or upgrade Glassbox
```

### Older Minor Version Warning

```
Warning: trace file "old-trace.json" uses schema_version "0.9"; current is "1.0"
  Consider re-exporting with the current CLI for full compatibility
```

### Legacy File Warning (No Envelope)

Files produced before the schema envelope was introduced load with a warning:

```
Warning: loaded legacy trace format (no version info)
  Consider re-exporting with current version for full compatibility
```

### ExportJSON Schema Validation Function

`ValidateJSONSchemaVersion(version string) error` can be called independently to validate any schema version string before file I/O. It rejects:

- Empty or whitespace-only strings
- Strings not in `MAJOR.MINOR` format
- Non-numeric components
- Version strings not present in `SupportedJSONSchemaVersions`

All errors include a `Fix:` hint and reference the current expected version.

---

## Path Normalization and Safety

All trace export output paths go through a security-aware validation layer before any file I/O begins. This section describes what is checked, what errors are produced, and how the checks differ from a simple trailing-slash guard.

### What is validated

Every output path flag (`--export`, `--output-json`, `--export-svg`, `--export-markdown`, `--snapshot`) and every input path flag (`--annotations`, `--gas-model`) is now processed through:

1. **Null-byte rejection** — paths containing `\x00` are rejected immediately (shell injection risk)
2. **`filepath.Clean` + `filepath.Abs`** — resolves `.` and `..` components to their absolute form before any existence check
3. **Symlink resolution** — `filepath.EvalSymlinks` is called so that a symlink pointing outside an allowed root is caught, not just the raw string
4. **Existing-directory guard** — if the path already refers to a directory on disk, the write is rejected with a message that includes the flag name and a suggested filename

For input paths the validator additionally checks that the file exists and is not a directory.

### Error message format

All path errors include the flag name (`--export`, `--output-json`, etc.) so the failing flag is unambiguous:

**Null byte:**
```
--export: path contains null bytes and cannot be used: "/path/to/trace\x00.html"
```

**Existing directory:**
```
--output-json: "/traces" is a directory; provide a full file path (e.g. "/traces/output.json")
```

**Missing input file:**
```
--annotations: file not found: "/path/to/annotations.json"
  Check that the path is correct and the file exists
```

**Path traversal (`--trace-output` via `ValidateTraceInputs`):**
```
--trace-output "../../../etc/passwd" contains directory traversal sequences (..)
  Fix: use absolute paths or relative paths without '..' for security
  Example: use './output/trace.html' instead of '../output/trace.html'
```

### The traversal detection fix

The old traversal check used `strings.Contains(path, "..")` which produced false positives for filenames that legitimately contain double dots (e.g. `my..trace.html`) and could miss Windows-style traversal paths.

The new check uses `filepath.Clean` first:

```go
cleaned := filepath.Clean(outputPath)
if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
    // traversal detected
}
```

This correctly:
- Accepts `my..trace.html` (double dot in filename, not a traversal component)
- Rejects `../trace.html` and `../../etc/passwd` after cleaning
- Handles both POSIX (`/`) and Windows (`\`) separators

### Coverage by command

| Flag | Command | Validator |
|------|---------|-----------|
| `--export` | `trace` | `ValidateOutputPath` → `NormalizePath` |
| `--output-json` | `trace` | `ValidateOutputPath` → `NormalizePath` |
| `--export-svg` | `trace` | `ValidateOutputPath` → `NormalizePath` |
| `--export-markdown` | `trace` | `ValidateOutputPath` → `NormalizePath` |
| `--annotations` | `trace` | `ValidateInputPath` → `NormalizePath` |
| `--gas-model` | `trace` | `ValidateInputPath` → `NormalizePath` |
| trace file argument | `trace` | `ValidateInputPath` → `NormalizePath` |
| `--snapshot` (export) | `export` | `ValidateOutputPath` → `NormalizePath` |
| `--audit-log` | all | `ValidateOutputPath` → `NormalizePath` |
| `--trace-output` | `debug` | `ValidateTraceInputs` + `ValidateDebugOutputPaths` |
| `--save-snapshots`, `--export-svg` | `debug` | `ValidateDebugOutputPaths` → `NormalizePath` |

---

## Snapshot Reliability Validation

Before exporting traces that depend on snapshot data (sandboxed replay, step
navigation), Glassbox validates snapshot coverage using
`ValidateSnapshotForExport`. This surfaces snapshot problems early — before
the expensive export operation begins — so users get a clear, actionable error
rather than a partial or silently broken export.

### Validation checks

| Condition | Error produced |
|-----------|---------------|
| Simulation hit an OOM condition | Snapshot capture failed due to memory pressure |
| Steps executed but zero snapshots captured | No snapshots were captured |
| Snapshot count far below expected coverage | Sparse snapshot coverage warning |

### OOM error

```
snapshot capture failed due to memory pressure (OOM) — trace export may be incomplete
  The simulation ran out of memory before all snapshots could be saved.
  Fix: re-run with a smaller transaction or increase the simulator memory limit
  Tip: use --trace-verbosity summary to reduce memory usage during capture
```

### No snapshots captured

```
no snapshots were captured during simulation (250 steps executed, 0 snapshots) —
sandboxed replay and step-navigation will be unavailable
  Possible causes:
    - Snapshot interval is set too high (no step hit the interval)
    - Simulator version does not support snapshot capture
  Fix: lower --snapshot-interval or re-run with a simulator that supports snapshots
  Tip: run 'glassbox doctor' to check simulator snapshot support
```

### Sparse snapshot coverage

```
sparse snapshot coverage: 2 snapshot(s) for 1000 steps (expected at least 5) —
step-navigation may jump large gaps
  Fix: lower --snapshot-interval to capture more frequent snapshots
  Current coverage: 1 snapshot per ~500 steps
```

### Coverage threshold

The validator expects at minimum **1 snapshot per 200 steps**. This matches the
default snapshot interval. If you use a higher `--snapshot-interval`, the sparse
coverage warning may trigger for long-running transactions — lower the interval
or acknowledge the reduced navigation granularity.

---

## See Also

- [Debug Command Reference](./debug-command.md)
- [Trace Export Annotations](./trace-export-annotations.md)
- [Trace Profiling and Performance](./trace-profiling.md)
- [Event Schemas](./event-schemas.md)
- [JSON Output Format](./json-output.md)
- [Snapshot Deduplication](./snapshot-deduplication.md)
- [Sandboxed Replay](./sandboxed-replay.md)
