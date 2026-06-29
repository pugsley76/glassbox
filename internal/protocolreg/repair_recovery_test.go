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

// ── Repair conflict recovery ──────────────────────────────────────────────────

// TestRepair_ConflictDetected_IncludesConflictAction verifies that when a
// conflicting registration exists, Repair records which handler is being
// displaced before attempting re-registration.
func TestRepair_ConflictDetected_IncludesConflictAction(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only: uses file-based registration artefacts")
	}

	r := newTestRegistrar(t)

	// Seed a foreign-binary wrapper to trigger ConflictDetected.
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

	result := r.Repair()

	// The actions list must mention the conflicting handler path.
	combined := strings.Join(result.Actions, " ")
	if !strings.Contains(combined, "/usr/bin/completely-other-app") {
		t.Errorf("Repair actions should name the conflicting handler; got: %v", result.Actions)
	}
	if !strings.Contains(combined, "Conflicting") {
		t.Errorf("Repair actions should describe the conflict; got: %v", result.Actions)
	}
}

// TestRepair_RecordsPreRepairStatus verifies that Repair always records at
// least the attempt action, regardless of the pre-repair diagnostic status.
func TestRepair_RecordsPreRepairStatus(t *testing.T) {
	r := newTestRegistrar(t)

	// No registration artefacts → StatusNotRegistered.
	result := r.Repair()

	if len(result.Actions) == 0 {
		t.Fatal("Repair must always record at least one action")
	}
	// There must be an action that describes the registration attempt.
	combined := strings.Join(result.Actions, " ")
	if !strings.Contains(combined, Scheme+"://") {
		t.Errorf("Repair actions should mention the protocol scheme; got: %v", result.Actions)
	}
}

// TestRepair_StatusError_ReturnsError verifies that Repair surfaces an error
// (not a panic or silent failure) when the pre-repair diagnostic is StatusError.
func TestRepair_StatusError_ReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		t.Skip("StatusError only occurs on unsupported platforms")
	}

	r := newTestRegistrar(t)
	result := r.Repair()

	if result.Err == nil {
		t.Fatal("Repair on an unsupported OS must return an error")
	}
}

// TestRepair_Idempotent_NoConflict verifies that calling Repair twice when
// there is no existing registration does not produce contradictory results.
func TestRepair_Idempotent_NoConflict(t *testing.T) {
	r := newTestRegistrar(t)

	first := r.Repair()
	second := r.Repair()

	// Neither call should panic; both must record actions.
	if len(first.Actions) == 0 || len(second.Actions) == 0 {
		t.Error("both Repair calls must record at least one action")
	}
}

// TestRepair_Linux_ConflictOverwrite_NoConflictFlag verifies that after
// Repair on a conflicting registration the ConflictDetected flag in a
// subsequent diagnostic is cleared (the conflict has been resolved).
func TestRepair_Linux_ConflictOverwrite_NoConflictFlag(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only: uses file-based registration artefacts")
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
	// Install a foreign script to simulate a conflict.
	foreignScript := "#!/bin/sh\nexec /usr/bin/another-app protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(foreignScript), 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-repair: conflict must be visible.
	preDiag := r.Diagnose()
	if !preDiag.ConflictDetected {
		t.Fatal("expected ConflictDetected=true before repair")
	}

	r.Repair()

	// Post-repair: the wrapper now references the current binary — no conflict.
	postDiag := r.Diagnose()
	if postDiag.ConflictDetected {
		t.Error("expected ConflictDetected=false after repair overwrote the conflicting handler")
	}
}

// ── Repair — empty executablePath guard ──────────────────────────────────────

// TestRepair_EmptyExecutablePath_ReturnsActionableError verifies that calling
// Repair on a Registrar with an empty executablePath fails immediately with a
// clear error instead of silently writing a broken registration.
func TestRepair_EmptyExecutablePath_ReturnsActionableError(t *testing.T) {
	r := &Registrar{
		executablePath: "", // intentionally empty
		homeDir:        t.TempDir(),
	}

	result := r.Repair()

	if result.Err == nil {
		t.Fatal("expected Repair to return an error for an empty executablePath")
	}
	msg := result.Err.Error()
	if !strings.Contains(msg, "executable path") {
		t.Errorf("error should mention 'executable path', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix: hint, got: %q", msg)
	}
	// Must not have recorded any registration actions before detecting the bad state.
	for _, action := range result.Actions {
		if strings.Contains(strings.ToLower(action), "register") {
			t.Errorf("Repair should not attempt registration with empty executablePath; action: %q", action)
		}
	}
}

// TestRepair_EmptyExecutablePath_RepairFalse verifies Repaired is false.
func TestRepair_EmptyExecutablePath_RepairFalse(t *testing.T) {
	r := &Registrar{executablePath: "", homeDir: t.TempDir()}
	result := r.Repair()
	if result.Repaired {
		t.Error("Repaired should be false when executable path is empty")
	}
}
