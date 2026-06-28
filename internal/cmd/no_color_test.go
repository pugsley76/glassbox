// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/dotandev/glassbox/internal/visualizer"
)

// TestNoColorFlagDisablesVisualizer verifies that setting NoColorFlag and
// calling the activation logic (mirroring what PersistentPreRunE does) causes
// visualizer.ColorEnabled() to return false and Colorize to emit plain text.
func TestNoColorFlagDisablesVisualizer(t *testing.T) {
	// Reset state after test.
	defer func() {
		NoColorFlag = false
		visualizer.SetNoColor(false)
		_ = os.Unsetenv("NO_COLOR")
	}()

	_ = os.Unsetenv("NO_COLOR")
	_ = os.Unsetenv("GLASSBOX_NO_COLOR")
	NoColorFlag = true

	// Simulate what PersistentPreRunE does when NoColorFlag is true.
	_ = os.Setenv("NO_COLOR", "1")
	visualizer.SetNoColor(true)

	if visualizer.ColorEnabled() {
		t.Error("visualizer.ColorEnabled() should be false after activating --no-color")
	}

	out := visualizer.Colorize("test", "red")
	if strings.Contains(out, "\033") {
		t.Errorf("Colorize should emit plain text when --no-color is active, got: %q", out)
	}
	if out != "test" {
		t.Errorf("expected plain %q, got %q", "test", out)
	}
}

// TestGlassboxNoColorEnvActivatesOnPreRun verifies that GLASSBOX_NO_COLOR in
// the environment triggers the same no-color path as the CLI flag.
func TestGlassboxNoColorEnvActivatesOnPreRun(t *testing.T) {
	defer func() {
		visualizer.SetNoColor(false)
		_ = os.Unsetenv("NO_COLOR")
		_ = os.Unsetenv("GLASSBOX_NO_COLOR")
	}()

	_ = os.Unsetenv("NO_COLOR")
	_ = os.Setenv("GLASSBOX_NO_COLOR", "1")

	// Simulate PersistentPreRunE logic for env-var path.
	if os.Getenv("GLASSBOX_NO_COLOR") != "" {
		_ = os.Setenv("NO_COLOR", "1")
		visualizer.SetNoColor(true)
	}

	if visualizer.ColorEnabled() {
		t.Error("visualizer.ColorEnabled() should be false when GLASSBOX_NO_COLOR is set")
	}

	out := visualizer.Colorize("ci-log", "cyan")
	if strings.Contains(out, "\033") {
		t.Errorf("expected plain text for CI log capture, got: %q", out)
	}
}

// TestNoColorFlagRegistered verifies that the --no-color persistent flag
// exists on the root command with the expected type.
func TestNoColorFlagRegistered(t *testing.T) {
	f := rootCmd.PersistentFlags().Lookup("no-color")
	if f == nil {
		t.Fatal("--no-color flag is not registered on rootCmd")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--no-color flag type: got %q, want %q", f.Value.Type(), "bool")
	}
}
