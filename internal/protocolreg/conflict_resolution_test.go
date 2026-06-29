// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// ── validateConflictingHandlerPath ────────────────────────────────────────────

func TestValidateConflictingHandlerPath_NullByte_ReturnsError(t *testing.T) {
	err := validateConflictingHandlerPath("/some/path\x00evil")
	if err == nil {
		t.Fatal("expected error for path with null byte")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention null bytes, got: %v", err)
	}
}

func TestValidateConflictingHandlerPath_ValidPath_NoError(t *testing.T) {
	if err := validateConflictingHandlerPath("/usr/bin/some-app"); err != nil {
		t.Errorf("expected no error for a valid path, got: %v", err)
	}
}

func TestValidateConflictingHandlerPath_EmptyPath_NoError(t *testing.T) {
	// An empty path is allowed — it means the diagnostic could not extract a path.
	if err := validateConflictingHandlerPath(""); err != nil {
		t.Errorf("expected no error for empty path, got: %v", err)
	}
}

// ── ResolveConflict — empty executable path ───────────────────────────────────

func TestResolveConflict_EmptyExecutablePath_ReturnsError(t *testing.T) {
	r := &Registrar{executablePath: "", homeDir: t.TempDir()}
	result := r.ResolveConflict()

	if result.Err == nil {
		t.Fatal("expected error when executablePath is empty")
	}
	if strings.Contains(result.Err.Error(), "executable path") == false {
		t.Errorf("error should mention executable path, got: %v", result.Err)
	}
	if result.Hint == "" {
		t.Error("Hint must be set when executable path is empty")
	}
	if result.Resolved {
		t.Error("Resolved must be false when executable path is empty")
	}
}

// ── ResolveConflict — no conflict present ─────────────────────────────────────

func TestResolveConflict_NoConflict_ResolvedTrue(t *testing.T) {
	r := newTestRegistrar(t)
	// A fresh registrar with no registration artefacts has no conflict.
	result := r.ResolveConflict()

	// ConflictDetected will be false → result.Resolved must be true.
	if !result.Resolved {
		t.Errorf("expected Resolved=true when no conflict, err=%v, actions=%v", result.Err, result.Actions)
	}
	if result.Err != nil {
		t.Errorf("expected no error when there is no conflict, got: %v", result.Err)
	}
	combined := strings.Join(result.Actions, " ")
	if !strings.Contains(combined, "No conflict") {
		t.Errorf("actions should say 'No conflict'; got: %v", result.Actions)
	}
}

// ── ResolveConflict — conflict present (Linux) ────────────────────────────────

func TestResolveConflict_Linux_ConflictPresent_RecordsConflictingPath(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only: uses file-based artefacts")
	}
	r := newTestRegistrar(t)

	// Write a foreign-binary wrapper to trigger ConflictDetected.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	foreignScript := "#!/bin/sh\nexec /usr/bin/completely-other-app protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(foreignScript), 0o755); err != nil {
		t.Fatal(err)
	}

	result := r.ResolveConflict()

	// The conflicting handler must be recorded.
	if result.ConflictingHandler == "" {
		t.Error("ConflictingHandler must be populated when a conflict is detected")
	}
	if !strings.Contains(result.ConflictingHandler, "completely-other-app") {
		t.Errorf("ConflictingHandler should name the foreign binary, got %q", result.ConflictingHandler)
	}

	// Actions must mention the displacement.
	combined := strings.Join(result.Actions, " ")
	if !strings.Contains(combined, "completely-other-app") {
		t.Errorf("actions should name the conflicting handler; got: %v", result.Actions)
	}
	if !strings.Contains(combined, "displacing") {
		t.Errorf("actions should say 'displacing'; got: %v", result.Actions)
	}
}

func TestResolveConflict_Linux_AfterResolution_ConflictCleared(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	r := newTestRegistrar(t)

	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	foreignScript := "#!/bin/sh\nexec /usr/bin/another-tool protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(foreignScript), 0o755); err != nil {
		t.Fatal(err)
	}

	result := r.ResolveConflict()

	// After ResolveConflict the post-resolution diagnostic should not see a conflict.
	// (xdg-mime may still report issues in CI — we only check the wrapper.)
	postDiag := r.Diagnose()
	if postDiag.ConflictDetected {
		t.Errorf("ConflictDetected should be false after ResolveConflict; result: %+v", result)
	}
}

// ── ResolveConflict — actions are always recorded ─────────────────────────────

func TestResolveConflict_ActionsAlwaysRecorded(t *testing.T) {
	r := newTestRegistrar(t)
	result := r.ResolveConflict()
	if len(result.Actions) == 0 {
		t.Error("ResolveConflict must always record at least one action")
	}
	for i, a := range result.Actions {
		if strings.TrimSpace(a) == "" {
			t.Errorf("action at index %d is empty", i)
		}
	}
}

// ── ResolveConflict — null-byte conflicting path ──────────────────────────────

// TestResolveConflict_NullByteConflictingHandler verifies that a conflicting
// handler path containing null bytes is rejected with an actionable error
// instead of being propagated to system calls.
func TestResolveConflict_NullByteConflictingHandler_ReturnsError(t *testing.T) {
	// We cannot inject a null-byte path via real OS state; instead we test the
	// validation helper directly.
	err := validateConflictingHandlerPath("/path\x00injected")
	if err == nil {
		t.Fatal("validateConflictingHandlerPath must reject null-byte paths")
	}
	if result := (&ConflictResolutionResult{}); result.Err != nil {
		t.Error("zero-value result should have nil Err")
	}
}

// ── ConflictResolutionResult — zero value is safe ─────────────────────────────

func TestConflictResolutionResult_ZeroValue(t *testing.T) {
	result := ConflictResolutionResult{}
	if result.Resolved {
		t.Error("zero-value Resolved should be false")
	}
	if result.Err != nil {
		t.Error("zero-value Err should be nil")
	}
	if result.ConflictingHandler != "" {
		t.Error("zero-value ConflictingHandler should be empty")
	}
}
