// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/signer"
	"github.com/spf13/cobra"
)

// clearPKCS11Env clears every PKCS#11 / provider environment variable so tests
// are not influenced by the host environment.
func clearPKCS11Env(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"GLASSBOX_PKCS11_MODULE", "GLASSBOX_PKCS11_PIN", "GLASSBOX_PKCS11_TOKEN_LABEL",
		"GLASSBOX_PKCS11_KEY_LABEL", "GLASSBOX_PKCS11_KEY_ID", "GLASSBOX_PKCS11_SLOT",
		"GLASSBOX_SIGNING_PROVIDER", "GLASSBOX_SIGNER_TYPE",
	} {
		t.Setenv(k, "")
	}
}

func TestValidatePKCS11SignInputs(t *testing.T) {
	tests := []struct {
		name       string
		cfg        signer.Pkcs11Config
		wantErr    bool
		wantSubstr string
	}{
		{
			name:    "module and pin present",
			cfg:     signer.Pkcs11Config{ModulePath: "/lib/softhsm.so", PIN: "1234"},
			wantErr: false,
		},
		{
			name:       "missing module",
			cfg:        signer.Pkcs11Config{PIN: "1234"},
			wantErr:    true,
			wantSubstr: "pkcs11-module",
		},
		{
			name:       "missing pin",
			cfg:        signer.Pkcs11Config{ModulePath: "/lib/softhsm.so"},
			wantErr:    true,
			wantSubstr: "pkcs11-pin",
		},
		{
			name:       "missing both",
			cfg:        signer.Pkcs11Config{},
			wantErr:    true,
			wantSubstr: "pkcs11-module",
		},
		{
			name:    "valid hex key id",
			cfg:     signer.Pkcs11Config{ModulePath: "/lib/softhsm.so", PIN: "1234", KeyIDHex: "a1b2c3"},
			wantErr: false,
		},
		{
			name:       "invalid hex key id",
			cfg:        signer.Pkcs11Config{ModulePath: "/lib/softhsm.so", PIN: "1234", KeyIDHex: "zzzz"},
			wantErr:    true,
			wantSubstr: "pkcs11-key-id",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := validatePKCS11SignInputs(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.wantSubstr != "" && !strings.Contains(err.Error(), tt.wantSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEffectivePKCS11Config_FlagOverridesEnv(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	t.Setenv("GLASSBOX_PKCS11_MODULE", "/env/module.so")
	t.Setenv("GLASSBOX_PKCS11_PIN", "envpin")
	t.Setenv("GLASSBOX_PKCS11_SLOT", "3")

	// Flag set for module only; PIN should fall back to env.
	auditSignPKCS11Module = "/flag/module.so"
	defer resetAuditSignFlags()

	cfg, err := effectivePKCS11Config()
	if err != nil {
		t.Fatalf("effectivePKCS11Config returned error: %v", err)
	}
	if cfg.ModulePath != "/flag/module.so" {
		t.Errorf("flag should override env: got ModulePath=%q", cfg.ModulePath)
	}
	if cfg.PIN != "envpin" {
		t.Errorf("PIN should fall back to env: got %q", cfg.PIN)
	}
	if cfg.SlotIndex != 3 {
		t.Errorf("SlotIndex from env not parsed: got %d", cfg.SlotIndex)
	}
}

func TestEffectivePKCS11Config_InvalidSlotRejected(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	t.Setenv("GLASSBOX_PKCS11_SLOT", "not-a-number")
	defer resetAuditSignFlags()

	_, err := effectivePKCS11Config()
	if err == nil {
		t.Fatal("expected invalid GLASSBOX_PKCS11_SLOT to be rejected")
	}
	if !strings.Contains(err.Error(), "GLASSBOX_PKCS11_SLOT") {
		t.Errorf("error should mention GLASSBOX_PKCS11_SLOT, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "non-negative integer") {
		t.Errorf("error should explain the required slot format, got %q", err.Error())
	}
}

func TestResolveProviderAndConfig_UsesEnvSlotIndex(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignProvider = "pkcs11"
	t.Setenv("GLASSBOX_PKCS11_SLOT", "4")
	defer resetAuditSignFlags()

	_, cfg := resolveProviderAndConfig()
	if cfg.PKCS11SlotIndex != 4 {
		t.Fatalf("expected resolveProviderAndConfig to carry PKCS11SlotIndex=4, got %d", cfg.PKCS11SlotIndex)
	}
}

func TestRunPkcs11Preflight_RejectsNonPkcs11Provider(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignValidateOnly = true
	defer func() {
		resetAuditSignFlags()
		auditSignValidateOnly = false
	}()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	// Default provider is software — preflight must refuse with a clear message.
	err := runPkcs11Preflight(cmd)
	if err == nil {
		t.Fatal("expected error for non-pkcs11 provider")
	}
	if !strings.Contains(err.Error(), "pkcs11 provider") {
		t.Errorf("error %q should mention the pkcs11 provider", err.Error())
	}
}

func TestRunPkcs11Preflight_AcceptsSigningProviderFlag(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignValidateOnly = true
	auditSignProvider = "pkcs11" // the recommended flag, not the deprecated --hsm-provider
	defer func() {
		resetAuditSignFlags()
		auditSignValidateOnly = false
	}()

	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	// With no module configured the preflight should fail on a check (module_path),
	// NOT be rejected for the wrong provider. So the error must not mention provider
	// selection.
	err := runPkcs11Preflight(cmd)
	if err != nil && strings.Contains(err.Error(), "applies only to the pkcs11 provider") {
		t.Errorf("--signing-provider pkcs11 should be accepted by preflight, got: %v", err)
	}
}

func TestAuditSignPreRunE_PKCS11MissingModuleRejected(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignProvider = "pkcs11"
	auditSignValidateOnly = false
	defer resetAuditSignFlags()

	err := auditSignPreRunE(auditSignCmd, nil)
	if err == nil {
		t.Fatal("expected PreRunE to reject pkcs11 signing without a module/pin")
	}
	if !strings.Contains(err.Error(), "pkcs11") {
		t.Errorf("error %q should explain the missing pkcs11 inputs", err.Error())
	}
}

func TestAuditSignPreRunE_ValidateOnlySkipsRequiredCheck(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignProvider = "pkcs11"
	auditSignValidateOnly = true // preflight reports missing inputs itself
	defer func() {
		resetAuditSignFlags()
		auditSignValidateOnly = false
	}()

	if err := auditSignPreRunE(auditSignCmd, nil); err != nil {
		t.Fatalf("--validate-only should skip the hard required-input check, got: %v", err)
	}
}

func TestAuditSignPreRunE_ValidateOnlyStillRejectsInvalidSlotEnv(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignProvider = "pkcs11"
	auditSignValidateOnly = true
	t.Setenv("GLASSBOX_PKCS11_SLOT", "-1")
	defer func() {
		resetAuditSignFlags()
		auditSignValidateOnly = false
	}()

	err := auditSignPreRunE(auditSignCmd, nil)
	if err == nil {
		t.Fatal("expected invalid GLASSBOX_PKCS11_SLOT to be rejected during PreRunE")
	}
	if !strings.Contains(err.Error(), "GLASSBOX_PKCS11_SLOT") {
		t.Errorf("error should mention GLASSBOX_PKCS11_SLOT, got %q", err.Error())
	}
}

func TestAuditSignPreRunE_SoftwareProviderOK(t *testing.T) {
	resetAuditSignFlags()
	clearPKCS11Env(t)
	auditSignPayload = `{"input":{}}`
	defer resetAuditSignFlags()

	if err := auditSignPreRunE(auditSignCmd, nil); err != nil {
		t.Fatalf("software provider PreRunE should pass, got: %v", err)
	}
}
