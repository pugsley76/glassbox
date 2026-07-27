// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"strings"
	"testing"
)

// ── ValidateTraceInputs: path traversal detection ──────────────────────────────
// These tests verify that the filepath.Clean-based traversal check correctly
// accepts safe paths that contain "dot-dot" substrings in the filename component
// (the old strings.Contains("..")  check would have falsely rejected these).

func TestValidateTraceInputs_TraversalWithDotDotInFilename_NoFalsePositive(t *testing.T) {
	// "my..trace.html" — contains ".." but is not a traversal attempt.
	// The old string-contains check would have rejected this.
	err := ValidateTraceInputs("normal", "html", "", "my..trace.html")
	if err != nil {
		t.Errorf("path %q should be accepted (not a traversal), got: %v", "my..trace.html", err)
	}
}

func TestValidateTraceInputs_TraversalWithDotDotInDirectory_Rejected(t *testing.T) {
	// "../etc/passwd" — genuine traversal.
	err := ValidateTraceInputs("normal", "html", "", "../etc/passwd")
	if err == nil {
		t.Fatal("expected error for directory traversal path")
	}
	msg := err.Error()
	if !strings.Contains(msg, "traversal") {
		t.Errorf("error should mention 'traversal', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

func TestValidateTraceInputs_TraversalDeepPath_Rejected(t *testing.T) {
	err := ValidateTraceInputs("normal", "html", "", "../../outputs/trace.html")
	if err == nil {
		t.Fatal("expected error for deep traversal path")
	}
	if !strings.Contains(err.Error(), "traversal") {
		t.Errorf("error should mention traversal, got: %q", err.Error())
	}
}

func TestValidateTraceInputs_RelativePathWithFilename_Accepted(t *testing.T) {
	// Normal relative paths that don't escape must pass.
	for _, p := range []string{
		"trace.html",
		"./traces/output.html",
		"artifacts/2026/debug.json",
	} {
		if err := ValidateTraceInputs("normal", "html", "", p); err != nil {
			t.Errorf("path %q should be accepted, got: %v", p, err)
		}
	}
}

func TestValidateTraceInputs_NullByteInPath_Rejected(t *testing.T) {
	err := ValidateTraceInputs("normal", "html", "", "/path/to/trace\x00.html")
	if err == nil {
		t.Fatal("expected error for null byte in path")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", err.Error())
	}
}

func TestValidateTraceInputs_DirectoryPath_Rejected(t *testing.T) {
	err := ValidateTraceInputs("normal", "html", "", "./traces/")
	if err == nil {
		t.Fatal("expected error for directory path")
	}
	if !strings.Contains(err.Error(), "directory path") {
		t.Errorf("error should mention 'directory path', got: %q", err.Error())
	}
}

func TestValidateTraceInputs_WindowsSeparatorDirectoryPath_Rejected(t *testing.T) {
	err := ValidateTraceInputs("normal", "html", "", `.\traces\`)
	if err == nil {
		t.Fatal("expected error for Windows-style directory path")
	}
	if !strings.Contains(err.Error(), "directory path") {
		t.Errorf("error should mention 'directory path', got: %q", err.Error())
	}
}

// ── Multiple failures collected together ──────────────────────────────────────

func TestValidateTraceInputs_TraversalAndBadFormat_BothReported(t *testing.T) {
	err := ValidateTraceInputs("normal", "csv", "", "../../trace.html")
	if err == nil {
		t.Fatal("expected errors for bad format and traversal path")
	}
	msg := err.Error()
	if !strings.Contains(msg, "traversal") {
		t.Errorf("error should include traversal failure, got: %q", msg)
	}
	if !strings.Contains(msg, "csv") {
		t.Errorf("error should include bad format, got: %q", msg)
	}
}
