// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for session schema stability and upgrade diagnostics.

package session

import (
	"strings"
	"testing"
)

func TestClassifySchemaVersion_CurrentVersion_IsNotNeedsUpgrade(t *testing.T) {
	r := classifySchemaVersion(SchemaVersion)
	if r.NeedsUpgrade {
		t.Error("current version should not require upgrade")
	}
	if r.Unsupported {
		t.Error("current version should not be unsupported")
	}
	if r.FromFuture {
		t.Error("current version should not be from future")
	}
}

func TestClassifySchemaVersion_OldUnsupportedVersion_MarksUnsupported(t *testing.T) {
	r := classifySchemaVersion(0)
	if !r.Unsupported {
		t.Error("version 0 should be marked unsupported")
	}
	if !strings.Contains(r.Message, "re-run") {
		t.Errorf("message should suggest re-running debug, got: %s", r.Message)
	}
}

func TestClassifySchemaVersion_FutureVersion_MarksFromFuture(t *testing.T) {
	r := classifySchemaVersion(SchemaVersion + 100)
	if !r.FromFuture {
		t.Error("future version should be marked from future")
	}
	if !r.Unsupported {
		t.Error("future version should also be marked unsupported")
	}
	if !strings.Contains(r.Message, "upgrade Glassbox") {
		t.Errorf("message should suggest upgrading Glassbox, got: %s", r.Message)
	}
}

func TestValidateSchemaVersion_CurrentVersion_ReturnsNil(t *testing.T) {
	if err := ValidateSchemaVersion(SchemaVersion, "sess-1"); err != nil {
		t.Fatalf("expected nil for current version, got: %v", err)
	}
}

func TestValidateSchemaVersion_OutdatedVersion_ReturnsNil(t *testing.T) {
	if SchemaVersion <= MinSupportedSchemaVersion {
		t.Skip("no upgradable version below current")
	}
	if err := ValidateSchemaVersion(SchemaVersion-1, "sess-1"); err != nil {
		t.Fatalf("outdated but supported version should be auto-upgraded, got: %v", err)
	}
}

func TestValidateSchemaVersion_FutureVersion_ReturnsSchemaError(t *testing.T) {
	err := ValidateSchemaVersion(SchemaVersion+1, "future-sess")
	if err == nil {
		t.Fatal("expected error for future schema version")
	}
	se := AsSchemaError(err)
	if se == nil {
		t.Fatalf("expected *SchemaError, got: %T", err)
	}
	if !se.Result.FromFuture {
		t.Error("SchemaError should indicate FromFuture")
	}
}

func TestValidateSchemaVersion_TooOld_ReturnsSchemaError(t *testing.T) {
	err := ValidateSchemaVersion(0, "old-sess")
	if err == nil {
		t.Fatal("expected error for version 0")
	}
	if !IsSchemaError(err) {
		t.Fatalf("expected *SchemaError, got: %T", err)
	}
}

func TestUpgradeSessionData_UpgradesOlderVersion(t *testing.T) {
	if SchemaVersion <= MinSupportedSchemaVersion {
		t.Skip("no upgradable version below current")
	}
	d := validData()
	d.SchemaVersion = SchemaVersion - 1
	d.EnvFingerprint = ""

	upgraded, err := UpgradeSessionData(d)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !upgraded {
		t.Error("expected upgraded=true")
	}
	if d.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", d.SchemaVersion, SchemaVersion)
	}
}

func TestUpgradeSessionData_FutureVersion_ReturnsError(t *testing.T) {
	d := validData()
	d.SchemaVersion = SchemaVersion + 99
	_, err := UpgradeSessionData(d)
	if err == nil {
		t.Fatal("expected error upgrading future-version session")
	}
	if !IsSchemaError(err) {
		t.Fatalf("expected *SchemaError, got: %T", err)
	}
}

func TestSchemaVersionSummary_IncludesVersion(t *testing.T) {
	s := SchemaVersionSummary(SchemaVersion)
	if !strings.Contains(s, "current") {
		t.Errorf("summary for current version should mention 'current', got: %q", s)
	}
}
