// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package simulator

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validBase64 returns a valid base64-encoded string from s.
func validBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// ── LoadLedgerOverrideManifest ────────────────────────────────────────────────

func TestLoadLedgerOverrideManifest_FileNotFound(t *testing.T) {
	_, err := LoadLedgerOverrideManifest("/nonexistent/manifest.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("error should mention --mock-ledger-manifest flag, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %v", err)
	}
	// Must include a remediation hint.
	if !strings.Contains(err.Error(), "Ensure the path") {
		t.Errorf("error should include a remediation hint, got: %v", err)
	}
}

func TestLoadLedgerOverrideManifest_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{bad json}"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLedgerOverrideManifest(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("error should mention --mock-ledger-manifest, got: %v", err)
	}
	if !strings.Contains(err.Error(), "JSON") {
		t.Errorf("error should mention JSON, got: %v", err)
	}
	if !strings.Contains(err.Error(), "ledger_entries") {
		t.Errorf("error should describe expected format, got: %v", err)
	}
}

func TestLoadLedgerOverrideManifest_EmptyValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	content := `{"ledger_entries": {"myKey": ""}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLedgerOverrideManifest(path)
	if err == nil {
		t.Fatal("expected error for empty ledger entry value")
	}
	if !strings.Contains(err.Error(), "empty value") {
		t.Errorf("error should mention 'empty value', got: %v", err)
	}
	if !strings.Contains(err.Error(), "myKey") {
		t.Errorf("error should include the offending key, got: %v", err)
	}
}

func TestLoadLedgerOverrideManifest_InvalidBase64Value(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	content := `{"ledger_entries": {"myKey": "not!!valid!!base64"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLedgerOverrideManifest(path)
	if err == nil {
		t.Fatal("expected error for invalid base64 value")
	}
	if !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("error should mention 'invalid base64', got: %v", err)
	}
	if !strings.Contains(err.Error(), "myKey") {
		t.Errorf("error should include the offending key, got: %v", err)
	}
}

func TestLoadLedgerOverrideManifest_Success(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	v := validBase64("xdr-payload")
	content := `{"ledger_entries": {"keyA": "` + v + `"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadLedgerOverrideManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries["keyA"] != v {
		t.Errorf("expected keyA=%q, got %q", v, entries["keyA"])
	}
}

func TestLoadLedgerOverrideManifest_EmptyLedgerEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(`{"ledger_entries": {}}`), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadLedgerOverrideManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadLedgerOverrideManifest_NullLedgerEntries(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := os.WriteFile(path, []byte(`{"ledger_entries": null}`), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadLedgerOverrideManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for null, got %d", len(entries))
	}
}

// ── ParseLedgerOverrideFlags ──────────────────────────────────────────────────

func TestParseLedgerOverrideFlags_InvalidFormat(t *testing.T) {
	_, err := ParseLedgerOverrideFlags([]string{"nokeyvalue"})
	if err == nil {
		t.Fatal("expected error for missing colon separator")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-entry") {
		t.Errorf("error should mention --mock-ledger-entry, got: %v", err)
	}
	if !strings.Contains(err.Error(), "key:value") {
		t.Errorf("error should show expected format key:value, got: %v", err)
	}
}

func TestParseLedgerOverrideFlags_EmptyKey(t *testing.T) {
	_, err := ParseLedgerOverrideFlags([]string{":somevalue"})
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "--mock-ledger-entry") {
		t.Errorf("error should mention --mock-ledger-entry, got: %v", err)
	}
}

func TestParseLedgerOverrideFlags_EmptyValue(t *testing.T) {
	_, err := ParseLedgerOverrideFlags([]string{"somekey:"})
	if err == nil {
		t.Fatal("expected error for empty value")
	}
	if !strings.Contains(err.Error(), "empty value") {
		t.Errorf("error should mention 'empty value', got: %v", err)
	}
}

func TestParseLedgerOverrideFlags_InvalidBase64Value(t *testing.T) {
	_, err := ParseLedgerOverrideFlags([]string{"somekey:not!!base64"})
	if err == nil {
		t.Fatal("expected error for invalid base64 value")
	}
	if !strings.Contains(err.Error(), "invalid base64") {
		t.Errorf("error should mention 'invalid base64', got: %v", err)
	}
}

func TestParseLedgerOverrideFlags_Success(t *testing.T) {
	v := validBase64("xdr-data")
	overrides, err := ParseLedgerOverrideFlags([]string{"key1:" + v})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overrides["key1"] != v {
		t.Errorf("expected key1=%q, got %q", v, overrides["key1"])
	}
}

func TestParseLedgerOverrideFlags_MultipleEntries(t *testing.T) {
	v1 := validBase64("entry-one")
	v2 := validBase64("entry-two")
	overrides, err := ParseLedgerOverrideFlags([]string{"k1:" + v1, "k2:" + v2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overrides["k1"] != v1 || overrides["k2"] != v2 {
		t.Errorf("unexpected overrides: %v", overrides)
	}
}

func TestParseLedgerOverrideFlags_Empty(t *testing.T) {
	overrides, err := ParseLedgerOverrideFlags([]string{})
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(overrides) != 0 {
		t.Errorf("expected empty map, got %v", overrides)
	}
}

// ── MergeLedgerOverrides ──────────────────────────────────────────────────────

func TestMergeLedgerOverrides_NilBase(t *testing.T) {
	result := MergeLedgerOverrides(nil, map[string]string{"k": "v"})
	if result["k"] != "v" {
		t.Errorf("expected k=v in result, got %v", result)
	}
}

func TestMergeLedgerOverrides_EmptyOverrides(t *testing.T) {
	base := map[string]string{"existing": "val"}
	result := MergeLedgerOverrides(base, map[string]string{})
	if result["existing"] != "val" {
		t.Errorf("base entry should be preserved, got %v", result)
	}
}

func TestMergeLedgerOverrides_OverrideWins(t *testing.T) {
	base := map[string]string{"k": "old"}
	result := MergeLedgerOverrides(base, map[string]string{"k": "new"})
	if result["k"] != "new" {
		t.Errorf("override should win, got %v", result)
	}
}

// ── MergeLedgerOverrides — isolation (base map must not be mutated) ───────────

// TestMergeLedgerOverrides_DoesNotMutateBase verifies that calling
// MergeLedgerOverrides never modifies the caller's original base map.
// Before the fix, overrides were applied directly to base, so a second call
// with a different override set would see stale keys from the first call.
func TestMergeLedgerOverrides_DoesNotMutateBase(t *testing.T) {
	base := map[string]string{"k": "original"}
	_ = MergeLedgerOverrides(base, map[string]string{"k": "overridden"})

	// base must still hold the original value after the merge.
	if base["k"] != "original" {
		t.Errorf("MergeLedgerOverrides mutated base map: base[k] = %q, want %q",
			base["k"], "original")
	}
}

// TestMergeLedgerOverrides_NewKeyInOverride verifies that a key present only
// in the override appears in the result without affecting base.
func TestMergeLedgerOverrides_NewKeyInOverride(t *testing.T) {
	base := map[string]string{"existing": "val"}
	result := MergeLedgerOverrides(base, map[string]string{"new": "added"})

	if result["existing"] != "val" {
		t.Errorf("existing base key missing from result, got %q", result["existing"])
	}
	if result["new"] != "added" {
		t.Errorf("new override key missing from result, got %q", result["new"])
	}
	// Base must be unmodified.
	if _, ok := base["new"]; ok {
		t.Error("MergeLedgerOverrides added override key to the base map")
	}
}

// TestMergeLedgerOverrides_BothNilAndEmpty verifies that nil base + nil overrides
// returns nil (not a nil-pointer panic).
func TestMergeLedgerOverrides_BothNilAndEmpty(t *testing.T) {
	result := MergeLedgerOverrides(nil, nil)
	if result != nil {
		t.Errorf("expected nil result for nil+nil merge, got %v", result)
	}
}

// TestMergeLedgerOverrides_ResultIsIndependent verifies that mutating the
// returned map does not affect the original base.
func TestMergeLedgerOverrides_ResultIsIndependent(t *testing.T) {
	base := map[string]string{"k": "v"}
	result := MergeLedgerOverrides(base, map[string]string{"x": "y"})

	// Mutate the result.
	result["k"] = "changed"

	if base["k"] != "v" {
		t.Errorf("mutating result map affected base: base[k] = %q, want %q", base["k"], "v")
	}
}

// ── LoadLedgerOverrideManifest — additional edge cases ────────────────────────

// TestLoadLedgerOverrideManifest_KeyWithColonInValue verifies that a manifest
// entry value containing a colon is loaded without error (colons are valid in
// base64-encoded XDR strings when padded).
func TestLoadLedgerOverrideManifest_KeysArePreserved(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/manifest.json"
	v1 := validBase64("entry-alpha")
	v2 := validBase64("entry-beta")
	content := `{"ledger_entries": {"keyA": "` + v1 + `", "keyB": "` + v2 + `"}}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	entries, err := LoadLedgerOverrideManifest(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries["keyA"] != v1 || entries["keyB"] != v2 {
		t.Errorf("entries not preserved: %v", entries)
	}
}

// TestLoadLedgerOverrideManifest_EmptyPath verifies that an empty path string
// returns a not-found error rather than panicking.
func TestLoadLedgerOverrideManifest_EmptyPath(t *testing.T) {
	_, err := LoadLedgerOverrideManifest("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	// Should mention the flag name so the error is actionable.
	if !strings.Contains(err.Error(), "--mock-ledger-manifest") {
		t.Errorf("error should mention --mock-ledger-manifest, got: %v", err)
	}
}

// ── ParseLedgerOverrideFlags — value containing colon ─────────────────────────

// TestParseLedgerOverrideFlags_ValueWithColon verifies that a value that itself
// contains a colon is parsed correctly (SplitN with n=2 handles this).
func TestParseLedgerOverrideFlags_ValueWithColon(t *testing.T) {
	// base64 of "a:b" — the encoded form doesn't contain a colon, but the raw
	// value portion after the first colon should be taken as-is.
	v := validBase64("xdr:payload")
	overrides, err := ParseLedgerOverrideFlags([]string{"mykey:" + v})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overrides["mykey"] != v {
		t.Errorf("expected mykey=%q, got %q", v, overrides["mykey"])
	}
}
