# Audit Signing Providers

Glassbox supports three audit signing providers. They differ in where the private key lives and what you must prepare before enabling signing. Use the comparison below to pick the guide that matches your custody requirements.

> **Glassbox does not store user private keys in any provider mode.** In software mode the key is read from the location you supply and used only in memory; in PKCS#11 mode the key never leaves the token; in KMS mode signing happens inside AWS KMS and Glassbox never receives the key material.

## Provider comparison

| Provider (`--signing-provider`) | Prerequisites | Key handling / custody | Setup guide |
| --- | --- | --- | --- |
| `software` | A PKCS#8 PEM-encoded Ed25519 private key, supplied via `--software-private-key` or the `GLASSBOX_AUDIT_PRIVATE_KEY_PEM` environment variable. | The key is read by the Glassbox process and used only in memory for signing. Glassbox never stores or persists the key. You are responsible for custody of the key file/secret. | [Software signing setup](audit-signing-software.md) |
| `pkcs11` | A PKCS#11 module path (`--pkcs11-module` or `GLASSBOX_PKCS11_MODULE`), the token PIN, and a key label or key ID identifying the signing key on the token. | Private key material stays on the hardware token/HSM and is never exported. Signing is performed by the token; Glassbox only sends the data to be signed. | [PKCS#11 signing setup](audit-signing-pkcs11.md) |
| `kms` | AWS credentials with permission to use the key, a KMS key ID or ARN (`--kms-key-id` or `GLASSBOX_KMS_KEY_ID`), and a signing algorithm. | Signing happens inside AWS KMS. Glassbox never receives the private key; only the signature is returned. Key custody remains with AWS KMS and your IAM policy. | [KMS signing setup](audit-signing-kms.md) |

## Flag reference

The flags above are implemented in `internal/cmd/audit_sign.go` (Go CLI) and `src/commands/audit.ts` (Node CLI). Both surfaces accept the same provider selection and credential inputs:

- `--signing-provider` — selects `software`, `pkcs11`, or `kms`.
- `--software-private-key` / `GLASSBOX_AUDIT_PRIVATE_KEY_PEM` — software provider key input.
- `--pkcs11-module` / `GLASSBOX_PKCS11_MODULE` — PKCS#11 module path.
- `--kms-key-id` / `GLASSBOX_KMS_KEY_ID` — KMS key identifier or ARN.

All of these flags are recorded in the API snapshot so that `scripts/validate-docs.sh` passes.