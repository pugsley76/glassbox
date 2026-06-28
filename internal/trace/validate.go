// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// TraceInputError is returned when one or more trace-related CLI inputs are
// invalid. Each element in Failures is an actionable description of a single
// problem, so users can fix all issues in one pass.
type TraceInputError struct {
	Failures []string
}

func (e *TraceInputError) Error() string {
	if len(e.Failures) == 1 {
		return e.Failures[0]
	}
	lines := make([]string, 0, len(e.Failures)+1)
	lines = append(lines, fmt.Sprintf("%d trace input validation error(s):", len(e.Failures)))
	for i, f := range e.Failures {
		lines = append(lines, fmt.Sprintf("  %d. %s", i+1, f))
	}
	return strings.Join(lines, "\n")
}

// ValidateTraceInputs checks trace-related CLI flags for validity before any
// simulation or network fetch occurs.
//
// Parameters:
//   - verbosity: value of --trace-verbosity (may be empty → default normal)
//   - exportFormat: value of --format (may be empty → default text)
//   - eventFilter: value of an event-type filter (may be empty → no filter)
//   - outputPath: path supplied to --trace-output (may be empty → no export)
//
// Returns nil when all inputs are valid, or a *TraceInputError listing every
// problem found.
func ValidateTraceInputs(verbosity, exportFormat, eventFilter, outputPath string) error {
	var failures []string

	// Verbosity.
	if verbosity != "" {
		if _, err := ParseVerbosity(verbosity); err != nil {
			failures = append(failures, fmt.Sprintf(
				"invalid --trace-verbosity %q — must be one of: summary, normal, verbose\n"+
					"  Fix: use --trace-verbosity normal (default), summary (minimal), or verbose (detailed)",
				verbosity,
			))
		}
	}

	// Export format.
	if exportFormat != "" {
		normalizedFormat := strings.ToLower(strings.TrimSpace(exportFormat))
		switch normalizedFormat {
		case "text", "json", "html", "markdown", "md":
			// valid
		default:
			failures = append(failures, fmt.Sprintf(
				"invalid trace export format %q — must be one of: text, json, html, markdown\n"+
					"  Fix: use --format html (interactive), json (machine-readable), markdown (shareable), or text (CLI output)",
				exportFormat,
			))
		}
	}

	// Event filter.
	if eventFilter != "" {
		valid := false
		for _, t := range AllFilterableEventTypes() {
			if strings.EqualFold(eventFilter, t) {
				valid = true
				break
			}
		}
		if !valid {
			failures = append(failures, fmt.Sprintf(
				"invalid event filter %q — must be one of: %s\n"+
					"  Fix: choose a valid event type to filter trace output\n"+
					"  Available types: %s",
				eventFilter,
				strings.Join(AllFilterableEventTypes(), ", "),
				strings.Join(AllFilterableEventTypes(), ", "),
			))
		}
	}

	// Output path sanity: must not be a bare directory path.
	if outputPath != "" {
		if strings.HasSuffix(outputPath, "/") || strings.HasSuffix(outputPath, "\\") {
			failures = append(failures, fmt.Sprintf(
				"--trace-output %q looks like a directory path; provide a full file path\n"+
					"  Fix: specify a complete file path (e.g. ./traces/trace.html or ./output/trace.json)\n"+
					"  Example: glassbox debug --trace-output ./traces/debug-$(date +%%Y%%m%%d).html <tx-hash>",
				outputPath,
			))
		}

		// Null bytes in paths are a shell-injection risk.
		if strings.ContainsRune(outputPath, 0) {
			failures = append(failures, fmt.Sprintf(
				"--trace-output contains null bytes which are not allowed in file paths\n"+
					"  Fix: remove any null bytes from the path specification",
			))
		}

		// Use filepath.Clean to reliably detect traversal after normalisation.
		// A string-contains("..")  check would falsely flag names like "..safe"
		// or legitimate double-dot-free paths on some platforms.
		cleaned := filepath.Clean(outputPath)
		if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			failures = append(failures, fmt.Sprintf(
				"--trace-output %q contains directory traversal sequences (..)\n"+
					"  Fix: use absolute paths or relative paths without '..' for security\n"+
					"  Example: use './output/trace.html' instead of '../output/trace.html'",
				outputPath,
			))
		}
	}

	if len(failures) > 0 {
		return &TraceInputError{Failures: failures}
	}
	return nil
}

// ValidateEventTypeField checks whether an explicitly supplied EventType value
// in an ExecutionState is a known, supported value. Unknown values are
// normalised to EventTypeOther by ClassifyEventType — calling this function
// allows callers to surface a warning when the simulator emits an unrecognised
// event type rather than silently discarding it.
//
// Returns a non-empty diagnostic string when the value is unrecognised.
func ValidateEventTypeField(eventType string) string {
	if eventType == "" {
		return "" // empty is fine; the type will be inferred
	}
	normalised := normalizeEventType(eventType)
	if normalised == EventTypeOther {
		return fmt.Sprintf(
			"unrecognised event type %q (normalised to %q); "+
				"expected one of: %s. Trace accuracy may be reduced for this step. "+
				"Check that your simulator version is compatible with this version of Glassbox",
			eventType,
			EventTypeOther,
			strings.Join(append(AllFilterableEventTypes(), EventTypeOther), ", "),
		)
	}
	return ""
}

// ValidateExecutionTrace checks an ExecutionTrace for structural correctness
// and returns a list of diagnostic messages (non-fatal unless otherwise noted).
//
// Checks:
//   - Trace is not nil.
//   - States slice is not empty (empty trace → diagnostic warning).
//   - Each state has a non-negative Step that matches its slice index.
//   - Unrecognised EventType fields are noted with their step index.
//
// This is deliberately permissive: it returns all issues at once so callers can
// choose whether to abort or merely warn.
func ValidateExecutionTrace(t *ExecutionTrace) []string {
	if t == nil {
		return []string{"trace is nil — execution trace must not be nil"}
	}

	var issues []string

	if len(t.States) == 0 {
		issues = append(issues, fmt.Sprintf(
			"execution trace for transaction %q contains no steps — "+
				"the simulator did not produce any diagnostic events. "+
				"Check that the transaction envelope is valid and the simulator binary is up-to-date",
			truncateForDiag(t.TransactionHash),
		))
		return issues // nothing further to check on an empty trace
	}

	// Per-step checks.
	for i, state := range t.States {
		if state.Step != i {
			issues = append(issues, fmt.Sprintf(
				"step index mismatch at position %d: state.Step=%d "+
					"(trace may have been modified after construction; trace accuracy may be affected)",
				i, state.Step,
			))
		}
		if diag := ValidateEventTypeField(state.EventType); diag != "" {
			issues = append(issues, fmt.Sprintf("step %d: %s", i, diag))
		}
	}

	return issues
}

// truncateForDiag trims a string for use in diagnostic messages.
func truncateForDiag(s string) string {
	if len(s) > 20 {
		return s[:17] + "..."
	}
	return s
}

// ValidateJSONSchemaVersion validates a schema_version string as found in the
// ExportJSON envelope produced by --output-json. It rejects empty, malformed
// (not MAJOR.MINOR), or unsupported version strings with actionable messages.
//
// This is a pure-function validator suitable for use in PreRunE or any point
// where a schema version string is known before file I/O begins.
func ValidateJSONSchemaVersion(version string) error {
	if strings.TrimSpace(version) == "" {
		return &TraceInputError{Failures: []string{
			"schema_version is empty — a valid version string is required\n" +
				"  Expected format: \"MAJOR.MINOR\" (e.g. \"1.0\")\n" +
				"  Fix: use the current schema version: \"" + CurrentJSONSchemaVersion + "\"",
		}}
	}

	// Must match MAJOR.MINOR pattern (digits only, exactly two components).
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		return &TraceInputError{Failures: []string{fmt.Sprintf(
			"schema_version %q is not in MAJOR.MINOR format\n"+
				"  Expected a two-component version string (e.g. \"1.0\")\n"+
				"  Fix: use the current schema version: %q",
			version, CurrentJSONSchemaVersion,
		)}}
	}
	for _, p := range parts {
		if len(p) == 0 {
			return &TraceInputError{Failures: []string{fmt.Sprintf(
				"schema_version %q contains an empty component\n"+
					"  Fix: use a valid version such as %q",
				version, CurrentJSONSchemaVersion,
			)}}
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return &TraceInputError{Failures: []string{fmt.Sprintf(
					"schema_version %q contains non-numeric characters\n"+
						"  Expected: digits only (e.g. \"1.0\")\n"+
						"  Fix: use a valid schema version such as %q",
					version, CurrentJSONSchemaVersion,
				)}}
			}
		}
	}

	if !IsJSONSchemaVersionSupported(version) {
		return &TraceInputError{Failures: []string{fmt.Sprintf(
			"schema_version %q is not supported by this version of Glassbox\n"+
				"  Supported versions: %s\n"+
				"  Fix: re-export the trace with the current CLI, which produces schema version %q\n"+
				"  Tip: run 'glassbox trace --output-json <file> <trace-file>' to re-export",
			version,
			joinSupportedVersions(),
			CurrentJSONSchemaVersion,
		)}}
	}

	return nil
}

// joinSupportedVersions formats SupportedJSONSchemaVersions for error messages.
func joinSupportedVersions() string {
	parts := make([]string, len(SupportedJSONSchemaVersions))
	for i, v := range SupportedJSONSchemaVersions {
		parts[i] = fmt.Sprintf("%q", v)
	}
	return strings.Join(parts, ", ")
}

// ValidateTraceFormatCompatibility is defined in export.go and checks
// whether the trace data is compatible with the target export format.
// It is documented here for discoverability: see export.go for the
// implementation.
