# Contract Fixture Provenance

This directory contains Soroban smart-contract source files used by integration
and unit tests.  None of the `.rust` files are compiled at test time; the tests
drive a `MockRunner` and treat these files as reference source for DWARF-path
resolution, source-mapping smoke tests, and regression coverage.

---

## Inventory

| File | Contract | Purpose |
|------|----------|---------|
| `counter.rust` | `Counter` | Minimal increment/get counter. Used by time-travel integration tests to verify that step indices align with WASM instruction trace offsets. |
| `auth_test.rust` | `AuthTestContract` | Two-signer authorization with a nonce counter. Used by auth-debug regression tests that require `require_auth()` call sites. |
| `panic.rust` | `PanicContract` | Increments a counter then panics with a message that includes the counter value. Used by snapshot-consistency tests to verify trap capture after state mutation. |

---

## Provenance

Every file in this directory is **hand-authored test-only source**.  No file
is generated, downloaded, or derived from a production contract.

| Property | Value |
|----------|-------|
| Ecosystem | Soroban (Stellar smart contracts) |
| SDK | `soroban-sdk` (Rust) |
| File extension | `.rust` — intentionally not `.rs` so the files are never picked up by an accidental `cargo build` in this directory |
| WASM compilation | **Not required at test time.** Tests use a `MockRunner`. |
| Real contract IDs | None — no on-chain contract is referenced |
| Real transaction hashes | None |
| Credentials / secrets | None |

---

## How tests reference these fixtures

Tests import the source text via a relative path constant rather than
embedding WASM bytes.  Example (Go):

```go
const contractFixtureDir = "tests/fixtures/contracts"

func TestSourceMap_Counter(t *testing.T) {
    src, err := os.ReadFile(filepath.Join(contractFixtureDir, "counter.rust"))
    // src is the Rust source; pass it to the mock source-resolver under test.
}
```

TypeScript tests reference the same directory:

```typescript
const FIXTURE_DIR = path.join(__dirname, '../../tests/fixtures/contracts');
const counterSrc  = fs.readFileSync(path.join(FIXTURE_DIR, 'counter.rust'), 'utf8');
```

---

## Regenerating / updating a fixture

When a contract's behaviour changes (new entry point, different storage key,
updated SDK):

1. Edit the `.rust` source directly in this directory.
2. Update the table above if the contract name, entry points, or test purpose
   change.
3. If any Go or TypeScript test asserts on the **text** of the source file
   (e.g. a golden-output test that shows a source excerpt), update the golden
   file at the same time.
4. Do **not** compile the contract or commit a `.wasm` artifact here — the
   WASM lives in `testdata/` for tests that need binary content.

To verify that a `.rust` file at least parses (requires a Rust toolchain,
optional):

```bash
# Rename temporarily, check syntax, rename back
cp tests/fixtures/contracts/counter.rust /tmp/counter_check.rs
rustc --edition 2021 --crate-type lib /tmp/counter_check.rs --emit=metadata -o /dev/null 2>&1 || true
```

---

## Adding a new fixture

1. Place the `.rust` source in this directory.
2. Follow the naming pattern: `<purpose>.rust` (snake\_case, all lowercase).
3. Add a row to the **Inventory** table above.
4. Add a `provenance` entry to `tests/fixtures/contracts/provenance.json`
   (see that file for the schema).
5. Write or update at least one test that loads the new fixture.

---

## Related documentation

- [`docs/source-mapping.md`](../../docs/source-mapping.md) — how DWARF paths
  are resolved at runtime
- [`docs/regression-test-guide.md`](../../docs/regression-test-guide.md) —
  general fixture authoring rules
- [`test/regression/fixtures/sourcemap/README.md`](../../test/regression/fixtures/sourcemap/README.md)
  — WASM binary stubs for source-map regression tests
