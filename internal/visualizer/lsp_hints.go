// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package visualizer

// lsp_hints.go provides the visualizer helpers consumed by the LSP server
// (internal/lsp/server.go) for hover content and inline diagnostic hints.
//
// These are deliberately kept in the visualizer package so the presentation
// layer is owned in one place: the same host-function descriptions and
// diagnostic messages that appear in the terminal output also drive the
// inline editor annotations, without duplicating the strings or the detection
// logic.

import (
	"fmt"
	"strings"
)

// SourceHint is an inline editor annotation for a single source line.
type SourceHint struct {
	// Line is the 0-based line index the annotation belongs to.
	Line int
	// Start is the 0-based start character index.
	Start uint32
	// End is the 0-based end character index (exclusive).
	End uint32
	// Message is the human-readable annotation text.
	Message string
}

// knownHostFunctions is the authoritative list of Soroban host function names
// that the LSP server recognises for hover and diagnostic hints. It includes
// both deprecated and unsafe functions so a developer gets inline warnings the
// moment they type a call.
//
// Keep in sync with internal/security/host_functions.go — that package owns
// the classification (deprecated vs unsafe) while this package owns the
// presentation.
var knownHostFunctions = []string{
	// Deprecated — scheduled for removal in a future protocol release.
	"bytes_copy_from_linear_memory",
	"bytes_copy_to_linear_memory",
	"bytes_new_from_linear_memory",
	"map_new_from_linear_memory",
	"map_unpack_to_linear_memory",
	"symbol_new_from_linear_memory",
	"string_new_from_linear_memory",
	"vec_new_from_linear_memory",
	"vec_unpack_to_linear_memory",
	// Unsafe — still permitted but risky without an explicit authorization check.
	"storage_put",
	"contract_call",
	// Commonly referenced non-deprecated host functions that benefit from
	// hover docs but generate no diagnostic.
	"require_auth",
	"storage_get",
	"storage_has",
	"storage_remove",
	"contract_id",
	"ledger_sequence",
	"ledger_timestamp",
}

// hostFunctionDocs holds the markdown hover documentation for each host
// function. The map key is the exact function name from knownHostFunctions.
var hostFunctionDocs = map[string]string{
	// ── Deprecated ────────────────────────────────────────────────────────────
	"bytes_copy_from_linear_memory": "**[DEPRECATED]** `bytes_copy_from_linear_memory`\n\n" +
		"Copies bytes from linear memory into a Soroban Bytes value.\n\n" +
		"**Migration:** Use `bytes_new_from_linear_memory` and copy the slice instead.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"bytes_copy_to_linear_memory": "**[DEPRECATED]** `bytes_copy_to_linear_memory`\n\n" +
		"Copies bytes from a Soroban Bytes value into linear memory.\n\n" +
		"**Migration:** Use `bytes_new_from_linear_memory` and copy the slice instead.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"bytes_new_from_linear_memory": "**[DEPRECATED]** `bytes_new_from_linear_memory`\n\n" +
		"Creates a new Soroban Bytes value from a linear memory slice.\n\n" +
		"**Migration:** Use `bytes_new_from_linear_memory_from_slice` where available.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"map_new_from_linear_memory": "**[DEPRECATED]** `map_new_from_linear_memory`\n\n" +
		"Creates a new Soroban Map value from linear memory key/value pairs.\n\n" +
		"**Migration:** Construct the map on the host and pass it as an argument.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"map_unpack_to_linear_memory": "**[DEPRECATED]** `map_unpack_to_linear_memory`\n\n" +
		"Unpacks a Soroban Map into linear memory key/value buffers.\n\n" +
		"**Migration:** Construct the map on the host and pass it as an argument.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"symbol_new_from_linear_memory": "**[DEPRECATED]** `symbol_new_from_linear_memory`\n\n" +
		"Creates a new Soroban Symbol from a linear memory byte slice.\n\n" +
		"**Migration:** Use `symbol_new_from_linear_memory` with an explicit length parameter.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"string_new_from_linear_memory": "**[DEPRECATED]** `string_new_from_linear_memory`\n\n" +
		"Creates a new Soroban String from a linear memory byte slice.\n\n" +
		"**Migration:** Use `string_new_from_linear_memory` with an explicit length parameter.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"vec_new_from_linear_memory": "**[DEPRECATED]** `vec_new_from_linear_memory`\n\n" +
		"Creates a new Soroban Vec from a linear memory element slice.\n\n" +
		"**Migration:** Construct the vector on the host and pass it as an argument.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	"vec_unpack_to_linear_memory": "**[DEPRECATED]** `vec_unpack_to_linear_memory`\n\n" +
		"Unpacks a Soroban Vec into a linear memory element buffer.\n\n" +
		"**Migration:** Construct the vector on the host and pass it as an argument.\n\n" +
		"This host function is scheduled for removal in a future Soroban protocol release.",

	// ── Unsafe ────────────────────────────────────────────────────────────────
	"storage_put": "**[UNSAFE]** `storage_put`\n\n" +
		"Writes a value into contract persistent storage.\n\n" +
		"**Warning:** This call writes to storage without an explicit authorization check. " +
		"Ensure `require_auth` has been called before writing to prevent unauthorized state mutations.\n\n" +
		"See: [Soroban authorization model](https://soroban.stellar.org/docs/fundamentals/authorization)",

	"contract_call": "**[UNSAFE]** `contract_call`\n\n" +
		"Calls another Soroban contract.\n\n" +
		"**Warning:** Cross-contract calls can create re-entrancy risks and unexpected auth contexts. " +
		"Validate that the callee is trusted and that your contract's invariants hold " +
		"before and after the call.",

	// ── Current ───────────────────────────────────────────────────────────────
	"require_auth": "`require_auth`\n\n" +
		"Requires that the given address has authorized the current contract invocation. " +
		"Panics with an auth error if the authorization is not present.\n\n" +
		"This is the standard Soroban authorization primitive — always call it before " +
		"performing a privileged operation.\n\n" +
		"See: [Soroban authorization model](https://soroban.stellar.org/docs/fundamentals/authorization)",

	"storage_get": "`storage_get`\n\n" +
		"Reads a value from contract persistent storage. " +
		"Returns the value associated with the given key, or panics if the key is absent.",

	"storage_has": "`storage_has`\n\n" +
		"Returns `true` if the given key exists in contract persistent storage.",

	"storage_remove": "`storage_remove`\n\n" +
		"Removes the given key from contract persistent storage. " +
		"No-ops if the key is absent.",

	"contract_id": "`contract_id`\n\n" +
		"Returns the current contract's address as a Soroban Address value.",

	"ledger_sequence": "`ledger_sequence`\n\n" +
		"Returns the current ledger sequence number as a `u32`.",

	"ledger_timestamp": "`ledger_timestamp`\n\n" +
		"Returns the current ledger close time as a Unix timestamp (`u64` seconds since epoch).",
}

// hostFunctionDiagnosticMessages holds the inline diagnostic messages for
// host functions that require a warning annotation. Functions not present in
// this map get no diagnostic, only hover docs.
var hostFunctionDiagnosticMessages = map[string]string{
	// Deprecated functions.
	"bytes_copy_from_linear_memory":  "deprecated host function: use bytes_new_from_linear_memory and copy the slice instead",
	"bytes_copy_to_linear_memory":    "deprecated host function: use bytes_new_from_linear_memory and copy the slice instead",
	"bytes_new_from_linear_memory":   "deprecated host function: use bytes_new_from_linear_memory_from_slice where available",
	"map_new_from_linear_memory":     "deprecated host function: construct the map on the host and pass it as an argument",
	"map_unpack_to_linear_memory":    "deprecated host function: construct the map on the host and pass it as an argument",
	"symbol_new_from_linear_memory":  "deprecated host function: use symbol_new_from_linear_memory with an explicit length",
	"string_new_from_linear_memory":  "deprecated host function: use string_new_from_linear_memory with an explicit length",
	"vec_new_from_linear_memory":     "deprecated host function: construct the vector on the host and pass it as an argument",
	"vec_unpack_to_linear_memory":    "deprecated host function: construct the vector on the host and pass it as an argument",
	// Unsafe functions.
	"storage_put":    "unsafe host function: storage_put writes to storage without an authorization check — call require_auth first",
	"contract_call":  "unsafe host function: cross-contract calls can create re-entrancy risks — validate the callee is trusted",
}

// KnownHostFunctions returns the list of Soroban host function names that the
// LSP server and hover provider recognise. The slice is a copy; callers may
// not modify it.
func KnownHostFunctions() []string {
	out := make([]string, len(knownHostFunctions))
	copy(out, knownHostFunctions)
	return out
}

// HostFunctionHoverContent returns a Markdown string suitable for an LSP hover
// response for the given host function name.  Returns a generic fallback when
// the function is not in the documentation table.
func HostFunctionHoverContent(name string) string {
	if doc, ok := hostFunctionDocs[name]; ok {
		return doc
	}
	return fmt.Sprintf("`%s`\n\nSoroban host function. See the [Soroban documentation](https://soroban.stellar.org/docs) for details.", name)
}

// DiagnosticsForSource scans source text and returns one SourceHint per line
// that contains a call to a host function which warrants an inline diagnostic
// (deprecated or unsafe functions).
//
// Matching is by substring: each line is scanned for each diagnostic function
// name, and the first match per function per line is reported. This is
// intentionally simple — the LSP server feeds full document text, so a
// word-boundary false-positive rate is acceptable for an informational hint.
func DiagnosticsForSource(source string) []SourceHint {
	if source == "" {
		return nil
	}

	lines := strings.Split(source, "\n")
	var hints []SourceHint

	for lineIdx, line := range lines {
		lower := strings.ToLower(line)
		for name, msg := range hostFunctionDiagnosticMessages {
			idx := strings.Index(lower, name)
			if idx < 0 {
				continue
			}
			hints = append(hints, SourceHint{
				Line:    lineIdx,
				Start:   uint32(idx),
				End:     uint32(idx + len(name)),
				Message: msg,
			})
		}
	}

	return hints
}
