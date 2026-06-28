// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStore_SaveLoad_RoundTrip(t *testing.T) {
	overrideTempHome(t)
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	original := &Data{
		ID:              "roundtrip-1",
		Name:            "payroll-bug",
		CreatedAt:       time.Now().Add(-time.Hour),
		LastAccessAt:    time.Now().Add(-time.Minute),
		Status:          "saved",
		Network:         "testnet",
		HorizonURL:      "https://horizon-testnet.stellar.org",
		TxHash:          "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		EnvelopeXdr:     "envelope-xdr",
		ResultXdr:       "result-xdr",
		ResultMetaXdr:   "meta-xdr",
		PinnedEndpoint:  "https://rpc.testnet.example",
		SimRequestJSON:  `{"envelope_xdr":"abc"}`,
		SimResponseJSON: `{"status":"ok"}`,
		ErstVersion:     "test-version",
		SchemaVersion:   SchemaVersion,
	}

	ctx := context.Background()
	if err := store.Save(ctx, original); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, original.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != original.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, original.Name)
	}
	if loaded.PinnedEndpoint != original.PinnedEndpoint {
		t.Errorf("PinnedEndpoint = %q, want %q", loaded.PinnedEndpoint, original.PinnedEndpoint)
	}
	if loaded.EnvFingerprint == "" {
		t.Error("EnvFingerprint should be populated on save")
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", loaded.SchemaVersion, SchemaVersion)
	}
}

func TestStore_Load_UpgradesOlderSchemaVersion(t *testing.T) {
	if SchemaVersion <= MinSupportedSchemaVersion {
		t.Skip("no upgradable version below current")
	}

	overrideTempHome(t)
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	d := makeValidSessionData(t, 0)
	d.ID = "upgrade-me"
	d.SchemaVersion = SchemaVersion - 1
	d.EnvFingerprint = ""

	if err := store.SavePreservingSchemaVersion(ctx, d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := store.Load(ctx, d.ID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d after upgrade", loaded.SchemaVersion, SchemaVersion)
	}
	if loaded.EnvFingerprint == "" {
		t.Error("EnvFingerprint should be populated after upgrade")
	}
}

func TestStore_Load_UnsupportedSchemaVersion_ReturnsSchemaError(t *testing.T) {
	overrideTempHome(t)
	store, err := NewStore()
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	d := makeValidSessionData(t, 0)
	d.ID = "too-old"
	d.SchemaVersion = 0

	if err := store.SavePreservingSchemaVersion(ctx, d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	_, err = store.Load(ctx, d.ID)
	if err == nil {
		t.Fatal("expected error loading unsupported schema version")
	}
	if !IsSchemaError(err) {
		t.Fatalf("expected *SchemaError, got: %T (%v)", err, err)
	}
	if !strings.Contains(err.Error(), "too old") {
		t.Errorf("error should mention too old schema, got: %v", err)
	}
}
