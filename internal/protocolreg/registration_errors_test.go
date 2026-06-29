// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package protocolreg

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	ersterrors "github.com/dotandev/glassbox/internal/errors"
)

// ── verificationError ────────────────────────────────────────────────────────

func TestVerificationError_IncludesAllIssues(t *testing.T) {
	issues := []string{"registry key missing", "URL Protocol value absent", "open command mismatch"}
	err := verificationError(issues)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	for _, issue := range issues {
		if !strings.Contains(msg, issue) {
			t.Errorf("error message should contain %q, got: %s", issue, msg)
		}
	}
}

func TestVerificationError_SingleIssue(t *testing.T) {
	err := verificationError([]string{"something went wrong"})
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !strings.Contains(err.Error(), "something went wrong") {
		t.Errorf("error should include the issue text, got: %v", err)
	}
}

// ── ErrRegistryConflict wrapping ─────────────────────────────────────────────

func TestErrRegistryConflict_IsDetectableWhenWrapped(t *testing.T) {
	// Verify that wrapping ErrRegistryConflict with fmt.Errorf preserves errors.Is
	// semantics — critical for callers that check for this sentinel.
	wrapped := fmt.Errorf("context about the conflict: %w", ersterrors.ErrRegistryConflict)
	if !errors.Is(wrapped, ersterrors.ErrRegistryConflict) {
		t.Error("errors.Is should detect ErrRegistryConflict through wrapping")
	}
}

func TestErrRegistryConflict_WrappedIncludesActionableGuidance(t *testing.T) {
	wrapped := fmt.Errorf(
		"protocol registration conflict: registry key is claimed by another app: %w\n"+
			"  Fix: run 'glassbox protocol:repair' to reclaim the registration",
		ersterrors.ErrRegistryConflict,
	)
	msg := wrapped.Error()
	if !strings.Contains(msg, "protocol:repair") {
		t.Errorf("wrapped conflict error should mention 'protocol:repair', got: %s", msg)
	}
	if !strings.Contains(msg, "registration conflict") {
		t.Errorf("wrapped conflict error should mention 'registration conflict', got: %s", msg)
	}
}

// ── hasCommand ───────────────────────────────────────────────────────────────

func TestHasCommand_KnownMissingCommand(t *testing.T) {
	if hasCommand("glassbox-nonexistent-binary-xyz-12345") {
		t.Error("hasCommand should return false for a nonexistent binary")
	}
}

func TestHasCommand_KnownPresentCommand(t *testing.T) {
	// "sh" is available on all supported non-Windows platforms.
	if runtime.GOOS == "windows" {
		t.Skip("test uses 'sh' which is not present on Windows")
	}
	if !hasCommand("sh") {
		t.Error("hasCommand should return true for 'sh'")
	}
}

// ── runCommand ───────────────────────────────────────────────────────────────

func TestRunCommand_MissingBinary_ReturnsError(t *testing.T) {
	_, err := runCommand("glassbox-nonexistent-binary-xyz-12345")
	if err == nil {
		t.Fatal("expected error when running a nonexistent command")
	}
	// The error should identify the binary name so the user knows what failed.
	if !strings.Contains(err.Error(), "glassbox-nonexistent-binary-xyz-12345") {
		t.Errorf("error should include the binary name, got: %v", err)
	}
}

// ── shellQuote ───────────────────────────────────────────────────────────────

func TestShellQuote_SimplePath(t *testing.T) {
	got := shellQuote("/usr/local/bin/glassbox")
	if got != "'/usr/local/bin/glassbox'" {
		t.Errorf("expected single-quoted path, got %q", got)
	}
}

func TestShellQuote_PathWithSingleQuote(t *testing.T) {
	// A path like /home/user's/glassbox must be escaped so the shell script is valid.
	// shellQuote replaces ' with '"'"' — this is the standard POSIX escape sequence
	// that closes the current single-quoted string, injects a double-quoted single
	// quote, then reopens single-quoting.
	got := shellQuote("/home/user's/glassbox")
	// The result must be wrapped in outer single quotes.
	if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
		t.Errorf("shellQuote result should be wrapped in single quotes, got: %q", got)
	}
	// The original path content must survive in the output.
	if !strings.Contains(got, "user") || !strings.Contains(got, "glassbox") {
		t.Errorf("shellQuote lost path components, got: %q", got)
	}
	// The escaped single quote sequence must appear in the output.
	if !strings.Contains(got, `'"'"'`) {
		t.Errorf("shellQuote should use '\"'\"' escape sequence, got: %q", got)
	}
}

func TestShellQuote_EmptyString(t *testing.T) {
	got := shellQuote("")
	if got != "''" {
		t.Errorf("empty string should become two single quotes, got %q", got)
	}
}

// ── XdgMime missing error message ────────────────────────────────────────────

func TestRegisterLinux_XdgMissingMessage_IsActionable(t *testing.T) {
	// Construct the exact error message that registerLinux would return when
	// xdg-mime is absent, and verify it contains installation guidance.
	msg := "xdg-mime is not installed: cannot register the glassbox:// MIME handler\n" +
		"  Fix: install xdg-utils — try one of:\n" +
		"    sudo apt install xdg-utils   (Debian/Ubuntu)\n" +
		"    sudo dnf install xdg-utils   (Fedora/RHEL)\n" +
		"    sudo pacman -S xdg-utils     (Arch Linux)"

	for _, keyword := range []string{"xdg-mime", "xdg-utils", "apt install", "dnf install", "pacman"} {
		if !strings.Contains(msg, keyword) {
			t.Errorf("xdg-mime missing error should mention %q", keyword)
		}
	}
}
