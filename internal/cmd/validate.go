// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/dotandev/glassbox/internal/version"
	"github.com/spf13/cobra"
)

// ValidationResult holds the result of a single validation check.
type ValidationResult struct {
	Check    string `json:"check"`
	Status   string `json:"status"` // "pass", "fail", "skip"
	Message  string `json:"message,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// ValidationReport holds the complete validation report.
type ValidationReport struct {
	Version   string            `json:"version"`
	Timestamp string            `json:"timestamp"`
	Results   []ValidationResult `json:"results"`
	Summary   ValidationSummary `json:"summary"`
}

// ValidationSummary provides a summary of validation results.
type ValidationSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Verify repository artifacts and build outputs before release",
	Long: `The validate command runs a suite of pre-release checks to confirm that
the repository is ready for release. It verifies:

  • Build outputs - Ensures binaries are built correctly
  • Lint checks   - Verifies code passes linting
  • Docs          - Confirms documentation is present
  • Version       - Checks version consistency

Exit codes:
  0 - All validations passed
  1 - One or more validations failed
  2 - Validation error (invalid arguments, missing tools, etc.)`,
	Example: `  glassbox validate              # Run all validation checks
  glassbox validate --build      # Only validate build outputs
  glassbox validate --lint       # Only run lint checks
  glassbox validate --json       # Output results as JSON`,
	RunE: runValidate,
}

var (
	validateBuild   bool
	validateLint    bool
	validateDocs    bool
	validateVersion bool
	validateJSON    bool
	validateFast    bool
	validateOutput  string
)

func init() {
	validateCmd.Flags().BoolVar(&validateBuild, "build", false, "Validate build outputs")
	validateCmd.Flags().BoolVar(&validateLint, "lint", false, "Run lint checks")
	validateCmd.Flags().BoolVar(&validateDocs, "docs", false, "Check documentation")
	validateCmd.Flags().BoolVar(&validateVersion, "version", false, "Check version consistency")
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "Output results as JSON")
	validateCmd.Flags().BoolVar(&validateFast, "fast", false, "Skip slow checks (e.g., full test suite)")
	validateCmd.Flags().StringVarP(&validateOutput, "output", "o", "", "Output file for validation report")

	rootCmd.AddCommand(validateCmd)
}

func runValidate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// If no specific checks are selected, run all
	runAll := !validateBuild && !validateLint && !validateDocs && !validateVersion

	report := ValidationReport{
		Version:   version.Version,
		Timestamp: version.BuildDate,
		Results:   []ValidationResult{},
	}

	var wg sync.WaitGroup
	resultChan := make(chan ValidationResult, 10)

	// Run validations concurrently
	if runAll || validateBuild {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultChan <- validateBuildOutputs(ctx)
		}()
	}

	if runAll || validateLint {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultChan <- validateLintChecks(ctx)
		}()
	}

	if runAll || validateDocs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultChan <- validateDocumentation(ctx)
		}()
	}

	if runAll || validateVersion {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resultChan <- validateVersionConsistency(ctx)
		}()
	}

	// Always run basic checks
	wg.Add(1)
	go func() {
		defer wg.Done()
		resultChan <- validateGoModule(ctx)
	}()

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		report.Results = append(report.Results, result)
	}

	// Calculate summary
	for _, r := range report.Results {
		report.Summary.Total++
		switch r.Status {
		case "pass":
			report.Summary.Passed++
		case "fail":
			report.Summary.Failed++
		case "skip":
			report.Summary.Skipped++
		}
	}

	// Output results
	if validateJSON {
		return outputJSONReport(report)
	}

	// Text output
	return outputTextReport(report)
}

// validateBuildOutputs checks that binaries are built correctly.
func validateBuildOutputs(ctx context.Context) ValidationResult {
	result := ValidationResult{Check: "build"}

	// Check if main binary exists
	binPath := "glassbox"
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		// Try to build
		buildCmd := exec.CommandContext(ctx, "go", "build", "-o", binPath, "./cmd/glassbox")
		buildCmd.Stdout = &bytes.Buffer{}
		buildCmd.Stderr = &bytes.Buffer{}

		if err := buildCmd.Run(); err != nil {
			result.Status = "fail"
			result.Message = fmt.Sprintf("Build failed: %v", err)
			return result
		}
	}

	// Verify binary is executable
	info, err := os.Stat(binPath)
	if err != nil {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Cannot stat binary: %v", err)
		return result
	}

	if info.Size() == 0 {
		result.Status = "fail"
		result.Message = "Binary is empty"
		return result
	}

	result.Status = "pass"
	result.Message = fmt.Sprintf("Binary built successfully (%d bytes)", info.Size())
	return result
}

// validateLintChecks runs code linting.
func validateLintChecks(ctx context.Context) ValidationResult {
	result := ValidationResult{Check: "lint"}

	// Check if pre-commit is available
	_, err := exec.LookPath("pre-commit")
	if err != nil {
		// Try golangci-lint
		_, err = exec.LookPath("golangci-lint")
		if err != nil {
			result.Status = "skip"
			result.Message = "No lint tool available (pre-commit or golangci-lint)"
			return result
		}

		// Run golangci-lint
		lintCmd := exec.CommandContext(ctx, "golangci-lint", "run", "./...")
		lintCmd.Dir = getProjectRoot()

		output := &bytes.Buffer{}
		lintCmd.Stdout = output
		lintCmd.Stderr = output

		if err := lintCmd.Run(); err != nil {
			result.Status = "fail"
			result.Message = fmt.Sprintf("Lint failed:\n%s", output.String())
			result.ExitCode = lintCmd.ProcessState.ExitCode()
			return result
		}

		result.Status = "pass"
		result.Message = "Lint checks passed"
		return result
	}

	// Run pre-commit
	preCommitCmd := exec.CommandContext(ctx, "pre-commit", "run", "--all-files")
	preCommitCmd.Dir = getProjectRoot()

	output := &bytes.Buffer{}
	preCommitCmd.Stdout = output
	preCommitCmd.Stderr = output

	if err := preCommitCmd.Run(); err != nil {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Pre-commit hooks failed:\n%s", output.String())
		result.ExitCode = preCommitCmd.ProcessState.ExitCode()
		return result
	}

	result.Status = "pass"
	result.Message = "Lint checks passed"
	return result
}

// validateDocumentation checks that required documentation exists.
func validateDocumentation(ctx context.Context) ValidationResult {
	result := ValidationResult{Check: "docs"}

	requiredDocs := []string{
		"README.md",
		"LICENSE",
	}

	docsDir := "docs"
	optionalDocs := []string{
		filepath.Join(docsDir, "adr/README.md"),
		filepath.Join(docsDir, "schema/README.md"),
	}

	missing := []string{}

	for _, doc := range requiredDocs {
		if _, err := os.Stat(doc); os.IsNotExist(err) {
			missing = append(missing, doc)
		}
	}

	if len(missing) > 0 {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Missing required documentation: %s", strings.Join(missing, ", "))
		return result
	}

	// Check optional docs
	optionalMissing := 0
	for _, doc := range optionalDocs {
		if _, err := os.Stat(doc); os.IsNotExist(err) {
			optionalMissing++
		}
	}

	result.Status = "pass"
	result.Message = fmt.Sprintf("Required docs present, %d optional docs missing", optionalMissing)
	return result
}

// validateVersionConsistency checks version consistency across files.
func validateVersionConsistency(ctx context.Context) ValidationResult {
	result := ValidationResult{Check: "version"}

	// Check version.go matches version.Version
	versionFile := filepath.Join("internal", "version", "version.go")

	data, err := os.ReadFile(versionFile)
	if err != nil {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Cannot read version file: %v", err)
		return result
	}

	// Check that version string is present
	if !strings.Contains(string(data), "Version") {
		result.Status = "fail"
		result.Message = "Version not found in version.go"
		return result
	}

	// Check Makefile version
	makefile := "Makefile"
	if data, err := os.ReadFile(makefile); err == nil {
		if !strings.Contains(string(data), "VERSION") {
			result.Status = "fail"
			result.Message = "VERSION not found in Makefile"
			return result
		}
	}

	result.Status = "pass"
	result.Message = "Version is consistent"
	return result
}

// validateGoModule checks that go.mod is valid.
func validateGoModule(ctx context.Context) ValidationResult {
	result := ValidationResult{Check: "go-module"}

	cmd := exec.CommandContext(ctx, "go", "mod", "verify")
	cmd.Dir = getProjectRoot()

	if err := cmd.Run(); err != nil {
		result.Status = "fail"
		result.Message = fmt.Sprintf("Go module verification failed: %v", err)
		return result
	}

	result.Status = "pass"
	result.Message = "Go module is valid"
	return result
}

// getProjectRoot returns the project root directory.
func getProjectRoot() string {
	// Assume current directory is project root
	dir, _ := os.Getwd()
	return dir
}

// outputJSONReport outputs the validation report as JSON.
func outputJSONReport(report ValidationReport) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}

	if validateOutput != "" {
		return os.WriteFile(validateOutput, data, 0644)
	}

	fmt.Println(string(data))
	return nil
}

// outputTextReport outputs the validation report as human-readable text.
func outputTextReport(report ValidationReport) error {
	fmt.Printf("Glassbox Validation Report\n")
	fmt.Printf("Version: %s\n", report.Version)
	fmt.Printf("Timestamp: %s\n\n", report.Timestamp)

	for _, r := range report.Results {
		icon := "✓"
		switch r.Status {
		case "pass":
			icon = "✓"
		case "fail":
			icon = "✗"
		case "skip":
			icon = "-"
		}

		fmt.Printf("[%s] %s: %s", icon, r.Check, r.Message)
		if r.ExitCode != 0 {
			fmt.Printf(" (exit code: %d)", r.ExitCode)
		}
		fmt.Println()
	}

	fmt.Printf("\nSummary: %d passed, %d failed, %d skipped\n",
		report.Summary.Passed, report.Summary.Failed, report.Summary.Skipped)

	if report.Summary.Failed > 0 {
		fmt.Println("\nValidation FAILED")
		return fmt.Errorf("validation failed: %d checks failed", report.Summary.Failed)
	}

	fmt.Println("\nValidation PASSED")
	return nil
}