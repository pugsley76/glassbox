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

// ── Protocol command help output and examples ─────────────────────────────────

// TestProtocolCommandExamples verifies that all protocol:* subcommands expose
// non-empty Example fields so that --help output includes usage guidance.
func TestProtocolCommandExamples(t *testing.T) {
	cases := []struct {
		name    string
		example string
	}{
		{"protocol:register", protocolRegisterCmd.Example},
		{"protocol:unregister", protocolUnregisterCmd.Example},
		{"protocol:status", protocolStatusCmd.Example},
		{"protocol:verify", protocolVerifyCmd.Example},
		{"protocol:handle", protocolHandlerCmd.Example},
		{"protocol:diagnose", protocolDiagnoseCmd.Example},
		{"protocol:repair", protocolRepairCmd.Example},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.TrimSpace(tc.example) == "" {
				t.Errorf("command %q has no Example field; add usage examples to reduce onboarding friction", tc.name)
			}
		})
	}
}

// TestProtocolCommandLongDescriptions verifies that all protocol:* subcommands
// have non-empty Long descriptions.
func TestProtocolCommandLongDescriptions(t *testing.T) {
	cases := []struct {
		name string
		long string
	}{
		{"protocol:register", protocolRegisterCmd.Long},
		{"protocol:unregister", protocolUnregisterCmd.Long},
		{"protocol:status", protocolStatusCmd.Long},
		{"protocol:verify", protocolVerifyCmd.Long},
		{"protocol:handle", protocolHandlerCmd.Long},
		{"protocol:diagnose", protocolDiagnoseCmd.Long},
		{"protocol:repair", protocolRepairCmd.Long},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.TrimSpace(tc.long) == "" {
				t.Errorf("command %q has no Long description; add one to improve help output", tc.name)
			}
		})
	}
}

// TestProtocolCommandExampleContent verifies that each protocol command's
// Example field references the command name and relevant flags/params.
func TestProtocolCommandExampleContent(t *testing.T) {
	cases := []struct {
		name        string
		example     string
		mustContain []string
	}{
		{
			name:        "protocol:register",
			example:     protocolRegisterCmd.Example,
			mustContain: []string{"glassbox protocol:register", "--dry-run"},
		},
		{
			name:        "protocol:unregister",
			example:     protocolUnregisterCmd.Example,
			mustContain: []string{"glassbox protocol:unregister"},
		},
		{
			name:        "protocol:status",
			example:     protocolStatusCmd.Example,
			mustContain: []string{"glassbox protocol:status"},
		},
		{
			name:        "protocol:verify",
			example:     protocolVerifyCmd.Example,
			mustContain: []string{"glassbox protocol:verify", "--probe"},
		},
		{
			name:        "protocol:handle",
			example:     protocolHandlerCmd.Example,
			mustContain: []string{"glassbox protocol:handle", "glassbox://debug/", "network="},
		},
		{
			name:        "protocol:diagnose",
			example:     protocolDiagnoseCmd.Example,
			mustContain: []string{"glassbox protocol:diagnose", "--json", "--format"},
		},
		{
			name:        "protocol:repair",
			example:     protocolRepairCmd.Example,
			mustContain: []string{"glassbox protocol:repair"},
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

// TestProtocolHandleCmd_LongDescribesURIFormat verifies the protocol:handle Long
// description documents the URI format so users can construct valid deep links.
func TestProtocolHandleCmd_LongDescribesURIFormat(t *testing.T) {
	long := protocolHandlerCmd.Long
	for _, required := range []string{
		"glassbox://debug/",
		"network",
		"testnet", "mainnet", "futurenet",
		"op",
		"view",
	} {
		if !strings.Contains(long, required) {
			t.Errorf("protocol:handle Long description should document %q; got:\n%s", required, long)
		}
	}
}

// TestProtocolRegisterCmd_LongDescribesDryRun verifies the Long description
// explains the --dry-run flag so users know it is available.
func TestProtocolRegisterCmd_LongDescribesDryRun(t *testing.T) {
	if !strings.Contains(protocolRegisterCmd.Long, "--dry-run") {
		t.Error("protocol:register Long description should mention --dry-run")
	}
}

// TestProtocolDiagnoseCmd_LongDescribesExitCodes verifies that the protocol:diagnose
// Long description documents the exit codes so scripts can rely on them.
func TestProtocolDiagnoseCmd_LongDescribesExitCodes(t *testing.T) {
	long := protocolDiagnoseCmd.Long
	if !strings.Contains(long, "0") || !strings.Contains(long, "1") {
		t.Error("protocol:diagnose Long description should document exit codes 0 and 1")
	}
	if !strings.Contains(long, "Exit") {
		t.Error("protocol:diagnose Long description should include an 'Exit codes' section")
	}
}

// TestProtocolRepairCmd_LongDescribesPlatforms verifies the repair Long
// description names all three supported platforms.
func TestProtocolRepairCmd_LongDescribesPlatforms(t *testing.T) {
	long := protocolRepairCmd.Long
	for _, platform := range []string{"Linux", "macOS", "Windows"} {
		if !strings.Contains(long, platform) {
			t.Errorf("protocol:repair Long description should mention platform %q; got:\n%s", platform, long)
		}
	}
}
