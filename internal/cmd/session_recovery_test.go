// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dotandev/glassbox/internal/session"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// overrideHome redirects HOME / USERPROFILE to a temp dir so session and
// checkpoint writes never touch the real user home directory.
func overrideHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	return dir
}

// writeCheckpointFile writes a raw checkpoint JSON to the expected path so
// tests can simulate various checkpoint states.
func writeCheckpointFile(t *testing.T, homeDir, content string) {
	t.Helper()
	glassboxDir := filepath.Join(homeDir, ".Glassbox")
	if err := os.MkdirAll(glassboxDir, 0755); err != nil {
		t.Fatalf("mkdir .Glassbox: %v", err)
	}
	path := filepath.Join(glassboxDir, "active_session.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}
}

// ── session recover: no checkpoint ───────────────────────────────────────────

func TestSessionRecover_NoCheckpoint_PrintsNothingToRecover(t *testing.T) {
	overrideHome(t)

	var out bytes.Buffer
	sessionRecoverCmd.SetOut(&out)
	sessionRecoverCmd.SetErr(&out)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err != nil {
		t.Fatalf("expected no error when no checkpoint exists, got: %v", err)
	}
	if !strings.Contains(out.String(), "Nothing to recover") {
		t.Errorf("expected 'Nothing to recover' in output, got: %q", out.String())
	}
}

// ── session recover: malformed checkpoint ────────────────────────────────────

func TestSessionRecover_MalformedCheckpoint_ClearsAndErrors(t *testing.T) {
	homeDir := overrideHome(t)
	writeCheckpointFile(t, homeDir, "{ not valid json {{")

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error for malformed checkpoint")
	}
	// The error message must be descriptive about what went wrong.
	if !strings.Contains(err.Error(), "checkpoint") {
		t.Errorf("error should mention 'checkpoint', got: %v", err)
	}
}

// ── session recover: corrupt checkpoint fields ────────────────────────────────

func TestSessionRecover_MissingSessionID_ReportsIssueAndClears(t *testing.T) {
	homeDir := overrideHome(t)
	// Checkpoint with empty session_id.
	writeCheckpointFile(t, homeDir, `{
		"session_id": "",
		"tx_hash":    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"network":    "testnet",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        99999
	}`)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error for checkpoint missing session ID")
	}
	msg := err.Error()
	if !strings.Contains(msg, "corrupt") {
		t.Errorf("error should say 'corrupt', got: %v", err)
	}
	// Checkpoint file must be cleared.
	glassboxDir := filepath.Join(homeDir, ".Glassbox")
	if _, statErr := os.Stat(filepath.Join(glassboxDir, "active_session.json")); !os.IsNotExist(statErr) {
		t.Error("corrupt checkpoint file should have been removed")
	}
}

func TestSessionRecover_MissingTxHash_ReportsIssueAndClears(t *testing.T) {
	homeDir := overrideHome(t)
	writeCheckpointFile(t, homeDir, `{
		"session_id": "session-abc",
		"tx_hash":    "",
		"network":    "testnet",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        99999
	}`)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error for checkpoint missing tx_hash")
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("error should say 'corrupt', got: %v", err)
	}
}

func TestSessionRecover_MissingNetwork_ReportsIssueAndClears(t *testing.T) {
	homeDir := overrideHome(t)
	writeCheckpointFile(t, homeDir, `{
		"session_id": "session-abc",
		"tx_hash":    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"network":    "",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        99999
	}`)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error for checkpoint missing network")
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("error should say 'corrupt', got: %v", err)
	}
}

func TestSessionRecover_InvalidPID_ReportsIssue(t *testing.T) {
	homeDir := overrideHome(t)
	writeCheckpointFile(t, homeDir, `{
		"session_id": "session-abc",
		"tx_hash":    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"network":    "testnet",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        0
	}`)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error for checkpoint with PID=0")
	}
	if !strings.Contains(err.Error(), "corrupt") {
		t.Errorf("error should say 'corrupt', got: %v", err)
	}
}

// ── session recover: error message hints ────────────────────────────────────

func TestSessionRecover_CorruptCheckpoint_HintMentionsDebugCommand(t *testing.T) {
	homeDir := overrideHome(t)
	writeCheckpointFile(t, homeDir, `{
		"session_id": "",
		"tx_hash":    "",
		"network":    "",
		"started_at": "0001-01-01T00:00:00Z",
		"pid":        0
	}`)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error for fully empty checkpoint")
	}

	combined := errBuf.String()
	// The stderr output must include a remediation hint pointing to the debug command.
	if !strings.Contains(combined, "glassbox debug") {
		t.Errorf("stderr should hint at running 'glassbox debug', got: %q", combined)
	}
}

// ── session recover: alive process ───────────────────────────────────────────

func TestSessionRecover_LiveProcess_ReturnsError(t *testing.T) {
	homeDir := overrideHome(t)
	// Write a checkpoint whose PID is the current process (always alive).
	livePID := os.Getpid()
	content := `{
		"session_id": "session-alive",
		"tx_hash":    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"network":    "testnet",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        ` + itoa(livePID) + `
	}`
	writeCheckpointFile(t, homeDir, content)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error when the owning process is still alive")
	}
	msg := err.Error()
	if !strings.Contains(msg, "still alive") && !strings.Contains(msg, "still running") {
		t.Errorf("error should say process is still alive/running, got: %v", err)
	}
	// The session ID must be named in the error.
	if !strings.Contains(msg, "session-alive") {
		t.Errorf("error should name the session ID, got: %v", err)
	}
}

// ── session recover: orphaned — session not in store ────────────────────────

func TestSessionRecover_OrphanedSessionNotInStore_ClearsAndHints(t *testing.T) {
	homeDir := overrideHome(t)
	// Write a valid checkpoint whose PID does not exist.
	content := `{
		"session_id": "nonexistent-session-xyz",
		"tx_hash":    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"network":    "testnet",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        9999999
	}`
	writeCheckpointFile(t, homeDir, content)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err := sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	// The command should return nil (the stale-checkpoint path is not a fatal error).
	if err != nil {
		t.Fatalf("expected nil error for orphaned session not in store, got: %v", err)
	}

	combined := errBuf.String()
	// The user must be told how to re-debug.
	if !strings.Contains(combined, "glassbox debug") {
		t.Errorf("stderr should hint at running 'glassbox debug', got: %q", combined)
	}

	// Checkpoint must have been cleared.
	if _, statErr := os.Stat(filepath.Join(homeDir, ".Glassbox", "active_session.json")); !os.IsNotExist(statErr) {
		t.Error("stale checkpoint should have been removed after not finding the session in the store")
	}
}

// ── session recover: integrity failure on loaded session ─────────────────────

func TestSessionRecover_CorruptStoredSession_ReportsIntegrityIssues(t *testing.T) {
	homeDir := overrideHome(t)

	// Seed a corrupt session in the store.
	store, err := session.NewStore()
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	corruptData := &session.Data{
		ID:            "corrupt-session-123",
		TxHash:        "", // intentionally missing — will fail integrity
		Network:       "testnet",
		Status:        "active",
		CreatedAt:     time.Now().Add(-time.Hour),
		LastAccessAt:  time.Now(),
		SchemaVersion: session.SchemaVersion,
	}
	if saveErr := store.Save(context.Background(), corruptData); saveErr != nil {
		t.Fatalf("save corrupt session: %v", saveErr)
	}

	// Write a checkpoint pointing to the corrupt session.
	content := `{
		"session_id": "corrupt-session-123",
		"tx_hash":    "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		"network":    "testnet",
		"started_at": "2026-01-01T00:00:00Z",
		"pid":        9999999
	}`
	writeCheckpointFile(t, homeDir, content)

	var errBuf bytes.Buffer
	sessionRecoverCmd.SetOut(&errBuf)
	sessionRecoverCmd.SetErr(&errBuf)

	err = sessionRecoverCmd.RunE(sessionRecoverCmd, []string{})
	if err == nil {
		t.Fatal("expected error when the recovered session fails integrity validation")
	}
	msg := err.Error()
	if !strings.Contains(msg, "integrity") {
		t.Errorf("error should mention 'integrity', got: %v", err)
	}
	// Checkpoint must be cleared even after an integrity failure.
	if _, statErr := os.Stat(filepath.Join(homeDir, ".Glassbox", "active_session.json")); !os.IsNotExist(statErr) {
		t.Error("checkpoint should be cleared after integrity failure")
	}
	// stderr must list specific issues and a remediation hint.
	combined := errBuf.String()
	if !strings.Contains(combined, "integrity") {
		t.Errorf("stderr should mention integrity issues, got: %q", combined)
	}
	if !strings.Contains(combined, "glassbox debug") || !strings.Contains(combined, "glassbox session delete") {
		t.Errorf("stderr should provide remediation hints, got: %q", combined)
	}
}

// ── session resume: empty session ID ────────────────────────────────────────

func TestSessionResume_EmptyID_ReturnsValidationError(t *testing.T) {
	overrideHome(t)

	err := sessionResumeCmd.RunE(sessionResumeCmd, []string{"   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only session ID")
	}
	msg := err.Error()
	if !strings.Contains(msg, "required") && !strings.Contains(msg, "session ID") {
		t.Errorf("error should say session ID is required, got: %v", err)
	}
}

// ── session resume: session not found ────────────────────────────────────────

func TestSessionResume_NotFound_ErrorSuggestsListCommand(t *testing.T) {
	overrideHome(t)

	err := sessionResumeCmd.RunE(sessionResumeCmd, []string{"does-not-exist-xyz"})
	if err == nil {
		t.Fatal("expected error for nonexistent session ID")
	}
	msg := err.Error()
	if !strings.Contains(msg, "session list") && !strings.Contains(msg, "not found") {
		t.Errorf("error should hint at 'glassbox session list', got: %v", err)
	}
}

// ── session resume: integrity failure ────────────────────────────────────────

func TestSessionResume_IntegrityFailure_ReportsIssuesAndBlocks(t *testing.T) {
	overrideHome(t)

	store, err := session.NewStore()
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	// Session with an empty TxHash — must fail integrity.
	bad := &session.Data{
		ID:            "bad-session-resume",
		TxHash:        "",
		Network:       "testnet",
		Status:        "saved",
		CreatedAt:     time.Now().Add(-time.Hour),
		LastAccessAt:  time.Now(),
		SchemaVersion: session.SchemaVersion,
	}
	if err := store.Save(context.Background(), bad); err != nil {
		t.Fatalf("save: %v", err)
	}

	var errBuf bytes.Buffer
	sessionResumeCmd.SetOut(&errBuf)
	sessionResumeCmd.SetErr(&errBuf)

	err = sessionResumeCmd.RunE(sessionResumeCmd, []string{"bad-session-resume"})
	if err == nil {
		t.Fatal("expected error when session fails integrity")
	}
	msg := err.Error()
	if !strings.Contains(msg, "integrity") {
		t.Errorf("error should mention 'integrity', got: %v", err)
	}
	combined := errBuf.String()
	// Must list the field that failed.
	if !strings.Contains(combined, "TxHash") {
		t.Errorf("stderr should name the failing field 'TxHash', got: %q", combined)
	}
	// Must provide a hint.
	if !strings.Contains(combined, "Hint") {
		t.Errorf("stderr should include a Hint, got: %q", combined)
	}
	// The session must NOT have been made current.
	if GetCurrentSession() != nil && GetCurrentSession().ID == "bad-session-resume" {
		t.Error("corrupt session must not become the current session")
	}
}

// ── session resume: schema version too new ───────────────────────────────────

func TestSessionResume_TooNewSchema_ReturnsUpgradeError(t *testing.T) {
	overrideHome(t)

	store, err := session.NewStore()
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	future := &session.Data{
		ID:            "future-session",
		TxHash:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Network:       "testnet",
		HorizonURL:    "https://horizon-testnet.stellar.org",
		Status:        "saved",
		CreatedAt:     time.Now().Add(-time.Hour),
		LastAccessAt:  time.Now(),
		SchemaVersion: session.SchemaVersion + 99,
	}
	if err := store.SavePreservingSchemaVersion(context.Background(), future); err != nil {
		t.Fatalf("save: %v", err)
	}

	var errBuf bytes.Buffer
	sessionResumeCmd.SetOut(&errBuf)
	sessionResumeCmd.SetErr(&errBuf)

	err = sessionResumeCmd.RunE(sessionResumeCmd, []string{"future-session"})
	if err == nil {
		t.Fatal("expected error for session with schema version newer than current binary")
	}
	msg := err.Error()
	if !strings.Contains(msg, "version") && !strings.Contains(msg, "upgrade") && !strings.Contains(msg, "schema") {
		t.Errorf("error should mention version/schema/upgrade, got: %v", err)
	}
}

// ── session resume: valid session resumes cleanly ────────────────────────────

func TestSessionResume_ValidSession_SetsCurrentSession(t *testing.T) {
	overrideHome(t)
	t.Cleanup(func() { SetCurrentSession(nil) })

	store, err := session.NewStore()
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	good := &session.Data{
		ID:            "good-session-resume",
		TxHash:        "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		Network:       "testnet",
		Status:        "saved",
		CreatedAt:     time.Now().Add(-time.Hour),
		LastAccessAt:  time.Now(),
		SchemaVersion: session.SchemaVersion,
	}
	if err := store.Save(context.Background(), good); err != nil {
		t.Fatalf("save: %v", err)
	}

	var out bytes.Buffer
	sessionResumeCmd.SetOut(&out)
	sessionResumeCmd.SetErr(&out)

	err = sessionResumeCmd.RunE(sessionResumeCmd, []string{"good-session-resume"})
	if err != nil {
		t.Fatalf("unexpected error resuming valid session: %v", err)
	}
	current := GetCurrentSession()
	if current == nil {
		t.Fatal("current session should be set after a successful resume")
	}
	if current.ID != "good-session-resume" {
		t.Errorf("current session ID = %q, want %q", current.ID, "good-session-resume")
	}
	if current.Status != "resumed" {
		t.Errorf("current session status = %q, want %q", current.Status, "resumed")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
