// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/dotandev/glassbox/internal/errors"
	"github.com/dotandev/glassbox/internal/trace"
	"github.com/spf13/cobra"
)

// traceCompareCmd implements glassbox trace compare
var traceCompareCmd = &cobra.Command{
	Use:     "compare <baseline_trace.json> <current_trace.json>",
	GroupID: "testing",
	Short:   "Compare two trace files for differences and regressions",
	Long: `Compare two execution trace files side-by-side to detect differences
in execution paths, state changes, and contract invocations.

This is useful for regression testing to ensure your contract changes don't
alter behavior in unexpected ways.

Examples:
  glassbox trace compare baseline.json current.json
  glassbox trace compare v1.json v2.json --baseline-name "v1.0" --current-name "v2.0"`,
	Args: cobra.ExactArgs(2),
	RunE: runTraceCompare,
}

// traceCompareParentCmd is the parent command for trace sub-commands
var traceCompareParentCmd = &cobra.Command{
	Use:     "trace",
	GroupID: "utility",
	Short:   "Manage execution traces",
	Long: `Save, load, and compare execution traces for debugging and regression testing.`,
}

var (
	traceCompareBaselineNameFlag string
	traceCompareCurrentNameFlag  string
	traceSaveOutputFlag          string
)

func init() {
	// Trace save command
	traceSaveCmd := &cobra.Command{
		Use:   "save <output_path>",
		Short: "Save the current execution trace to a file",
		RunE:  runTraceSave, // TODO: implement this if we have a current trace
	}
	traceSaveCmd.Flags().StringVar(&traceSaveOutputFlag, "output", "", "Output file path (required)")

	// Trace compare command flags
	traceCompareCmd.Flags().StringVar(&traceCompareBaselineNameFlag, "baseline-name", "Baseline", "Name for the baseline trace in output")
	traceCompareCmd.Flags().StringVar(&traceCompareCurrentNameFlag, "current-name", "Current", "Name for the current trace in output")

	traceCompareParentCmd.AddCommand(traceCompareCmd)
	traceCompareParentCmd.AddCommand(traceSaveCmd)
	rootCmd.AddCommand(traceCompareParentCmd)
}

func runTraceCompare(cmd *cobra.Command, args []string) error {
	baselinePath := args[0]
	currentPath := args[1]

	fmt.Printf("Loading traces...\n")
	fmt.Printf("  Baseline: %s\n", baselinePath)
	fmt.Printf("  Current:  %s\n\n", currentPath)

	baselineTrace, err := trace.LoadExecutionTrace(baselinePath)
	if err != nil {
		return errors.WrapValidationError(fmt.Sprintf("failed to load baseline trace: %v", err))
	}

	currentTrace, err := trace.LoadExecutionTrace(currentPath)
	if err != nil {
		return errors.WrapValidationError(fmt.Sprintf("failed to load current trace: %v", err))
	}

	fmt.Printf("Comparing traces...\n\n")
	diffResult := trace.CompareTraces(baselineTrace, currentTrace, traceCompareBaselineNameFlag, traceCompareCurrentNameFlag)
	diffResult.Render()

	if diffResult.HasDivergence {
		return errors.WrapValidationError("Traces differ - regression detected")
	}

	return nil
}

func runTraceSave(cmd *cobra.Command, args []string) error {
	// TODO: This would need access to a current trace, perhaps from a session or debug command
	return errors.WrapValidationError("trace save is not implemented yet - use debug command to generate traces")
}
