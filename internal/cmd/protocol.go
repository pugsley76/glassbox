// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dotandev/glassbox/internal/clioutput"
	"github.com/dotandev/glassbox/internal/deeplink"
	"github.com/dotandev/glassbox/internal/protocolreg"
	"github.com/spf13/cobra"
)

// protocolDiagnoseJSON controls whether protocol:diagnose emits JSON output.
var protocolDiagnoseJSON bool

// protocolDiagnoseFormat controls the output format for protocol:diagnose ("text" or "json").
var protocolDiagnoseFormat string

// protocolVerifyProbe runs an optional glassbox://doctor-probe handler check.
var protocolVerifyProbe bool

// protocolRegisterDryRun previews what register would do without modifying OS state.
var protocolRegisterDryRun bool

var protocolRegisterCmd = &cobra.Command{
	Use:     "protocol:register",
	Aliases: []string{"pb:register"},
	Short:   "Register the glassbox:// protocol handler in the operating system",
	Long: `Register the glassbox:// URI scheme so the OS dispatches deep links to Glassbox.

Use --dry-run to preview what would be written without modifying any system state.`,
	GroupID: "utility",
	RunE: func(cmd *cobra.Command, args []string) error {
		registrar, err := protocolreg.NewRegistrar()
		if err != nil {
			return fmt.Errorf("initialise registrar: %w\n  Fix: ensure the glassbox binary is installed and accessible", err)
		}

		if protocolRegisterDryRun {
			diag := registrar.Diagnose()
			if diag.Status == protocolreg.StatusOK {
				fmt.Fprintf(cmd.OutOrStdout(), "[DRY-RUN] Protocol handler is already registered — no changes needed.\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "[DRY-RUN] Would register %s:// handler on %s.\n", protocolreg.Scheme, diag.Platform)
				fmt.Fprintf(cmd.OutOrStdout(), "[DRY-RUN] Current status: %s\n", diag.Status)
				if len(diag.Issues) > 0 {
					fmt.Fprintf(cmd.OutOrStdout(), "[DRY-RUN] Issues to fix:\n")
					for _, issue := range diag.Issues {
						fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", issue)
					}
				}
			}
			return nil
		}

		if err := registrar.Register(); err != nil {
			return fmt.Errorf("%w\n  Tip: run 'glassbox protocol:diagnose' for a detailed breakdown, or 'glassbox protocol:repair' to attempt automatic repair", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Registered GLASSBOX Protocol handler for %s://\n", protocolreg.Scheme)
		fmt.Fprintln(cmd.OutOrStdout(), "Tip: run 'glassbox protocol:verify' to confirm the registration is working.")
		return nil
	},
}

var protocolUnregisterCmd = &cobra.Command{
	Use:     "protocol:unregister",
	Aliases: []string{"pb:unregister"},
	Short:   "Unregister the glassbox:// protocol handler from the operating system",
	GroupID: "utility",
	RunE: func(cmd *cobra.Command, args []string) error {
		registrar, err := protocolreg.NewRegistrar()
		if err != nil {
			return err
		}
		if err := registrar.Unregister(); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Unregistered GLASSBOX Protocol handler")
		return nil
	},
}

var protocolStatusCmd = &cobra.Command{
	Use:     "protocol:status",
	Aliases: []string{"pb:status"},
	Short:   "Check current registration status of the glassbox:// protocol handler",
	GroupID: "utility",
	RunE: func(cmd *cobra.Command, args []string) error {
		registrar, err := protocolreg.NewRegistrar()
		if err != nil {
			return err
		}

		diag := registrar.Diagnose()
		for _, check := range diag.Checks {
			fmt.Fprintf(cmd.OutOrStdout(), "[OK] %s\n", check)
		}

		if diag.Status == protocolreg.StatusOK {
			fmt.Fprintln(cmd.OutOrStdout(), "GLASSBOX Protocol handler is currently REGISTERED")
			return nil
		}

		fmt.Fprintln(cmd.ErrOrStderr(), "GLASSBOX Protocol handler is NOT REGISTERED")
		if len(diag.RemediationSteps) > 0 {
			fmt.Fprintln(cmd.ErrOrStderr(), "\nTo register the protocol handler:")
			for i, step := range diag.RemediationSteps {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %d. %s\n", i+1, step)
			}
		}
		return fmt.Errorf("GLASSBOX Protocol handler is NOT REGISTERED")
	},
}

var protocolVerifyCmd = &cobra.Command{
	Use:     "protocol:verify",
	Aliases: []string{"pb:verify", "verify-protocol-registration"},
	Short:   "Verify the native OS registration for the glassbox:// protocol handler",
	Long: `Verify that the glassbox:// protocol handler is registered correctly on this system.

The command inspects OS registration artefacts and reports pass/fail for each check.
Use --probe to simulate a glassbox://doctor-probe deep link and confirm the handler
responds without side effects.

On failure, remediation steps are printed to help repair the registration.`,
	GroupID: "utility",
	RunE: func(cmd *cobra.Command, args []string) error {
		registrar, err := protocolreg.NewRegistrar()
		if err != nil {
			return err
		}

		report, err := registrar.Verify()
		for _, check := range report.Checks {
			fmt.Fprintf(cmd.OutOrStdout(), "[OK] %s\n", check)
		}
		for _, issue := range report.Issues {
			fmt.Fprintf(cmd.ErrOrStderr(), "[FAIL] %s\n", issue)
		}
		probePassed := true
		if protocolVerifyProbe {
			exePath, exeErr := os.Executable()
			if exeErr != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "[FAIL] Handler probe: cannot resolve executable: %v\n", exeErr)
				probePassed = false
			} else if deeplink.ProbeHandler(exePath) {
				fmt.Fprintf(cmd.OutOrStdout(), "[OK] Handler probe (%s) exited cleanly\n", deeplink.MockURL)
			} else {
				probePassed = false
				fmt.Fprintf(cmd.ErrOrStderr(), "[FAIL] Handler probe (%s) did not complete successfully\n", deeplink.MockURL)
				fmt.Fprintf(cmd.ErrOrStderr(), "       Ensure the binary handles --deep-link %s\n", deeplink.MockURL)
			}
		}

		if err != nil || !probePassed {
			diag := registrar.Diagnose()
			if len(diag.RemediationSteps) > 0 {
				fmt.Fprintf(cmd.ErrOrStderr(), "\nRemediation steps:\n")
				for i, step := range diag.RemediationSteps {
					fmt.Fprintf(cmd.ErrOrStderr(), "  %d. %s\n", i+1, step)
				}
			}
		}

		if err != nil {
			return err
		}
		if !probePassed {
			return fmt.Errorf("protocol handler probe failed")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Verified GLASSBOX Protocol registration on %s (%dms)\n", report.Platform, report.ElapsedMs)
		return nil
	},
}

var protocolHandlerCmd = &cobra.Command{
	Use:     "protocol:handle <uri>",
	Aliases: []string{"protocol-handler", "pb:handle"},
	Short:   "Handle an glassbox:// protocol URI and dispatch it to the debugger",
	GroupID: "utility",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parsed, err := protocolreg.ParseDebugURI(args[0])
		if err != nil {
			return fmt.Errorf(
				"%w\n"+
					"  Expected format: glassbox://debug/<64-char-hex>?network=<testnet|mainnet|futurenet>[&op=<n>][&view=<mode>]\n"+
					"  Run 'glassbox protocol:handle --help' for full parameter documentation",
				err,
			)
		}

		executablePath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("resolve executable path: %w", err)
		}

		debugArgs := []string{"debug", parsed.TransactionHash, "--network", parsed.Network}

		// Forward the operation index when present (prefer Op, fall back to Operation).
		opIdx := parsed.Op
		if opIdx == nil {
			opIdx = parsed.Operation
		}
		if opIdx != nil {
			debugArgs = append(debugArgs, "--op", fmt.Sprintf("%d", *opIdx))
		}

		// Forward the view mode when present.
		if parsed.View != "" {
			debugArgs = append(debugArgs, "--view", parsed.View)
		}

		child := exec.CommandContext(cmd.Context(), executablePath, debugArgs...)
		child.Stdout = cmd.OutOrStdout()
		child.Stderr = cmd.ErrOrStderr()
		return child.Run()
	},
}

var protocolDiagnoseCmd = &cobra.Command{
	Use:     "protocol:diagnose",
	Aliases: []string{"pb:diagnose"},
	Short:   "Inspect the glassbox:// protocol registration and report root causes",
	GroupID: "utility",
	Long: `Inspect the glassbox:// protocol registration on the current OS and produce
a structured diagnostic report.

The command checks:
  • Whether the protocol handler is registered with the OS
  • Whether the registered handler points to the current executable
  • Platform-specific registration artefacts (.desktop file, registry key, app bundle)
  • xdg-mime / LaunchServices / registry consistency

Exit codes:
  0  — registration is healthy
  1  — registration is missing or broken (issues are printed to stderr)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		registrar, err := protocolreg.NewRegistrar()
		if err != nil {
			return err
		}

		report := registrar.Diagnose()

		// Validate and resolve the output format.
		normalizedFormat := strings.ToLower(strings.TrimSpace(protocolDiagnoseFormat))
		if normalizedFormat != "" && normalizedFormat != "text" && normalizedFormat != "json" {
			return fmt.Errorf("invalid --format %q: must be 'text' or 'json'", protocolDiagnoseFormat)
		}
		if clioutput.WantsJSON(protocolDiagnoseJSON, normalizedFormat) {
			return clioutput.Write(cmd.OutOrStdout(), "protocol:diagnose", report)
		}

		for _, check := range report.Checks {
			fmt.Fprintf(cmd.OutOrStdout(), "[OK]   %s\n", check)
		}
		for _, issue := range report.Issues {
			fmt.Fprintf(cmd.ErrOrStderr(), "[FAIL] %s\n", issue)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "\nStatus: %s  (platform: %s, scheme: %s://)\n",
			report.Status, report.Platform, report.Scheme)

		if report.RegisteredHandler != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Registered handler: %s\n", report.RegisteredHandler)
			if report.HandlerMatchesSelf {
				fmt.Fprintf(cmd.OutOrStdout(), "Handler matches current executable: yes\n")
			} else if report.ConflictDetected {
				// Conflict: a foreign binary owns the registration.
				fmt.Fprintf(cmd.ErrOrStderr(),
					"Handler matches current executable: NO (conflict — registered handler is %s)\n",
					report.ConflictingHandler)
				fmt.Fprintf(cmd.ErrOrStderr(),
					"⚠  Protocol conflict detected: the glassbox:// scheme is currently handled by\n"+
						"   a different binary (%s).\n"+
						"   Run 'glassbox protocol:repair' to reclaim the registration.\n",
					report.ConflictingHandler)
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Handler matches current executable: NO (stale path)\n")
			}
		}

		if len(report.RemediationSteps) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nRemediation steps:\n")
			for i, step := range report.RemediationSteps {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %d. %s\n", i+1, step)
			}
		}

		if report.Status != protocolreg.StatusOK {
			if report.ConflictDetected {
				return fmt.Errorf(
					"protocol registration conflict: glassbox:// is claimed by %s — run 'glassbox protocol:repair' to resolve",
					report.ConflictingHandler,
				)
			}
			return fmt.Errorf("protocol registration is %s", report.Status)
		}
		return nil
	},
}

var protocolRepairCmd = &cobra.Command{
	Use:     "protocol:repair",
	Aliases: []string{"pb:repair"},
	Short:   "Repair a broken or missing glassbox:// protocol registration",
	GroupID: "utility",
	Long: `Attempt to repair the glassbox:// protocol handler registration.

The command first runs a diagnostic check to understand what is broken, then
re-registers the protocol handler using the best available platform mechanism:

  • Linux  — recreates the .desktop file and updates xdg-mime
  • macOS  — rebuilds the app bundle and re-registers with LaunchServices
  • Windows — updates the HKEY_CURRENT_USER registry keys

After repair, a verification pass confirms the fix was successful.

PERMISSION NOTES
  On Windows, registry writes to HKEY_CURRENT_USER do not require elevation.
  On Linux and macOS, the handler is installed per-user (~/.local/share or
  ~/Applications) and does not require root.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		registrar, err := protocolreg.NewRegistrar()
		if err != nil {
			return err
		}

		result := registrar.Repair()

		for _, action := range result.Actions {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", action)
		}

		if result.PermissionHint != "" {
			fmt.Fprintf(cmd.ErrOrStderr(), "\nPermission hint: %s\n", result.PermissionHint)
		}

		if result.Err != nil {
			return result.Err
		}

		if result.Repaired {
			fmt.Fprintf(cmd.OutOrStdout(), "\nRepair successful. The %s:// protocol handler is now registered.\n",
				protocolreg.Scheme)
		}
		return nil
	},
}

func init() {
	protocolDiagnoseCmd.Flags().BoolVar(&protocolDiagnoseJSON, "json", false,
		"Emit diagnostic report as JSON (shorthand for --format json)")
	protocolDiagnoseCmd.Flags().StringVar(&protocolDiagnoseFormat, "format", "",
		"Output format: 'text' (default) or 'json'")
	protocolVerifyCmd.Flags().BoolVar(&protocolVerifyProbe, "probe", false,
		"Run a dry-run glassbox://doctor-probe handler check after registration verification")
	protocolRegisterCmd.Flags().BoolVar(&protocolRegisterDryRun, "dry-run", false,
		"Preview what would be registered without modifying OS state")

	rootCmd.AddCommand(protocolRegisterCmd)
	rootCmd.AddCommand(protocolUnregisterCmd)
	rootCmd.AddCommand(protocolStatusCmd)
	rootCmd.AddCommand(protocolVerifyCmd)
	rootCmd.AddCommand(protocolHandlerCmd)
	rootCmd.AddCommand(protocolDiagnoseCmd)
	rootCmd.AddCommand(protocolRepairCmd)
}
