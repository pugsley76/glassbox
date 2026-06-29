// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package signer

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"strings"
	"testing"
)

// encodeHex encodes bytes to a lowercase hex string.
func encodeHex(b []byte) string { return hex.EncodeToString(b) }

// ── SoftwareProvider.Validate — early content checks ──────────────────────────

func TestSoftwareProvider_Validate_OpenSSHFormat_Rejected(t *testing.T) {
	p := &SoftwareProvider{}
	// Minimal fake OpenSSH-format header (content doesn't matter for header detection).
	fakeOpenSSH := "-----BEGIN OPENSSH PRIVATE KEY-----\nfakebase64\n-----END OPENSSH PRIVATE KEY-----\n"
	err := p.Validate(ProviderConfig{SoftwareKeyPEM: fakeOpenSSH})
	if err == nil {
		t.Fatal("expected error for OpenSSH-format key")
	}
	msg := err.Error()
	if !strings.Contains(msg, "OpenSSH") {
		t.Errorf("error should mention 'OpenSSH', got: %q", msg)
	}
	if !strings.Contains(msg, "PKCS#8") {
		t.Errorf("error should mention 'PKCS#8' as the expected format, got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

func TestSoftwareProvider_Validate_ECPrivateKeyFormat_Rejected(t *testing.T) {
	p := &SoftwareProvider{}
	fakeEC := "-----BEGIN EC PRIVATE KEY-----\nfakebase64\n-----END EC PRIVATE KEY-----\n"
	err := p.Validate(ProviderConfig{SoftwareKeyPEM: fakeEC})
	if err == nil {
		t.Fatal("expected error for SEC1 EC PRIVATE KEY format")
	}
	msg := err.Error()
	if !strings.Contains(msg, "SEC1") || !strings.Contains(msg, "PKCS#8") {
		t.Errorf("error should mention SEC1 and PKCS#8, got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

func TestSoftwareProvider_Validate_RSAKeyFormat_Rejected(t *testing.T) {
	p := &SoftwareProvider{}
	fakeRSA := "-----BEGIN RSA PRIVATE KEY-----\nfakebase64\n-----END RSA PRIVATE KEY-----\n"
	err := p.Validate(ProviderConfig{SoftwareKeyPEM: fakeRSA})
	if err == nil {
		t.Fatal("expected error for RSA key")
	}
	msg := err.Error()
	if !strings.Contains(msg, "RSA") {
		t.Errorf("error should mention RSA, got: %q", msg)
	}
	if !strings.Contains(msg, "Ed25519") {
		t.Errorf("error should mention Ed25519, got: %q", msg)
	}
}

func TestSoftwareProvider_Validate_GarbledPEM_Rejected(t *testing.T) {
	p := &SoftwareProvider{}
	// Structurally looks like PRIVATE KEY but has invalid content.
	garbled := "-----BEGIN PRIVATE KEY-----\nnot-valid-base64!!!\n-----END PRIVATE KEY-----\n"
	err := p.Validate(ProviderConfig{SoftwareKeyPEM: garbled})
	if err == nil {
		t.Fatal("expected error for garbled PKCS#8 PEM")
	}
	if !strings.Contains(err.Error(), "invalid Ed25519 private key") {
		t.Errorf("error should mention 'invalid Ed25519 private key', got: %q", err.Error())
	}
}

func TestSoftwareProvider_Validate_ValidPKCS8PEM_Accepted(t *testing.T) {
	p := &SoftwareProvider{}
	pemText := newTestPEM(t)
	if err := p.Validate(ProviderConfig{SoftwareKeyPEM: pemText}); err != nil {
		t.Errorf("valid PKCS#8 PEM should be accepted, got: %v", err)
	}
}

// ── SoftwareProvider.Validate — hex key validation ────────────────────────────

func TestSoftwareProvider_Validate_HexKeyInvalidChars_Rejected(t *testing.T) {
	p := &SoftwareProvider{}
	t.Setenv("GLASSBOX_AUDIT_PRIVATE_KEY_PEM", "")
	err := p.Validate(ProviderConfig{SoftwareKeyHex: "not-valid-hex-!!!"})
	if err == nil {
		t.Fatal("expected error for non-hex SoftwareKeyHex")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not valid hexadecimal") {
		t.Errorf("error should mention 'not valid hexadecimal', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

func TestSoftwareProvider_Validate_HexKeyWrongLength_Rejected(t *testing.T) {
	p := &SoftwareProvider{}
	t.Setenv("GLASSBOX_AUDIT_PRIVATE_KEY_PEM", "")
	// 10 bytes = 20 hex chars — not a valid Ed25519 key length.
	shortHex := "aabbccddeeff00112233"
	err := p.Validate(ProviderConfig{SoftwareKeyHex: shortHex})
	if err == nil {
		t.Fatal("expected error for wrong-length hex key")
	}
	msg := err.Error()
	if !strings.Contains(msg, "wrong key length") {
		t.Errorf("error should mention 'wrong key length', got: %q", msg)
	}
	// Should mention expected lengths.
	if !strings.Contains(msg, "32") || !strings.Contains(msg, "64") {
		t.Errorf("error should mention expected byte counts 32 and 64, got: %q", msg)
	}
}

func TestSoftwareProvider_Validate_HexSeedLength_Accepted(t *testing.T) {
	p := &SoftwareProvider{}
	t.Setenv("GLASSBOX_AUDIT_PRIVATE_KEY_PEM", "")
	_, priv, _ := ed25519.GenerateKey(nil)
	// Encode the 32-byte seed as 64 hex characters.
	seedHex := encodeHex(priv.Seed())
	if err := p.Validate(ProviderConfig{SoftwareKeyHex: seedHex}); err != nil {
		t.Errorf("32-byte hex seed should be accepted, got: %v", err)
	}
}

// ── PKCS11Provider.Validate — new checks ─────────────────────────────────────

func TestPKCS11Provider_Validate_ModuleIsDirectory_Rejected(t *testing.T) {
	p := &PKCS11Provider{}
	dir := t.TempDir() // a real directory, not a .so file
	err := p.Validate(ProviderConfig{
		PKCS11ModulePath: dir,
		PKCS11PIN:        "1234",
	})
	if err == nil {
		t.Fatal("expected error when module path is a directory")
	}
	msg := err.Error()
	if !strings.Contains(msg, "directory") {
		t.Errorf("error should mention 'directory', got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

func TestPKCS11Provider_Validate_KeyIDHexInvalid_Rejected(t *testing.T) {
	p := &PKCS11Provider{}
	// Write a real file so the module-exists check passes.
	dir := t.TempDir()
	modFile := dir + "/libfake.so"
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	err := p.Validate(ProviderConfig{
		PKCS11ModulePath: modFile,
		PKCS11PIN:        "1234",
		PKCS11KeyIDHex:   "not-hex-!!!",
	})
	if err == nil {
		t.Fatal("expected error for non-hex KeyIDHex")
	}
	msg := err.Error()
	if !strings.Contains(msg, "pkcs11-key-id") && !strings.Contains(msg, "GLASSBOX_PKCS11_KEY_ID") {
		t.Errorf("error should mention the key-id flag/env, got: %q", msg)
	}
	if !strings.Contains(msg, "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", msg)
	}
}

func TestPKCS11Provider_Validate_KeyIDHexValid_Accepted(t *testing.T) {
	p := &PKCS11Provider{}
	dir := t.TempDir()
	modFile := dir + "/libfake.so"
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := p.Validate(ProviderConfig{
		PKCS11ModulePath: modFile,
		PKCS11PIN:        "1234",
		PKCS11KeyIDHex:   "a1b2c3",
	}); err != nil {
		t.Errorf("valid hex KeyIDHex should be accepted, got: %v", err)
	}
}

func TestPKCS11Provider_Validate_NegativeSlotIndex_Rejected(t *testing.T) {
	p := &PKCS11Provider{}
	dir := t.TempDir()
	modFile := dir + "/libfake.so"
	if err := os.WriteFile(modFile, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	err := p.Validate(ProviderConfig{
		PKCS11ModulePath: modFile,
		PKCS11PIN:        "1234",
		PKCS11SlotIndex:  -1,
	})
	if err == nil {
		t.Fatal("expected error for negative slot index")
	}
	msg := err.Error()
	if !strings.Contains(msg, "slot index") {
		t.Errorf("error should mention 'slot index', got: %q", msg)
	}
	if !strings.Contains(msg, "-1") {
		t.Errorf("error should echo the invalid value, got: %q", msg)
	}
}

func TestPKCS11Provider_Validate_ModulePathWithFix_Rejected(t *testing.T) {
	p := &PKCS11Provider{}
	err := p.Validate(ProviderConfig{
		PKCS11ModulePath: "/nonexistent/libpkcs11.so",
		PKCS11PIN:        "1234",
	})
	if err == nil {
		t.Fatal("expected error for non-existent module")
	}
	if !strings.Contains(err.Error(), "Fix:") {
		t.Errorf("error should include a Fix hint, got: %q", err.Error())
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// pkcs8PEM generates a PKCS#8 PEM from an Ed25519 key for use in tests.
func pkcs8PEM(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
}
