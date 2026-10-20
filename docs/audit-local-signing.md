# Local Audit Signing (Test-Only Workflow)

This guide shows contributors how to exercise audit signing and verification locally
**without production keys, hardware, or real payloads**. Everything here uses a
throwaway key pair generated into a temporary directory and a minimal stub payload.

> **Never commit private keys or real audit payloads.** The workflow below writes
> all generated material to a `mktemp` directory that you delete when finished.

---

## Prerequisites

- The `glassbox` CLI built from your working tree (see the project README for build steps).
- A POSIX shell with `mktemp` available.

---

## Step 1 — Generate a throwaway key pair

Create a temporary directory and generate a test-only key pair into it:

```sh
TMPDIR="$(mktemp -d)"
glassbox audit:keygen --output-dir "$TMPDIR" --key-name local-test
```

This writes `local-test.private.pem` (and the corresponding public key) into the
temporary directory. The private key is **test-only** and must never leave this
directory or be committed.

---

## Step 2 — Sign a minimal test payload

Sign a stub payload that contains no production data. The shape below matches the
canonical fixture used by the test suite (`testhelpers.AuditPayloadFixture` in
`internal/testhelpers/audit.go`):

```sh
glassbox audit:sign \
  --payload '{"input":{},"state":{},"events":[],"timestamp":"1970-01-01T00:00:00Z"}' \
  --software-private-key "$TMPDIR/local-test.private.pem" \
  --output signed-audit.json
```

The signed audit log is written to `signed-audit.json` in the current directory.

---

## Step 3 — Verify the signed audit log

Verify the signature using the CLI:

```sh
glassbox audit:verify --audit-log signed-audit.json
```

A successful run confirms the signature is valid for the stub payload.

---

## Step 4 — Clean up

Remove the temporary key material and the signed output:

```sh
rm -rf "$TMPDIR"
rm -f signed-audit.json
```

---

## Keeping private material out of commits

Follow these rules whenever you generate keys or sign payloads locally:

- **Use `mktemp` directories.** Generate keys into a temporary directory (as shown
  above) rather than into the repository tree.
- **Ignore private key files.** Add a pattern such as `*.private.pem` to your
  `.gitignore` so stray keys are never staged.
- **Never commit keys or real payloads.** Only stub payloads like the one in Step 2
  belong in tests or documentation. Real audit payloads and production keys must
  never appear in the repository.
- **Clean up after yourself.** Delete temporary directories and signed outputs when
  you are done, as shown in Step 4.

---

## Reference

- The canonical stub payload shape is defined by `testhelpers.AuditPayloadFixture`
  in `internal/testhelpers/audit.go`. Reuse it when writing new tests so local
  signing stays consistent with the test suite.