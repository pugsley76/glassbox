// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ── CurrentJSONSchemaVersion constant ─────────────────────────────────────────

func TestCurrentJSONSchemaVersion_Format(t *testing.T) {
	// Must be MAJOR.MINOR (exactly two numeric components).
	if err := ValidateJSONSchemaVersion(CurrentJSONSchemaVersion); err != nil {
		t.Errorf("CurrentJSONSchemaVersion %q fails its own validator: %v", CurrentJSONSchemaVersion, err)
	}
}

func TestCurrentJSONSchemaVersion_IsInSupportedList(t *testing.T) {
	if !IsJSONSchemaVersionSupported(CurrentJSONSchemaVersion) {
		t.Errorf("CurrentJSONSchemaVersion %q is not in SupportedJSONSchemaVersions", CurrentJSONSchemaVersion)
	}
}

// ── ValidateJSONSchemaVersion ──────────────────────────────────────────────────

func TestValidateJSONSchemaVersion_Empty(t *testing.T) {
	err := ValidateJSONSchemaVersion("")
	if err == nil {
		t.Fatal("expected error for empty schema version")
	}
	msg := err.Error()
	if !strings.Contains(msg, "empty") {
		t.Errorf("error should say 'empty', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include Fix hint, got: %q", msg)
	}
}

func TestValidateJSONSchemaVersion_WhitespaceOnly(t *testing.T) {
	if err := ValidateJSONSchemaVersion("   "); err == nil {
		t.Fatal("expected error for whitespace-only schema version")
	}
}

func TestValidateJSONSchemaVersion_TooManyComponents(t *testing.T) {
	err := ValidateJSONSchemaVersion("1.0.0")
	if err == nil {
		t.Fatal("expected error for three-component version (MAJOR.MINOR.PATCH not accepted)")
	}
	if !strings.Contains(err.Error(), "MAJOR.MINOR") {
		t.Errorf("error should mention expected MAJOR.MINOR format, got: %q", err.Error())
	}
}

func TestValidateJSONSchemaVersion_TooFewComponents(t *testing.T) {
	err := ValidateJSONSchemaVersion("1")
	if err == nil {
		t.Fatal("expected error for single-component version")
	}
}

func TestValidateJSONSchemaVersion_NonNumeric(t *testing.T) {
	err := ValidateJSONSchemaVersion("one.zero")
	if err == nil {
		t.Fatal("expected error for non-numeric components")
	}
	if !strings.Contains(err.Error(), "non-numeric") {
		t.Errorf("error should mention non-numeric characters, got: %q", err.Error())
	}
}

func TestValidateJSONSchemaVersion_EmptyComponent(t *testing.T) {
	err := ValidateJSONSchemaVersion(".0")
	if err == nil {
		t.Fatal("expected error for empty leading component")
	}
}

func TestValidateJSONSchemaVersion_UnsupportedVersion(t *testing.T) {
	err := ValidateJSONSchemaVersion("99.0")
	if err == nil {
		t.Fatal("expected error for unsupported version 99.0")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not supported") {
		t.Errorf("error should say 'not supported', got: %q", msg)
	}
	if !strings.Contains(msg, CurrentJSONSchemaVersion) {
		t.Errorf("error should mention current version %q, got: %q", CurrentJSONSchemaVersion, msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include Fix hint, got: %q", msg)
	}
}

func TestValidateJSONSchemaVersion_CurrentVersionPasses(t *testing.T) {
	if err := ValidateJSONSchemaVersion(CurrentJSONSchemaVersion); err != nil {
		t.Errorf("current version %q should pass validation, got: %v", CurrentJSONSchemaVersion, err)
	}
}

// ── ExportJSON schema_version field ───────────────────────────────────────────

func TestExportJSON_ContainsCurrentSchemaVersion(t *testing.T) {
	tr := NewExecutionTrace("schema-version-test", 10)
	tr.AddState(ExecutionState{Operation: "op", Timestamp: time.Now()})

	data, err := tr.ExportJSON(CurrentJSONSchemaVersion, time.Now())
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to parse ExportJSON output: %v", err)
	}

	got, ok := envelope["schema_version"].(string)
	if !ok {
		t.Fatalf("schema_version field missing or not a string: %v", envelope["schema_version"])
	}
	if got != CurrentJSONSchemaVersion {
		t.Errorf("schema_version = %q, want %q", got, CurrentJSONSchemaVersion)
	}
}

func TestExportJSON_ContainsGeneratedAt(t *testing.T) {
	tr := NewExecutionTrace("generated-at-test", 10)
	tr.AddState(ExecutionState{Operation: "op", Timestamp: time.Now()})

	fixed := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	data, err := tr.ExportJSON(CurrentJSONSchemaVersion, fixed)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to parse ExportJSON output: %v", err)
	}

	genAt, ok := envelope["generated_at"].(string)
	if !ok || genAt == "" {
		t.Errorf("generated_at missing or empty: %v", envelope["generated_at"])
	}
	// Should be 2026-06-01T12:00:00Z (truncated to second).
	if !strings.HasPrefix(genAt, "2026-06-01") {
		t.Errorf("generated_at = %q, expected prefix 2026-06-01", genAt)
	}
}

func TestExportJSON_TraceFieldPresent(t *testing.T) {
	tr := NewExecutionTrace("trace-field-test", 10)
	tr.AddState(ExecutionState{Operation: "op", Timestamp: time.Now()})

	data, err := tr.ExportJSON(CurrentJSONSchemaVersion, time.Now())
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to parse ExportJSON output: %v", err)
	}

	traceObj, ok := envelope["trace"].(map[string]interface{})
	if !ok {
		t.Fatalf("trace field missing or wrong type: %v", envelope["trace"])
	}

	// Required sub-fields.
	for _, field := range []string{"transaction_hash", "start_time", "states"} {
		if _, exists := traceObj[field]; !exists {
			t.Errorf("trace.%s is missing from ExportJSON output", field)
		}
	}
}

func TestExportJSON_FingerprintedTransactionHash(t *testing.T) {
	// The raw transaction hash must NOT appear verbatim; it should be SHA-256 fingerprinted.
	rawHash := "ACTUAL_TX_HASH_SHOULD_NOT_APPEAR"
	tr := NewExecutionTrace(rawHash, 10)
	tr.AddState(ExecutionState{Operation: "op", Timestamp: time.Now()})

	data, err := tr.ExportJSON(CurrentJSONSchemaVersion, time.Now())
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	if strings.Contains(string(data), rawHash) {
		t.Errorf("raw transaction hash %q should not appear in ExportJSON output (should be fingerprinted)", rawHash)
	}

	var envelope map[string]interface{}
	if err := json.Unmarshal(data, &envelope); err != nil {
		t.Fatalf("failed to parse ExportJSON output: %v", err)
	}
	traceObj := envelope["trace"].(map[string]interface{})
	fp := traceObj["transaction_hash"].(string)
	if !strings.HasPrefix(fp, "sha256:") {
		t.Errorf("transaction_hash fingerprint should start with 'sha256:', got: %q", fp)
	}
}

func TestExportJSON_Deterministic(t *testing.T) {
	tr := NewExecutionTrace("det-test", 10)
	tr.AddState(ExecutionState{Operation: "op1", Timestamp: time.Now()})
	tr.AddState(ExecutionState{Operation: "op2", HostState: map[string]interface{}{"b": 2, "a": 1}})

	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	a, err := tr.ExportJSON(CurrentJSONSchemaVersion, fixed)
	if err != nil {
		t.Fatalf("first ExportJSON failed: %v", err)
	}
	b, err := tr.ExportJSON(CurrentJSONSchemaVersion, fixed)
	if err != nil {
		t.Fatalf("second ExportJSON failed: %v", err)
	}
	if string(a) != string(b) {
		t.Error("ExportJSON should be deterministic: two calls with same input produced different output")
	}
}

// ── LoadVersionedTrace: ExportJSON envelope ────────────────────────────────────

func TestLoadVersionedTrace_ExportJSONEnvelope(t *testing.T) {
	// Write an ExportJSON-format file and verify LoadVersionedTrace reads it.
	tr := NewExecutionTrace("load-export-json-test", 10)
	tr.AddState(ExecutionState{Operation: "load-test", Timestamp: time.Now()})

	data, err := tr.ExportJSON(CurrentJSONSchemaVersion, time.Now())
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "export-json.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	loaded, err := LoadVersionedTrace(path, DefaultCompatibilityOptions())
	if err != nil {
		t.Fatalf("LoadVersionedTrace failed on ExportJSON envelope: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded trace is nil")
	}
	// Transaction hash in ExportJSON is fingerprinted, not raw, so we can't
	// compare it to the original. We verify the loaded trace is structurally sound.
	if len(loaded.States) != 1 {
		t.Errorf("expected 1 state, got %d", len(loaded.States))
	}
}

func TestLoadVersionedTrace_ExportJSONEnvelope_UnsupportedVersion(t *testing.T) {
	// An ExportJSON file with an unsupported schema_version should fail with a clear error.
	tr := NewExecutionTrace("unsupported-schema", 10)
	tr.AddState(ExecutionState{Operation: "op", Timestamp: time.Now()})

	data, err := tr.ExportJSON("99.0", time.Now()) // future/unsupported version
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "future-schema.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err = LoadVersionedTrace(path, DefaultCompatibilityOptions())
	if err == nil {
		t.Fatal("expected error for unsupported schema_version")
	}
	msg := err.Error()
	if !strings.Contains(msg, "unsupported schema_version") {
		t.Errorf("error should say 'unsupported schema_version', got: %q", msg)
	}
	if !strings.Contains(msg, "99.0") {
		t.Errorf("error should echo the invalid version, got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include Fix hint, got: %q", msg)
	}
}

func TestLoadVersionedTrace_ExportJSONEnvelope_ValidFormatVersionedAlsoWorks(t *testing.T) {
	// ExportVersionedTrace (VersionedTrace envelope) should still load correctly.
	tr := NewExecutionTrace("versioned-compat-test", 10)
	tr.AddState(ExecutionState{Operation: "test", Timestamp: time.Now()})

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "versioned.json")
	if err := ExportVersionedTrace(tr, "json", path, ExportOptions{}, DefaultCompatibilityOptions()); err != nil {
		t.Fatalf("ExportVersionedTrace failed: %v", err)
	}

	loaded, err := LoadVersionedTrace(path, DefaultCompatibilityOptions())
	if err != nil {
		t.Fatalf("LoadVersionedTrace failed on VersionedTrace envelope: %v", err)
	}
	if loaded.TransactionHash != tr.TransactionHash {
		t.Errorf("transaction hash mismatch: got %q, want %q", loaded.TransactionHash, tr.TransactionHash)
	}
}

// ── SupportedJSONSchemaVersions ───────────────────────────────────────────────

func TestSupportedJSONSchemaVersions_NotEmpty(t *testing.T) {
	if len(SupportedJSONSchemaVersions) == 0 {
		t.Error("SupportedJSONSchemaVersions must not be empty")
	}
}

func TestSupportedJSONSchemaVersions_AllPassValidation(t *testing.T) {
	for _, v := range SupportedJSONSchemaVersions {
		if err := ValidateJSONSchemaVersion(v); err != nil {
			t.Errorf("SupportedJSONSchemaVersions entry %q fails ValidateJSONSchemaVersion: %v", v, err)
		}
	}
}

func TestIsJSONSchemaVersionSupported_KnownVersions(t *testing.T) {
	if !IsJSONSchemaVersionSupported("1.0") {
		t.Error("version \"1.0\" should be supported")
	}
	if IsJSONSchemaVersionSupported("0.0") {
		t.Error("version \"0.0\" should not be supported")
	}
	if IsJSONSchemaVersionSupported("99.9") {
		t.Error("version \"99.9\" should not be supported")
	}
}

// ── SchemaCompatibilityReport ──────────────────────────────────────────────

func TestCheckSchemaCompatibility_SameVersion(t *testing.T) {
	r := CheckSchemaCompatibility(
		TraceFormatVersion{Major: 1, Minor: 0, Patch: 0},
		TraceFormatVersion{Major: 1, Minor: 0, Patch: 0},
	)
	if !r.Compatible {
		t.Error("expected same version to be compatible")
	}
	if r.RequiresMigration {
		t.Error("expected same version to not require migration")
	}
	if len(r.Warnings) != 0 {
		t.Errorf("expected no warnings for same version, got: %v", r.Warnings)
	}
}

func TestCheckSchemaCompatibility_MajorMismatch(t *testing.T) {
	r := CheckSchemaCompatibility(
		TraceFormatVersion{Major: 2, Minor: 0, Patch: 0},
		TraceFormatVersion{Major: 1, Minor: 0, Patch: 0},
	)
	if r.Compatible {
		t.Error("expected major mismatch to be incompatible")
	}
	if !r.RequiresMigration {
		t.Error("expected major mismatch to require migration")
	}
	if len(r.Actions) == 0 {
		t.Error("expected at least one action for major mismatch")
	}
	if !strings.Contains(r.Actions[0], "v2.x.x") {
		t.Errorf("action should reference major version 2, got: %s", r.Actions[0])
	}
}

func TestCheckSchemaCompatibility_OlderMinorCompatible(t *testing.T) {
	r := CheckSchemaCompatibility(
		TraceFormatVersion{Major: 1, Minor: 0, Patch: 0},
		TraceFormatVersion{Major: 1, Minor: 1, Patch: 0},
	)
	if !r.Compatible {
		t.Error("expected older minor to be compatible")
	}
	if !r.RequiresMigration {
		t.Error("expected older minor to require migration")
	}
	if len(r.Actions) == 0 {
		t.Error("expected at least one action")
	}
}

func TestCheckSchemaCompatibility_NewerMinorIncompatible(t *testing.T) {
	r := CheckSchemaCompatibility(
		TraceFormatVersion{Major: 1, Minor: 2, Patch: 0},
		TraceFormatVersion{Major: 1, Minor: 1, Patch: 0},
	)
	if r.Compatible {
		t.Error("expected newer minor to be incompatible without AllowNewerMinor")
	}
	if !r.RequiresMigration {
		t.Error("expected newer minor to require migration")
	}
	if len(r.Actions) == 0 {
		t.Error("expected at least one action for newer minor")
	}
}

func TestCheckSchemaCompatibility_PatchIgnored(t *testing.T) {
	r := CheckSchemaCompatibility(
		TraceFormatVersion{Major: 1, Minor: 0, Patch: 5},
		TraceFormatVersion{Major: 1, Minor: 0, Patch: 0},
	)
	if !r.Compatible {
		t.Error("patch differences should not affect compatibility")
	}
	if r.RequiresMigration {
		t.Error("patch differences should not require migration")
	}
}
