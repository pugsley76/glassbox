// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/signer"
)

// ── validatePKCS11SignInputs: module file existence check ─────────────────────

func TestValidatePKCS11SignInputs_ModuleNotFound_Rejected(t *testing.T) {
	cfg := signer.Pkcs11Config{
		ModulePath: "/nonexistent/libpkcs11.so",
		PIN:        "1234",
		KeyLabel:   "signing-key",
	}
	err := validatePKCS11SignInputs(cfg)
	if err == nil {
		t.Fatal("expected error for non-existent module file")
	}
	msg := err.Error()
	if !strings.Contains(msg, "--pkcs11-module") {
		t.Errorf("error should mention --pkcs11-module, got: %q", msg)
	}
	if !strings.Contains(msg, "not found") {
		t.Errorf("error should say 'not found', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include Fix hint, got: %q", msg)
	}
}

func TestValidatePKCS11SignInputs_ModuleIsDirectory_Rejected(t *testing.T) {
	dir := t.TempDir()
	cfg := signer.Pkcs11Config{
		ModulePath: dir,
		PIN:        "1234",
		KeyLabel:   "signing-key",
	}
	err := validatePKCS11SignInputs(cfg)
	if err == nil {
		t.Fatal("expected error when module path is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "Fix:") {
		t.Errorf("error should include Fix hint, got: %q", err.Error())
	}
}

func TestValidatePKCS11SignInputs_NoKeySelector_Rejected(t *testing.T) {
	dir := t.TempDir()
	modFile := filepath.Join(dir, "libfake.so")
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := signer.Pkcs11Config{
		ModulePath: modFile,
		PIN:        "1234",
		// KeyLabel and KeyIDHex both empty
	}
	err := validatePKCS11SignInputs(cfg)
	if err == nil {
		t.Fatal("expected error when no key selector is provided")
	}
	msg := err.Error()
	if !strings.Contains(msg, "key selector") {
		t.Errorf("error should mention 'key selector', got: %q", msg)
	}
	// Should mention both options.
	if !strings.Contains(msg, "--pkcs11-key-label") || !strings.Contains(msg, "--pkcs11-key-id") {
		t.Errorf("error should mention both --pkcs11-key-label and --pkcs11-key-id, got: %q", msg)
	}
	if !strings.Contains(msg, "Tip:") {
		t.Errorf("error should include a Tip, got: %q", msg)
	}
}

func TestValidatePKCS11SignInputs_ValidModuleWithKeyLabel_Passes(t *testing.T) {
	dir := t.TempDir()
	modFile := filepath.Join(dir, "libfake.so")
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := signer.Pkcs11Config{
		ModulePath: modFile,
		PIN:        "1234",
		KeyLabel:   "signing-key",
	}
	if err := validatePKCS11SignInputs(cfg); err != nil {
		t.Errorf("expected no error for valid config with key label, got: %v", err)
	}
}

func TestValidatePKCS11SignInputs_ValidModuleWithKeyIDHex_Passes(t *testing.T) {
	dir := t.TempDir()
	modFile := filepath.Join(dir, "libfake.so")
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := signer.Pkcs11Config{
		ModulePath: modFile,
		PIN:        "1234",
		KeyIDHex:   "a1b2c3",
	}
	if err := validatePKCS11SignInputs(cfg); err != nil {
		t.Errorf("expected no error for valid config with key ID hex, got: %v", err)
	}
}

// ── validateAuditSignArgs: dynamic registry list ──────────────────────────────

func TestValidateAuditSignArgs_ProviderListDynamic(t *testing.T) {
	// The registered providers should be accepted; an unregistered name should
	// be rejected. Importantly, the accepted list is now derived from the
	// registry rather than a hardcoded literal.
	for _, name := range signer.DefaultRegistry.Names() {
		if err := validateAuditSignArgs("", "", name); err != nil {
			t.Errorf("registered provider %q should be accepted, got: %v", name, err)
		}
	}
}

func TestValidateAuditSignArgs_UnregisteredProvider_Rejected(t *testing.T) {
	err := validateAuditSignArgs("", "", "aws-kms-not-registered")
	if err == nil {
		t.Fatal("expected error for unregistered provider")
	}
	msg := err.Error()
	if !strings.Contains(msg, "aws-kms-not-registered") {
		t.Errorf("error should echo the invalid provider name, got: %q", msg)
	}
	// Error must list the valid options (from the registry).
	for _, name := range signer.DefaultRegistry.Names() {
		if !strings.Contains(msg, name) {
			t.Errorf("error should list registered provider %q, got: %q", name, msg)
		}
	}
}

func TestValidateAuditSignArgs_NewlyRegisteredProvider_Accepted(t *testing.T) {
	// Register a temporary provider and verify it becomes valid immediately.
	tempProvider := &mockProvider{name: "temp-hsm", s: nil}
	signer.DefaultRegistry.RegisterOrReplace(tempProvider)
	t.Cleanup(func() {
		// Re-register the originals to restore state.
		signer.DefaultRegistry.RegisterOrReplace(&signer.SoftwareProvider{})
		signer.DefaultRegistry.RegisterOrReplace(&signer.PKCS11Provider{})
	})

	if err := validateAuditSignArgs("", "", "temp-hsm"); err != nil {
		t.Errorf("newly registered provider 'temp-hsm' should be accepted, got: %v", err)
	}
}

// ── PreRunE: module existence wired end-to-end ────────────────────────────────

func TestAuditSignPreRunE_PKCS11ModuleNotFound_Rejected(t *testing.T) {
	defer resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignProvider = "pkcs11"
	auditSignPKCS11Module = "/nonexistent/libpkcs11.so"
	auditSignPKCS11PIN = "1234"
	auditSignPKCS11KeyLabel = "signing-key"

	err := auditSignCmd.PreRunE(auditSignCmd, nil)
	if err == nil {
		t.Fatal("expected error for non-existent PKCS#11 module in PreRunE")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "--pkcs11-module") {
		t.Errorf("error should mention --pkcs11-module, got: %q", err.Error())
	}
}

func TestAuditSignPreRunE_PKCS11ModuleIsDirectory_Rejected(t *testing.T) {
	defer resetAuditSignFlags()
	clearPKCS11Env(t)
	dir := t.TempDir()
	auditSignProvider = "pkcs11"
	auditSignPKCS11Module = dir
	auditSignPKCS11PIN = "1234"
	auditSignPKCS11KeyLabel = "signing-key"

	err := auditSignCmd.PreRunE(auditSignCmd, nil)
	if err == nil {
		t.Fatal("expected error when PKCS#11 module path is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error should mention 'directory', got: %q", err.Error())
	}
}

func TestAuditSignPreRunE_PKCS11NoKeySelector_Rejected(t *testing.T) {
	defer resetAuditSignFlags()
	clearPKCS11Env(t)
	dir := t.TempDir()
	modFile := filepath.Join(dir, "libfake.so")
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	auditSignProvider = "pkcs11"
	auditSignPKCS11Module = modFile
	auditSignPKCS11PIN = "1234"
	// No KeyLabel or KeyIDHex

	err := auditSignCmd.PreRunE(auditSignCmd, nil)
	if err == nil {
		t.Fatal("expected error for missing key selector")
	}
	if !strings.Contains(err.Error(), "key selector") {
		t.Errorf("error should mention 'key selector', got: %q", err.Error())
	}
}

func TestAuditSignPreRunE_PKCS11ValidConfig_Passes(t *testing.T) {
	defer resetAuditSignFlags()
	clearPKCS11Env(t)
	dir := t.TempDir()
	modFile := filepath.Join(dir, "libfake.so")
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	auditSignProvider = "pkcs11"
	auditSignPKCS11Module = modFile
	auditSignPKCS11PIN = "1234"
	auditSignPKCS11KeyLabel = "signing-key"

	// PreRunE should pass (actual PKCS#11 ops run at signing time, not here).
	if err := auditSignCmd.PreRunE(auditSignCmd, nil); err != nil {
		t.Errorf("valid PKCS#11 config should pass PreRunE, got: %v", err)
	}
}
