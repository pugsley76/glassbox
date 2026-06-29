# Trace Export Annotations

Trace exports can include collaborator comments and session metadata:

```bash
glassbox trace execution.json \
  --export trace.html \
  --comment "Reviewed with Alice" \
  --meta session=payroll-bug \
  --meta network=testnet
```

Supported fields:

- `comments`: free-form notes, repeatable with `--comment`
- `session_metadata`: key/value metadata supplied with `--meta key=value`
- `generated_at`: timestamp added when annotations are merged into exports

Annotations are included in HTML and Markdown trace artifacts and preserved in
JSON trace exports under the `annotations` object.

---

## Validation

All annotation flags are validated in `PreRunE` before the trace file is loaded,
so every error is surfaced in a single pass.

### `--comment`

**Validation:**
- Value must not be empty or whitespace-only

**Error example:**
```
--comment value at position 0 is empty or whitespace-only
  Fix: provide non-empty comment text or omit the empty --comment flag
```

**Limits (enforced at export time):**
- Maximum 100 comments per trace export
- Maximum 10,000 characters per individual comment

### `--meta key=value`

**Validation:**
- Every value must be in `key=value` format (contains at least one `=`)
- Key must not be empty or whitespace-only

**Error example:**
```
--meta value "no-equals-sign" is not in key=value format
  Fix: supply metadata as key=value pairs, e.g. --meta env=testnet --meta version=1.2
```

Values containing `=` are parsed correctly — only the first `=` is used as a
separator, so `--meta filter=type=contract_call` produces `filter` → `type=contract_call`.

---

## Multiple validation errors

When `--comment` and `--meta` failures are combined with other flag errors, all
failures are numbered and reported together:

```
2 trace command validation error(s):
  1. --meta value "no-equals" is not in key=value format
     Fix: supply metadata as key=value pairs, e.g. --meta env=testnet --meta version=1.2
  2. --export "./traces/" looks like a directory path; provide a full file path
     Fix: specify a filename (e.g. --export ./traces/output.html)
```
