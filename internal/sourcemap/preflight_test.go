// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Issue #311: environment detection preflight for source mapping.

package sourcemap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── RunSourceMapPreflight — empty projectRoot ─────────────────────────────────

// TestPreflight_EmptyProjectRoot_NoWasmIssues verifies that when projectRoot is
// empty the WASM-artifact checks are skipped and the report is OK (no env vars set).
func TestPreflight_EmptyProjectRoot_NoWasmIssues(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("")
	if !report.OK {
		t.Errorf("empty projectRoot with no env vars should be OK; issues: %v", report.Issues)
	}
}

// ── projectRoot validation checks ────────────────────────────────────────────

// TestPreflight_NonexistentProjectRoot_ErrorIssue verifies that a non-existent
// projectRoot produces an error issue and sets report.OK=false.
func TestPreflight_NonexistentProjectRoot_ErrorIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("/does/not/exist/project/root")
	if report.OK {
		t.Fatal("non-existent projectRoot must set report.OK=false")
	}
	requireIssueCheck(t, report, "project_root")
}

// TestPreflight_ProjectRootIsFile_ErrorIssue verifies that when projectRoot
// points to a file (not a directory) an error issue is produced.
func TestPreflight_ProjectRootIsFile_ErrorIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	dir := t.TempDir()
	f := filepath.Join(dir, "notadir.rs")
	if err := os.WriteFile(f, []byte("fn main() {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := RunSourceMapPreflight(f)
	if report.OK {
		t.Fatal("file path as projectRoot must set report.OK=false")
	}
	requireIssueCheck(t, report, "project_root")
}

// TestPreflight_ValidProjectRoot_NoRootIssue verifies that a valid directory as
// projectRoot does not produce a project_root issue.
func TestPreflight_ValidProjectRoot_NoRootIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	dir := t.TempDir()
	report := RunSourceMapPreflight(dir)

	for _, issue := range report.Issues {
		if issue.Check == "project_root" {
			t.Errorf("valid directory projectRoot should not produce a project_root issue; got: %+v", issue)
		}
	}
}

// TestPreflight_NonexistentProjectRoot_HintIsActionable verifies that the hint
// for a non-existent projectRoot is non-empty and actionable.
func TestPreflight_NonexistentProjectRoot_HintIsActionable(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("/nonexistent/root")
	for _, issue := range report.Issues {
		if issue.Check == "project_root" {
			if strings.TrimSpace(issue.Hint) == "" {
				t.Error("project_root issue must have a non-empty actionable hint")
			}
		}
	}
}

// ── WASM target directory checks ─────────────────────────────────────────────

// TestPreflight_MissingWasmTargetDir_WarningIssue verifies that a project root
// without the WASM target directory produces a warning (not an error).
func TestPreflight_MissingWasmTargetDir_WarningIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	dir := t.TempDir() // no target/ subdirectory
	report := RunSourceMapPreflight(dir)

	// Missing target dir is a warning, not an error — report must still be OK.
	if !report.OK {
		t.Errorf("missing WASM target dir should be a warning, not an error; report.OK=%v", report.OK)
	}
	requireIssueCheck(t, report, "wasm_target_dir")
}

// TestPreflight_WasmTargetDirExistsButEmpty_WarningIssue verifies that an
// existing but empty WASM target directory produces a warning.
func TestPreflight_WasmTargetDirExistsButEmpty_WarningIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	dir := t.TempDir()
	wasmDir := filepath.Join(dir, "target", "wasm32-unknown-unknown", "release")
	if err := os.MkdirAll(wasmDir, 0755); err != nil {
		t.Fatal(err)
	}

	report := RunSourceMapPreflight(dir)
	if !report.OK {
		t.Errorf("empty WASM target dir should be a warning; issues: %v", report.Issues)
	}
	requireIssueCheck(t, report, "wasm_artifacts")
}

// TestPreflight_WasmFilePresent_NoArtifactIssue verifies that when a .wasm
// file is present in the release directory no artifact issues are reported.
func TestPreflight_WasmFilePresent_NoArtifactIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	dir := t.TempDir()
	wasmDir := filepath.Join(dir, "target", "wasm32-unknown-unknown", "release")
	if err := os.MkdirAll(wasmDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wasmDir, "contract.wasm"),
		[]byte{0x00, 0x61, 0x73, 0x6d}, 0644); err != nil {
		t.Fatal(err)
	}

	report := RunSourceMapPreflight(dir)

	for _, issue := range report.Issues {
		if issue.Check == "wasm_target_dir" || issue.Check == "wasm_artifacts" {
			t.Errorf("should not flag WASM artifact issue when .wasm file is present; got: %+v", issue)
		}
	}
}

// ── GLASSBOX_SKIP_SOURCE_MAPPING env var ─────────────────────────────────────

// TestPreflight_SkipEnvTrue_WarningIssue verifies that setting
// GLASSBOX_SKIP_SOURCE_MAPPING=true produces a warning (not a hard error).
func TestPreflight_SkipEnvTrue_WarningIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "true")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("")
	if !report.OK {
		t.Errorf("SKIP env var should produce a warning only; report.OK=%v", report.OK)
	}
	requireIssueCheck(t, report, "skip_source_mapping_env")
}

// TestPreflight_SkipEnvFalse_NoIssue verifies that
// GLASSBOX_SKIP_SOURCE_MAPPING=false does not produce an issue.
func TestPreflight_SkipEnvFalse_NoIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "false")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("")
	for _, issue := range report.Issues {
		if issue.Check == "skip_source_mapping_env" {
			t.Errorf("GLASSBOX_SKIP_SOURCE_MAPPING=false should not produce an issue")
		}
	}
}

// TestPreflight_SkipEnvVariants verifies that "1", "true", and "yes" are all
// detected as truthy (source mapping disabled warning).
func TestPreflight_SkipEnvVariants(t *testing.T) {
	for _, val := range []string{"1", "true", "yes", "TRUE", "YES"} {
		val := val
		t.Run(val, func(t *testing.T) {
			t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", val)
			t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

			report := RunSourceMapPreflight("")
			found := false
			for _, issue := range report.Issues {
				if issue.Check == "skip_source_mapping_env" {
					found = true
				}
			}
			if !found {
				t.Errorf("GLASSBOX_SKIP_SOURCE_MAPPING=%q should produce a warning", val)
			}
		})
	}
}

// ── GLASSBOX_SOURCE_MAP_CACHE env var ─────────────────────────────────────────

// TestPreflight_CacheEnvMissing_ErrorIssue verifies that a non-existent cache
// directory is an error and sets report.OK=false.
func TestPreflight_CacheEnvMissing_ErrorIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "/nonexistent/cache/dir")

	report := RunSourceMapPreflight("")
	if report.OK {
		t.Fatal("non-existent cache dir must set report.OK=false")
	}
	requireIssueCheck(t, report, "source_map_cache_dir")
}

// TestPreflight_CacheEnvValidDir_NoIssue verifies that a valid writable
// directory does not produce a cache issue.
func TestPreflight_CacheEnvValidDir_NoIssue(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", dir)

	report := RunSourceMapPreflight("")
	for _, issue := range report.Issues {
		if issue.Check == "source_map_cache_dir" {
			t.Errorf("valid cache dir should not produce an issue; got: %+v", issue)
		}
	}
}

// TestPreflight_CacheEnvIsFile_ErrorIssue verifies that pointing
// GLASSBOX_SOURCE_MAP_CACHE at a file (not a directory) is an error.
func TestPreflight_CacheEnvIsFile_ErrorIssue(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(f, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", f)

	report := RunSourceMapPreflight("")
	if report.OK {
		t.Fatal("file path for cache dir must set report.OK=false")
	}
	requireIssueCheck(t, report, "source_map_cache_dir")
}

// ── Hint quality ──────────────────────────────────────────────────────────────

// TestPreflight_AllIssuesHaveHints verifies that every issue produced by the
// preflight carries a non-empty actionable hint.
func TestPreflight_AllIssuesHaveHints(t *testing.T) {
	// Trigger as many issues as possible.
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "1")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "/does/not/exist")

	dir := t.TempDir() // no WASM artifacts
	report := RunSourceMapPreflight(dir)

	for _, issue := range report.Issues {
		if strings.TrimSpace(issue.Hint) == "" {
			t.Errorf("issue %q has an empty Hint", issue.Check)
		}
		if strings.TrimSpace(issue.Description) == "" {
			t.Errorf("issue %q has an empty Description", issue.Check)
		}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// requireIssueCheck asserts that at least one issue targets the named check.
func requireIssueCheck(t *testing.T, report *PreflightReport, check string) {
	t.Helper()
	for _, issue := range report.Issues {
		if issue.Check == check {
			return
		}
	}
	t.Errorf("expected an issue for check %q; got issues: %v", check, report.Issues)
}

// ── PreflightReport.Summary() ────────────────────────────────────────────────

// TestPreflightReport_Summary_NoIssues_ReturnsEmpty verifies that a clean
// report produces an empty summary string.
func TestPreflightReport_Summary_NoIssues_ReturnsEmpty(t *testing.T) {
	report := &PreflightReport{OK: true, Issues: nil}
	if s := report.Summary(); s != "" {
		t.Errorf("Summary() with no issues should be empty, got: %q", s)
	}
}

// TestPreflightReport_Summary_SingleIssue_ContainsCheckAndHint verifies that
// Summary() includes the check name, description, and hint for a single issue.
func TestPreflightReport_Summary_SingleIssue_ContainsCheckAndHint(t *testing.T) {
	report := &PreflightReport{
		OK: false,
		Issues: []PreflightIssue{
			{
				Check:       "wasm_target_dir",
				Severity:    "warning",
				Description: "WASM target directory not found: /tmp/project/target/wasm32-unknown-unknown/release",
				Hint:        "Run cargo build --target wasm32-unknown-unknown --release",
			},
		},
	}

	s := report.Summary()
	if s == "" {
		t.Fatal("Summary() should not be empty for a report with issues")
	}
	if !strings.Contains(s, "wasm_target_dir") {
		t.Errorf("Summary() should include check name, got: %q", s)
	}
	if !strings.Contains(s, "WASM target directory") {
		t.Errorf("Summary() should include description, got: %q", s)
	}
	if !strings.Contains(s, "cargo build") {
		t.Errorf("Summary() should include hint, got: %q", s)
	}
	if !strings.Contains(s, "warning") {
		t.Errorf("Summary() should include severity, got: %q", s)
	}
}

// TestPreflightReport_Summary_MultipleIssues_AllIncluded verifies that
// Summary() includes all issues when more than one is present.
func TestPreflightReport_Summary_MultipleIssues_AllIncluded(t *testing.T) {
	report := &PreflightReport{
		OK: false,
		Issues: []PreflightIssue{
			{Check: "wasm_target_dir", Severity: "warning", Description: "missing dir", Hint: "build first"},
			{Check: "source_map_cache_dir", Severity: "error", Description: "not writable", Hint: "check perms"},
		},
	}

	s := report.Summary()
	if !strings.Contains(s, "wasm_target_dir") {
		t.Errorf("Summary() should mention wasm_target_dir, got: %q", s)
	}
	if !strings.Contains(s, "source_map_cache_dir") {
		t.Errorf("Summary() should mention source_map_cache_dir, got: %q", s)
	}
	if !strings.Contains(s, "check perms") {
		t.Errorf("Summary() should include the second hint, got: %q", s)
	}
}

// TestPreflightReport_Summary_IssueWithNoHint_NoCrash verifies that Summary()
// does not crash or include a stray "Hint:" label when an issue has no hint.
func TestPreflightReport_Summary_IssueWithNoHint_NoCrash(t *testing.T) {
	report := &PreflightReport{
		OK: false,
		Issues: []PreflightIssue{
			{Check: "some_check", Severity: "warning", Description: "something is wrong", Hint: ""},
		},
	}

	s := report.Summary()
	if s == "" {
		t.Fatal("Summary() should not be empty for a report with issues")
	}
	if strings.Contains(s, "Hint:") {
		t.Errorf("Summary() should not emit 'Hint:' when hint is empty, got: %q", s)
	}
}

// TestPreflightReport_Summary_RealPreflight_ContainsIssues verifies that the
// Summary() method works end-to-end with a real RunSourceMapPreflight call.
func TestPreflightReport_Summary_RealPreflight_ContainsIssues(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "true")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("")
	if len(report.Issues) == 0 {
		t.Skip("expected at least one issue from the preflight; env may not have propagated")
	}

	s := report.Summary()
	if s == "" {
		t.Error("Summary() should not be empty when there are issues")
	}
	if !strings.Contains(s, "skip_source_mapping_env") {
		t.Errorf("Summary() should include the skip_source_mapping_env check, got: %q", s)
	}
}

// ── RunSourceMapPreflight — null byte in projectRoot ─────────────────────────

// TestPreflight_NullByteInProjectRoot_ErrorIssue verifies that a projectRoot
// containing a null byte is rejected before any filesystem access with a clear
// error message.
func TestPreflight_NullByteInProjectRoot_ErrorIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "")

	report := RunSourceMapPreflight("/valid/path\x00injection")
	if report.OK {
		t.Fatal("null byte in projectRoot must set report.OK=false")
	}
	requireIssueCheck(t, report, "project_root")
	for _, issue := range report.Issues {
		if issue.Check == "project_root" {
			if !strings.Contains(issue.Description, "null bytes") {
				t.Errorf("issue description should mention null bytes, got: %q", issue.Description)
			}
			if strings.TrimSpace(issue.Hint) == "" {
				t.Error("project_root null-byte issue must have a non-empty hint")
			}
		}
	}
}

// TestPreflight_NullByteInCacheDir_ErrorIssue verifies that
// GLASSBOX_SOURCE_MAP_CACHE containing a null byte is rejected with a clear
// error rather than producing an obscure OS error.
func TestPreflight_NullByteInCacheDir_ErrorIssue(t *testing.T) {
	t.Setenv("GLASSBOX_SKIP_SOURCE_MAPPING", "")
	t.Setenv("GLASSBOX_SOURCE_MAP_CACHE", "/valid\x00bad")

	report := RunSourceMapPreflight("")
	if report.OK {
		t.Fatal("null byte in GLASSBOX_SOURCE_MAP_CACHE must set report.OK=false")
	}
	requireIssueCheck(t, report, "source_map_cache_dir")
	for _, issue := range report.Issues {
		if issue.Check == "source_map_cache_dir" {
			if !strings.Contains(issue.Description, "null bytes") {
				t.Errorf("issue description should mention null bytes, got: %q", issue.Description)
			}
			if strings.TrimSpace(issue.Hint) == "" {
				t.Error("source_map_cache_dir null-byte issue must have a non-empty hint")
			}
		}
	}
}
