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

// ── ArtifactReport fields ─────────────────────────────────────────────────────

func TestArtifactReport_UnsupportedPlatform_HasEntry(t *testing.T) {
	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("test only applies to unsupported platforms")
	}
	r := newTestRegistrar(t)
	report := r.DiscoverArtifacts()
	if len(report.Artifacts) == 0 {
		t.Error("DiscoverArtifacts must produce at least one entry on unsupported platforms")
	}
	if report.AllPresent {
		t.Error("AllPresent must be false on an unsupported platform")
	}
}

func TestArtifactReport_Platform_IsPopulated(t *testing.T) {
	r := newTestRegistrar(t)
	report := r.DiscoverArtifacts()
	if report.Platform != runtime.GOOS {
		t.Errorf("Platform should be %q, got %q", runtime.GOOS, report.Platform)
	}
}

// ── DiscoverArtifacts — Linux no artefacts ────────────────────────────────────

func TestDiscoverArtifacts_Linux_NoArtefacts_AllMissing(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only test")
	}
	r := newTestRegistrar(t)
	report := r.DiscoverArtifacts()

	if report.AllPresent {
		t.Error("AllPresent must be false when no artefacts have been written")
	}
	for _, entry := range report.Artifacts {
		if entry.Present {
			t.Errorf("entry %q should not be present in a clean state", entry.Name)
		}
		if entry.Issue == "" {
			t.Errorf("missing entry %q must describe the issue", entry.Name)
		}
		if entry.Hint == "" {
			t.Errorf("missing entry %q must provide a Hint", entry.Name)
		}
	}
}

// ── DiscoverArtifacts — Linux fully written ───────────────────────────────────

func TestDiscoverArtifacts_Linux_FullyWritten_AllPresent(t *testing.T) {
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

	report := r.DiscoverArtifacts()

	if !report.AllPresent {
		for _, e := range report.Artifacts {
			if !e.Present {
				t.Logf("missing artefact: %s (%s): %s", e.Name, e.Path, e.Issue)
			}
		}
		t.Error("AllPresent must be true when all artefacts are written")
	}
	for _, entry := range report.Artifacts {
		if !entry.Valid {
			t.Errorf("entry %q should be Valid when content is correct; issue: %s", entry.Name, entry.Issue)
		}
	}
}

// ── DiscoverArtifacts — Linux stale wrapper ───────────────────────────────────

func TestDiscoverArtifacts_Linux_StaleWrapper_PresentButInvalid(t *testing.T) {
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
	stale := "#!/bin/sh\nexec /old/glassbox protocol-handler \"$1\"\n"
	if err := os.WriteFile(r.linuxWrapperPath(), []byte(stale), 0o755); err != nil {
		t.Fatal(err)
	}

	report := r.DiscoverArtifacts()

	// Wrapper is present but invalid (stale path).
	var wrapper *ArtifactEntry
	for i := range report.Artifacts {
		if report.Artifacts[i].Name == "wrapper script" {
			wrapper = &report.Artifacts[i]
		}
	}
	if wrapper == nil {
		t.Fatal("expected 'wrapper script' entry in ArtifactReport")
	}
	if !wrapper.Present {
		t.Error("wrapper script should be Present=true (file exists)")
	}
	if wrapper.Valid {
		t.Error("wrapper script should be Valid=false (stale path)")
	}
	if !strings.Contains(wrapper.Issue, "stale") {
		t.Errorf("issue should mention 'stale', got: %q", wrapper.Issue)
	}
	if wrapper.Hint == "" {
		t.Error("Hint must be set for an invalid wrapper")
	}
}

// ── DiscoverArtifacts — macOS ─────────────────────────────────────────────────

func TestDiscoverArtifacts_Darwin_NoArtefacts_AllMissing(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-only test")
	}
	r := newTestRegistrar(t)
	report := r.DiscoverArtifacts()

	if report.AllPresent {
		t.Error("AllPresent must be false when no artefacts exist")
	}
	for _, entry := range report.Artifacts {
		if entry.Present {
			t.Errorf("entry %q should not be present in a clean state", entry.Name)
		}
		if entry.Hint == "" {
			t.Errorf("missing entry %q must provide a Hint", entry.Name)
		}
	}
}

// ── ArtifactEntry — Issue and Hint invariants ─────────────────────────────────

func TestArtifactEntry_MissingAlwaysHasHint(t *testing.T) {
	r := newTestRegistrar(t)
	report := r.DiscoverArtifacts()

	for _, entry := range report.Artifacts {
		if !entry.Present && entry.Hint == "" {
			t.Errorf("ArtifactEntry %q: Present=false but Hint is empty", entry.Name)
		}
	}
}

func TestArtifactEntry_InvalidAlwaysHasIssue(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only: need file-based artefacts to produce Invalid entries")
	}
	r := newTestRegistrar(t)

	// Write a desktop file with incorrect content to produce an invalid entry.
	if err := os.MkdirAll(filepath.Dir(r.linuxDesktopPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(r.linuxDesktopPath(), []byte("[Desktop Entry]\nName=wrong\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := r.DiscoverArtifacts()
	for _, entry := range report.Artifacts {
		if entry.Present && !entry.Valid && entry.Issue == "" {
			t.Errorf("ArtifactEntry %q: Present=true, Valid=false but Issue is empty", entry.Name)
		}
	}
}

// ── allArtifactsPresent helper ────────────────────────────────────────────────

func TestAllArtifactsPresent_EmptySlice_ReturnsFalse(t *testing.T) {
	if allArtifactsPresent(nil) {
		t.Error("allArtifactsPresent(nil) should return false")
	}
	if allArtifactsPresent([]ArtifactEntry{}) {
		t.Error("allArtifactsPresent([]) should return false")
	}
}

func TestAllArtifactsPresent_OneAbsent_ReturnsFalse(t *testing.T) {
	entries := []ArtifactEntry{
		{Name: "a", Present: true},
		{Name: "b", Present: false},
	}
	if allArtifactsPresent(entries) {
		t.Error("allArtifactsPresent should return false when any entry is missing")
	}
}

func TestAllArtifactsPresent_AllPresent_ReturnsTrue(t *testing.T) {
	entries := []ArtifactEntry{
		{Name: "a", Present: true},
		{Name: "b", Present: true},
	}
	if !allArtifactsPresent(entries) {
		t.Error("allArtifactsPresent should return true when all entries are present")
	}
}

// ── validateDesktopFileContent ────────────────────────────────────────────────

func TestValidateDesktopFileContent_Valid(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only: linuxMimeType and linuxWrapperPath are Linux-specific")
	}
	r := newTestRegistrar(t)
	issues := validateDesktopFileContent(r.linuxDesktopEntry(), r)
	if len(issues) != 0 {
		t.Errorf("expected no issues for a freshly generated desktop entry, got: %v", issues)
	}
}

func TestValidateDesktopFileContent_MissingMimeType(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	r := newTestRegistrar(t)
	content := "[Desktop Entry]\nExec=" + r.linuxWrapperPath() + " %u\n"
	issues := validateDesktopFileContent(content, r)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "MimeType") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected MimeType issue, got: %v", issues)
	}
}

func TestValidateDesktopFileContent_MissingExec(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-only")
	}
	r := newTestRegistrar(t)
	content := "[Desktop Entry]\nMimeType=" + linuxMimeType + ";\n"
	issues := validateDesktopFileContent(content, r)
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "Exec") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected Exec issue, got: %v", issues)
	}
}
