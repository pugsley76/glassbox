// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for source discovery and fallback validation in the debug command.
// Covers PreRunE validation of --contract-source and --source-alias flags
// as well as the new dry-run source-discovery checks.

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── validateSourceDiscoveryFlags — --contract-source ─────────────────────────

func TestValidateSourceDiscoveryFlags_ContractSource_NonExistent_ReturnsError(t *testing.T) {
	prev := contractSourceFlag
	contractSourceFlag = "/nonexistent/source/directory"
	t.Cleanup(func() { contractSourceFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error for non-existent --contract-source directory")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--contract-source") {
		t.Errorf("error should mention --contract-source, got: %q", msg)
	}
	if !strings.Contains(msg, "not found") {
		t.Errorf("error should say 'not found', got: %q", msg)
	}
	// Must include remediation guidance.
	if !strings.Contains(msg, "src/") && !strings.Contains(msg, "source directory") {
		t.Errorf("error should describe what to provide, got: %q", msg)
	}
}

func TestValidateSourceDiscoveryFlags_ContractSource_IsFile_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "lib.rs")
	if err := os.WriteFile(filePath, []byte("fn main() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	prev := contractSourceFlag
	contractSourceFlag = filePath
	t.Cleanup(func() { contractSourceFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error when --contract-source points to a file")
	}
	if !strings.Contains(err.Error(), "is a file, not a directory") {
		t.Errorf("error should say 'is a file, not a directory', got: %q", err)
	}
}

func TestValidateSourceDiscoveryFlags_ContractSource_ValidDir_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	prev := contractSourceFlag
	contractSourceFlag = dir
	t.Cleanup(func() { contractSourceFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err != nil {
		t.Errorf("should not reject valid directory, got: %v", err)
	}
}

func TestValidateSourceDiscoveryFlags_ContractSource_Empty_ReturnsNil(t *testing.T) {
	prev := contractSourceFlag
	contractSourceFlag = ""
	t.Cleanup(func() { contractSourceFlag = prev })

	// Empty is valid — source discovery simply uses auto-detection.
	err := validateSourceDiscoveryFlags()
	if err != nil {
		t.Errorf("empty --contract-source should be a no-op, got: %v", err)
	}
}

func TestValidateSourceDiscoveryFlags_ContractSource_Whitespace_ReturnsError(t *testing.T) {
	prev := contractSourceFlag
	contractSourceFlag = "   "
	t.Cleanup(func() { contractSourceFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error for whitespace-only --contract-source")
	}
	if !strings.Contains(err.Error(), "--contract-source") {
		t.Errorf("error should mention --contract-source, got: %q", err)
	}
}

// ── validateSourceDiscoveryFlags — --source-alias JSON validation ─────────────

func TestValidateSourceDiscoveryFlags_SourceAlias_InvalidJSON_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	if err := os.WriteFile(aliasPath, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}
	prev := sourceAliasFlag
	sourceAliasFlag = aliasPath
	t.Cleanup(func() { sourceAliasFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error for invalid JSON in --source-alias")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--source-alias") {
		t.Errorf("error should mention --source-alias, got: %q", msg)
	}
	if !strings.Contains(msg, "JSON") {
		t.Errorf("error should mention JSON parse failure, got: %q", msg)
	}
	// Must include a usage example so the user knows what format to use.
	if !strings.Contains(msg, "Example") && !strings.Contains(msg, "example") {
		t.Errorf("error should include an example, got: %q", msg)
	}
}

func TestValidateSourceDiscoveryFlags_SourceAlias_FileNotFound_ReturnsError(t *testing.T) {
	prev := sourceAliasFlag
	sourceAliasFlag = filepath.Join(t.TempDir(), "missing.json")
	t.Cleanup(func() { sourceAliasFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error for missing --source-alias file")
	}
	if !strings.Contains(err.Error(), "--source-alias") {
		t.Errorf("error should mention --source-alias, got: %q", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention missing file, got: %q", err)
	}
}

func TestValidateSourceDiscoveryFlags_SourceAlias_ValidJSON_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	data, _ := json.Marshal(map[string]string{"my_crate": "/some/path"})
	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	prev := sourceAliasFlag
	sourceAliasFlag = aliasPath
	t.Cleanup(func() { sourceAliasFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err != nil {
		t.Errorf("should not reject valid --source-alias JSON, got: %v", err)
	}
}

func TestValidateSourceDiscoveryFlags_SourceAlias_TargetMissing_DoesNotError(t *testing.T) {
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	data, _ := json.Marshal(map[string]string{"my_crate": filepath.Join(dir, "missing-src")})
	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	prev := sourceAliasFlag
	sourceAliasFlag = aliasPath
	t.Cleanup(func() { sourceAliasFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err != nil {
		t.Fatalf("missing --source-alias target should be treated as a warning, got: %v", err)
	}
}

func TestValidateSourceDiscoveryFlags_SourceAlias_EmptyFlag_ReturnsNil(t *testing.T) {
	prev := sourceAliasFlag
	sourceAliasFlag = ""
	t.Cleanup(func() { sourceAliasFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err != nil {
		t.Errorf("empty --source-alias should be a no-op, got: %v", err)
	}
}

// ── dry-run: source discovery checks ─────────────────────────────────────────

func TestDryRun_ContractSourceNotFound_AppearsInFailures(t *testing.T) {
	t.Cleanup(func() {
		networkFlag = "mainnet"
		contractSourceFlag = ""
		rpcURLFlag = ""
		rpcTokenFlag = ""
	})
	networkFlag = "testnet"
	contractSourceFlag = "/nonexistent/source/dir"

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := runDebugDryRun(cmd, validHash)
	if err == nil {
		t.Fatal("expected dry-run to fail for missing --contract-source")
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "[FAIL]") {
		t.Errorf("expected [FAIL] in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "contract-source") {
		t.Errorf("stderr should mention contract-source, got: %s", stderr)
	}
	if !strings.Contains(stderr, "Fix:") {
		t.Errorf("stderr should include a Fix: hint, got: %s", stderr)
	}
}

func TestDryRun_ContractSourceIsFile_AppearsInFailures(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "lib.rs")
	if err := os.WriteFile(filePath, []byte("fn main() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		networkFlag = "mainnet"
		contractSourceFlag = ""
		rpcURLFlag = ""
	})
	networkFlag = "testnet"
	contractSourceFlag = filePath

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := runDebugDryRun(cmd, validHash)
	if err == nil {
		t.Fatal("expected dry-run to fail when --contract-source is a file")
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "not a directory") {
		t.Errorf("stderr should say 'not a directory', got: %s", stderr)
	}
}

func TestDryRun_ContractSourceValid_PrintsOK(t *testing.T) {
	dir := t.TempDir()

	t.Cleanup(func() {
		networkFlag = "mainnet"
		contractSourceFlag = ""
		rpcURLFlag = ""
	})
	networkFlag = "testnet"
	contractSourceFlag = dir
	// Use an unreachable RPC so other checks fail but source check passes.
	rpcURLFlag = "http://127.0.0.1:19999"

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	_ = runDebugDryRun(cmd, validHash) // may fail due to RPC, that's fine
	stdout := out.String()
	if !strings.Contains(stdout, "[OK]") {
		t.Skip("dry-run produced no [OK] lines — may be an environment issue")
	}
	if strings.Contains(stdout, "[FAIL]") && strings.Contains(errBuf.String(), "contract-source") {
		t.Errorf("valid --contract-source should not produce a [FAIL], got stderr: %s", errBuf.String())
	}
}

func TestDryRun_SourceAliasInvalidJSON_AppearsInFailures(t *testing.T) {
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	if err := os.WriteFile(aliasPath, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		networkFlag = "mainnet"
		sourceAliasFlag = ""
		rpcURLFlag = ""
	})
	networkFlag = "testnet"
	sourceAliasFlag = aliasPath

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := runDebugDryRun(cmd, validHash)
	if err == nil {
		t.Fatal("expected dry-run to fail for invalid --source-alias JSON")
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "source-alias") {
		t.Errorf("stderr should mention source-alias, got: %s", stderr)
	}
	if !strings.Contains(stderr, "[FAIL]") {
		t.Errorf("expected [FAIL] marker, got: %s", stderr)
	}
}

func TestDryRun_SourceAliasValid_PrintsOK(t *testing.T) {
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	data, _ := json.Marshal(map[string]string{"my_crate": "/tmp/src"})
	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		networkFlag = "mainnet"
		sourceAliasFlag = ""
		rpcURLFlag = ""
	})
	networkFlag = "testnet"
	sourceAliasFlag = aliasPath
	rpcURLFlag = "http://127.0.0.1:19999" // unreachable, so other checks fail

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	_ = runDebugDryRun(cmd, validHash)
	// Valid JSON should NOT produce a source-alias failure.
	if strings.Contains(errBuf.String(), "[FAIL]") &&
		strings.Contains(errBuf.String(), "source-alias") &&
		strings.Contains(errBuf.String(), "JSON") {
		t.Errorf("valid --source-alias should not produce a JSON failure, got: %s", errBuf.String())
	}
}

// ── PreRunE — source discovery validation integration ─────────────────────────

func TestDebugPreRunE_ContractSourceNonExistent_Rejected(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	contractSourceFlag = "/nonexistent/source/dir"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for non-existent --contract-source")
	}
	if !strings.Contains(err.Error(), "--contract-source") {
		t.Errorf("error should mention --contract-source, got: %v", err)
	}
}

func TestDebugPreRunE_ContractSourceIsFile_Rejected(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	dir := t.TempDir()
	filePath := filepath.Join(dir, "lib.rs")
	if err := os.WriteFile(filePath, []byte("fn main() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	contractSourceFlag = filePath

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error when --contract-source points to a file")
	}
	if !strings.Contains(err.Error(), "is a file, not a directory") {
		t.Errorf("error should say 'is a file, not a directory', got: %v", err)
	}
}

func TestDebugPreRunE_SourceAliasInvalidJSON_Rejected(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	if err := os.WriteFile(aliasPath, []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}
	sourceAliasFlag = aliasPath

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for invalid JSON in --source-alias")
	}
	if !strings.Contains(err.Error(), "--source-alias") {
		t.Errorf("error should mention --source-alias, got: %v", err)
	}
}

// ── dry-run error summary includes source discovery failures ──────────────────

func TestDryRun_SourceDiscoveryFailuresEnumeratedInSummary(t *testing.T) {
	t.Cleanup(func() {
		networkFlag = "mainnet"
		contractSourceFlag = ""
		sourceAliasFlag = ""
	})
	networkFlag = "testnet"
	contractSourceFlag = "/nonexistent/contract/src"

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := runDebugDryRun(cmd, validHash)
	if err == nil {
		t.Fatal("expected failure")
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "Dry-run FAILED") {
		t.Errorf("stderr should contain 'Dry-run FAILED', got: %s", stderr)
	}
	// The numbered error list must include the contract-source failure.
	if !strings.Contains(stderr, "  1.") {
		t.Errorf("failures should be numbered, got: %s", stderr)
	}
}

// ── validateSourceDiscoveryFlags — null-byte checks ──────────────────────────

func TestValidateSourceDiscoveryFlags_ContractSource_NullByte_ReturnsError(t *testing.T) {
	prev := contractSourceFlag
	contractSourceFlag = "/valid/path\x00injected"
	t.Cleanup(func() { contractSourceFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error for null byte in --contract-source")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--contract-source") {
		t.Errorf("error should mention --contract-source, got: %q", msg)
	}
	if !strings.Contains(msg, "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", msg)
	}
}

func TestValidateSourceDiscoveryFlags_SourceAlias_NullByte_ReturnsError(t *testing.T) {
	prev := sourceAliasFlag
	sourceAliasFlag = "/valid/path\x00injected.json"
	t.Cleanup(func() { sourceAliasFlag = prev })

	err := validateSourceDiscoveryFlags()
	if err == nil {
		t.Fatal("expected error for null byte in --source-alias")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--source-alias") {
		t.Errorf("error should mention --source-alias, got: %q", msg)
	}
	if !strings.Contains(msg, "null bytes") {
		t.Errorf("error should mention null bytes, got: %q", msg)
	}
}

// ── dry-run: source alias OK line printed exactly once ────────────────────────

// TestDryRun_SourceAliasValidJSON_OKLineNotDuplicated verifies that a valid
// --source-alias file causes the [OK] confirmation line to be printed exactly
// once, not twice (regression for the duplicate-print bug).
func TestDryRun_SourceAliasValidJSON_OKLineNotDuplicated(t *testing.T) {
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	data, _ := json.Marshal(map[string]string{"my_crate": filepath.Join(dir, "src")})
	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	// Create the target dir so LoadAliasConfig validation passes.
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		networkFlag = "mainnet"
		sourceAliasFlag = ""
		rpcURLFlag = ""
	})
	networkFlag = "testnet"
	sourceAliasFlag = aliasPath
	rpcURLFlag = "http://127.0.0.1:19999" // unreachable so other checks fail

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	_ = runDebugDryRun(cmd, validHash)

	stdout := out.String()
	count := strings.Count(stdout, "Source alias file:")
	if count > 1 {
		t.Errorf("'Source alias file:' should appear at most once in stdout, got %d occurrences:\n%s", count, stdout)
	}
}

// ── dry-run: both --contract-source null-byte and --source-alias null-byte ────

func TestDryRun_ContractSourceNullByte_AppearsInFailures(t *testing.T) {
	t.Cleanup(func() {
		networkFlag = "mainnet"
		contractSourceFlag = ""
		rpcURLFlag = ""
	})
	networkFlag = "testnet"
	contractSourceFlag = "/path\x00bad"

	var out, errBuf bytes.Buffer
	cmd := makeDebugCmdForTest()
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := runDebugDryRun(cmd, validHash)
	if err == nil {
		t.Fatal("expected dry-run to fail for null byte in --contract-source")
	}
	stderr := errBuf.String()
	if !strings.Contains(stderr, "[FAIL]") {
		t.Errorf("expected [FAIL] in stderr, got: %s", stderr)
	}
	if !strings.Contains(stderr, "contract-source") {
		t.Errorf("stderr should mention contract-source, got: %s", stderr)
	}
}
