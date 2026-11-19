# Automating on `error.code`

Glassbox failures carry a stable code from the
[stable code catalogue](stable-error-codes.md#stable-code-catalogue). Scripts
and CI jobs should branch on that code and on the process exit code, never on
the human-readable message.

There are two places a code can appear:

| Surface | Where | Produced by |
|---|---|---|
| stderr line | `Error: <CODE>: <message>` | `cmd/glassbox/main.go` for every failing command (current CLI behavior) |
| JSON envelope | `.data.error.code` | `clioutput.WriteError` (see [json-output.md](json-output.md#error-envelope)) |

No command currently writes the JSON error envelope on failure, so for the
CLI today use the exit code and the stderr prefix. The JSON recipes below
apply to any tool that consumes envelopes written by `clioutput.WriteError`.

## Decision order

1. **Exit code** (`0`, `1`, `2`, `3`, `130`): always available. See the
   [exit code taxonomy](stable-error-codes.md#exit-code-taxonomy).
2. **Stable code**: from `.data.error.code` or the `Error: <CODE>:` prefix.
3. **Fallback**: if no code is present (the error is not an `ErstError`, for
   example `session name is too long`), treat it as `UNKNOWN` and use the exit
   code alone.

## Filtering the stderr prefix

Codes are uppercase letters and underscores. Extract the code from stderr:

```bash
glassbox session save --name payroll-bug 2> err.txt
status=$?
code=$(sed -n 's/^Error: \([A-Z_]*\): .*/\1/p' err.txt | head -n1)
echo "exit=${status} code=${code:-UNKNOWN}"
```

Expected output in a fresh process (no active session, see
[session-bookmarking.md](session-bookmarking.md#save-without-an-active-session)):

```text
exit=1 code=SIMULATION_LOGIC_ERROR
```

Branch on the result:

```bash
case "${code:-UNKNOWN}" in
  RPC_CONNECTION_FAILED|RPC_TIMEOUT|ALL_RPC_FAILED|RATE_LIMIT_EXCEEDED)
    echo "transient RPC failure, retrying" ;;
  SIMULATOR_NOT_FOUND|CONFIG_ERROR)
    echo "environment problem, see docs/troubleshooting-index.md"; exit 2 ;;
  TRANSACTION_NOT_FOUND|INVALID_NETWORK|NETWORK_NOT_FOUND|VALIDATION_FAILED)
    echo "bad input, not retrying"; exit 1 ;;
  *)
    exit "${status}" ;;
esac
```

## Filtering the JSON envelope

Golden error envelope, written by `clioutput.WriteError` for
`WrapRPCConnectionFailed` with `ctx = {"network": "testnet"}`
(`generated_at` and `glassbox_version` vary per build and run):

```json
{
  "schema_version": "1.0",
  "glassbox_version": "0.0.0-dev",
  "generated_at": "2026-11-19T09:17:25.654487226Z",
  "command": "debug",
  "data": {
    "error": {
      "code": "RPC_CONNECTION_FAILED",
      "severity": "error",
      "message": "RPC_CONNECTION_FAILED: RPC connection failed: dial tcp 127.0.0.1:8000: connect: connection refused",
      "remediation": "The RPC endpoint could not be reached. Check your internet connection and the endpoint, pass a known-good one with --rpc-url <url>, and make sure it serves the selected --network.",
      "context": {
        "network": "testnet"
      }
    }
  }
}
```

Go's JSON encoder escapes `<`, `>`, and `&` as `<`, `>`, and `&`.
Decode with a JSON parser instead of grepping raw text.

Success envelopes have the same top-level shape with command data under
`data` and no `data.error` key:

```json
{
  "schema_version": "1.0",
  "glassbox_version": "0.0.0-dev",
  "generated_at": "2026-11-19T09:17:25.65458634Z",
  "command": "debug",
  "data": {
    "status": "ok"
  }
}
```

With `jq` (assuming the envelope is saved as `out.json`):

```bash
# Extract the code; prints nothing for a success envelope
jq -r '.data.error.code // empty' out.json
# → RPC_CONNECTION_FAILED

# Success vs failure as an exit status (0 = error envelope, 1 = success)
jq -e '.data | has("error")' out.json >/dev/null

# Filter one code out of newline-delimited envelopes (jq -c output, one per line)
jq -c 'select(.data.error.code == "RPC_TIMEOUT")' envelopes.ndjson

# Pull only fields that are safe to index
jq '{command, code: .data.error.code, network: .data.error.context.network}' out.json
```

Check `schema_version` before parsing: `1.0` is the only version emitted today
(`clioutput.SchemaVersion`).

## Rules

- Key on `code`, never on `message`; the message includes wrapped causes and
  may change between releases.
- Treat a missing or `UNKNOWN` code as "use the exit code".
- Do not rely on `ARGUMENT_REQUIRED`: missing flags currently report
  `VALIDATION_FAILED` (see [stable-error-codes.md](stable-error-codes.md#general)).
- Before shipping `message` to a log store, read the
  [log aggregation guidance](stable-error-codes.md#context-fields-and-log-aggregation).

## Verifying

```bash
go test ./internal/clioutput/   # envelope shape, code extraction, context map
go test ./internal/errors/      # code ↔ sentinel mapping
bash scripts/validate-errors.sh # error standardization checks
```
