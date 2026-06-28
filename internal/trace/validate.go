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
		return []string{"execution trace is nil"}
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

// ValidateTraceExportParams performs comprehensive validation of trace export parameters.
// It checks the trace, format, output path, and export options for validity before export.
// Returns a detailed error if validation fails, or nil if all checks pass.
func ValidateTraceExportParams(trace *ExecutionTrace, format, outputPath string, opts ExportOptions) error {
	var failures []string

	// 1. Validate trace is not nil
	if trace == nil {
		failures = append(failures, "trace is nil — cannot export a nil trace\n"+
			"  Fix: ensure a valid trace object is provided to the export function\n"+
			"  This typically means the trace failed to load or deserialize correctly")
	} else {
		// 2. Validate trace has states
		if len(trace.States) == 0 {
			failures = append(failures, "trace has no execution states — no steps in trace to export\n"+
				"  Possible causes:\n"+
				"    - The trace was captured successfully but no execution steps were recorded\n"+
				"    - The diagnostic events are empty or filtered out\n"+
				"    - The trace file may be truncated or corrupted\n"+
				"  Fix: re-run the simulation with --verbose to capture more detailed trace data")
		}

		// 3. Validate transaction hash is present
		if strings.TrimSpace(trace.TransactionHash) == "" {
			failures = append(failures, "trace has no transaction hash — transaction context is missing\n"+
				"  Fix: ensure the trace was created with a valid transaction hash\n"+
				"  This is usually set automatically when loading a trace from a file")
		}

		// 4. Validate time ordering if both times are set
		if !trace.StartTime.IsZero() && !trace.EndTime.IsZero() && trace.EndTime.Before(trace.StartTime) {
			failures = append(failures, "end time is before start time — trace has invalid temporal ordering\n"+
				"  Fix: verify the trace timestamps were recorded correctly\n"+
				"  Start: "+trace.StartTime.String()+", End: "+trace.EndTime.String())
		}
	}

	// 5. Validate format string
	if strings.TrimSpace(format) == "" {
		failures = append(failures, "--export-format is empty — must specify a format\n"+
			"  Fix: use --export-format with one of: html, markdown, json, text\n"+
			"  Default is html if not specified during export")
	} else {
		normalizedFormat := strings.ToLower(strings.TrimSpace(format))
		validFormats := map[string]bool{"html": true, "markdown": true, "md": true, "json": true, "text": true}
		if !validFormats[normalizedFormat] {
			failures = append(failures, fmt.Sprintf(
				"invalid --export-format %q — unsupported format; must be one of: html, markdown, json, text\n"+
					"  Fix: use a supported format\n"+
					"  html     — interactive HTML (best for browsers)\n"+
					"  markdown — GitHub-friendly markdown report\n"+
					"  json     — machine-readable JSON\n"+
					"  text     — plain text ASCII output",
				format))
		}
	}

	// 6. Validate output path
	if strings.TrimSpace(outputPath) == "" {
		failures = append(failures, "--export output path is empty — must specify a target file\n"+
			"  Fix: provide an output file path (e.g., ./trace.html or /tmp/report.md)\n"+
			"  Example: glassbox trace --export ./output/trace.html --format html input.json")
	} else {
		if strings.HasSuffix(outputPath, "/") || strings.HasSuffix(outputPath, "\\") {
			failures = append(failures, fmt.Sprintf(
				"--export path %q looks like a directory (ends with %q); provide a full file path\n"+
					"  Fix: specify a complete filename\n"+
					"  Example: --export ./output/trace.html instead of --export ./output/",
				outputPath, string(outputPath[len(outputPath)-1])))
		}
		if strings.Contains(outputPath, "\x00") {
			failures = append(failures, "output path contains null bytes — invalid file path\n"+
				"  Fix: remove any null bytes from the path")
		}
	}

	// 7. Validate Comments count and length
	if len(opts.Comments) > 100 {
		failures = append(failures, fmt.Sprintf(
			"too many comments (%d) — maximum is 100 comments per export\n"+
				"  Fix: reduce the number of comments or split the export into multiple files",
			len(opts.Comments)))
	}
	for i, comment := range opts.Comments {
		if strings.TrimSpace(comment) == "" {
			failures = append(failures, fmt.Sprintf(
				"comment #%d is empty or whitespace-only\n"+
					"  Fix: provide non-empty comments or remove empty entries from the list",
				i+1))
		}
		if len(comment) > 10000 {
			failures = append(failures, fmt.Sprintf(
				"comment #%d exceeds maximum length of 10000 characters (got %d)\n"+
					"  Fix: shorten the comment or split it into multiple comments",
				i+1, len(comment)))
		}
	}

	// 8. Validate ExportOptions.SessionMetadata keys and values
	for key, value := range opts.SessionMetadata {
		if strings.TrimSpace(key) == "" {
			failures = append(failures, "session metadata key is empty or whitespace-only\n"+
				"  Fix: provide non-empty keys for all metadata entries")
		}
		if strings.TrimSpace(value) == "" {
			failures = append(failures, fmt.Sprintf(
				"session metadata value for key %q is empty or whitespace-only\n"+
					"  Fix: provide non-empty values or omit the metadata entry", key))
		}
	}

	if len(failures) > 0 {
		if len(failures) == 1 {
			return &TraceInputError{Failures: []string{failures[0]}}
		}
		return &TraceInputError{Failures: failures}
	}
	return nil
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

// ValidateTraceFormatCompatibility checks if the trace data is suitable for the target export format.
// Some formats have specific requirements or may produce suboptimal results with certain trace data.
// Returns an error if the trace is fundamentally incompatible with the format, or nil if compatible.
func ValidateTraceFormatCompatibility(trace *ExecutionTrace, format string) error {
	if trace == nil {
		return fmt.Errorf("trace is nil — cannot check format compatibility")
	}

	normalizedFormat := strings.ToLower(strings.TrimSpace(format))

	// Format-specific validation checks
	switch normalizedFormat {
	case "html":
		if len(trace.States) > 50000 {
			return fmt.Errorf(
				"trace has %d steps — too large for HTML export (browser may become unresponsive)\n"+
					"  Fix: use --format json for large traces or filter the trace verbosity\n"+
					"  Alternatively: use --trace-verbosity summary to reduce output size",
				len(trace.States))
		}

		for i, state := range trace.States {
			argStr := fmt.Sprintf("%v", state.Arguments)
			if len(argStr) > 50000 {
				return fmt.Errorf(
					"step %d has very large arguments (%d chars) that may cause browser rendering issues in HTML format — consider using JSON format instead",
					i, len(argStr))
			}
		}

	case "markdown", "md":
		if len(trace.States) > 10000 {
			return fmt.Errorf(
				"trace has %d steps — markdown output will be extremely long (>1MB) and difficult to view\n"+
					"  Fix: use --format json for large traces or filter the trace verbosity",
				len(trace.States))
		}

		for i, state := range trace.States {
			if strings.Count(state.Error, "```") > 0 {
				return fmt.Errorf(
					"trace step %d error contains markdown code fence markers (```) — may break markdown formatting\n"+
						"  This is usually OK and will be handled gracefully, but you may want to review the step details",
					i)
			}
		}

	case "json":
		for i, state := range trace.States {
			if state.Step != i {
				return fmt.Errorf(
					"trace step mismatch at position %d: expected step %d but got %d — trace may be corrupted",
					i, i, state.Step)
			}
		}

	case "text":
		if len(trace.States) > 100000 {
			return fmt.Errorf(
				"trace has %d steps — plain text export will be extremely large (likely >5MB) and slow to generate\n"+
					"  Fix: use --format json for very large traces or filter the trace verbosity",
				len(trace.States))
		}

	default:
		return fmt.Errorf(
			"unsupported trace export format: %q\n"+
				"  Supported formats: html, markdown, json, text\n"+
				"  Fix: use --export-format with one of the supported values",
			format)
	}

	return nil
}
