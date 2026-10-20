# Audit Payload Validation Errors

This page is a compact reference for the validation failures that can occur when an audit payload is submitted for signing. Malformed payloads are rejected **before** signing; validation behaviour is unchanged by this document. Each entry below is backed by existing validation code and tests.

## Stable codes

| Code | Exit code | Meaning |
| --- | --- | --- |
| `VALIDATION_ERROR` | `2` | The audit payload failed validation. The request was rejected before signing. |

The `VALIDATION_ERROR` code is defined in `internal/clioutput/error.go` and mapped to exit code `2` in `src/exit-codes.ts`. When a payload is rejected, the CLI prints the stable code together with a field-level message so callers can correct the offending field.

## Where validation happens

- `internal/cmd/audit.go` — command-level validation of the audit payload before it is handed to the signer (payload presence, JSON decoding, required fields, timestamp format, size limits).
- `internal/audit/verify.go` — verification of audit record structure and field constraints.
- `internal/testhelpers/audit.go` — fixtures used by the tests that exercise each failure below.

## Common failures and corrective actions

### 1. Empty payload

- **Symptom:** The command reports `VALIDATION_ERROR` (exit code `2`) and states that the audit payload is empty.
- **Cause:** No payload was supplied, or the supplied payload contained only whitespace.
- **Corrective action:** Provide a non-empty JSON object for the audit payload. Ensure the input is not blank.
- **Example (invalid):**

  ```json
  {}
  ```

  An empty object has no fields to sign and is rejected. Supply the required fields described below.

### 2. Invalid JSON

- **Symptom:** `VALIDATION_ERROR` (exit code `2`) with a message indicating the payload could not be decoded as JSON.
- **Cause:** The payload is not well-formed JSON (for example, a trailing comma, an unquoted key, or a truncated object).
- **Corrective action:** Fix the JSON syntax. Validate the payload with a JSON parser before submitting it.
- **Example (invalid):**

  ```json
  {
    "action": "deploy",
  }
  ```

  The trailing comma after `"deploy"` makes this invalid JSON. Remove the trailing comma:

  ```json
  {
    "action": "deploy"
  }
  ```

### 3. Missing required fields

- **Symptom:** `VALIDATION_ERROR` (exit code `2`) naming the missing field.
- **Cause:** One or more required fields are absent from the payload.
- **Corrective action:** Add the named field with a valid value. Re-run validation after each correction until no missing-field errors remain.
- **Example (invalid):**

  ```json
  {
    "action": "deploy"
  }
  ```

  If a required field such as `timestamp` is absent, the payload is rejected. Add the missing field:

  ```json
  {
    "action": "deploy",
    "timestamp": "2024-01-01T00:00:00Z"
  }
  ```

### 4. Malformed timestamp

- **Symptom:** `VALIDATION_ERROR` (exit code `2`) indicating the timestamp field is not in the expected format.
- **Cause:** The `timestamp` value is not a valid RFC 3339 / ISO 8601 timestamp.
- **Corrective action:** Reformat the timestamp as RFC 3339 in UTC, for example `2024-01-01T00:00:00Z`. Do not use local time without an offset.
- **Example (invalid):**

  ```json
  {
    "action": "deploy",
    "timestamp": "01/01/2024"
  }
  ```

  Correct it to:

  ```json
  {
    "action": "deploy",
    "timestamp": "2024-01-01T00:00:00Z"
  }
  ```

### 5. Oversized payload

- **Symptom:** `VALIDATION_ERROR` (exit code `2`) stating the payload exceeds the maximum permitted size.
- **Cause:** The serialized payload is larger than the configured limit.
- **Corrective action:** Reduce the payload size. Remove unnecessary fields, trim large embedded values, or split the work into multiple smaller audit records.
- **Example:** A payload that embeds a large blob in a single field will exceed the limit. Move the blob out of the audit payload and reference it by identifier instead, keeping the signed record small.

## Notes

- All examples above are credential-free; they contain only field names and placeholder values.
- Validation is performed before signing, so a rejected payload is never signed.
- The stable code `VALIDATION_ERROR` and exit code `2` are the values to check in scripts and automation.
- For the authoritative field constraints, see the validation code in `internal/cmd/audit.go` and `internal/audit/verify.go`, and the fixtures in `internal/testhelpers/audit.go`.