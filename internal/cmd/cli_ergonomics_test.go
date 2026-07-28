// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// ── session list: empty store ─────────────────────────────────────────────────

func TestSessionList_EmptyStore_PrintsTipMessage(t *testing.T) {
	overrideHome(t)

	var out bytes.Buffer
	sessionListCmd.SetOut(&out)
	sessionListCmd.SetErr(&out)

	err := sessionListCmd.RunE(sessionListCmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error listing empty store: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "No saved sessions found") {
		t.Errorf("expected 'No saved sessions found' in output, got: %q", output)
	}
	// Should tell the user how to start a session.
	if !strings.Contains(output, "glassbox debug") {
		t.Errorf("empty list output should hint at 'glassbox debug', got: %q", output)
	}
	if !strings.Contains(output, "glassbox session save") {
		t.Errorf("empty list output should hint at 'glassbox session save', got: %q", output)
	}
}

// ── session delete: empty ID ──────────────────────────────────────────────────

func TestSessionDelete_EmptyID_ReturnsValidationError(t *testing.T) {
	overrideHome(t)

	err := sessionDeleteCmd.RunE(sessionDeleteCmd, []string{"   "})
	if err == nil {
		t.Fatal("expected error for whitespace-only session ID")
	}
	msg := err.Error()
	if !strings.Contains(msg, "session ID") && !strings.Contains(msg, "required") {
		t.Errorf("error should mention session ID is required, got: %v", err)
	}
}

// ── session delete: not found ─────────────────────────────────────────────────

func TestSessionDelete_NotFound_HintsAtListCommand(t *testing.T) {
	overrideHome(t)

	err := sessionDeleteCmd.RunE(sessionDeleteCmd, []string{"does-not-exist-xyz"})
	if err == nil {
		t.Fatal("expected error for nonexistent session ID")
	}
	msg := err.Error()
	if !strings.Contains(msg, "session list") && !strings.Contains(msg, "not found") {
		t.Errorf("error should hint at 'glassbox session list', got: %v", err)
	}
}

// ── session save: no active session ──────────────────────────────────────────

func TestSessionSave_NoActiveSession_ReturnsHelpfulError(t *testing.T) {
	overrideHome(t)
	// Ensure no current session is set.
	SetCurrentSession(nil)

	var out bytes.Buffer
	sessionSaveCmd.SetOut(&out)
	sessionSaveCmd.SetErr(&out)

	err := sessionSaveCmd.RunE(sessionSaveCmd, []string{})
	if err == nil {
		t.Fatal("expected error when no active session to save")
	}
	msg := err.Error()
	if !strings.Contains(msg, "glassbox debug") {
		t.Errorf("error should suggest running 'glassbox debug', got: %v", err)
	}
}
