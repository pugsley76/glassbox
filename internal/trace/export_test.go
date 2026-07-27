// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExportExecutionTrace_HTMLAndMarkdown(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	trace.EndTime = trace.StartTime.Add(5 * time.Minute)

	trace.AddState(ExecutionState{
		Operation:   "contract_call",
		EventType:   "contract_call",
		ContractID:  "C123",
		Function:    "transfer",
		Arguments:   []interface{}{"100", "XLM"},
		ReturnValue: "ok",
		SourceFile:  "src/contract.rs",
		SourceLine:  42,
		GitHubLink:  "https://github.com/example/repo/blob/main/src/contract.rs#L42",
	})
	trace.AddState(ExecutionState{
		Operation: "host_function",
		Error:     "insufficient balance",
	})

	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "trace-export.html")
	if err := ExportExecutionTrace(trace, "html", htmlPath); err != nil {
		t.Fatalf("ExportExecutionTrace(html) failed: %v", err)
	}
	data, err := os.ReadFile(htmlPath)
	if err != nil {
		t.Fatalf("failed to read exported html file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Glassbox Trace Export") {
		t.Fatalf("exported html missing expected header")
	}
	if !strings.Contains(content, "contract_call") {
		t.Fatalf("exported html missing step operation")
	}

	mdPath := filepath.Join(tmpDir, "trace-export.md")
	if err := ExportExecutionTrace(trace, "markdown", mdPath); err != nil {
		t.Fatalf("ExportExecutionTrace(markdown) failed: %v", err)
	}
	data, err = os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read exported markdown file: %v", err)
	}
	content = string(data)
	if !strings.Contains(content, "# Glassbox Trace Export") {
		t.Fatalf("exported markdown missing expected header")
	}
	if !strings.Contains(content, "transfer") {
		t.Fatalf("exported markdown missing expected function name")
	}

	txtPath := filepath.Join(tmpDir, "trace-export.txt")
	if err := ExportExecutionTrace(trace, "text", txtPath); err != nil {
		t.Fatalf("ExportExecutionTrace(text) failed: %v", err)
	}
	data, err = os.ReadFile(txtPath)
	if err != nil {
		t.Fatalf("failed to read exported text file: %v", err)
	}
	content = string(data)
	if !strings.Contains(content, "Glassbox Trace Export") {
		t.Fatalf("exported text missing expected header")
	}
	if !strings.Contains(content, "  Contract:  C123") {
		t.Fatalf("exported text missing indented contract field")
	}
	if !strings.Contains(content, "  Source:") {
		t.Fatalf("exported text missing source reference section")
	}
}

// Test ValidateTraceExportParams with nil trace
func TestValidateTraceExportParams_NilTrace(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	err := ValidateTraceExportParams(nil, "html", outputPath, ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject nil trace")
	}
	if !strings.Contains(err.Error(), "trace is nil") {
		t.Fatalf("error message should mention nil trace, got: %v", err)
	}
}

// Test ValidateTraceExportParams with empty trace (no states)
func TestValidateTraceExportParams_EmptyTrace(t *testing.T) {
	trace := NewExecutionTrace("test-tx", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	err := ValidateTraceExportParams(trace, "html", outputPath, ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject trace with no states")
	}
	if !strings.Contains(err.Error(), "no execution states") {
		t.Fatalf("error should mention empty trace, got: %v", err)
	}
}

// Test ValidateTraceExportParams with missing transaction hash
func TestValidateTraceExportParams_MissingTransactionHash(t *testing.T) {
	trace := NewExecutionTrace("", 10) // empty tx hash
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	err := ValidateTraceExportParams(trace, "html", outputPath, ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject trace without transaction hash")
	}
	if !strings.Contains(err.Error(), "transaction hash") {
		t.Fatalf("error should mention missing tx hash, got: %v", err)
	}
}

// Test ValidateTraceExportParams with invalid time ordering (end before start)
func TestValidateTraceExportParams_InvalidTimeOrdering(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	now := time.Now()
	trace.StartTime = now
	trace.EndTime = now.Add(-1 * time.Second) // end time before start
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	err := ValidateTraceExportParams(trace, "html", outputPath, ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject trace with invalid time ordering")
	}
	if !strings.Contains(err.Error(), "end time is before start time") {
		t.Fatalf("error should mention invalid time ordering, got: %v", err)
	}
}

// Test ValidateTraceExportParams with invalid format
func TestValidateTraceExportParams_InvalidFormat(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	err := ValidateTraceExportParams(trace, "invalid-format", outputPath, ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject invalid format")
	}
	if !strings.Contains(err.Error(), "invalid --export-format") {
		t.Fatalf("error should mention invalid format, got: %v", err)
	}
}

// Test ValidateTraceExportParams with empty format (should accept, defaults to html)
func TestValidateTraceExportParams_EmptyFormat(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	err := ValidateTraceExportParams(trace, "", outputPath, ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject empty format string")
	}
}

// Test ValidateTraceExportParams with empty output path
func TestValidateTraceExportParams_EmptyOutputPath(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	err := ValidateTraceExportParams(trace, "html", "", ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject empty output path")
	}
	if !strings.Contains(err.Error(), "output path is empty") {
		t.Fatalf("error should mention empty output path, got: %v", err)
	}
}

// Test ValidateTraceExportParams with directory-like path
func TestValidateTraceExportParams_DirectoryLikePath(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	// Test with trailing slash
	err := ValidateTraceExportParams(trace, "html", "/tmp/output/", ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject directory-like path with /")
	}

	// Test with trailing backslash
	err = ValidateTraceExportParams(trace, "html", "C:\\output\\", ExportOptions{})
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject directory-like path with \\")
	}
}

// Test ValidateTraceExportParams with valid inputs
func TestValidateTraceExportParams_ValidInputs(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	// Should succeed with valid inputs
	err := ValidateTraceExportParams(trace, "html", outputPath, ExportOptions{})
	if err != nil {
		t.Fatalf("ValidateTraceExportParams should accept valid inputs, got: %v", err)
	}

	// Test with json format
	err = ValidateTraceExportParams(trace, "json", outputPath, ExportOptions{})
	if err != nil {
		t.Fatalf("ValidateTraceExportParams should accept json format, got: %v", err)
	}

	// Test with markdown format
	err = ValidateTraceExportParams(trace, "markdown", outputPath, ExportOptions{})
	if err != nil {
		t.Fatalf("ValidateTraceExportParams should accept markdown format, got: %v", err)
	}

	// Test with text format
	err = ValidateTraceExportParams(trace, "text", outputPath, ExportOptions{})
	if err != nil {
		t.Fatalf("ValidateTraceExportParams should accept text format, got: %v", err)
	}
}

// Test ValidateTraceExportParams with empty comments in ExportOptions
func TestValidateTraceExportParams_EmptyComments(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	opts := ExportOptions{
		Comments: []string{"valid comment", "   ", "another valid"},
	}

	err := ValidateTraceExportParams(trace, "html", outputPath, opts)
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject empty comments")
	}
}

// Test ValidateTraceExportParams with empty metadata keys/values
func TestValidateTraceExportParams_EmptyMetadata(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.html")

	// Empty key
	opts := ExportOptions{
		SessionMetadata: map[string]string{"": "value"},
	}
	err := ValidateTraceExportParams(trace, "html", outputPath, opts)
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject empty metadata keys")
	}

	// Empty value
	opts = ExportOptions{
		SessionMetadata: map[string]string{"key": ""},
	}
	err = ValidateTraceExportParams(trace, "html", outputPath, opts)
	if err == nil {
		t.Fatalf("ValidateTraceExportParams should reject empty metadata values")
	}
}

// Test ValidateTraceFormatCompatibility with nil trace
func TestValidateTraceFormatCompatibility_NilTrace(t *testing.T) {
	err := ValidateTraceFormatCompatibility(nil, "html")
	if err == nil {
		t.Fatalf("ValidateTraceFormatCompatibility should reject nil trace")
	}
}

// Test ValidateTraceFormatCompatibility with very large trace and HTML format
func TestValidateTraceFormatCompatibility_LargeTraceHTML(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)

	// Add 60,000 states to exceed HTML format limit
	for i := 0; i < 60000; i++ {
		trace.AddState(ExecutionState{Operation: "op", Function: "f"})
	}

	err := ValidateTraceFormatCompatibility(trace, "html")
	if err == nil {
		t.Fatalf("ValidateTraceFormatCompatibility should warn about large traces with HTML")
	}
	if !strings.Contains(err.Error(), "too large for HTML export") {
		t.Fatalf("error should mention HTML size limit, got: %v", err)
	}
}

// Test ValidateTraceFormatCompatibility with large trace and markdown format
func TestValidateTraceFormatCompatibility_LargeTraceMarkdown(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)

	// Add 15,000 states to exceed markdown format limit
	for i := 0; i < 15000; i++ {
		trace.AddState(ExecutionState{Operation: "op", Function: "f"})
	}

	err := ValidateTraceFormatCompatibility(trace, "markdown")
	if err == nil {
		t.Fatalf("ValidateTraceFormatCompatibility should warn about large traces with markdown")
	}
}

// Test ValidateTraceFormatCompatibility with valid trace and various formats
func TestValidateTraceFormatCompatibility_ValidFormats(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	validFormats := []string{"html", "markdown", "md", "json", "text"}
	for _, format := range validFormats {
		err := ValidateTraceFormatCompatibility(trace, format)
		if err != nil {
			t.Fatalf("ValidateTraceFormatCompatibility should accept %s format, got: %v", format, err)
		}
	}
}

// Test ValidateTraceFormatCompatibility with invalid format
func TestValidateTraceFormatCompatibility_InvalidFormat(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Now()
	trace.EndTime = trace.StartTime.Add(time.Second)
	trace.AddState(ExecutionState{Operation: "test"})

	err := ValidateTraceFormatCompatibility(trace, "invalid-format")
	if err == nil {
		t.Fatalf("ValidateTraceFormatCompatibility should reject invalid format")
	}
}

// Test end-to-end export with validation
func TestExportExecutionTrace_WithValidation(t *testing.T) {
	trace := NewExecutionTrace("test-tx-hash", 10)
	trace.StartTime = time.Date(2026, time.January, 2, 15, 4, 5, 0, time.UTC)
	trace.EndTime = trace.StartTime.Add(5 * time.Minute)

	trace.AddState(ExecutionState{
		Operation:   "contract_call",
		EventType:   "contract_call",
		ContractID:  "C123",
		Function:    "transfer",
		Arguments:   []interface{}{"100", "XLM"},
		ReturnValue: "ok",
	})

	tmpDir := t.TempDir()

	// Test with various formats
	formats := []string{"html", "markdown", "json", "text"}
	for _, format := range formats {
		outputPath := filepath.Join(tmpDir, "trace."+format)
		if err := ExportExecutionTrace(trace, format, outputPath); err != nil {
			t.Fatalf("ExportExecutionTrace failed for %s format: %v", format, err)
		}

		// Verify file was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("export file not created for %s format", format)
		}
	}
}
