// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/dotandev/glassbox/internal/errors"
	"github.com/dotandev/glassbox/internal/rpc"
	"github.com/dotandev/glassbox/internal/simulator"
	"github.com/spf13/cobra"
)

var (
	regressionTestCount       int
	regressionProtocolVersion uint32
	regressionStartSeq        uint32
	regressionMaxWorkers      int
)

// maxRegressionCount is the upper bound accepted by --count to prevent
// accidental large RPC batches.
const maxRegressionCount = 1000

var regressionTestCmd = &cobra.Command{
	Use:     "regression-test",
	GroupID: "testing",
	Short:   "Run protocol regression tests against historic transactions",
	Long: `Execute a comprehensive regression test suite by downloading historic failed
transactions from Mainnet and ensuring glassbox-sim yields identical results.

This command fetches up to the specified number of failed transactions and
simulates them in parallel, verifying that the simulator produces the same
traps and events as the original network execution.

The tests help ensure that protocol changes don't introduce regressions.

Validation:
  --count must be between 1 and 1000 (inclusive).
  --workers must be >= 0; when 0 it defaults to 4.
  --protocol-version is optional; when provided it must be a supported version.
  --network must be one of: testnet, mainnet, futurenet.`,
	Example: `  # Run 100 regression tests on mainnet (default)
  glassbox regression-test --count 100

  # Use more parallel workers for faster runs
  glassbox regression-test --count 1000 --workers 8

  # Test against a specific protocol version
  glassbox regression-test --count 500 --network mainnet --protocol-version 22

  # Verbose output shows per-transaction progress
  glassbox regression-test --count 50 --verbose`,
	PreRunE: validateRegressionFlags,
	RunE:    runRegressionTest,
}

// validateRegressionFlags runs early validation so issues are surfaced before
// any network or simulator calls are made.
func validateRegressionFlags(cmd *cobra.Command, args []string) error {
	if regressionTestCount <= 0 {
		return errors.WrapValidationError(fmt.Sprintf(
			"--count must be greater than 0 (got %d); specify the number of "+
				"historic failed transactions to test",
			regressionTestCount,
		))
	}
	if regressionTestCount > maxRegressionCount {
		return errors.WrapValidationError(fmt.Sprintf(
			"--count %d exceeds the maximum of %d; reduce --count to avoid "+
				"large RPC batch requests",
			regressionTestCount, maxRegressionCount,
		))
	}

	if regressionMaxWorkers < 0 {
		return errors.WrapValidationError(fmt.Sprintf(
			"--workers must be a positive integer (got %d)", regressionMaxWorkers,
		))
	}

	// Validate --network early before creating any client.
	if err := validateNetworkName(networkFlag); err != nil {
		return errors.WrapValidationError(fmt.Sprintf(
			"invalid --network %q; must be one of: testnet, mainnet, futurenet\n"+
				"Use 'glassbox regression-test --help' for all available flags",
			networkFlag,
		))
	}

	// Validate --protocol-version when explicitly provided.
	if regressionProtocolVersion > 0 {
		if err := simulator.Validate(regressionProtocolVersion); err != nil {
			return errors.WrapValidationError(fmt.Sprintf(
				"invalid --protocol-version %d: %v\n"+
					"Run 'glassbox version' to see the protocol versions supported by "+
					"this build",
				regressionProtocolVersion, err,
			))
		}
	}

	return nil
}

func runRegressionTest(cmd *cobra.Command, args []string) error {
	if regressionMaxWorkers <= 0 {
		regressionMaxWorkers = 4
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Starting regression test suite\n")
	fmt.Fprintf(out, "  Target count:   %d transactions\n", regressionTestCount)
	fmt.Fprintf(out, "  Network:        %s\n", networkFlag)
	fmt.Fprintf(out, "  Workers:        %d\n", regressionMaxWorkers)

	if regressionProtocolVersion > 0 {
		fmt.Fprintf(out, "  Protocol ver.:  %d\n", regressionProtocolVersion)
	}

	// Create RPC client
	opts := []rpc.ClientOption{
		rpc.WithNetwork(rpc.Network(networkFlag)),
		rpc.WithToken(rpcTokenFlag),
	}
	if rpcURLFlag != "" {
		opts = append(opts, rpc.WithHorizonURL(rpcURLFlag))
	}

	client, err := rpc.NewClient(opts...)
	if err != nil {
		return errors.WrapValidationError(fmt.Sprintf(
			"failed to create RPC client for network %q: %v\n"+
				"Check your --network and --rpc-url flags, and ensure the RPC "+
				"endpoint is reachable",
			networkFlag, err,
		))
	}

	// Create simulator runner
	runner, err := simulator.NewRunner("", false)
	if err != nil {
		return errors.WrapSimulatorNotFound(fmt.Sprintf(
			"failed to initialize simulator: %v\n"+
				"Run 'glassbox doctor --fix' to install or repair the simulator binary",
			err,
		))
	}

	// Create regression harness
	harness := simulator.NewRegressionHarness(runner, client, regressionMaxWorkers)
	harness.Verbose = verbose

	// Run the regression tests
	ctx := cmd.Context()

	var protVersion *uint32
	if regressionProtocolVersion > 0 {
		protVersion = &regressionProtocolVersion
	}

	suite, err := harness.RunRegressionTests(ctx, regressionTestCount, protVersion, regressionStartSeq)
	if err != nil {
		// Surface a clear message with remediation guidance.
		errMsg := err.Error()
		hint := ""
		switch {
		case strings.Contains(errMsg, "no failed transactions found"):
			hint = "\nTip: try a different --start-seq value or verify the network has recent failed transactions."
		case strings.Contains(errMsg, "fetch transaction hashes"):
			hint = "\nTip: check your --rpc-url and --network settings, or run 'glassbox doctor' to verify connectivity."
		}
		return errors.WrapValidationError(fmt.Sprintf("regression tests failed: %v%s", err, hint))
	}

	// Print summary
	fmt.Fprintln(out, "\n"+suite.Summary())

	// Print failed results if any
	failed := suite.FailedResults()
	if len(failed) > 0 {
		fmt.Fprintf(out, "\n%d test(s) failed:\n", len(failed))
		const maxDisplayed = 10
		for i, result := range failed {
			if i >= maxDisplayed {
				fmt.Fprintf(out, "  ... and %d more failures (re-run with --verbose for full output)\n",
					len(failed)-maxDisplayed)
				break
			}
			fmt.Fprintf(out, "  [%d] %s\n      Error: %s\n", i+1, result.TransactionHash, result.ErrorMessage)
		}
		return errors.WrapValidationError(fmt.Sprintf(
			"regression test completed with %d failure(s) out of %d tests\n"+
				"Run 'glassbox debug <tx-hash>' on any failing transaction for a detailed trace",
			len(failed), suite.TotalTests,
		))
	}

	fmt.Fprintln(out, "\nAll regression tests passed!")
	return nil
}

func init() {
	regressionTestCmd.Flags().IntVar(
		&regressionTestCount,
		"count",
		100,
		fmt.Sprintf("Number of historic failed transactions to test (1–%d)", maxRegressionCount),
	)

	regressionTestCmd.Flags().Uint32Var(
		&regressionStartSeq,
		"start-seq",
		0,
		"Starting ledger sequence number for fetching transactions (0 = most recent)",
	)

	regressionTestCmd.Flags().IntVar(
		&regressionMaxWorkers,
		"workers",
		4,
		"Number of parallel workers for testing (default: 4)",
	)

	regressionTestCmd.Flags().Uint32Var(
		&regressionProtocolVersion,
		"protocol-version",
		0,
		"Optional protocol version override for all tests (0 = use default)",
	)

	regressionTestCmd.Flags().StringVarP(
		&networkFlag,
		"network",
		"n",
		string(rpc.Mainnet),
		"Stellar network to fetch transactions from (testnet, mainnet, futurenet)",
	)

	regressionTestCmd.Flags().StringVar(
		&rpcURLFlag,
		"rpc-url",
		"",
		"Custom RPC URL (overrides the default for the selected network)",
	)

	regressionTestCmd.Flags().StringVar(
		&rpcTokenFlag,
		"rpc-token",
		"",
		"RPC authentication token (or set GLASSBOX_RPC_TOKEN)",
	)

	regressionTestCmd.Flags().BoolVarP(
		&verbose,
		"verbose",
		"v",
		false,
		"Enable verbose per-transaction progress output",
	)

	rootCmd.AddCommand(regressionTestCmd)
}
