// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"
)

// TestCommandExamples verifies that key subcommands expose non-empty Example
// fields so that `--help` output includes representative usage guidance.
func TestCommandExamples(t *testing.T) {
	cases := []struct {
		name    string
		example string
	}{
		{"trace", traceCmd.Example},
		{"compare", compareCmd.Example},
		{"heuristic", heuristicCmd.Example},
		{"debug", debugCmd.Example},
		{"session", sessionCmd.Example},
		{"cache", cacheCmd.Example},
		{"report", reportCmd.Example},
		{"regression-test", regressionTestCmd.Example},
		{"version", versionCmd.Example},
		{"bench", benchCmd.Example},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.TrimSpace(tc.example) == "" {
				t.Errorf("command %q has no Example field; add usage examples to help reduce onboarding friction", tc.name)
			}
		})
	}
}

// TestDryRunCommandExample verifies the dry-run subcommand has examples.
func TestDryRunCommandExample(t *testing.T) {
	if strings.TrimSpace(dryRunCmd.Example) == "" {
		t.Error("dry-run command has no Example field")
	}
}

// TestCommandExampleContent verifies that examples reference real flag names.
func TestCommandExampleContent(t *testing.T) {
	cases := []struct {
		name        string
		example     string
		mustContain []string
	}{
		{
			name:        "trace",
			example:     traceCmd.Example,
			mustContain: []string{"glassbox trace", "--print", "--theme"},
		},
		{
			name:        "compare",
			example:     compareCmd.Example,
			mustContain: []string{"glassbox compare", "--wasm", "--network"},
		},
		{
			name:        "dry-run",
			example:     dryRunCmd.Example,
			mustContain: []string{"glassbox dry-run", "--network"},
		},
		{
			name:        "heuristic",
			example:     heuristicCmd.Example,
			mustContain: []string{"glassbox heuristic list", "glassbox heuristic validate"},
		},
		{
			name:        "report",
			example:     reportCmd.Example,
			mustContain: []string{"glassbox report", "--file", "--format"},
		},
		{
			name:        "regression-test",
			example:     regressionTestCmd.Example,
			mustContain: []string{"glassbox regression-test", "--count", "--workers"},
		},
		{
			name:        "version",
			example:     versionCmd.Example,
			mustContain: []string{"glassbox version", "--json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, substr := range tc.mustContain {
				if !strings.Contains(tc.example, substr) {
					t.Errorf("command %q Example missing expected content %q", tc.name, substr)
				}
			}
		})
	}
}

// TestCommandLongDescriptions verifies that the key improved commands have
// non-empty Long descriptions that mention validation behavior.
func TestCommandLongDescriptions(t *testing.T) {
	cases := []struct {
		name        string
		long        string
		mustContain []string
	}{
		{
			name:        "regression-test",
			long:        regressionTestCmd.Long,
			mustContain: []string{"--count", "--network"},
		},
		{
			name:        "report",
			long:        reportCmd.Long,
			mustContain: []string{"--file", "--format"},
		},
		{
			name:        "version",
			long:        versionCmd.Long,
			mustContain: []string{"--json"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.TrimSpace(tc.long) == "" {
				t.Errorf("command %q has no Long description", tc.name)
			}
			for _, substr := range tc.mustContain {
				if !strings.Contains(tc.long, substr) {
					t.Errorf("command %q Long description missing %q", tc.name, substr)
				}
			}
		})
	}
}

// ── Source mapping commands — help output quality ─────────────────────────────

// TestBenchCmd_ExamplePresent verifies that the bench command (which includes
// the sourcemap pipeline stage) has a non-empty Example field.
func TestBenchCmd_ExamplePresent(t *testing.T) {
	if strings.TrimSpace(benchCmd.Example) == "" {
		t.Error("bench command must have a non-empty Example field")
	}
}

// TestBenchCmd_ExampleMentionsSourcemap verifies the bench Example field
// references the sourcemap mode so users know it exists from --help output.
func TestBenchCmd_ExampleMentionsSourcemap(t *testing.T) {
	if !strings.Contains(benchCmd.Example, "sourcemap") {
		t.Error("bench Example should reference the sourcemap mode")
	}
}

// TestBenchCmd_LongDescriptionMentionsSourcemap verifies the bench Long
// description explains the source mapping benchmark purpose.
func TestBenchCmd_LongDescriptionMentionsSourcemap(t *testing.T) {
	long := benchCmd.Long
	if !strings.Contains(long, "sourcemap") {
		t.Error("bench Long description should mention the sourcemap mode")
	}
	if !strings.Contains(long, "--mode") {
		t.Error("bench Long description should mention --mode validation")
	}
}

// TestBenchCmd_LongDescriptionMentionsCount verifies that --count is described.
func TestBenchCmd_LongDescriptionMentionsCount(t *testing.T) {
	if !strings.Contains(benchCmd.Long, "--count") {
		t.Error("bench Long description should mention --count validation")
	}
}

// TestWasmDiffCmd_ExamplePresent verifies wasm-diff has a non-empty Example.
func TestWasmDiffCmd_ExamplePresent(t *testing.T) {
	if strings.TrimSpace(wasmDiffCmd.Example) == "" {
		t.Error("wasm-diff command must have a non-empty Example field")
	}
}

// TestWasmDiffCmd_LongDescriptionMentionsSourceMapping verifies the wasm-diff
// Long description explains its relevance to source mapping.
func TestWasmDiffCmd_LongDescriptionMentionsSourceMapping(t *testing.T) {
	if !strings.Contains(wasmDiffCmd.Long, "source mapping") {
		t.Error("wasm-diff Long description should mention source mapping")
	}
}
