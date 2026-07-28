# Deep Link Parameter Semantics

## Overview

The `glassbox://` custom URL scheme lets external tools (browsers, CI dashboards,
IDEs) open a transaction debug session directly in Glassbox. The URI parser
validates all parameters and produces actionable error messages for malformed links.

---

## URI Format

```
glassbox://debug/<txhash>?network=<network>[&op=<index>][&view=<mode>][&source=<id>][&signature=<hint>][&protocol-version=<ver>][&mock-ledger-manifest=<path>][&mock-ledger-entry=<key:val>]
```

### Components

| Component | Required | Description |
|---|---|---|
| `debug` | yes | Fixed host segment. Only `debug` is supported. |
| `<txhash>` | yes | 64-character hexadecimal transaction hash (case-insensitive). |
| `network` | yes | Network to query. One of: `testnet`, `mainnet`, `futurenet`. |
| `op` | no | Zero-based operation index to focus on. Alias for `operation`. |
| `operation` | no | Zero-based operation index (legacy alias; `op` takes precedence when both are present). |
| `view` | no | Initial view mode to open. See [View Modes](#view-modes) below. |
| `source` | no | Free-form source identifier (e.g. `"dashboard"`, `"ci"`). Not validated. |
| `signature` | no | Free-form signature hint. Not validated. |
| `protocol-version` | no | Protocol version override for simulation. One of: `20`, `21`, `22`. |
| `mock-ledger-manifest` | no | Path to a JSON manifest containing ledger_entries for override state. |
| `mock-ledger-entry` | no | Override ledger entries before simulation (repeatable, format: `key:value`). |

---

## Query Parameters

### `network` (required)

Specifies which Stellar network to fetch the transaction from.

| Value | Network |
|---|---|
| `testnet` | Stellar Testnet |
| `mainnet` | Stellar Mainnet (Public Network) |
| `futurenet` | Stellar Futurenet |

Any other value produces an error listing the allowed values.

### `op` / `operation` (optional)

Zero-based index of the operation within the transaction to focus on.

- Must be a non-negative integer (`0`, `1`, `2`, …).
- `op` takes precedence over `operation` when both are present.
- `operation` is retained for backward compatibility.
- When omitted, the debugger opens on the first failing operation (or the
  transaction overview if no specific operation failed).

**Examples:**
```
glassbox://debug/<hash>?network=testnet&op=0      # first operation
glassbox://debug/<hash>?network=testnet&op=2      # third operation
```

### `view` (optional)

Selects the initial view panel to display when the debug session opens.

| Value | Panel |
|---|---|
| `trace` | Execution trace / call tree |
| `flamegraph` | CPU flamegraph |
| `events` | Contract events and diagnostic events |
| `auth` | Authorization trace |
| `budget` | CPU and memory budget breakdown |
| `storage` | Ledger storage reads and writes |

When omitted, the default view is shown (typically the execution trace).

Any unrecognised value produces an error listing the allowed values.

### `source` (optional)

A free-form string identifying where the link originated (e.g. `"dashboard"`,
`"ci"`, `"explorer"`). Used for analytics and session labelling. Not validated
beyond URL encoding.

### `signature` (optional)

A free-form hint string. Not validated. Reserved for future use.

### `protocol-version` (optional)

Overrides the Soroban protocol version for simulation. Must be a supported protocol version (e.g. `20`, `21`, `22`).

### `mock-ledger-manifest` (optional)

Specifies a path to a JSON manifest containing `ledger_entries` for override state. Rejects paths containing null bytes.

### `mock-ledger-entry` (optional, repeatable)

Enables overriding specific ledger entries using `key:value` pairs before simulation. Each override must be a non-empty key and a valid base64-encoded value.

---

## Examples

**Minimal valid link (testnet, no optional params):**
```
glassbox://debug/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef?network=testnet
```

**Acceptance-criteria example — network + op:**
```
glassbox://debug/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef?network=testnet&op=0
```

**Full example — all parameters:**
```
glassbox://debug/0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef?network=futurenet&op=2&view=flamegraph&source=ci&protocol-version=22&mock-ledger-manifest=/path/to/manifest.json&mock-ledger-entry=AAAAAQ==:BBBBQg==
```

---

## Error Messages

The parser produces specific, actionable errors for each class of invalid input:

| Problem | Example error |
|---|---|
| Empty URI | `protocol URI must not be empty` |
| Wrong scheme | `invalid protocol URI: expected glassbox://` |
| Wrong host | `invalid protocol host "inspect": expected "debug"` |
| Missing/empty hash | `invalid transaction hash "": must be a 64-character hex string` |
| Short/invalid hash | `invalid transaction hash "abc": must be a 64-character hex string` |
| Missing network | `missing required query parameter: network` |
| Invalid network | `invalid network "staging": must be one of testnet, mainnet, futurenet` |
| Negative op | `invalid operation index "-1": must be a non-negative integer` |
| Non-numeric op | `invalid operation index "abc": must be a non-negative integer` |
| Invalid view | `invalid view "raw": must be one of trace, flamegraph, events, auth, budget, storage` |
| Invalid protocol-version | `invalid protocol-version "99": unsupported version` |
| Null byte in mock-ledger-manifest | `mock-ledger-manifest parameter contains null bytes and cannot be used` |
| Invalid mock-ledger-entry format | `invalid mock-ledger-entry format "keyonly" — expected key:value` |
| Invalid mock-ledger-entry base64 | `mock-ledger-entry "key:value" has an invalid base64 value` |

---

## CLI Dispatch

When the OS dispatches a `glassbox://` link to the binary, the
`protocol:handle <uri>` command validates the URI and re-invokes the binary:

```
glassbox debug <txhash> --network <network> [--op <index>] [--view <mode>]
```

The `--deep-link` internal flag (used by the doctor probe) also handles
`glassbox://debug/...` URIs via the same `ParseDebugURI` validation path.

---

## Source Location

- Parser: `internal/protocolreg/uri.go`
- Tests: `internal/protocolreg/uri_test.go`
- CLI handler: `internal/cmd/protocol.go`
- Deep link probe: `internal/cmd/root.go` (`handleDeepLinkProbe`)
