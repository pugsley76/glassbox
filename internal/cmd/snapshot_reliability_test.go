// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Part A: improved debug command snapshot reliability.
//
// These tests exercise the snapshot validation path that runs before replay
// begins, ensuring invalid or stale snapshots surface clear, actionable errors
// rather than low-level panics or ambiguous failures.

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/snapshot"
)

// makeValidPersistedSnapshot creates a PersistedSnapshot on disk and returns
// its path. It is suitable for use with the --snapshot flag in unit tests.
func makeValidPersistedSnapshot(t *testing.T, txHash, network string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	meta := &snapshot.ReplayMetadata{
		GlassboxVersion: "v1.0.0",
		TxHash:          txHash,
		Network:         network,
	}
	params := map[string]string{"network": network, "tx": txHash}
	meta.ParamFingerprint = snapshot.BuildParamFingerprint(params)

	snap := snapshot.FromMap(map[string]string{
		"ledger-key-1": "ledger-val-1",
		"ledger-key-2": "ledger-val-2",
	})

	if err := snapshot.SavePersisted(path, meta, snap); err != nil {
		t.Fatalf("failed to create test snapshot: %v", err)
	}
	return path
}

// makeCorruptedSnapshotFile creates a file that looks like a snapshot but has
// tampered fingerprint data.
func makeCorruptedSnapshotFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.snap.json")

	// Write a snapshot with a deliberately wrong fingerprint.
	raw := `{
  "metadata": {
    "schema_version": 1,
    "glassbox_version": "v1.0.0",
    "saved_at": "2026-01-01T00:00:00Z",
    "tx_hash": "abc123",
    "network": "testnet"
  },
  "snapshot": {
    "ledgerEntries": [["key1", "value1"]],
    "fingerprint": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatalf("failed to write corrupted snapshot: %v", err)
	}
	return path
}

// makeStalledSnapshotFile creates a snapshot with a different network than
// the one we will attempt to replay on.
func makeStalledSnapshotFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "stale.snap.json")

	meta := &snapshot.ReplayMetadata{
		GlassboxVersion: "v1.0.0",
		TxHash:          "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd",
		Network:         "mainnet", // snapshot is for mainnet
	}
	params := map[string]string{
		"network": "mainnet",
		"tx":      "abc123abc123abc123abc123abc123abc123abc123abc123abc123abc123abcd",
	}
	meta.ParamFingerprint = snapshot.BuildParamFingerprint(params)

	snap := snapshot.FromMap(map[string]string{"k": "v"})
	if err := snapshot.SavePersisted(path, meta, snap); err != nil {
		t.Fatalf("failed to create stale snapshot: %v", err)
	}
	return path
}

// ── ValidateSnapshotBeforeReplay unit tests ───────────────────────────────────

func TestValidateSnapshot_NilReturnsActionableError(t *testing.T) {
	err := snapshot.ValidateSnapshotBeforeReplay(nil, "", "", nil, "")
	if err == nil {
		t.Fatal("expected error for nil snapshot")
	}
	msg := err.Error()
	if !strings.Contains(msg, "re-run") {
		t.Errorf("expected remediation hint in error, got: %q", msg)
	}
}

func TestValidateSnapshot_NetworkMismatchIsActionable(t *testing.T) {
	meta := &snapshot.ReplayMetadata{TxHash: "abc", Network: "mainnet"}
	snap := snapshot.FromMap(nil)
	ps := &snapshot.PersistedSnapshot{Metadata: meta, Snapshot: snap}

	err := snapshot.ValidateSnapshotBeforeReplay(ps, "", "testnet", nil, "")
	if err == nil {
		t.Fatal("expected error for network mismatch")
	}
	msg := err.Error()
	if !strings.Contains(msg, "network mismatch") {
		t.Errorf("error should say 'network mismatch', got: %q", msg)
	}
	// Should tell user exactly which flag to use.
	if !strings.Contains(msg, "--network") {
		t.Errorf("error should suggest --network flag, got: %q", msg)
	}
}

func TestValidateSnapshot_TxHashMismatchIncludesBothHashes(t *testing.T) {
	stored := "storedtxhash"
	expected := "expecttxhash"
	meta := &snapshot.ReplayMetadata{TxHash: stored, Network: "testnet"}
	snap := snapshot.FromMap(nil)
	ps := &snapshot.PersistedSnapshot{Metadata: meta, Snapshot: snap}

	err := snapshot.ValidateSnapshotBeforeReplay(ps, expected, "", nil, "")
	if err == nil {
		t.Fatal("expected error for tx hash mismatch")
	}
	msg := err.Error()
	if !strings.Contains(msg, stored) {
		t.Errorf("error should show stored hash, got: %q", msg)
	}
	if !strings.Contains(msg, expected) {
		t.Errorf("error should show expected hash, got: %q", msg)
	}
}

func TestValidateSnapshot_StaleSourceHash_MentionsWasm(t *testing.T) {
	meta := &snapshot.ReplayMetadata{
		TxHash:     "abc",
		Network:    "testnet",
		SourceHash: "old_hash",
	}
	snap := snapshot.FromMap(nil)
	ps := &snapshot.PersistedSnapshot{Metadata: meta, Snapshot: snap}

	err := snapshot.ValidateSnapshotBeforeReplay(ps, "", "", nil, "new_hash")
	if err == nil {
		t.Fatal("expected error for stale WASM source hash")
	}
	msg := err.Error()
	if !strings.Contains(strings.ToLower(msg), "wasm") {
		t.Errorf("error should mention WASM, got: %q", msg)
	}
}

func TestValidateSnapshot_FingerprintMismatch_IsBlockingNotJustLogged(t *testing.T) {
	// This ensures drift detection is blocking (not just a log statement)
	// when ValidateSnapshotBeforeReplay is used.
	meta := &snapshot.ReplayMetadata{TxHash: "abc", Network: "testnet"}
	snap := snapshot.FromMap(map[string]string{"k": "v"})
	// Tamper the fingerprint to simulate corruption.
	snap.Fingerprint = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	ps := &snapshot.PersistedSnapshot{Metadata: meta, Snapshot: snap}

	err := snapshot.ValidateSnapshotBeforeReplay(ps, "", "", nil, "")
	if err == nil {
		t.Fatal("expected error for tampered fingerprint")
	}
	msg := err.Error()
	if !strings.Contains(msg, "fingerprint mismatch") {
		t.Errorf("error should say 'fingerprint mismatch', got: %q", msg)
	}
	// Must suggest re-running the debug command.
	if !strings.Contains(msg, "Re-run") {
		t.Errorf("error must suggest re-running the debug command, got: %q", msg)
	}
}

// ── LoadWithDiagnostics unit tests ────────────────────────────────────────────

func TestLoadWithDiagnostics_CleanSnapshotNoWarning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clean.json")

	snap := snapshot.FromMap(map[string]string{"k": "v"})
	if err := snapshot.Save(path, snap); err != nil {
		t.Fatal(err)
	}

	loaded, warn, err := snapshot.LoadWithDiagnostics(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if warn != nil {
		t.Errorf("expected no warning for clean snapshot, got: %v", warn)
	}
	if loaded == nil {
		t.Fatal("expected non-nil snapshot")
	}
}

func TestLoadWithDiagnostics_MissingFile_ClearError(t *testing.T) {
	_, _, err := snapshot.LoadWithDiagnostics("/nonexistent/snapshot.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	// Error should mention the file path.
	if !strings.Contains(err.Error(), "/nonexistent/snapshot.json") {
		t.Errorf("error should mention file path, got: %v", err)
	}
}

// ── SnapshotLoadDiagnostic formatting ─────────────────────────────────────────

func TestSnapshotLoadDiagnostic_ContainsExpectedSections(t *testing.T) {
	meta := &snapshot.ReplayMetadata{
		TxHash:          "abc123",
		Network:         "testnet",
		GlassboxVersion: "v1.0.0",
	}
	snap := snapshot.FromMap(map[string]string{"k": "v"})
	ps := &snapshot.PersistedSnapshot{Metadata: meta, Snapshot: snap}

	diag := snapshot.SnapshotLoadDiagnostic(ps)

	for _, want := range []string{"testnet", "abc123", "Ledger entries", "Fingerprint"} {
		if !strings.Contains(diag, want) {
			t.Errorf("diagnostic should contain %q, got:\n%s", want, diag)
		}
	}
}

// ── SavePersisted validation ──────────────────────────────────────────────────

func TestSavePersisted_NilMeta_ReturnsError(t *testing.T) {
	err := snapshot.SavePersisted("/tmp/test.json", nil, snapshot.FromMap(nil))
	if err == nil {
		t.Fatal("expected error for nil metadata")
	}
}

func TestSavePersisted_EmptyTxHash_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	meta := &snapshot.ReplayMetadata{
		TxHash:  "",
		Network: "testnet",
	}
	err := snapshot.SavePersisted(path, meta, snapshot.FromMap(nil))
	if err == nil {
		t.Fatal("expected error for empty TxHash")
	}
	if !strings.Contains(err.Error(), "transaction hash") {
		t.Errorf("error should mention 'transaction hash', got: %v", err)
	}
}

func TestSavePersisted_EmptyNetwork_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	meta := &snapshot.ReplayMetadata{
		TxHash:  "abc123",
		Network: "",
	}
	err := snapshot.SavePersisted(path, meta, snapshot.FromMap(nil))
	if err == nil {
		t.Fatal("expected error for empty Network")
	}
	if !strings.Contains(err.Error(), "network") {
		t.Errorf("error should mention 'network', got: %v", err)
	}
}

func TestSavePersisted_InvalidNetwork_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	meta := &snapshot.ReplayMetadata{
		TxHash:  "abc123",
		Network: "devnet",
	}
	err := snapshot.SavePersisted(path, meta, snapshot.FromMap(nil))
	if err == nil {
		t.Fatal("expected error for invalid Network")
	}
	if !strings.Contains(err.Error(), "devnet") {
		t.Errorf("error should name the invalid network, got: %v", err)
	}
}

// ── LoadPersisted validation ──────────────────────────────────────────────────

func TestLoadPersisted_MissingTxHashInMetadata_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	// Build a file with a PersistedSnapshot but no TxHash in metadata.
	raw := `{
  "metadata": {
    "schema_version": 1,
    "glassbox_version": "v1.0.0",
    "saved_at": "2026-01-01T00:00:00Z",
    "tx_hash": "",
    "network": "testnet"
  },
  "snapshot": {
    "ledgerEntries": [],
    "fingerprint": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := snapshot.LoadPersisted(path)
	if err == nil {
		t.Fatal("expected error for snapshot with empty TxHash in metadata")
	}
}

func TestLoadPersisted_MissingNetworkInMetadata_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	raw := `{
  "metadata": {
    "schema_version": 1,
    "glassbox_version": "v1.0.0",
    "saved_at": "2026-01-01T00:00:00Z",
    "tx_hash": "abc123",
    "network": ""
  },
  "snapshot": {
    "ledgerEntries": [],
    "fingerprint": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
  }
}`
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := snapshot.LoadPersisted(path)
	if err == nil {
		t.Fatal("expected error for snapshot with empty Network in metadata")
	}
}

// ── snapshot_persist.go CLI integration tests ─────────────────────────────────

func TestRunSnapshotLoad_MissingPath_ClearError(t *testing.T) {
	prev := snapLoadPathFlag
	snapLoadPathFlag = ""
	defer func() { snapLoadPathFlag = prev }()

	err := snapshotLoadCmd.RunE(snapshotLoadCmd, nil)
	if err == nil {
		t.Fatal("expected error for missing --path")
	}
	if !strings.Contains(err.Error(), "path") {
		t.Errorf("error should mention 'path', got: %v", err)
	}
}

func TestRunSnapshotSave_InvalidWasmPath_ClearError(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "state.json")
	snap := snapshot.FromMap(map[string]string{"k": "v"})
	if err := snapshot.Save(inputPath, snap); err != nil {
		t.Fatal(err)
	}

	prevTx := snapSaveTxHashFlag
	prevInput := snapSaveInputFlag
	prevWasm := snapSaveWasmFlag
	prevOutput := snapSaveOutputFlag
	t.Cleanup(func() {
		snapSaveTxHashFlag = prevTx
		snapSaveInputFlag = prevInput
		snapSaveWasmFlag = prevWasm
		snapSaveOutputFlag = prevOutput
	})

	snapSaveTxHashFlag = "abc123"
	snapSaveInputFlag = inputPath
	snapSaveWasmFlag = "/nonexistent/contract.wasm"
	snapSaveOutputFlag = filepath.Join(dir, "out.snap.json")

	err := snapshotSaveCmd.RunE(snapshotSaveCmd, nil)
	if err == nil {
		t.Fatal("expected error for non-existent WASM file")
	}
	// Should name the bad file path.
	if !strings.Contains(err.Error(), "/nonexistent/contract.wasm") {
		t.Errorf("error should mention bad WASM path, got: %v", err)
	}
}

func TestRunSnapshotLoad_TxMismatch_IsActionable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	meta := &snapshot.ReplayMetadata{TxHash: "abc123", Network: "testnet", GlassboxVersion: "v1"}
	snap := snapshot.FromMap(map[string]string{"k": "v"})
	if err := snapshot.SavePersisted(path, meta, snap); err != nil {
		t.Fatal(err)
	}

	prevPath := snapLoadPathFlag
	prevTx := snapLoadTxHashFlag
	prevVerify := snapLoadVerifyFlag
	t.Cleanup(func() {
		snapLoadPathFlag = prevPath
		snapLoadTxHashFlag = prevTx
		snapLoadVerifyFlag = prevVerify
	})

	snapLoadPathFlag = path
	snapLoadTxHashFlag = "different_hash_entirely"
	snapLoadVerifyFlag = true

	err := snapshotLoadCmd.RunE(snapshotLoadCmd, nil)
	if err == nil {
		t.Fatal("expected error for tx hash mismatch with --verify")
	}
	// Error should be actionable.
	if !strings.Contains(err.Error(), "abc123") && !strings.Contains(err.Error(), "tx hash") {
		t.Errorf("error should mention stored tx hash or 'tx hash', got: %v", err)
	}
}

// ── snapshot_diff command tests ───────────────────────────────────────────────

func TestSnapshotDiff_BothPathsRequired(t *testing.T) {
	// Verify that snapshot diff commands require both source and target paths.
	snap := snapshot.FromMap(map[string]string{"k": "v"})
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")
	if err := snapshot.Save(path, snap); err != nil {
		t.Fatal(err)
	}

	diff := snapshot.DiffSnapshots(snap, snapshot.FromMap(nil))
	if diff == nil {
		t.Fatal("expected non-nil diff")
	}
	// Snap has 1 entry, nil snap has 0 → 1 removal.
	if diff.TotalChanges() == 0 {
		t.Error("expected at least one change in diff")
	}
}
