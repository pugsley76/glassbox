// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"os"
	"strings"
	"testing"
)

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	_ = os.Unsetenv(key)
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}

func TestANSIRenderer_IsTTY(t *testing.T) {
	t.Run("NO_COLOR disables color", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		r := NewANSIRenderer()
		if r.IsTTY() {
			t.Error("IsTTY() should be false when NO_COLOR is set")
		}
	})

	t.Run("FORCE_COLOR enables color", func(t *testing.T) {
		unsetEnv(t, "NO_COLOR")
		t.Setenv("FORCE_COLOR", "1")
		r := NewANSIRenderer()
		if !r.IsTTY() {
			t.Error("IsTTY() should be true when FORCE_COLOR is set")
		}
	})

	t.Run("TERM=dumb disables color", func(t *testing.T) {
		unsetEnv(t, "FORCE_COLOR")
		t.Setenv("TERM", "dumb")
		r := NewANSIRenderer()
		if r.IsTTY() {
			t.Error("IsTTY() should be false when TERM=dumb")
		}
	})
}

func TestANSIRenderer_Colorize(t *testing.T) {
	t.Run("colorized when forced", func(t *testing.T) {
		unsetEnv(t, "NO_COLOR")
		t.Setenv("FORCE_COLOR", "1")
		r := NewANSIRenderer()

		colored := r.Colorize("hello", "red")
		if !strings.Contains(colored, "\033[31m") {
			t.Errorf("Expected red color code, got %q", colored)
		}
	})

	t.Run("NO_COLOR overrides force", func(t *testing.T) {
		t.Setenv("FORCE_COLOR", "1")
		t.Setenv("NO_COLOR", "1")
		r := NewANSIRenderer()

		plain := r.Colorize("hello", "red")
		if strings.Contains(plain, "\033") {
			t.Errorf("Expected plain text when NO_COLOR is set, got %q", plain)
		}
	})
}

func TestANSIRenderer_Symbols(t *testing.T) {
	t.Run("forced color symbols", func(t *testing.T) {
		t.Setenv("FORCE_COLOR", "1")
		r := NewANSIRenderer()

		if r.Symbol("check") != "[OK]" {
			t.Errorf("Expected [OK] for check symbol, got %q", r.Symbol("check"))
		}
	})

	t.Run("NO_COLOR symbols", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		r := NewANSIRenderer()

		if r.Symbol("check") != "[OK]" {
			t.Errorf("Expected [OK] for check symbol when NO_COLOR, got %q", r.Symbol("check"))
		}
	})
}

// TestANSIRenderer_NOColorOverridesFORCECOLOR verifies that NO_COLOR takes
// precedence over FORCE_COLOR when both are set simultaneously. This is the
// critical precedence rule documented in docs/no-color.md and tested here in
// the ANSI renderer unit test (the cross-platform variant lives in
// platform_tty_test.go).
func TestANSIRenderer_NOColorOverridesFORCECOLOR(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	t.Setenv("FORCE_COLOR", "1")
	r := NewANSIRenderer()
	if r.IsTTY() {
		t.Error("IsTTY() must return false when both NO_COLOR and FORCE_COLOR are set: NO_COLOR wins")
	}
	// Colorize must return plain text, not ANSI sequences.
	got := r.Colorize("text", "red")
	if strings.Contains(got, "\033") {
		t.Errorf("Colorize must return plain text when NO_COLOR is set; got %q", got)
	}
}

// TestANSIRenderer_Symbol_CrossDivergence documents and verifies the
// intentional TTY/non-TTY output difference for Symbol("cross"):
//
//   TTY mode    → "[FAIL]"  (descriptive, for terminal display)
//   non-TTY mode → "[X]"    (compact, for log files and plain text)
//
// All other documented symbol names return the same string in both modes.
// This divergence is documented in docs/no-color.md §"Symbol divergence".
func TestANSIRenderer_Symbol_CrossDivergence(t *testing.T) {
	t.Run("TTY mode returns [FAIL]", func(t *testing.T) {
		unsetEnv(t, "NO_COLOR")
		t.Setenv("FORCE_COLOR", "1")
		r := NewANSIRenderer()
		got := r.Symbol("cross")
		if got != "[FAIL]" {
			t.Errorf("Symbol(cross) in TTY mode = %q, want [FAIL]", got)
		}
	})

	t.Run("non-TTY mode returns [X]", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		unsetEnv(t, "FORCE_COLOR")
		r := NewANSIRenderer()
		got := r.Symbol("cross")
		if got != "[X]" {
			t.Errorf("Symbol(cross) in non-TTY mode = %q, want [X]", got)
		}
	})
}

// TestANSIRenderer_Symbol_ConsistentForNonCross verifies that for every
// documented symbol name other than "cross", Symbol() returns the same
// string regardless of TTY mode. This confirms the cross divergence is the
// only intentional difference.
func TestANSIRenderer_Symbol_ConsistentForNonCross(t *testing.T) {
	// Symbols that must be identical in both TTY and non-TTY modes.
	sameBothModes := []struct {
		name string
		want string
	}{
		{"check", "[OK]"},
		{"warn", "[!]"},
		{"arrow_r", "->"},
		{"arrow_l", "<-"},
		{"pin", "*"},
	}

	for _, sym := range sameBothModes {
		t.Run(sym.name, func(t *testing.T) {
			// TTY mode
			unsetEnv(t, "NO_COLOR")
			t.Setenv("FORCE_COLOR", "1")
			ttyRenderer := NewANSIRenderer()
			ttyGot := ttyRenderer.Symbol(sym.name)

			// non-TTY mode
			t.Setenv("NO_COLOR", "1")
			unsetEnv(t, "FORCE_COLOR")
			noColorRenderer := NewANSIRenderer()
			noColorGot := noColorRenderer.Symbol(sym.name)

			if ttyGot != sym.want {
				t.Errorf("Symbol(%q) TTY = %q, want %q", sym.name, ttyGot, sym.want)
			}
			if noColorGot != sym.want {
				t.Errorf("Symbol(%q) non-TTY = %q, want %q", sym.name, noColorGot, sym.want)
			}
			if ttyGot != noColorGot {
				t.Errorf("Symbol(%q) differs between TTY (%q) and non-TTY (%q) — only 'cross' should diverge",
					sym.name, ttyGot, noColorGot)
			}
		})
	}
}
