// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// resetTracePathFlags clears every flag that the path-validation tests touch.
// It is called in t.Cleanup so each sub-test starts from a known-clean state.
func resetTracePathFlags() {
	traceExportPath = ""
	traceExportFormat = "html"
	traceFormatAlias = ""
	traceExportMarkdown = ""
	traceOutputJSON = ""
	traceExportSVG = ""
	traceAnnotationsFlag = ""
	traceGasModelPath = ""
	traceVerbosity = "normal"
	tracePrint = false
	traceDryRunFlag = false
	traceForceFlag = false
	traceShowTimingFlag = false
	traceComments = nil
	traceMetadata = nil
	traceFile = ""
	traceThemeFlag = ""
	traceNoColor = false
}

// ── --export path: null byte rejection ────────────────────────────────────────

func TestTracePreRunE_ExportNullByte_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceExportPath = "/path/to/trace\x00.html"
	traceExportFormat = "html"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error for null byte in --export path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

// ── --export path: existing directory rejected ────────────────────────────────

func TestTracePreRunE_ExportExistingDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	dir := t.TempDir()
	traceExportPath = dir // a real existing directory
	traceExportFormat = "html"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --export points to an existing directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

// ── --export path: valid new file in existing dir ─────────────────────────────

func TestTracePreRunE_ExportValidNewFile_Passes(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	dir := t.TempDir()
	traceExportPath = filepath.Join(dir, "output.html")
	traceExportFormat = "html"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err != nil {
		t.Errorf("unexpected error for valid --export path: %v", err)
	}
}

// ── --output-json path: null byte rejection ───────────────────────────────────

func TestTracePreRunE_OutputJSONNullByte_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceOutputJSON = "/path/to/trace\x00.json"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error for null byte in --output-json path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

// ── --output-json path: existing directory rejected ───────────────────────────

func TestTracePreRunE_OutputJSONExistingDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceOutputJSON = t.TempDir()

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --output-json points to an existing directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

// ── --export-svg path: null byte rejection ────────────────────────────────────

func TestTracePreRunE_ExportSVGNullByte_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceExportSVG = "/path/to/graph\x00.svg"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error for null byte in --export-svg path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

// ── --export-svg path: existing directory rejected ────────────────────────────

func TestTracePreRunE_ExportSVGExistingDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceExportSVG = t.TempDir()

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --export-svg points to an existing directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

// ── --export-markdown path: null byte rejection ───────────────────────────────

func TestTracePreRunE_ExportMarkdownNullByte_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceExportMarkdown = "/path/to/report\x00.md"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error for null byte in --export-markdown path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

// ── --annotations path: null byte rejection ───────────────────────────────────

func TestTracePreRunE_AnnotationsNullByte_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceAnnotationsFlag = "/path/to/annotations\x00.json"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error for null byte in --annotations path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

// ── --gas-model path: null byte rejection ─────────────────────────────────────

func TestTracePreRunE_GasModelNullByte_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceGasModelPath = "/path/to/gas\x00.json"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error for null byte in --gas-model path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

// ── --annotations: directory rejected ────────────────────────────────────────

func TestTracePreRunE_AnnotationsIsDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceAnnotationsFlag = t.TempDir() // a directory, not a file

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --annotations is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

// ── --gas-model: directory rejected ──────────────────────────────────────────

func TestTracePreRunE_GasModelIsDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	traceGasModelPath = t.TempDir()

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected error when --gas-model is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

// ── multiple path errors collected together ───────────────────────────────────

func TestTracePreRunE_MultiplePathErrors_AllReported(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	// --export points to an existing directory; --output-json has a null byte.
	traceExportPath = t.TempDir()
	traceExportFormat = "html"
	traceOutputJSON = "/trace\x00.json"

	err := traceCmd.PreRunE(traceCmd, []string{})
	if err == nil {
		t.Fatal("expected errors for both bad paths")
	}
	msg := err.Error()
	if !strings.Contains(msg, "directory") {
		t.Errorf("error should include directory failure, got: %q", msg)
	}
	if !strings.Contains(msg, "null bytes") {
		t.Errorf("error should include null-byte failure, got: %q", msg)
	}
}

// ── RunE: trace file input path safety ───────────────────────────────────────

func TestTraceRunE_NullByteInTraceFile_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)

	err := traceCmd.RunE(traceCmd, []string{"/path/to/trace\x00.json"})
	if err == nil {
		t.Fatal("expected error for null byte in trace file path")
	}
	if !strings.Contains(err.Error(), "null bytes") && !strings.Contains(err.Error(), "not found") {
		// ValidateInputPath may surface "null bytes" or produce a normalized
		// path that then fails with "not found" — both are acceptable.
		t.Errorf("error should mention null bytes or not-found, got: %q", err.Error())
	}
}

func TestTraceRunE_TraceFileIsDirectory_Rejected(t *testing.T) {
	t.Cleanup(resetTracePathFlags)
	dir := t.TempDir()

	err := traceCmd.RunE(traceCmd, []string{dir})
	if err == nil {
		t.Fatal("expected error when trace file arg is a directory")
	}
	if !strings.Contains(err.Error(), "directory") && !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention directory or not-found, got: %q", err.Error())
	}
}

// ── export command: --snapshot path safety ────────────────────────────────────

func TestExportCmd_SnapshotNullByte_Rejected(t *testing.T) {
	prev := exportSnapshotFlag
	exportSnapshotFlag = "/path/snap\x00.json"
	t.Cleanup(func() { exportSnapshotFlag = prev })

	setTestSession(t, map[string]string{"k": "v"})

	err := exportCmd.RunE(exportCmd, nil)
	if err == nil {
		t.Fatal("expected error for null byte in --snapshot path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

func TestExportCmd_SnapshotIsDirectory_Rejected(t *testing.T) {
	dir := t.TempDir()
	prev := exportSnapshotFlag
	exportSnapshotFlag = dir
	t.Cleanup(func() { exportSnapshotFlag = prev })

	setTestSession(t, map[string]string{"k": "v"})

	err := exportCmd.RunE(exportCmd, nil)
	if err == nil {
		t.Fatal("expected error when --snapshot path is an existing directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

// ── operation audit log: path safety ─────────────────────────────────────────

func TestWriteOperationAuditLog_NullByte_Rejected(t *testing.T) {
	prev := AuditLogPathFlag
	AuditLogPathFlag = "/path/audit\x00.json"
	t.Cleanup(func() { AuditLogPathFlag = prev })

	err := writeOperationAuditLog([]string{"glassbox", "debug"}, nil)
	if err == nil {
		t.Fatal("expected error for null byte in audit log path")
	}
	if !strings.Contains(err.Error(), "null bytes") && !strings.Contains(err.Error(), "audit-log") {
		t.Errorf("error should mention null bytes or audit-log, got: %q", err.Error())
	}
}

func TestWriteOperationAuditLog_DirectoryPath_Rejected(t *testing.T) {
	dir := t.TempDir()
	prev := AuditLogPathFlag
	AuditLogPathFlag = dir
	t.Cleanup(func() { AuditLogPathFlag = prev })

	err := writeOperationAuditLog([]string{"glassbox", "debug"}, nil)
	if err == nil {
		t.Fatal("expected error when audit log path is an existing directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

func TestWriteOperationAuditLog_ValidPath_Succeeds(t *testing.T) {
	dir := t.TempDir()
	prev := AuditLogPathFlag
	AuditLogPathFlag = filepath.Join(dir, "audit.json")
	t.Cleanup(func() { AuditLogPathFlag = prev })

	// writeOperationAuditLog will fail to sign (no key configured), but path
	// validation succeeds before that, so the error should NOT be path-related.
	err := writeOperationAuditLog([]string{"glassbox", "debug"}, nil)
	if err != nil {
		// Acceptable: signing may fail without a key — just not a path error.
		if strings.Contains(err.Error(), "null bytes") || strings.Contains(err.Error(), "directory") {
			t.Errorf("error should not be a path error for a valid path, got: %q", err.Error())
		}
	}

	// If signing succeeded, verify the file was written.
	if err == nil {
		if _, statErr := os.Stat(AuditLogPathFlag); os.IsNotExist(statErr) {
			t.Error("audit log file was not created despite success return")
		}
	}
}
