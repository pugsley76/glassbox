// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dotandev/glassbox/internal/authtrace"
	"github.com/dotandev/glassbox/internal/config"
	"github.com/dotandev/glassbox/internal/errors"
	"github.com/dotandev/glassbox/internal/logger"
	"github.com/dotandev/glassbox/internal/rpc"
	"github.com/spf13/cobra"
)

var (
	authNetworkFlag    string
	authRPCURLFlag     string
	authDetailedFlag   bool
	authJSONOutputFlag bool
)

var authDebugCmd = &cobra.Command{
	Use:     "auth-debug <transaction-hash>",
	GroupID: "core",
	Short:   "Debug multi-signature and threshold-based authorization failures",
	Long: `Analyze multi-signature authorization flows and identify which signatures or thresholds failed.

The transaction hash, --network, and --rpc-url are all validated before any
network call is made, so malformed input fails fast with an explicit message
instead of a low-level RPC error.

Examples:
  glassbox auth-debug <tx-hash>
  glassbox auth-debug --detailed <tx-hash>
  glassbox auth-debug --json <tx-hash>
  glassbox auth-debug --network testnet <tx-hash>`,
	Args:    cobra.ExactArgs(1),
	PreRunE: authDebugPreRunE,
	RunE:    authDebugRunE,
}

// authDebugPreRunE validates all inputs before any network connection is made.
// Invalid conditions (malformed transaction hash, unsupported network, or a
// badly formed --rpc-url) are rejected here with explicit, actionable messages.
func authDebugPreRunE(cmd *cobra.Command, args []string) error {
	// Validate the transaction hash and --rpc-url up front, before touching the
	// network. validateNetworkName is re-checked below against the possibly
	// auto-resolved network value.
	if err := validateAuthDebugInputs(args[0], authNetworkFlag, authRPCURLFlag); err != nil {
		return err
	}

	// When --network was not explicitly provided, try to auto-detect it from the
	// transaction. This only runs after the hash is known to be well-formed.
	if !cmd.Flags().Changed("network") {
		token := authResolveRPCToken()
		probeCtx, probeCancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer probeCancel()
		if resolved, err := rpc.ResolveNetwork(probeCtx, args[0], token); err == nil {
			authNetworkFlag = string(resolved)
			fmt.Printf("Resolved network: %s\n", authNetworkFlag)
		}
	}

	return validateNetworkName(authNetworkFlag)
}

// validateAuthDebugInputs performs the input validation for the auth-debug
// command without any network access, so it can be exercised directly in tests.
// It rejects, in order: a malformed transaction hash, a badly formed --rpc-url,
// and an unknown network name.
func validateAuthDebugInputs(txHash, network, rpcURL string) error {
	if err := rpc.ValidateTransactionHash(txHash); err != nil {
		return errors.WrapValidationError(fmt.Sprintf(
			"invalid transaction hash %q: %v\n"+
				"  Transaction hashes must be exactly 64 hexadecimal characters.\n"+
				"  Example: glassbox auth-debug 5c0a1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab",
			txHash, err))
	}

	if rpcURL != "" {
		if err := validateRPCURL(rpcURL); err != nil {
			return errors.WrapValidationError(fmt.Sprintf(
				"--rpc-url %q is not valid: %v\n"+
					"  Provide an http(s) URL, e.g. --rpc-url https://horizon-testnet.stellar.org",
				rpcURL, err))
		}
	}

	if err := validateNetworkName(network); err != nil {
		return err
	}

	return nil
}

// authResolveRPCToken resolves the RPC token from the environment first and then
// from config, matching the precedence used elsewhere in the CLI.
func authResolveRPCToken() string {
	if token := os.Getenv("GLASSBOX_RPC_TOKEN"); token != "" {
		return token
	}
	if cfg, err := config.LoadConfig(); err == nil && cfg.RPCToken != "" {
		return cfg.RPCToken
	}
	return ""
}

func authDebugRunE(cmd *cobra.Command, args []string) error {
	txHash := args[0]

	// --detailed only affects the human-readable report; in JSON mode the full
	// detail is always emitted. Surface the no-op so the user is not surprised.
	if authJSONOutputFlag && authDetailedFlag {
		fmt.Fprintln(os.Stderr, "note: --detailed has no effect together with --json; JSON output already includes full detail")
	}

	opts := []rpc.ClientOption{rpc.WithToken(authResolveRPCToken())}
	if authRPCURLFlag != "" {
		opts = append(opts, rpc.WithHorizonURL(authRPCURLFlag))
	}

	// networkClientOptions already returns an explicit, actionable error for an
	// unknown network, so it is propagated unchanged rather than re-wrapped.
	client, err := newClientForNetwork(authNetworkFlag, opts...)
	if err != nil {
		return err
	}

	logger.Logger.Info("Fetching transaction for auth analysis", "tx_hash", txHash)

	// GetTransaction already returns precise, hinted errors (transaction not
	// found vs. RPC connection failure). Propagate them verbatim instead of
	// flattening every failure into a generic "connection failed".
	resp, err := client.GetTransaction(cmd.Context(), txHash)
	if err != nil {
		return err
	}

	if len(resp.EnvelopeXdr) == 0 {
		return errors.WrapValidationError(fmt.Sprintf(
			"transaction %s was fetched but its envelope is empty; authorization cannot be analyzed\n"+
				"  Confirm the hash is correct and that --network matches where the transaction was submitted, then retry.",
			txHash))
	}

	fmt.Printf("Transaction Envelope: %d bytes\n", len(resp.EnvelopeXdr))

	traceConfig := authtrace.Config{
		TraceCustomContracts: true,
		CaptureSigDetails:    true,
		MaxEventDepth:        1000,
	}

	tracker := authtrace.NewTracker(traceConfig)
	trace := tracker.GenerateTrace()
	reporter := authtrace.NewDetailedReporter(trace)

// When no authorization events were extracted, the report's "SUCCEEDED"
// status only means "no failures were recorded" — not that authorization was
// verified. Make that explicit so the output is not misread as a pass.
// Also surface diagnostic hints from the trace, including source mapping guidance.
if !authTraceHasData(trace) {
	fmt.Fprintln(os.Stderr, emptyAuthTraceNote(txHash))
	if trace.Diagnostics != nil && trace.Diagnostics.EmptyTraceReason != "" {
		fmt.Fprintf(os.Stderr, "  Detail: %s\n", trace.Diagnostics.EmptyTraceReason)
	}
}

// When some events were recorded but source mapping is missing, provide
// a targeted hint so users can improve trace-to-source correlation.
if trace.Diagnostics != nil && !trace.Diagnostics.SourceMappingAvailable && len(trace.AuthEvents) > 0 {
	fmt.Fprintf(os.Stderr, "  Hint: %s\n", trace.Diagnostics.SourceMappingHint)
}

	if authJSONOutputFlag {
		jsonStr, err := reporter.GenerateJSONString()
		if err != nil {
			return errors.WrapMarshalFailed(err)
		}
		fmt.Println(jsonStr)
		return nil
	}

	fmt.Println(reporter.GenerateReport())
	if authDetailedFlag {
		printDetailedAnalysis(reporter)
	}

	return nil
}

// authTraceHasData reports whether the trace carries any authorization signal
// (events or recorded failures). An empty trace produces a misleading
// "SUCCEEDED" report, so callers use this to attach an explanatory note.
func authTraceHasData(trace *authtrace.AuthTrace) bool {
	if trace == nil {
		return false
	}
	return len(trace.AuthEvents) > 0 || len(trace.Failures) > 0
}

// emptyAuthTraceNote returns the diagnostic shown when no authorization data was
// extracted from a transaction, so an empty "SUCCEEDED" report is not mistaken
// for a verified-successful authorization.
func emptyAuthTraceNote(txHash string) string {
	return fmt.Sprintf(
		"warning: no authorization events were extracted from transaction %s.\n"+
			"  The report below reflects \"no failures recorded\", not a verified-successful authorization.\n"+
			"  This is expected for transactions that contain no Soroban authorization entries.\n"+
			"  Verify the hash and --network, or run 'glassbox doctor' if you expected auth data.",
		txHash)
}

func printDetailedAnalysis(reporter *authtrace.DetailedReporter) {
	metrics := reporter.SummaryMetrics()
	fmt.Println("\n--- SUMMARY METRICS ---")
	for key, value := range metrics {
		fmt.Printf("%s: %v\n", key, value)
	}

	missingKeys := reporter.IdentifyMissingKeys()
	if len(missingKeys) > 0 {
		fmt.Println("\n--- MISSING SIGNATURES ---")
		for _, signer := range missingKeys {
			fmt.Printf("  - %s (required weight: %d)\n", signer.SignerKey, signer.Weight)
		}
	}
}

func init() {
	authDebugCmd.Flags().StringVarP(&authNetworkFlag, "network", "n", string(rpc.Mainnet), "Stellar network (testnet, mainnet, futurenet)")
	authDebugCmd.Flags().StringVar(&authRPCURLFlag, "rpc-url", "", "Custom Horizon RPC URL")
	authDebugCmd.Flags().BoolVar(&authDetailedFlag, "detailed", false, "Show detailed analysis and missing signatures")
	authDebugCmd.Flags().BoolVar(&authJSONOutputFlag, "json", false, "Output as JSON")

	_ = authDebugCmd.RegisterFlagCompletionFunc("network", completeNetworkFlag)

	rootCmd.AddCommand(authDebugCmd)
}
