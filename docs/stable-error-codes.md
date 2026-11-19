# Stable Error Code Policy

Glassbox commands surface failures through a shared error type so that
automation scripts and CI pipelines can key on a **stable code** rather than
parsing free-form text.

## Error type

Every structured error is an `*errors.ErstError` (defined in
`internal/errors/errors.go`):

```go
type ErstError struct {
    Code    ErstErrorCode  // stable string, e.g. "RPC_CONNECTION_FAILED"
    Message string         // human-readable; may change between releases
    OrigErr error          // original cause, preserved via Unwrap()
    Hint    string         // optional actionable remediation for the user
}
```

- `Code` is the **automation contract** — do not key on `Message`.
- `Hint` is never included in `Error()` so it does not pollute log entries.
  It is surfaced by `errors.Hint(err)` and printed separately.
- `OrigErr` is returned by `Unwrap()` so `errors.Is` / `errors.As` chains
  work transparently.

## Output adapters (internal/clioutput)

| Function | When to use |
|---|---|
| `clioutput.WriteError(w, cmd, err, ctx)` | JSON mode — writes a schema-versioned envelope with `error.code` |
| `clioutput.FormatErrorText(err)` | Text mode — returns `[CODE] CODE: message\nHint: …` |

Both adapters extract the same `Code` from the error, so text and JSON modes
are always consistent. Because `ErstError.Error()` already starts with
`CODE: `, the code appears twice in `FormatErrorText` output and once at the
start of the JSON `message` field.

### What the CLI process prints today

No command currently routes its failure through `WriteError`; the adapters are
covered by `internal/clioutput/error_test.go`. When a command returns an error,
`cmd/glassbox/main.go` writes to **stderr**:

```text
Error: <CODE>: <message>[: <cause>]
Hint: <remediation>          # only when the error carries a Hint
```

and exits with `ExitCodeFor(err)`. Errors that are not `ErstError` print
`Error: <message>` with no code and exit `3`. See
[json-output-automation.md](json-output-automation.md) for filtering on
`error.code` from scripts.

### JSON envelope shape

```jsonc
{
  "schema_version": "1.0",
  "glassbox_version": "1.2.3",
  "generated_at": "2026-07-24T12:00:00Z",
  "command": "debug",
  "data": {
    "error": {
      "code": "RPC_CONNECTION_FAILED",   // ← stable, key on this
      "severity": "error",
      "message": "RPC_CONNECTION_FAILED: RPC connection failed: dial tcp …",
      "remediation": "The RPC endpoint could not be reached. …",
      "context": {
        "network": "testnet"
      }
    }
  }
}
```

### Error payload fields

Defined by `ErrorPayload` in `internal/clioutput/error.go`:

| Field | Type | Always present | Notes |
|---|---|---|---|
| `code` | string | yes | Stable `ErstErrorCode`; `UNKNOWN` when the error is not an `ErstError` |
| `severity` | string | yes | Currently always `error` |
| `message` | string | yes | Full `err.Error()` text including wrapped causes; may change between releases |
| `remediation` | string | no (`omitempty`) | The error's `Hint`; static text written in `internal/errors` or the command |
| `context` | object of string → string | no (`omitempty`) | Command-supplied `ctx` map passed to `WriteError`; omitted when `nil` or empty |

## Context fields and log aggregation

`context` is a flat `map[string]string`. Glassbox does not add keys on its
own: every key comes from the `ctx` argument of `clioutput.WriteError`. The
only keys exercised today are `tx_hash` and `network`
(`TestWriteError_ContextIncluded` in `internal/clioutput/error_test.go`).

Use this classification when indexing error envelopes in a log aggregator,
and when choosing keys for new `WriteError` call sites:

| Field / key | Safe to index | Why |
|---|---|---|
| `code` | yes | Fixed vocabulary from the [catalogue](#stable-code-catalogue); low cardinality |
| `severity` | yes | Fixed vocabulary |
| `command` (envelope) | yes | Cobra command name |
| `schema_version`, `glassbox_version` | yes | Build metadata |
| `context.network` | yes | One of `testnet`, `mainnet`, `futurenet` or a configured network name |
| `context.tx_hash` | yes | Public on-chain identifier; high cardinality, so index as a keyword, not a label |
| `remediation` | store, don't index | Static prose; can change between releases |
| `message` | store only, redact first | Contains the wrapped cause verbatim, which can include RPC URLs (query-string tokens), local file paths, and user input |

Rules for new context keys:

- Use `snake_case` keys and keep values to identifiers (network, hash,
  ledger sequence, command name).
- Never put secrets, tokens, private keys, full RPC URLs, or absolute home
  paths in `context`. The same redaction expectations apply as in
  `glassbox doctor --bundle` (see [diagnostics-bundle.md](diagnostics-bundle.md))
  and execution plans (`internal/plan`, which redacts URL tokens and keys).
- Add the key and its classification to the table above in the same change.

Verify the payload contract with:

```bash
go test ./internal/clioutput/ -run 'WriteError|FormatErrorText'
```

## Stable code catalogue

### General

| Code | Meaning | Exit |
|---|---|---|
| `VALIDATION_FAILED` | Bad flag, argument, or config value | 1 |
| `ARGUMENT_REQUIRED` | Reserved; not currently emitted. Missing flags (`WrapCliArgumentRequired`) report `VALIDATION_FAILED` | 1 |
| `CONFIG_ERROR` | Config file unreadable or invalid | 2 |
| `UNKNOWN` | Unclassified error | 3 |

### RPC

| Code | Meaning | Exit |
|---|---|---|
| `RPC_CONNECTION_FAILED` | Endpoint unreachable | 3 |
| `RPC_TIMEOUT` | Request timed out | 3 |
| `ALL_RPC_FAILED` | All failover endpoints failed | 3 |
| `RPC_ERROR` | RPC server returned a non-200 / error response | 3 |
| `TRANSACTION_NOT_FOUND` | TX hash not found on the selected network | 1 |
| `LEDGER_NOT_FOUND` | Ledger sequence unavailable | 1 |
| `LEDGER_ARCHIVED` | Ledger has been archived | 1 |
| `RATE_LIMIT_EXCEEDED` | RPC rate limit hit | 1 |
| `UNAUTHORIZED` | Missing or invalid auth token | 1 |
| `INVALID_NETWORK` | Unrecognised `--network` value | 1 |
| `NETWORK_NOT_FOUND` | Named network has no known endpoint | 1 |

### Simulator

| Code | Meaning | Exit |
|---|---|---|
| `SIMULATOR_NOT_FOUND` | Simulator binary absent or not executable | 2 |
| `SIMULATION_FAILED` | Simulator process returned non-zero | 3 |
| `SIMULATOR_CRASH` | Simulator process crashed unexpectedly | 3 |
| `SIMULATION_LOGIC_ERROR` | Contract logic error detected | 1 |

### Other

| Code | Meaning | Exit |
|---|---|---|
| `SOURCE_DISCOVERY_FAILED` | Source mapping unavailable for contract | 3 |

## Adding a new code

1. Add the constant to `internal/errors/glassbox_error_code.go`.
2. Add it to the `codeToSentinel` map with a matching sentinel error.
3. Add it to `errorCodeRegistry` in `errors.go` if a sentinel should map to it.
4. Update the table above.
5. Decide whether it maps to exit code 1, 2, or 3 and add it to
   `userErrorCodes` or `configErrorCodes` in `internal/cmd/exitcode.go`.

## Exit code taxonomy

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | User error (bad input, validation failure) |
| 2 | Config / environment error |
| 3 | Internal / runtime error |
| 130 | Interrupted (SIGINT / Ctrl-C) |

## Migrated commands

The following commands emit `ErstError` for all failure paths and have test
coverage of at least one validation error and one runtime error:

- `debug`
- `trace`
- `audit:sign`
- `audit:verify`
- `session` (save, resume, list, delete, recover, doctor)
- `protocol:register`, `protocol:diagnose`, `protocol:verify`
- `config show` (load and explain paths)
