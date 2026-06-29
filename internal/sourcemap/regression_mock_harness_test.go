// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Tests for Issue #309: source mapping validation in the regression and mock harness.
// These tests cover scenarios not addressed in resolver_test.go:
//   - Mock-harness source override integration
//   - Concurrent Resolve calls (regression suite parallelism safety)
//   - ClearCache / InvalidateCache no-ops when cache is absent
//   - ResolveFilePath passthrough without alias resolver
//   - AutoDiscoverLocalSymbols with a populated (but non-matching) build dir

package sourcemap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// ── Concurrent Resolve — parallelism safety (regression harness uses goroutines) ─

// TestResolve_ConcurrentNonInteractive_NoDataRace verifies that concurrent
// Resolve calls in non-interactive mode (as invoked by the regression harness)
// do not race on shared Resolver state.
func TestResolve_ConcurrentNonInteractive_NoDataRace(t *testing.T) {
	srv := notFoundServer(t)
	rc := NewRegistryClient(WithBaseURL(srv.URL))
	r := NewResolver(
		WithRegistryClient(rc),
		WithNonInteractive(),
	)

	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, errs[idx] = r.Resolve(context.Background(), testContractID)
		}(i)
	}
	wg.Wait()

	// Every goroutine must have received a proper error (ErrSourceNotFound),
	// never a nil return that would indicate a silent failure.
	for i, err := range errs {
		if err == nil {
			t.Errorf("worker %d: expected error from exhausted stages, got nil", i)
		}
		if !errors.Is(err, ErrSourceNotFound) {
			t.Errorf("worker %d: expected ErrSourceNotFound, got: %v", i, err)
		}
	}
}

// ── ResolveFilePath — no alias resolver passthrough ───────────────────────────

// TestResolveFilePath_NoAlias_ReturnsInputUnchanged verifies that without an
// alias resolver the path is returned verbatim (no panic, no mutation).
func TestResolveFilePath_NoAlias_ReturnsInputUnchanged(t *testing.T) {
	r := NewResolver()
	input := "src/lib.rs"
	got := r.ResolveFilePath(input)
	if got != input {
		t.Errorf("ResolveFilePath without alias resolver: got %q, want %q", got, input)
	}
}

// TestResolveFilePath_EmptyPath_ReturnsEmpty verifies the edge case of an
// empty path with no alias resolver set.
func TestResolveFilePath_EmptyPath_ReturnsEmpty(t *testing.T) {
	r := NewResolver()
	if got := r.ResolveFilePath(""); got != "" {
		t.Errorf("ResolveFilePath(\"\") = %q, want \"\"", got)
	}
}

// ── InvalidateCache / ClearCache — no-op when cache is nil ───────────────────

// TestInvalidateCache_NilCache_ReturnsNil verifies that InvalidateCache with no
// configured cache is a safe no-op.
func TestInvalidateCache_NilCache_ReturnsNil(t *testing.T) {
	r := NewResolver() // no WithCache → cache is nil
	if err := r.InvalidateCache("CABC3J7GYCCX3S7LX63P6R7EAL477J26C356X6E5A4XERAD7UXD6I7Y3N"); err != nil {
		t.Errorf("InvalidateCache with nil cache: got %v, want nil", err)
	}
}

// TestClearCache_NilCache_ReturnsNil verifies that ClearCache with no
// configured cache is a safe no-op.
func TestClearCache_NilCache_ReturnsNil(t *testing.T) {
	r := NewResolver()
	if err := r.ClearCache(); err != nil {
		t.Errorf("ClearCache with nil cache: got %v, want nil", err)
	}
}

// ── AutoDiscoverLocalSymbols — populated build dir, no hash match ─────────────

// TestAutoDiscoverLocalSymbols_NoMatchInBuildDir_ReturnsNil verifies that when
// a build directory exists and contains WASM files but none match the expected
// hash, the function returns nil (non-fatal — the harness can continue without
// local symbols).
func TestAutoDiscoverLocalSymbols_NoMatchInBuildDir_ReturnsNil(t *testing.T) {
	dir := t.TempDir()
	releaseDir := filepath.Join(dir, "target", "wasm32-unknown-unknown", "release")
	if err := os.MkdirAll(releaseDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a minimal valid WASM magic header.
	wasmData := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(filepath.Join(releaseDir, "my_contract.wasm"), wasmData, 0644); err != nil {
		t.Fatal(err)
	}

	r := NewResolver()
	// Use an arbitrary hash that won't match the empty WASM above.
	err := r.AutoDiscoverLocalSymbols(dir, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Errorf("no-match case must be non-fatal; got: %v", err)
	}
}

// ── Mock-harness override: ValidDir accepted, SourceCode populated ────────────

// TestMockHarnessOverride_ValidDir_SourceCodePopulated verifies that the
// mock harness pattern (non-interactive + --contract-source override) correctly
// populates a SourceCode record for the regression suite's per-transaction
// source resolution step.
func TestMockHarnessOverride_ValidDir_SourceCodePopulated(t *testing.T) {
	srcDir := t.TempDir()
	srv := notFoundServer(t)
	rc := NewRegistryClient(WithBaseURL(srv.URL))
	r := NewResolver(
		WithRegistryClient(rc),
		WithNonInteractive(),
		WithContractSource(srcDir),
	)

	src, err := r.Resolve(context.Background(), testContractID)
	if err != nil {
		t.Fatalf("override dir should succeed; got: %v", err)
	}
	if src == nil {
		t.Fatal("expected non-nil SourceCode")
	}
	if src.ContractID != testContractID {
		t.Errorf("ContractID = %q, want %q", src.ContractID, testContractID)
	}
	if src.Repository != srcDir {
		t.Errorf("Repository = %q, want %q", src.Repository, srcDir)
	}
}

// TestMockHarnessOverride_MissingDir_ErrorNamesFlag verifies that a missing
// override path surfaces an actionable message that names --contract-source,
// matching the regression harness's expected error format.
func TestMockHarnessOverride_MissingDir_ErrorNamesFlag(t *testing.T) {
	srv := notFoundServer(t)
	rc := NewRegistryClient(WithBaseURL(srv.URL))
	r := NewResolver(
		WithRegistryClient(rc),
		WithNonInteractive(),
		WithContractSource("/does/not/exist"),
	)

	_, err := r.Resolve(context.Background(), testContractID)
	if err == nil {
		t.Fatal("expected error for missing override dir")
	}
	if err.Error() == "" {
		t.Fatal("error message must not be empty")
	}
	// Must name the flag for actionable output.
	for _, want := range []string{"--contract-source", "not found"} {
		if !contains(err.Error(), want) {
			t.Errorf("error should contain %q, got: %q", want, err.Error())
		}
	}
}

// contains is a local helper to avoid importing strings in this file.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
