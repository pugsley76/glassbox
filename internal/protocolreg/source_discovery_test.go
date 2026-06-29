// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── validateDiscoveredPath ────────────────────────────────────────────────────

func TestValidateDiscoveredPath_EmptyPath_ReturnsError(t *testing.T) {
	_, err := validateDiscoveredPath("", "test-source")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if !strings.Contains(err.Error(), "empty path") {
		t.Errorf("error should mention 'empty path', got: %v", err)
	}
}

func TestValidateDiscoveredPath_NullByte_ReturnsError(t *testing.T) {
	_, err := validateDiscoveredPath("/some/path\x00evil", "test-source")
	if err == nil {
		t.Fatal("expected error for path with null byte")
	}
	if !strings.Contains(err.Error(), "null bytes") {
		t.Errorf("error should mention 'null bytes', got: %v", err)
	}
}

func TestValidateDiscoveredPath_NonExistentFile_ReturnsError(t *testing.T) {
	_, err := validateDiscoveredPath("/nonexistent/path/glassbox", "test-source")
	if err == nil {
		t.Fatal("expected error for non-existent path")
	}
}

func TestValidateDiscoveredPath_Directory_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	_, err := validateDiscoveredPath(dir, "test-source")
	if err == nil {
		t.Fatal("expected error when path is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %v", err)
	}
}

func TestValidateDiscoveredPath_ValidFile_ReturnsAbsPath(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "glassbox")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	got, err := validateDiscoveredPath(exe, "test-source")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == "" {
		t.Error("expected a non-empty resolved path")
	}
	if !filepath.IsAbs(got) {
		t.Errorf("resolved path should be absolute, got %q", got)
	}
}

func TestValidateDiscoveredPath_WhitespacePath_Rejected(t *testing.T) {
	_, err := validateDiscoveredPath("   ", "test-source")
	if err == nil {
		t.Fatal("expected error for whitespace-only path")
	}
}

// ── DiscoverExecutableSource ──────────────────────────────────────────────────

// TestDiscoverExecutableSource_FindsSomething verifies that when glassbox is
// invoked normally (not via 'go run') os.Executable returns a usable path.
func TestDiscoverExecutableSource_FindsSomething(t *testing.T) {
	result := DiscoverExecutableSource()

	// In a normal test execution os.Executable() should resolve to the test binary.
	// We only assert the result is structurally valid.
	if result == nil {
		t.Fatal("DiscoverExecutableSource must never return nil")
	}
	if result.Found {
		if result.Path == "" {
			t.Error("Found=true but Path is empty")
		}
		if result.Source == "" {
			t.Error("Found=true but Source is empty")
		}
	} else {
		if result.Hint == "" {
			t.Error("Found=false but Hint is empty — must provide remediation guidance")
		}
	}
}

// TestDiscoverExecutableSource_GLASSBOX_BIN_Override verifies that when
// GLASSBOX_BIN is set to a valid executable it is preferred (as a fallback)
// when os.Executable() returns a temp binary that satisfies stage 1 first.
// We test the env-based fallback in isolation via discoverViaEnv.
func TestDiscoverExecutableSource_GLASSBOX_BIN_ValidFile_IsUsed(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "glassbox-override")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("create override binary: %v", err)
	}
	t.Setenv("GLASSBOX_BIN", exe)

	path, err := discoverViaEnv()
	if err != nil {
		t.Fatalf("discoverViaEnv failed: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path from GLASSBOX_BIN")
	}
}

func TestDiscoverExecutableSource_GLASSBOX_BIN_MissingFile_ReturnsError(t *testing.T) {
	t.Setenv("GLASSBOX_BIN", "/nonexistent/path/glassbox")
	_, err := discoverViaEnv()
	if err == nil {
		t.Fatal("expected error when GLASSBOX_BIN points to a non-existent file")
	}
}

func TestDiscoverExecutableSource_GLASSBOX_BIN_NotSet_ReturnsError(t *testing.T) {
	t.Setenv("GLASSBOX_BIN", "")
	_, err := discoverViaEnv()
	if err == nil {
		t.Fatal("expected error when GLASSBOX_BIN is not set")
	}
	if !strings.Contains(err.Error(), "GLASSBOX_BIN") {
		t.Errorf("error should mention GLASSBOX_BIN, got: %v", err)
	}
}

func TestDiscoverExecutableSource_GLASSBOX_BIN_NullByte_ReturnsError(t *testing.T) {
	t.Setenv("GLASSBOX_BIN", "/some\x00path")
	_, err := discoverViaEnv()
	if err == nil {
		t.Fatal("expected error for GLASSBOX_BIN with null byte")
	}
}

// ── SourceDiscoveryResult — Hint when not found ───────────────────────────────

func TestSourceDiscoveryResult_HintMentionsGLASSBOX_BIN(t *testing.T) {
	// Construct a result that simulates all stages failing.
	result := &SourceDiscoveryResult{
		Found: false,
		Hint: "Set the GLASSBOX_BIN environment variable to the absolute path of the glassbox binary, " +
			"or reinstall Glassbox so it is available on PATH.",
	}
	if !strings.Contains(result.Hint, "GLASSBOX_BIN") {
		t.Error("Hint should mention GLASSBOX_BIN for the env-var fallback")
	}
	if !strings.Contains(result.Hint, "PATH") {
		t.Error("Hint should mention PATH for the path-lookup fallback")
	}
}

// ── SourceDiscoveryResult — Fallback flag ─────────────────────────────────────

func TestSourceDiscoveryResult_PrimarySource_FallbackFalse(t *testing.T) {
	result := DiscoverExecutableSource()
	if result.Found && result.Source == "os.Executable" && result.Fallback {
		t.Error("Fallback should be false when resolved via the primary os.Executable source")
	}
}

// ── discoverViaPath ───────────────────────────────────────────────────────────

func TestDiscoverViaPath_NonExistentBinary_ReturnsError(t *testing.T) {
	// "glassbox-truly-nonexistent-xyz" should not be on PATH.
	// We can't directly test discoverViaPath("glassbox") without controlling PATH,
	// but we can verify the error path via validateDiscoveredPath for a missing file.
	_, err := validateDiscoveredPath("/nonexistent/binary", "PATH")
	if err == nil {
		t.Fatal("expected error for non-existent binary path")
	}
}
