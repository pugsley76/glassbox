# SARIF diagnostic rule families

Alongside the hash-derived security rules (`GB-XXXX`) documented in
[sarif-export.md](sarif-export.md), the SARIF writer emits two **enumerated**
diagnostic rule families for findings that are not `security.Finding` objects.

They are enumerated rather than derived because each one corresponds to a
distinct remediation, and because CI code scanning needs rule IDs that stay
stable across runs in order to track a rule's history.

## Source map diagnostics (GB1xxx)

Reported when a WASM instruction offset cannot be mapped back to a Rust source
line. Previously these were printed as human-readable warnings on stderr, so a
CI run produced no durable record of them.

| Rule | Name | Level | Meaning |
| ---- | ---- | ----- | ------- |
| `GB1001` | `SourceMapOffsetNotFound` | `warning` | No DWARF line table entry for the offset |
| `GB1002` | `SourceMapDwarfMissing` | `error` | The WASM binary has no usable debug sections at all |
| `GB1003` | `SourceMapNoGitRepository` | `note` | No Git repository detected, so no source links were generated |

`GB1002` is an error rather than a warning because it is not an isolated
mapping miss: without debug sections, *no* offset in the binary can be mapped,
so every failure in the run is unattributable.

### Location handling

A source map failure has no source file to point at — resolving the offset is
precisely what failed. Emitting an artifact location anyway would produce a
result pointing at a file that does not contain the problem, so these results
carry a SARIF `logicalLocation` keyed on the WASM offset instead:

```json
{
  "ruleId": "GB1001",
  "level": "warning",
  "message": { "text": "WASM offset 0x1a4c: no DWARF line table entry for this offset. Fix: build the contract with debug = true" },
  "locations": [
    {
      "logicalLocation": {
        "name": "0x1a4c",
        "kind": "wasmOffset",
        "fullyQualifiedName": "wasm::offset::0x1a4c"
      }
    }
  ]
}
```

`logicalLocation` is only emitted when there is genuinely no source file. When
the caller did resolve a location after all, the result carries both a
`physicalLocation` and the logical offset, so a consumer can use whichever it
has a base URI for.

## Security warnings (GB2xxx)

Reported for host function calls that are permitted but undesirable. The
deprecated list previously lived as a package-private variable inside
`internal/cmd`, so only `glassbox debug` could report it; it now lives in
`internal/security` and is shared with the LSP diagnostics publisher.

| Rule | Name | Level | Meaning |
| ---- | ---- | ----- | ------- |
| `GB2001` | `DeprecatedHostFunction` | `warning` | A Soroban host function scheduled for removal |
| `GB2002` | `AllowlistedUnsafeHostFunction` | `error` | A permitted host function used unsafely |

### Location handling

A security warning points at a real Rust source line when source mapping
succeeded, and falls back to the logical WASM offset when it did not — the
opposite of the source map failures, which never have a source file. When a
physical location is used, a `logicalLocation` of kind `hostFunction` carries
the function name alongside it.

## API

```go
log := sarif.ExportDiagnosticsWithFindings(
    findings,  // []security.Finding, as before
    failures,  // []sarif.SourceMapFailure
    warnings,  // []sarif.SecurityWarning
    sarif.DiagnosticOptions{ArtifactURI: "contract.wasm"},
)

encoded, err := sarif.Marshal(log)
```

`ExportDiagnostics` is the same call without security findings. Both are pure
functions: no I/O, and two exports of the same input are byte-identical, which
is what keeps a SARIF diff meaningful.

Results and rules are sorted by rule ID and message, and each result carries a
`primaryLocationLineHash/v1` fingerprint so a code-scanning platform can track
a finding across runs.

## Rule metadata

Every enumerated rule carries a `shortDescription`, a `fullDescription`, a
`helpUri` and a `defaultConfiguration.level`, plus `tags` and a `precision`
label. `sarif.RuleByID` looks one up, and `sarif.DiagnosticRules` returns them
all in rule-ID order.
