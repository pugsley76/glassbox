// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// wasmMagicBytes returns the 8-byte minimal valid WASM binary header.
func wasmMagicBytes() []byte {
	return []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
}

// writeWasmFile writes a minimal valid WASM file and returns its path.
func writeWasmFile(t *testing.T, dir string) string {
	t.Helper()
	p := filepath.Join(dir, "contract.wasm")
	if err := os.WriteFile(p, wasmMagicBytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// resetArtifactFlags resets all flags touched by this test file.
func resetArtifactFlags(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		wasmPath = ""
		contractSourceFlag = ""
		mockLedgerManifest = ""
		sourceAliasFlag = ""
		hotReloadFlag = false
		xdrFileFlag = ""
		jsonFileFlag = ""
		demoMode = false
		loadSnapshotsFlag = ""
		opIndexFlag = -1
		watchFlag = false
		watchTimeoutFlag = 30
		networkFlag = "mainnet"
		compareNetworkFlag = ""
		secureWorkspaceFlag = false
		traceVerbosityFlag = "normal"
		debugFormatFlag = "text"
	})
}

// ── --wasm validation ─────────────────────────────────────────────────────────

func TestDebugPreRunE_WasmFileNotFound(t *testing.T) {
	resetArtifactFlags(t)
	wasmPath = "/nonexistent/contract.wasm"

	err := debugCmd.PreRunE(debugCmd, []string{})
	if err == nil {
		t.Fatal("expected error for missing --wasm file")
	}
	if !strings.Contains(err.Error(), "--wasm") {
		t.Errorf("error should mention --wasm, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %v", err)
	}
	// Must include a remediation hint.
	if !strings.Contains(err.Error(), "cargo build") {
		t.Errorf("error should suggest 'cargo build', got: %v", err)
	}
}

func TestDebugPreRunE_WasmFileInvalidMagic(t *testing.T) {
	resetArtifactFlags(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "not_wasm.wasm")
	// Write a non-WASM file (e.g., ELF header)
	if err := os.WriteFile(p, []byte{0x7f, 0x45, 0x4c, 0x46, 0x00, 0x00, 0x00, 0x00}, 0644); err != nil {
		t.Fatal(err)
	}
	wasmPath = p

	err := debugCmd.PreRunE(debugCmd, []string{})
	if err == nil {
		t.Fatal("expected error for non-WASM file")
	}
	if !strings.Contains(err.Error(), "not a valid WASM binary") {
		t.Errorf("error should say 'not a valid WASM binary', got: %v", err)
	}
}

func TestDebugPreRunE_WasmFileTooSmall(t *testing.T) {
	resetArtifactFlags(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "tiny.wasm")
	if err := os.WriteFile(p, []byte{0x00, 0x61}, 0644); err != nil {
		t.Fatal(err)
	}
	wasmPath = p

	err := debugCmd.PreRunE(debugCmd, []string{})
	if err == nil {
		t.Fatal("expected error for too-small WASM file")
	}
	if !strings.Contains(err.Error(), "not a valid WASM binary") {
		t.Errorf("error should say 'not a valid WASM binary', got: %v", err)
	}
}

func TestDebugPreRunE_WasmFileValid(t *testing.T) {
	resetArtifactFlags(t)
	dir := t.TempDir()
	wasmPath = writeWasmFile(t, dir)

	err := debugCmd.PreRunE(debugCmd, []string{})
	// Other errors may fire (no hash, no network) but NOT wasm-related ones.
	if err != nil && strings.Contains(err.Error(), "--wasm") {
		t.Errorf("should not reject valid WASM file, got: %v", err)
	}
}

// ── --contract-source validation ──────────────────────────────────────────────

func TestDebugPreRunE_ContractSourceNotFound(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	contractSourceFlag = "/nonexistent/src/dir"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for missing --contract-source directory")
	}
	if !strings.Contains(err.Error(), "--contract-source") {
		t.Errorf("error should mention --contract-source, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %v", err)
	}
}

func TestDebugPreRunE_ContractSourceIsFile(t *testing.T) {
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

func TestDebugPreRunE_ContractSourceValidDirectory(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	dir := t.TempDir()
	contractSourceFlag = dir

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	// May fail for other reasons but NOT for contract-source.
	if err != nil && strings.Contains(err.Error(), "--contract-source") {
		t.Errorf("should not reject valid --contract-source directory, got: %v", err)
	}
}

// ── --mock-ledger-manifest validation ────────────────────────────────────────

func TestDebugPreRunE_MockLedgerManifestNotFound(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	mockLedgerManifest = "/nonexistent/manifest.json"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for missing --mock-ledger-manifest file")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("error should mention --mock-ledger-manifest, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %v", err)
	}
}

func TestDebugPreRunE_MockLedgerManifestValidFile(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	content := `{"ledger_entries": {}}`
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	mockLedgerManifest = manifestPath

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err != nil && strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("should not reject valid manifest file, got: %v", err)
	}
}

// ── --source-alias validation ─────────────────────────────────────────────────

func TestDebugPreRunE_SourceAliasNotFound(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	sourceAliasFlag = "/nonexistent/aliases.json"

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for missing --source-alias file")
	}
	if !strings.Contains(err.Error(), "--source-alias") {
		t.Errorf("error should mention --source-alias, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %v", err)
	}
}

func TestDebugPreRunE_SourceAliasInvalidJSON(t *testing.T) {
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
		t.Fatal("expected error for invalid JSON in --source-alias file")
	}
	if !strings.Contains(err.Error(), "--source-alias") {
		t.Errorf("error should mention --source-alias, got: %v", err)
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("error should mention JSON parse failure, got: %v", err)
	}
	// Must include a remediation example.
	if !strings.Contains(err.Error(), "Example") {
		t.Errorf("error should include an Example hint, got: %v", err)
	}
}

func TestDebugPreRunE_SourceAliasValid(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")
	// Alias targets don't need to exist — we only warn, don't error.
	aliasMap := map[string]string{"my_crate": "/some/path/that/may/not/exist"}
	data, _ := json.Marshal(aliasMap)
	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	sourceAliasFlag = aliasPath

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	// Must NOT produce a --source-alias error for valid JSON.
	if err != nil && strings.Contains(err.Error(), "--source-alias") {
		t.Errorf("should not error on valid --source-alias JSON, got: %v", err)
	}
}

// ── loadMockLedgerOverrides error messages ────────────────────────────────────

func TestLoadMockLedgerOverrides_ManifestNotFound(t *testing.T) {
	prev := mockLedgerManifest
	mockLedgerManifest = "/nonexistent/manifest.json"
	t.Cleanup(func() { mockLedgerManifest = prev })

	_, err := loadMockLedgerOverrides()
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("error should mention --mock-ledger-manifest, got: %v", err)
	}
}

func TestLoadMockLedgerOverrides_ManifestInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	prev := mockLedgerManifest
	mockLedgerManifest = path
	t.Cleanup(func() { mockLedgerManifest = prev })

	_, err := loadMockLedgerOverrides()
	if err == nil {
		t.Fatal("expected error for invalid JSON manifest")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("error should mention --mock-ledger-manifest, got: %v", err)
	}
}

func TestLoadMockLedgerOverrides_ManifestEmptyValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	// Entry with empty value.
	content := `{"ledger_entries": {"validKey": ""}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	prev := mockLedgerManifest
	mockLedgerManifest = path
	t.Cleanup(func() { mockLedgerManifest = prev })

	_, err := loadMockLedgerOverrides()
	if err == nil {
		t.Fatal("expected error for empty ledger entry value")
	}
	if !strings.Contains(err.Error(), "empty value") {
		t.Errorf("error should say 'empty value', got: %v", err)
	}
}

func TestLoadMockLedgerOverrides_ManifestInvalidBase64Value(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	content := `{"ledger_entries": {"validKey": "not_valid_base64!!!"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	prev := mockLedgerManifest
	mockLedgerManifest = path
	t.Cleanup(func() { mockLedgerManifest = prev })

	_, err := loadMockLedgerOverrides()
	if err == nil {
		t.Fatal("expected error for invalid base64 value")
	}
	if !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("error should mention 'invalid base64', got: %v", err)
	}
}

func TestLoadMockLedgerOverrides_FlagEmptyValue(t *testing.T) {
	prev := mockLedgerEntryFlags
	mockLedgerEntryFlags = []string{"someKey:"}
	t.Cleanup(func() { mockLedgerEntryFlags = prev })

	_, err := loadMockLedgerOverrides()
	if err == nil {
		t.Fatal("expected error for empty value in --mock-ledger-entry")
	}
	if !strings.Contains(err.Error(), "empty value") {
		t.Errorf("error should say 'empty value', got: %v", err)
	}
}

func TestLoadMockLedgerOverrides_FlagInvalidBase64Value(t *testing.T) {
	prev := mockLedgerEntryFlags
	mockLedgerEntryFlags = []string{"someKey:not!valid!base64"}
	t.Cleanup(func() { mockLedgerEntryFlags = prev })

	_, err := loadMockLedgerOverrides()
	if err == nil {
		t.Fatal("expected error for invalid base64 in --mock-ledger-entry")
	}
	if !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("error should mention 'invalid base64', got: %v", err)
	}
}

func TestLoadMockLedgerOverrides_FlagValidEntry(t *testing.T) {
	prev := mockLedgerEntryFlags
	validBase64 := base64.StdEncoding.EncodeToString([]byte("ledger-entry-xdr"))
	mockLedgerEntryFlags = []string{"myKey:" + validBase64}
	t.Cleanup(func() { mockLedgerEntryFlags = prev })

	overrides, err := loadMockLedgerOverrides()
	if err != nil {
		t.Fatalf("unexpected error for valid entry: %v", err)
	}
	if overrides["myKey"] != validBase64 {
		t.Errorf("expected entry 'myKey' to be set, got: %v", overrides)
	}
}

func TestLoadMockLedgerOverrides_ManifestSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	validBase64 := base64.StdEncoding.EncodeToString([]byte("xdr-payload"))
	content, _ := json.Marshal(map[string]interface{}{
		"ledger_entries": map[string]string{
			"keyA": validBase64,
		},
	})
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	prev := mockLedgerManifest
	mockLedgerManifest = path
	t.Cleanup(func() { mockLedgerManifest = prev })

	overrides, err := loadMockLedgerOverrides()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overrides["keyA"] != validBase64 {
		t.Errorf("expected keyA to be set, got: %v", overrides)
	}
}

// ── --wasm whitespace path validation ────────────────────────────────────────

// TestDebugPreRunE_WasmWhitespaceOnly verifies that a whitespace-only --wasm
// path is treated as a missing file rather than producing a confusing OS error.
// The path is non-empty so the PreRunE validation branch fires, but the file
// obviously doesn't exist.
func TestDebugPreRunE_WasmWhitespacePath_NotFound(t *testing.T) {
	resetArtifactFlags(t)
	wasmPath = "   " // whitespace only — treated as invalid path

	err := debugCmd.PreRunE(debugCmd, []string{})
	// Expect an error — either "not found" or another path-related error.
	// The key requirement is that it doesn't silently succeed.
	if err == nil {
		t.Fatal("expected error for whitespace-only --wasm path")
	}
}

// ── --contract-source whitespace validation (inline PreRunE path) ─────────────

// TestDebugPreRunE_ContractSourceWhitespace_InlinePath verifies that the
// inline PreRunE check (used for the transaction-hash code path) also rejects
// whitespace-only --contract-source values.
func TestDebugPreRunE_ContractSourceWhitespace_Rejected(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	contractSourceFlag = "   " // whitespace only

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	if err == nil {
		t.Fatal("expected error for whitespace-only --contract-source")
	}
	if !strings.Contains(err.Error(), "--contract-source") {
		t.Errorf("error should mention --contract-source, got: %v", err)
	}
	if !strings.Contains(err.Error(), "empty") && !strings.Contains(err.Error(), "whitespace") {
		t.Errorf("error should mention empty/whitespace, got: %v", err)
	}
}

// ── loadMockLedgerOverrides — no-flags baseline ───────────────────────────────

// TestLoadMockLedgerOverrides_NoFlags_ReturnsEmptyMap verifies that calling
// loadMockLedgerOverrides when neither --mock-ledger-manifest nor
// --mock-ledger-entry flags are set returns an empty map without error.
// This is the baseline "no overrides" case that every normal debug run uses.
func TestLoadMockLedgerOverrides_NoFlags_ReturnsEmptyMap(t *testing.T) {
	prevManifest := mockLedgerManifest
	prevEntries := mockLedgerEntryFlags
	mockLedgerManifest = ""
	mockLedgerEntryFlags = []string{}
	t.Cleanup(func() {
		mockLedgerManifest = prevManifest
		mockLedgerEntryFlags = prevEntries
	})

	overrides, err := loadMockLedgerOverrides()
	if err != nil {
		t.Fatalf("expected no error when no flags set, got: %v", err)
	}
	if len(overrides) != 0 {
		t.Errorf("expected empty overrides map, got: %v", overrides)
	}
}

// ── --source-alias target validation (warning, not error) ─────────────────────

// TestDebugPreRunE_SourceAlias_MissingTarget_IsWarnNotError verifies that a
// valid --source-alias JSON file where a target path doesn't exist on disk
// produces a warning but does NOT cause PreRunE to return an error.
// Users should still be able to debug even when some alias targets are stale.
func TestDebugPreRunE_SourceAlias_MissingTarget_IsWarnNotError(t *testing.T) {
	resetArtifactFlags(t)
	networkFlag = "testnet"
	dir := t.TempDir()
	aliasPath := filepath.Join(dir, "aliases.json")

	// Alias target does not exist on disk — should warn, not error.
	aliasData := map[string]string{
		"my_crate": "/nonexistent/path/that/does/not/exist",
	}
	data, _ := json.Marshal(aliasData)
	if err := os.WriteFile(aliasPath, data, 0644); err != nil {
		t.Fatal(err)
	}
	sourceAliasFlag = aliasPath

	validHash := "5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab"
	err := debugCmd.PreRunE(debugCmd, []string{validHash})
	// Must NOT return a --source-alias error for a missing target path.
	if err != nil && strings.Contains(err.Error(), "--source-alias") {
		t.Errorf("missing alias target should be a warning, not an error; got: %v", err)
	}
}
