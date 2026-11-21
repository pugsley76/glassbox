# Cross-Language Canonical JSON Conformance Tests

Glassbox uses deterministic, canonical JSON serialisation for audit log hashing
and signature verification. Go and TypeScript both hash the same logical payload
and must produce byte-identical output. This document explains the conformance
test infrastructure that enforces that guarantee.

## Why this matters

An audit log is signed over a canonical hash. If Go and TypeScript canonicalize
a payload differently — even by a single byte — the TypeScript verifier will
reject a log produced by the Go signer, or vice versa. The conformance corpus
makes this class of bug impossible to introduce silently.

## Corpus location

```
testdata/canonical-conformance/corpus.json
```

The corpus is language-neutral JSON. Every test case has:

| Field | Description |
|---|---|
| `id` | Unique fixture name used in test output |
| `description` | Human-readable explanation of what property is tested |
| `input` | The raw JSON value to canonicalize |
| `canonical` | The expected byte-exact canonical string |
| `sha256` | SHA-256 hex digest of the `canonical` string encoded as UTF-8 |
| `notes` | Optional implementation notes |

## What the corpus covers

| Case ID | Property tested |
|---|---|
| `basic-key-ordering` | Top-level object keys sorted lexicographically |
| `nested-key-ordering` | Recursive key sorting at every nesting depth |
| `no-whitespace` | No whitespace between tokens |
| `array-insertion-order` | Arrays preserve insertion order (not sorted) |
| `null-value` | `null` values serialise as JSON `null` |
| `boolean-values` | `true` / `false` preserved |
| `integer-numbers` | Integers have no trailing `.0` |
| `float-numbers` | IEEE 754 double-precision floating point |
| `unicode-strings` | Non-ASCII characters (emoji, CJK, Arabic) preserved as UTF-8 |
| `empty-object` | `{}` |
| `empty-array` | `{"items":[]}` |
| `deeply-nested` | Key sorting at all nesting levels simultaneously |
| `audit-payload-typical` | Realistic AuditLogger payload |
| `array-of-objects` | Mixed arrays: element order preserved, object keys sorted |
| `mixed-types-in-array` | Arrays may contain different JSON types |
| `string-escaping-basics` | Backslash, quote, newline, tab per JSON spec |

### Worked example

The `basic-key-ordering` corpus case shows the full round trip. Given this
input in any key order:

```json
{"z": 1, "a": 2, "m": 3}
```

both runtimes produce the same byte-exact canonical string:

```json
{"a":2,"m":3,"z":1}
```

Hashing the UTF-8 bytes of that string with SHA-256 yields:

```
ebba85cfdc0a724b6cc327ecc545faeb38b9fe02eca603b430eb872f5cf75370
```

The corpus records all three values under the `id` `basic-key-ordering`; the
Go and Jest conformance suites compare the computed digest against the stored
`sha256` without printing the payload itself.

### Invalid cases

Cases in the `invalid` array describe inputs that must be **rejected** before
serialisation. They have only `id`, `description`, and `notes`. There is no
`input`, because none of these values can be written as JSON.

| Case ID | Reason | What the tests actually assert |
|---|---|---|
| `nan-value` | NaN cannot be expressed in JSON | Go: none (rejected by `validateCanonicalNumber` in `canonical.go`, but no corpus-driven test). TS: `stringify({value: NaN})` returns `{"value":null}`, so rejection is the application layer's job |
| `infinity-value` | Infinity cannot be expressed in JSON | Go: none (same as above). TS: `stringify({value: Infinity})` returns `{"value":null}` |
| `circular-reference` | Circular references cannot be serialised | Go: none. TS: `stringify` throws |

Neither test suite loops over the `invalid` array. The TypeScript tests for
these three cases are hand-written. If you add an entry to `invalid`, you must
also add a matching hand-written test in each runtime, or nothing exercises it.

### Fixture index rules

The two tables above are the index of every fixture in `corpus.json`. Keep
them complete:

- Every `id` in `cases` and `invalid` appears in exactly one table row, in
  corpus order.
- `id` values are unique across `cases` and `invalid`.
- Every `sha256` is the SHA-256 of the UTF-8 bytes of `canonical`.
- `canonical` equals `input` with keys sorted, no whitespace, and non-ASCII
  characters left unescaped.

#### Checking the index without a Go or Node toolchain

The command below needs only `python3`. Run it from the repository root. It
checks all four rules and works even when the Go or Jest suites cannot run:

```bash
python3 - <<'EOF'
import hashlib, json, re, sys
c = json.load(open("testdata/canonical-conformance/corpus.json", encoding="utf-8"))
doc = open("docs/canonical-conformance.md", encoding="utf-8").read()
doc = doc.split("## What the corpus covers", 1)[1].split("### Fixture index rules", 1)[0]
errors = []
ids = [x["id"] for x in c["cases"] + c["invalid"]]
for d in sorted({i for i in ids if ids.count(i) > 1}):
    errors.append(f"duplicate id: {d}")
for x in c["cases"]:
    want = json.dumps(x["input"], sort_keys=True, separators=(",", ":"), ensure_ascii=False)
    if want != x["canonical"]:
        errors.append(f"{x['id']}: canonical does not match input")
    if hashlib.sha256(x["canonical"].encode("utf-8")).hexdigest() != x["sha256"]:
        errors.append(f"{x['id']}: sha256 does not match canonical")
indexed = set(re.findall(r"^\| `([a-z0-9-]+)` \|", doc, re.M))
for i in ids:
    if i not in indexed:
        errors.append(f"{i}: missing from docs/canonical-conformance.md tables")
for i in sorted(indexed - set(ids)):
    errors.append(f"{i}: listed in docs but not in corpus.json")
print("\n".join(errors) or f"OK: {len(c['cases'])} cases, {len(c['invalid'])} invalid, index complete")
sys.exit(1 if errors else 0)
EOF
```

Expected output on a clean checkout (exit code 0):

```
OK: 16 cases, 3 invalid, index complete
```

Failure example: if you add a case with `"sha256": "placeholder"` and leave it
out of the table, the command exits 1 and prints:

```
my-new-case: sha256 does not match canonical
my-new-case: missing from docs/canonical-conformance.md tables
```

The `input` check uses Python's `sort_keys`, which sorts by code point and
matches Go's `canonicalJSON` for every current case. One known gap is
Unicode normalisation: Go also applies NFC to strings and TypeScript does not.
A case that relies on NFC needs a hand-written `canonical` value and a note in
its `notes` field.

## Running the demo

`examples/canonical_json_demo.go` is a self-contained, runnable walkthrough of
the canonical JSON guarantees described above. It needs only the Go toolchain
and does not depend on the conformance corpus or the Jest suite.

```bash
go run examples/canonical_json_demo.go
```

The demo prints three sections, each demonstrating one property:

| Section | What it shows |
|---|---|
| Deterministic hashing | The same logical payload, built with keys inserted in different orders, canonicalizes to one byte-exact string and one SHA-256 digest |
| Tampering detection | A modified payload produces a different digest, so a verifier rejects it |
| Cross-platform consistency | The canonical string uses sorted keys, no whitespace, and raw UTF-8, matching the TypeScript implementation |

A successful run exits 0. The tampering section is the focused failure path:
it shows the original and tampered digests side by side and reports that the
tampered payload is detected, for example:

```
Tampering detected: digest mismatch
  original: <hex digest>
  tampered: <different hex digest>
```

If the tampered payload were *not* detected, the demo would report a failure
and exit non-zero. Treat any non-zero exit as a regression in canonicalization,
not as an expected outcome.

> **Known baseline (pre-existing, unrelated to the demo):** on current `main`,
> `go test ./internal/cmd/...` reports `[build failed]` because of compile
> errors in packages it imports (`internal/demangle`, `internal/cache`,
> `internal/decoder`, `internal/snapshot`, `internal/deterministic`). The demo
> in `examples/` does not import those packages, so `go run
> examples/canonical_json_demo.go` is the narrowest check that works on a clean
> checkout until those build errors are fixed.

## Running the conformance suite

### Go

```bash
# Run only the conformance tests
go test ./internal/cmd/... -run TestCanonicalConformance -v

# Run corpus integrity check (verifies corpus.json is self-consistent)
go test ./internal/cmd/... -run TestCanonicalConformance_CorpusIntegrity -v
```

> **Known baseline (pre-existing, unrelated to the corpus):** on current `main`,
> `go test ./internal/cmd/...` reports `[build failed]`. The cause is compile
> errors in packages it imports (`internal/demangle`, `internal/cache`,
> `internal/decoder`, `internal/snapshot`, `internal/deterministic`). Until
> those are fixed, use the
> [index check](#checking-the-index-without-a-go-or-node-toolchain) to
> validate corpus edits.

### TypeScript / Jest

Run `npm ci` first. Jest is not available until `node_modules/` is installed.

```bash
# Run only the conformance tests
npx jest src/audit/__tests__/canonical-conformance.spec.ts --verbose

# Or via npm
npm test -- --testPathPattern=canonical-conformance
```

### Both at once (CI)

```bash
go test ./internal/cmd/... -run TestCanonicalConformance && \
  npx jest src/audit/__tests__/canonical-conformance.spec.ts
```

## Failure output

Both test suites are designed to emit **only safe diagnostic information** on
failure. Raw payload values are never printed. Failure messages contain:

- The fixture `id`
- The first 16 characters of the computed and expected SHA-256 hashes
- A reminder to enable debug logging if the full canonical string is needed

Example failure:

```
--- FAIL: TestCanonicalConformance_ValidCases/audit-payload-typical
    canonical_conformance_test.go:107:
        [audit-payload-typical] SHA-256 hash mismatch
          fixture hash: d95ddcc344dbee5…
          computed: 1a2b3c4d5e6f7a8b…
```

## Adding a new fixture

1. Add a new entry to the `cases` array in `testdata/canonical-conformance/corpus.json`.
2. Set `sha256` to `"placeholder"` initially.
3. Compute the full hash of your `canonical` string (replace `my-new-case`
   with your case's `id`):
   ```bash
   python3 -c 'import json,hashlib,sys; c=json.load(open("testdata/canonical-conformance/corpus.json",encoding="utf-8")); print(next(hashlib.sha256(x["canonical"].encode()).hexdigest() for x in c["cases"] if x["id"]==sys.argv[1]))' my-new-case
   ```
   The test suites cannot give you this value. `TestCanonicalConformance_CorpusIntegrity`
   **skips** any case whose `sha256` is `""` or `"placeholder"`, and on a
   mismatch every test prints only the first 16 hex characters (`safeHash`).
4. Update the `sha256` field with the printed value.
5. Add a row for the new `id` to the table in [What the corpus covers](#what-the-corpus-covers).
6. Run the [index check](#checking-the-index-without-a-go-or-node-toolchain),
   then both test suites, to confirm the new fixture passes.

Do **not** duplicate a fixture `id`. The test suites do **not** detect
duplicates: Go's `t.Run` renames the second one to `<id>#01`, and Jest accepts
repeated test names. The index check above is the only thing that catches a
duplicate.

## Canonicalization implementations

### Go

Located in `internal/cmd/canonical.go`. The function `canonicalJSON(v interface{})` accepts
any Go value, sorts all map keys recursively, and returns compact JSON bytes with no
trailing newline.

Key properties:
- Keys sorted with `sort.Strings`
- No HTML escaping (`SetEscapeHTML(false)`)
- No indentation
- Unicode preserved as raw UTF-8 bytes

### TypeScript

Uses `fast-json-stable-stringify` (npm package). Called as `stringify(value)` in
`src/audit/AuditLogger.ts` and `src/audit/AuditVerifier.ts`.

Key properties:
- Keys sorted lexicographically at every level
- Unicode preserved as raw UTF-8 (no `\uXXXX` escaping beyond JSON minimum)
- No whitespace

Both implementations produce byte-identical output for all corpus cases.

## Related documentation

- [Audit Canonicalization](audit-canonicalization.md) — canonicalization design and rationale
- [Audit Log Signing](audit-signing.md) — how signed logs are produced and verified
- [Binding Validation](binding-validation.md) — ABI binding generation and staleness detection
- [Byte-Stable Binding Output Verification](binding-validation.md#byte-stable-binding-output-verification---byte-verify) — deterministic generation and byte-level diff verification
