# Source Mapping

Glassbox maps WASM instruction failures back to Rust source code lines using
DWARF debug symbols embedded in the compiled WASM binary.

## Automatic Discovery

When a contract fails, Glassbox attempts to resolve the source location through
the following pipeline:

1. **Local cache** — previously resolved source is returned immediately.
2. **Registry** — queries [stellar.expert](https://stellar.expert) for a
   verified source link.
3. **GitHub fallback** — downloads source from the linked repository when a
   `GitHubRetriever` is configured.
4. **`--contract-source` override** — uses the explicitly provided local path
   (see below).
5. **Interactive prompt** — asks the user for a WASM path when all automatic
   methods fail. In non-interactive environments (CI pipelines) this stage is
   skipped and an explicit error is returned instead of hanging on stdin.

### Non-interactive / CI mode

In CI pipelines and other non-interactive environments, the interactive prompt
is disabled automatically. When all discovery stages fail, Glassbox returns an
explicit error:

```
contract source not found: all discovery stages exhausted for contract "C..."
  Stages tried: cache, registry (stellar.expert), GitHub retriever, --contract-source override
  To resolve: provide --contract-source <path> pointing to the contract source directory,
  or verify the contract on stellar.expert to enable registry lookup.
  Use --skip-source-mapping to proceed without source mapping.
```

Set `--skip-source-mapping` to bypass source discovery entirely when you only
need raw trace output.

## `--contract-source` Override

When automatic discovery fails (e.g. the contract is not yet verified on
stellar.expert, or you are working with a private repository), you can provide
the path to the contract source directory explicitly:

```bash
glassbox debug --wasm ./target/wasm32-unknown-unknown/release/my_contract.wasm \
               --contract-source ./src \
               <transaction-hash>
```

Or for local WASM replay:

```bash
glassbox debug --wasm ./contract.wasm \
               --contract-source /path/to/contract/src
```

### Validation

The `--contract-source` path is validated before any network or simulator work begins:

| Condition | Error |
|-----------|-------|
| Path does not exist | `--contract-source: directory not found: "<path>"` |
| Path is a file, not a directory | `--contract-source: "<path>" is a file, not a directory` |
| Path is not accessible | `--contract-source: cannot access "<path>": <os error>` |

Each error includes a remediation hint so you know exactly what to fix.

### WASM binary validation

When checking hash mismatches, Glassbox validates that the local file is a valid WASM binary before computing its hash. Files that do not start with the WASM magic bytes (`\0asm`) produce a clear error:

```
not a valid WASM binary: "<path>" does not start with WASM magic bytes
```

This catches corrupted or non-WASM files early, before the hash comparison step.

### How it works

- When `--contract-source <path>` is set and automatic source resolution fails,
  Glassbox uses `<path>` as the root directory for resolving source file
  references from DWARF debug info.
- The path is tried directly, then as a prefix for the relative file path
  reported by the DWARF info, and finally as a prefix for just the filename.
- The path is also forwarded to the simulator via `ContractSourcePath` in the
  `SimulationRequest`, allowing the Rust simulator to resolve source lines
  during execution.

### When to use it

| Situation | Recommendation |
|-----------|---------------|
| Contract not verified on stellar.expert | `--contract-source ./src` |
| Private repository | `--contract-source /path/to/repo/src` |
| Monorepo with multiple contracts | `--contract-source ./contracts/my_contract/src` |
| CI/CD pipeline (non-interactive) | `--contract-source $CONTRACT_SRC_DIR` |

### Compiling with debug symbols

For best results, compile your contract with debug symbols:

```toml
# Cargo.toml
[profile.release]
debug = true
```

Then build:

```bash
cargo build --target wasm32-unknown-unknown --release
```

See [docs/debug-symbols-guide.md](debug-symbols-guide.md) for more details.

## Cross-repository source links

When contract sources live in another Git repository, map local path prefixes to
remote GitHub URLs in `.glassbox.toml`:

```toml
external_source_map = '[{"prefix":"/path/to/vendor/lib","remote_url":"https://github.com/org/lib","branch":"main"}]'
```

Glassbox uses these mappings when a source file path falls outside the workspace
repository but under the configured prefix.

## Skip source mapping

For faster raw replay when you only need WASM offsets and traces:

```bash
glassbox debug --wasm ./contract.wasm --skip-source-mapping
```

This bypasses DWARF parsing and Git link generation in the simulator.

## Trace verbosity

Control trace detail with `--trace-verbosity`:

| Level | Output |
|-------|--------|
| `summary` | Step names and status only |
| `normal` | Source locations and links (default) |
| `verbose` | Arguments, WASM instructions, and full event payloads |

```bash
glassbox debug --wasm ./contract.wasm --trace-verbosity summary
glassbox trace --print --trace-verbosity verbose execution.json
```

## Fallback pipeline

When no DWARF symbols are available, Glassbox uses a multi-stage fallback
pipeline to provide a best-effort source location:

| Stage | Mechanism | Quality |
|-------|-----------|---------|
| 1 | Full DWARF line-number tables | `full` |
| 2 | Partial DWARF — extract file names from `.debug_line` even when `.debug_info` is stripped | `partial` |
| 3 | Symbol heuristics — infer source paths from Rust mangled symbol names | `heuristic` |
| 4 | Cargo manifest discovery — walk the repo for `Cargo.toml` files | `heuristic` |
| 5 | Unknown — no mapping possible; WAT disassembly shown instead | `unknown` |

Each fallback stage emits a `Warning:` field in the result explaining what was
used and why the mapping may be inaccurate, along with a `debug = true`
remediation hint.

## Local WASM build discovery

Glassbox scans `target/wasm32-unknown-unknown/release/` for WASM files whose
SHA-256 hash matches the on-chain contract bytecode. When a match is found,
DWARF symbols are loaded automatically.

If the build directory is missing, Glassbox logs a debug-level message and
continues without local symbols. The message includes a suggestion to run
`cargo build` if local symbols are needed.

## `--source-alias` Alias Mapping

When source file paths embedded in DWARF symbols don't match your local
directory layout, remap them with an alias file:

```bash
glassbox debug --source-alias ./aliases.json <tx-hash>
```

The alias file must be a flat JSON object:

```json
{
  "my_crate": "/path/to/my_crate/src",
  "vendor_lib": "/path/to/vendor/lib/src"
}
```

**Validation:** The file must be readable and contain valid JSON. Invalid JSON
produces an explicit error:

```
--source-alias: failed to parse "<path>" as JSON: <detail>
  The file must be a flat JSON object mapping alias strings to local paths.
  Example: {"my_crate": "/path/to/my_crate/src"}
```

Alias target directories that don't exist on disk produce a **warning** (not
an error) so debugging can continue if only some aliases are stale.

## Dry-run source discovery checks

`glassbox debug --dry-run` validates source discovery configuration before any
simulation runs:

```
[OK]   Source directory: ./src
[OK]   Source alias file: ./aliases.json (2 mapping(s))
       Warning: source-alias target for "old_crate" does not exist: "/tmp/old_crate/src"
```

Failures appear as numbered items in the `Dry-run FAILED` summary with a
`Fix:` hint for each.
