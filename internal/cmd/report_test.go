// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Part B: debug command reporting and summaries improvements.
// Covers validateReportFlags, reportExec error paths, and summary helpers.

package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/trace"
)

// ── validateReportFlags ───────────────────────────────────────────────────────

// TestValidateReportFlags_MissingFile verifies that omitting --file produces
// a clear actionable error with usage guidance.
func TestValidateReportFlags_MissingFile(t *testing.T) {
	prev := reportFile
	t.Cleanup(func() { reportFile = prev })
	reportFile = ""

	err := validateReportFlags(reportCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --file is not set")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--file") {
		t.Errorf("error should mention --file, got: %q", msg)
	}
	// Must include usage guidance.
	if !strings.Contains(msg, "Usage") && !strings.Contains(msg, "glassbox report") {
		t.Errorf("error should include usage hint, got: %q", msg)
	}
}

// TestValidateReportFlags_InvalidFormat verifies that an unsupported --format
// value surfaces a clear message listing valid options.
func TestValidateReportFlags_InvalidFormat(t *testing.T) {
	prevFile := reportFile
	prevFmt := reportFormat
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
	})
	reportFile = "trace.json" // non-empty so we get to format validation
	reportFormat = "xml"      // invalid

	err := validateReportFlags(reportCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid --format")
	}
	msg := err.Error()
	if !strings.Contains(msg, "xml") {
		t.Errorf("error should echo the invalid value, got: %q", msg)
	}
	if !strings.Contains(msg, "html") {
		t.Errorf("error should list valid formats, got: %q", msg)
	}
}

// TestValidateReportFlags_ValidFormats verifies that all valid format strings
// pass without error.
func TestValidateReportFlags_ValidFormats(t *testing.T) {
	validFormats := []string{"html", "pdf", "html,pdf", "pdf,html", "json", "text"}
	for _, f := range validFormats {
		f := f
		t.Run(f, func(t *testing.T) {
			prevFile := reportFile
			prevFmt := reportFormat
			t.Cleanup(func() {
				reportFile = prevFile
				reportFormat = prevFmt
			})
			reportFile = "trace.json"
			reportFormat = f

			err := validateReportFlags(reportCmd, []string{})
			if err != nil {
				t.Errorf("validateReportFlags() should accept --format=%s, got: %v", f, err)
			}
		})
	}
}

// ── reportExec error paths ────────────────────────────────────────────────────

// TestReportExec_TraceFileNotFound verifies that a missing trace file produces
// a clear, actionable error naming the file path.
func TestReportExec_TraceFileNotFound(t *testing.T) {
	prevFile := reportFile
	prevFmt := reportFormat
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
	})
	reportFile = "/nonexistent/path/trace.json"
	reportFormat = "text"

	err := reportExec(reportCmd, []string{})
	if err == nil {
		t.Fatal("expected error for non-existent trace file")
	}
	msg := err.Error()
	if !strings.Contains(msg, "trace file not found") && !strings.Contains(msg, "not found") {
		t.Errorf("error should say file not found, got: %q", msg)
	}
	if !strings.Contains(msg, "/nonexistent") {
		t.Errorf("error should include the file path, got: %q", msg)
	}
}

// TestReportExec_EmptyFileError verifies that an existing but empty file
// produces a clear error rather than a panic or nil-pointer crash.
func TestReportExec_EmptyFileError(t *testing.T) {
	tmpDir := t.TempDir()
	tmp := tmpDir + "/empty.json"
	if err := os.WriteFile(tmp, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	prevFile := reportFile
	prevFmt := reportFormat
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
	})
	reportFile = tmp
	reportFormat = "text"

	err := reportExec(reportCmd, []string{})
	if err == nil {
		t.Fatal("expected error for empty trace file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should say file is empty, got: %q", err.Error())
	}
}

// TestReportExec_InvalidJSONError verifies that a malformed JSON trace file
// produces an informative parse error with a remediation hint.
func TestReportExec_InvalidJSONError(t *testing.T) {
	tmpDir := t.TempDir()
	tmp := tmpDir + "/bad.json"
	if err := os.WriteFile(tmp, []byte("{invalid json}"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	prevFile := reportFile
	prevFmt := reportFormat
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
	})
	reportFile = tmp
	reportFormat = "text"

	err := reportExec(reportCmd, []string{})
	if err == nil {
		t.Fatal("expected error for invalid JSON trace file")
	}
	// Error must tell user how to produce a valid trace file.
	msg := err.Error()
	if !strings.Contains(msg, "glassbox debug") && !strings.Contains(msg, "glassbox trace") &&
		!strings.Contains(msg, "parse") && !strings.Contains(msg, "JSON") &&
		!strings.Contains(msg, "unmarshal") {
		t.Errorf("error should guide user toward fixing the trace, got: %q", msg)
	}
}

// ── summary helpers ──────────────────────────────────────────────────────────

// TestCountErrors_Empty verifies countErrors returns 0 for empty state list.
func TestCountErrors_Empty(t *testing.T) {
	if got := countErrors(nil); got != 0 {
		t.Errorf("countErrors(nil) = %d; want 0", got)
	}
}

// TestCountErrors_Mixed verifies countErrors counts only steps with non-empty errors.
func TestCountErrors_Mixed(t *testing.T) {
	states := []trace.ExecutionState{
		{Error: ""},
		{Error: "panic"},
		{Error: ""},
		{Error: "trap"},
	}
	if got := countErrors(states); got != 2 {
		t.Errorf("countErrors() = %d; want 2", got)
	}
}

// TestCalculateSuccessRate_AllSuccess verifies 100% when no errors.
func TestCalculateSuccessRate_AllSuccess(t *testing.T) {
	states := []trace.ExecutionState{{Error: ""}, {Error: ""}}
	if got := calculateSuccessRate(states); got != 100.0 {
		t.Errorf("calculateSuccessRate() = %v; want 100.0", got)
	}
}

// TestCalculateSuccessRate_Empty verifies 100% for empty states (safe default).
func TestCalculateSuccessRate_Empty(t *testing.T) {
	if got := calculateSuccessRate(nil); got != 100.0 {
		t.Errorf("calculateSuccessRate(nil) = %v; want 100.0", got)
	}
}

// TestCalculateSuccessRate_HalfErrors verifies 50% when half the steps fail.
func TestCalculateSuccessRate_HalfErrors(t *testing.T) {
	states := []trace.ExecutionState{
		{Error: "err"},
		{Error: ""},
	}
	if got := calculateSuccessRate(states); got != 50.0 {
		t.Errorf("calculateSuccessRate() = %v; want 50.0", got)
	}
}

// TestAssessRisk_NoCritical verifies low risk when no errors.
func TestAssessRisk_NoCritical(t *testing.T) {
	states := []trace.ExecutionState{{Error: ""}, {Error: ""}}
	if got := assessRisk(states); got != "low" {
		t.Errorf("assessRisk() = %q; want low", got)
	}
}

// TestAssessRisk_CriticalMajority verifies critical risk when >= half fail.
func TestAssessRisk_CriticalMajority(t *testing.T) {
	states := []trace.ExecutionState{
		{Error: "err"}, {Error: "err"}, {Error: ""},
	}
	got := assessRisk(states)
	// 2 out of 3 errors → >= half → critical
	if got != "critical" {
		t.Errorf("assessRisk() = %q; want critical", got)
	}
}

// TestCountContracts_Deduplication verifies unique contract IDs are counted once.
func TestCountContracts_Deduplication(t *testing.T) {
	states := []trace.ExecutionState{
		{ContractID: "contractA"},
		{ContractID: "contractA"},
		{ContractID: "contractB"},
		{ContractID: ""},
	}
	if got := countContracts(states); got != 2 {
		t.Errorf("countContracts() = %d; want 2", got)
	}
}

// TestCalculateRiskScore_ZeroForEmpty verifies 0 risk score for empty states.
func TestCalculateRiskScore_ZeroForEmpty(t *testing.T) {
	if got := calculateRiskScore(nil); got != 0 {
		t.Errorf("calculateRiskScore(nil) = %v; want 0", got)
	}
}

// TestCalculateRiskScore_AllErrors verifies 100% risk for all-error states.
func TestCalculateRiskScore_AllErrors(t *testing.T) {
	states := []trace.ExecutionState{
		{Error: "err1"},
		{Error: "err2"},
	}
	if got := calculateRiskScore(states); got != 100.0 {
		t.Errorf("calculateRiskScore() = %v; want 100.0", got)
	}
}

// ── output directory behavior ────────────────────────────────────────────────

// TestReportExec_OutputDirAutoCreated verifies that a non-existent --output
// directory is created automatically and the report is written successfully.
func TestReportExec_OutputDirAutoCreated(t *testing.T) {
	// Build a minimal valid trace file.
	tmpBase := t.TempDir()
	traceFile := tmpBase + "/trace.json"
	traceJSON := `{"transaction_hash":"abc","states":[{"step":0,"operation":"test"}]}`
	if err := os.WriteFile(traceFile, []byte(traceJSON), 0644); err != nil {
		t.Fatalf("failed to create trace file: %v", err)
	}

	newOutputDir := tmpBase + "/auto-created-reports"

	prevFile := reportFile
	prevFmt := reportFormat
	prevOut := reportOutput
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
		reportOutput = prevOut
	})
	reportFile = traceFile
	reportFormat = "text"
	reportOutput = newOutputDir

	err := reportExec(reportCmd, []string{})
	if err != nil {
		t.Fatalf("reportExec should succeed with auto-created output dir, got: %v", err)
	}

	// Verify directory was created.
	if info, statErr := os.Stat(newOutputDir); statErr != nil || !info.IsDir() {
		t.Errorf("expected output directory to be created at %q", newOutputDir)
	}

	// Verify the report file exists inside the directory.
	reportPath := newOutputDir + "/report.txt"
	if _, statErr := os.Stat(reportPath); statErr != nil {
		t.Errorf("expected report file at %q, got: %v", reportPath, statErr)
	}
}

// TestReportExec_OutputIsFile_Error verifies that --output pointing to an
// existing regular file (not a directory) produces a clear error.
func TestReportExec_OutputIsFile_Error(t *testing.T) {
	tmpBase := t.TempDir()
	traceFile := tmpBase + "/trace.json"
	traceJSON := `{"transaction_hash":"abc","states":[{"step":0,"operation":"test"}]}`
	if err := os.WriteFile(traceFile, []byte(traceJSON), 0644); err != nil {
		t.Fatalf("failed to create trace file: %v", err)
	}

	// Create a regular file where the output directory is expected.
	existingFile := tmpBase + "/not-a-dir"
	if err := os.WriteFile(existingFile, []byte("file"), 0644); err != nil {
		t.Fatalf("failed to create placeholder file: %v", err)
	}

	prevFile := reportFile
	prevFmt := reportFormat
	prevOut := reportOutput
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
		reportOutput = prevOut
	})
	reportFile = traceFile
	reportFormat = "text"
	reportOutput = existingFile

	err := reportExec(reportCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --output points to an existing file")
	}
	if !strings.Contains(err.Error(), "not a directory") && !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention the path is not a directory, got: %q", err.Error())
	}
}

// TestReportExec_TextReport_OutputFileCreated verifies that the text report
// file is created inside the output directory with the expected name.
func TestReportExec_TextReport_OutputFileCreated(t *testing.T) {
	tmpBase := t.TempDir()
	traceFilePath := tmpBase + "/trace.json"
	traceJSON := `{"transaction_hash":"abc","states":[{"step":0,"operation":"test"}]}`
	if err := os.WriteFile(traceFilePath, []byte(traceJSON), 0644); err != nil {
		t.Fatalf("failed to create trace file: %v", err)
	}

	prevFile := reportFile
	prevFmt := reportFormat
	prevOut := reportOutput
	t.Cleanup(func() {
		reportFile = prevFile
		reportFormat = prevFmt
		reportOutput = prevOut
	})
	reportFile = traceFilePath
	reportFormat = "text"
	reportOutput = tmpBase

	var out strings.Builder
	reportCmd.SetOut(&out)
	t.Cleanup(func() { reportCmd.SetOut(nil) })

	if err := reportExec(reportCmd, []string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// report.txt must be present in the output directory.
	entries, readErr := os.ReadDir(tmpBase)
	if readErr != nil {
		t.Fatalf("failed to read output directory: %v", readErr)
	}
	found := false
	for _, e := range entries {
		if e.Name() == "report.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected report.txt inside %q", tmpBase)
	}
}
