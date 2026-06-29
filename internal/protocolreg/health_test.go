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

// ── HealthStatus constants ────────────────────────────────────────────────────

func TestHealthStatusConstants_Distinct(t *testing.T) {
	statuses := []HealthStatus{HealthReady, HealthDegraded, HealthNotReady}
	seen := make(map[HealthStatus]bool)
	for _, s := range statuses {
		if s == "" {
			t.Fatal("HealthStatus must not be empty string")
		}
		if seen[s] {
			t.Fatalf("duplicate HealthStatus: %q", s)
		}
		seen[s] = true
	}
}

// ── HealthCheck — empty executable path ──────────────────────────────────────

func TestHealthCheck_EmptyExecutablePath_NotReady(t *testing.T) {
	r := &Registrar{executablePath: "", homeDir: t.TempDir()}
	report := r.HealthCheck()

	if report.Ready {
		t.Error("HealthCheck must return Ready=false when executablePath is empty")
	}
	if report.Status != HealthNotReady {
		t.Errorf("expected HealthNotReady, got %q", report.Status)
	}
	if report.Hint == "" {
		t.Error("HealthCheck must provide a Hint when not ready")
	}
	combined := strings.Join(report.Failures, " ")
	if !strings.Contains(combined, "executable path") {
		t.Errorf("failure should mention 'executable path', got: %v", report.Failures)
	}
}

// ── HealthCheck — unreachable executable ─────────────────────────────────────

func TestHealthCheck_MissingExecutable_NotReady(t *testing.T) {
	r := &Registrar{
		executablePath: "/nonexistent/path/glassbox",
		homeDir:        t.TempDir(),
	}
	report := r.HealthCheck()

	if report.Ready {
		t.Error("expected Ready=false for a missing executable")
	}
	if report.Status != HealthNotReady {
		t.Errorf("expected HealthNotReady, got %q", report.Status)
	}
	combined := strings.Join(report.Failures, " ")
	if !strings.Contains(combined, "executable not reachable") {
		t.Errorf("failure should mention 'executable not reachable', got: %v", report.Failures)
	}
}

// ── HealthCheck — unsupported OS ─────────────────────────────────────────────

func TestHealthCheck_UnsupportedOS_ReturnsNotReady(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("test only applies to unsupported platforms")
	}
	r := newTestRegistrar(t)
	report := r.HealthCheck()
	if report.Status != HealthNotReady {
		t.Errorf("expected HealthNotReady on unsupported OS, got %q", report.Status)
	}
}

// ── HealthCheck — Linux ───────────────────────────────────────────────────────

func TestHealthCheck_Linux_NoArtefacts_NotReady(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	r := newTestRegistrar(t)
	report := r.HealthCheck()

	if report.Ready {
		t.Error("expected Ready=false when no artefacts exist")
	}
	if report.Hint == "" {
		t.Error("HealthCheck must provide a Hint when not ready")
	}
}

func TestHealthCheck_Linux_FullyReady(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	r := newTestRegistrar(t)

	// Write all artefacts correctly.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(r.linuxWrapperPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(r.unixHandlerScript()), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.HealthCheck()

	if !report.Ready {
		t.Errorf("expected Ready=true with all artefacts in place; failures: %v", report.Failures)
	}
	if report.Status != HealthReady {
		t.Errorf("expected HealthReady, got %q", report.Status)
	}
	if len(report.Checks) == 0 {
		t.Error("HealthCheck must record passing checks")
	}
}

func TestHealthCheck_Linux_StaleWrapper_Degraded(t *testing.T) {
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
	// Write a stale wrapper that points to a different binary.
	stale := "#!/bin/sh\nexec /old/path/glassbox protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(stale), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.HealthCheck()

	if report.Ready {
		t.Error("expected Ready=false when wrapper is stale")
	}
	if report.Status == HealthReady {
		t.Errorf("expected Degraded or NotReady, got %q", report.Status)
	}
	combined := strings.Join(report.Failures, " ")
	if !strings.Contains(combined, "stale") {
		t.Errorf("failure should mention 'stale', got: %v", report.Failures)
	}
	if report.Hint == "" {
		t.Error("Hint must be set when artefact is stale")
	}
}

func TestHealthCheck_Linux_MissingWrapper_Degraded(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	r := newTestRegistrar(t)

	// Write only the desktop file — wrapper is absent.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte(r.linuxDesktopEntry()), 0o644); err != nil {
		t.Fatal(err)
	}

	report := r.HealthCheck()

	if report.Ready {
		t.Error("expected Ready=false when wrapper script is missing")
	}
	combined := strings.Join(report.Failures, " ")
	if !strings.Contains(combined, "wrapper") {
		t.Errorf("failure should mention 'wrapper', got: %v", report.Failures)
	}
}

// ── HealthReport — Hint always set when not ready ────────────────────────────

func TestHealthCheck_HintNonEmpty_WhenNotReady(t *testing.T) {
	r := newTestRegistrar(t)
	report := r.HealthCheck()
	if report.Ready {
		t.Skip("handler already registered; skipping hint test")
	}
	if report.Hint == "" {
		t.Error("Hint must be non-empty when HealthCheck returns Ready=false")
	}
}

// ── HealthReport — Checks non-empty when ready ───────────────────────────────

func TestHealthCheck_Linux_ChecksPopulated(t *testing.T) {
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
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(r.unixHandlerScript()), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.HealthCheck()
	if len(report.Checks) == 0 {
		t.Error("Checks must be populated when HealthCheck passes")
	}
	for i, c := range report.Checks {
		if strings.TrimSpace(c) == "" {
			t.Errorf("check entry %d is empty", i)
		}
	}
}
