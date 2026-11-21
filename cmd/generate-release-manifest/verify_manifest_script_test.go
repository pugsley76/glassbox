// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// manifestHasBash reports whether bash is available on PATH.
func manifestHasBash() bool {
	_, err := exec.LookPath("bash")
	return err == nil
}

// manifestHasPython3 reports whether python3 is available on PATH.
func manifestHasPython3() bool {
	_, err := exec.LookPath("python3")
	return err == nil
}

// manifestRepoRoot walks up from this test file to locate the repository root
// (the directory containing go.mod).
func manifestRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}

// TestVerifyManifestScript_EndToEnd generates a real signed manifest with the
// Go tool, then runs scripts/verify-manifest.sh against it and asserts the
// offline verifier accepts it. This guards the byte-level contract between
// manifest.CanonicalJSON (Go: json.MarshalIndent) and the script's re-derivation
// of manifest_hash.
//
// The Ed25519 signature check requires the 'cryptography' Python package; when
// it is absent the script reports [SKIP] for the signature and still exits 0,
// so this test passes with or without it. The integrity checks (structure,
// presence, SHA-256, no-unlisted, and manifest_hash re-derivation) need only
// bash and python3, which is why the skip guard checks just those two.
func TestVerifyManifestScript_EndToEnd(t *testing.T) {
	if !manifestHasBash() || !manifestHasPython3() {
		t.Skip("bash and python3 required for verify-manifest.sh end-to-end test")
	}
	root := manifestRepoRoot(t)
	script := filepath.Join(root, "scripts", "verify-manifest.sh")

	dir := t.TempDir()
	// Release artifacts.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "glassbox-linux-amd64.tar.gz"), []byte("fake archive content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "checksums.sha256"), []byte("aabbcc  glassbox-linux-amd64.tar.gz\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "version.txt"), []byte("version=v1.0.0\ncommit=abc\nbuild_date=2026-01-01T00:00:00Z\n"), 0644))

	keyPath := generateTestKey(t)
	outPath := filepath.Join(dir, "manifest.json")

	require.NoError(t, run([]string{
		"--dist", dir,
		"--version", "v1.0.0",
		"--commit", strings.Repeat("a", 40),
		"--build-date", "2026-01-01T00:00:00Z",
		"--signing-key", keyPath,
		"--output", outPath,
		"--json-only",
		"--signer-identity", "test-pipeline",
	}), "generate-release-manifest must produce a manifest")

	cmd := exec.Command("bash", script, outPath, dir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	assert.Equal(t, 0, exitCode, "verify-manifest.sh must exit 0 against a freshly signed manifest; output:\n%s", out)
	// The critical assertion: the script's re-derived hash must match the
	// Go-produced manifest_hash. If this fails, the script's canonical JSON
	// serialisation has drifted from manifest.CanonicalJSON.
	assert.Contains(t, string(out), "manifest_hash matches re-derived hash",
		"re-derived hash must match the Go-produced manifest_hash; output:\n%s", out)
	assert.Contains(t, string(out), "all manifest verification checks passed",
		"output:\n%s", out)
}

// TestVerifyManifestScript_TamperedHashFails generates a manifest, corrupts its
// manifest_hash, and confirms the script reports the mismatch rather than
// silently passing. This is the most important failure path for the verifier.
func TestVerifyManifestScript_TamperedHashFails(t *testing.T) {
	if !manifestHasBash() || !manifestHasPython3() {
		t.Skip("bash and python3 required for verify-manifest.sh end-to-end test")
	}
	root := manifestRepoRoot(t)
	script := filepath.Join(root, "scripts", "verify-manifest.sh")

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "glassbox-linux-amd64.tar.gz"), []byte("fake archive content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "checksums.sha256"), []byte("aabbcc  glassbox-linux-amd64.tar.gz\n"), 0644))

	keyPath := generateTestKey(t)
	outPath := filepath.Join(dir, "manifest.json")

	require.NoError(t, run([]string{
		"--dist", dir,
		"--version", "v1.0.0",
		"--commit", strings.Repeat("b", 40),
		"--build-date", "2026-01-01T00:00:00Z",
		"--signing-key", keyPath,
		"--output", outPath,
		"--json-only",
	}))

	// Corrupt the stored manifest_hash so the re-derivation cannot match.
	raw, err := os.ReadFile(outPath)
	require.NoError(t, err)
	corrupted := strings.Replace(string(raw),
		`"manifest_hash": "`, `"manifest_hash": "0000000000000000000000000000000000000000000000000000000000000000`, 1)
	require.NotEqual(t, string(raw), corrupted, "manifest_hash field must have been found and replaced")
	require.NoError(t, os.WriteFile(outPath, []byte(corrupted), 0644))

	cmd := exec.Command("bash", script, outPath, dir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	assert.NotEqual(t, 0, exitCode, "a tampered manifest_hash must cause a non-zero exit; output:\n%s", out)
	assert.Contains(t, string(out), "manifest_hash mismatch",
		"the script must report the hash mismatch; output:\n%s", out)
}
