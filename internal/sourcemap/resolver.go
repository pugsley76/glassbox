// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package sourcemap

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotandev/glassbox/internal/dwarf"
	"github.com/dotandev/glassbox/internal/logger"
)

// Resolver coordinates fetching verified source code from a registry,
// with optional local caching and auto-discovery of local DWARF symbols.
type Resolver struct {
	registry        *RegistryClient
	cache           *SourceCache
	githubRetriever *GitHubRetriever
	// contractSourceOverride is an explicit local path to the contract source
	// directory. When set, it is used as a fallback before prompting the user.
	contractSourceOverride string
	// aliasResolver translates workspace-relative path prefixes to real
	// filesystem paths before source files are opened.
	aliasResolver *AliasResolver
	// nonInteractive disables the stdin prompt so Resolve never blocks in CI
	// pipelines or automated environments. When true and all automatic stages
	// fail, Resolve returns ErrSourceNotFound instead of reading from stdin.
	nonInteractive bool
}

// ErrSourceNotFound is a sentinel returned by Resolve when all discovery
// stages are exhausted and no interactive prompt is available (e.g. CI mode).
// Callers should check for it with errors.Is and surface a diagnostic.
var ErrSourceNotFound = fmt.Errorf("contract source not found: all discovery stages exhausted")

// ResolverOption is a functional option for configuring the Resolver.
type ResolverOption func(*Resolver)

// WithCache enables caching with the specified directory.
func WithCache(cacheDir string) ResolverOption {
	return func(r *Resolver) {
		cache, err := NewSourceCache(filepath.Join(cacheDir, "sourcemap"))
		if err != nil {
			logger.Logger.Warn("Failed to create source cache, caching disabled", "error", err)
			return
		}
		r.cache = cache
	}
}

// WithRegistryClient sets a custom registry client.
func WithRegistryClient(rc *RegistryClient) ResolverOption {
	return func(r *Resolver) {
		r.registry = rc
	}
}

// WithContractSource sets an explicit local path to the contract source
// directory. When automatic discovery fails, this path is used before
// prompting the user interactively (Issue #117).
func WithContractSource(path string) ResolverOption {
	return func(r *Resolver) {
		r.contractSourceOverride = path
	}
}

// WithAliasResolver sets an AliasResolver that translates workspace-relative
// path prefixes to real filesystem paths when resolving source file locations.
func WithAliasResolver(ar *AliasResolver) ResolverOption {
	return func(r *Resolver) {
		r.aliasResolver = ar
	}
}

// WithNonInteractive disables the stdin prompt so Resolve never blocks
// waiting for user input. Use this in CI pipelines and automated environments.
// When all discovery stages fail in non-interactive mode, Resolve returns
// ErrSourceNotFound instead of prompting.
func WithNonInteractive() ResolverOption {
	return func(r *Resolver) {
		r.nonInteractive = true
	}
}

// NewResolver creates a Resolver with the given options.
func NewResolver(opts ...ResolverOption) *Resolver {
	r := &Resolver{
		registry: NewRegistryClient(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Resolve attempts to find verified source code for the given contract ID
// through a multi-stage discovery pipeline:
//
//  1. Local cache
//  2. Registry (stellar.expert)
//  3. GitHub retriever (when configured)
//  4. --contract-source override (Issue #117)
//  5. Interactive stdin prompt — or ErrSourceNotFound in non-interactive mode
//
// Failure modes:
//   - Invalid contractID: returns a validation error immediately.
//   - All stages exhausted, non-interactive mode: returns ErrSourceNotFound.
//   - Prompt read failure (interactive): returns a wrapped I/O error.
//   - Returns (nil, nil) only when the user provides an empty path at the
//     interactive prompt (explicit opt-out of source mapping).
func (r *Resolver) Resolve(ctx context.Context, contractID string) (*SourceCode, error) {
	if err := validateContractID(contractID); err != nil {
		return nil, fmt.Errorf("invalid contract ID %q: %w\n"+
			"  Contract IDs must start with 'C' and be exactly 56 characters long.", contractID, err)
	}

	// 1. Check cache first.
	if r.cache != nil {
		if cached := r.cache.Get(contractID); cached != nil {
			logger.Logger.Info("Source resolved from cache", "contract_id", contractID)
			return cached, nil
		}
	}

	// 2. Fetch from registry.
	source, err := r.registry.FetchVerifiedSource(ctx, contractID)
	if err != nil {
		logger.Logger.Debug("Registry lookup failed", "contract_id", contractID, "error", err)
	}

	// 3. GitHub fallback.
	if source == nil && r.githubRetriever != nil {
		ghSource, ghErr := r.githubRetriever.Retrieve(ctx, contractID)
		if ghErr != nil {
			logger.Logger.Debug("GitHub source retrieval failed",
				"contract_id", contractID, "error", ghErr)
		} else if ghSource != nil {
			logger.Logger.Info("Source resolved from GitHub",
				"contract_id", contractID,
				"repository", ghSource.Repository,
				"file_count", len(ghSource.Files),
			)
			if r.cache != nil {
				if cacheErr := r.cache.Put(ghSource); cacheErr != nil {
					logger.Logger.Warn("Failed to cache GitHub source",
						"contract_id", contractID, "error", cacheErr)
				}
			}
			return ghSource, nil
		}
	}

	// 4. Fallback: use explicit override path if provided (Issue #117).
	//    Validate it here so callers get actionable errors rather than a
	//    silent no-op if the path is wrong.
	if source == nil && r.contractSourceOverride != "" {
		overridePath := strings.TrimSpace(r.contractSourceOverride)
		if overridePath == "" {
			return nil, fmt.Errorf(
				"--contract-source: value must not be empty or whitespace\n" +
					"  Provide the path to your contract's source directory (the one containing src/).",
			)
		}
		info, statErr := os.Stat(overridePath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, fmt.Errorf(
					"--contract-source: directory not found: %q\n"+
						"  Provide the path to your contract's source directory (the one containing src/).\n"+
						"  Source mapping will be unavailable without a valid path.",
					overridePath,
				)
			}
			return nil, fmt.Errorf(
				"--contract-source: cannot access %q: %w", overridePath, statErr)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf(
				"--contract-source: %q is a file, not a directory\n"+
					"  Provide the path to your contract's source directory, not a file.",
				overridePath,
			)
		}
		logger.Logger.Info("Using --contract-source override for source mapping",
			"contract_id", contractID,
			"path", overridePath,
		)
		return &SourceCode{
			ContractID: contractID,
			Repository: overridePath,
			Files:      map[string]string{},
			FetchedAt:  time.Now(),
		}, nil
	}

	// 5. Prompt or non-interactive failure.
	if source == nil {
		if r.nonInteractive {
			logger.Logger.Warn("Source discovery exhausted all stages (non-interactive mode)",
				"contract_id", contractID)
			return nil, fmt.Errorf(
				"%w for contract %q\n"+
					"  Stages tried: cache, registry (stellar.expert), GitHub retriever, --contract-source override\n"+
					"  To resolve: provide --contract-source <path> pointing to the contract source directory,\n"+
					"  or verify the contract on stellar.expert to enable registry lookup.\n"+
					"  Use --skip-source-mapping to proceed without source mapping.",
				ErrSourceNotFound, contractID,
			)
		}

		logger.Logger.Info("Contract source unresolved automatically; prompting user",
			"contract_id", contractID)
		manualPath, promptErr := r.PromptForWasmPath()
		if promptErr != nil {
			return nil, fmt.Errorf(
				"failed to read manual WASM path from stdin: %w\n"+
					"  In non-interactive environments use --contract-source <path> or --skip-source-mapping.",
				promptErr,
			)
		}
		manualPath = strings.TrimSpace(manualPath)
		if manualPath == "" {
			logger.Logger.Info("User chose to skip manual WASM path entry")
			return nil, nil
		}
		info, statErr := os.Stat(manualPath)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, fmt.Errorf(
					"manual WASM path %q does not exist\n"+
						"  Provide a valid path to the contract WASM file or use --contract-source <path> or --skip-source-mapping.",
					manualPath,
				)
			}
			return nil, fmt.Errorf("manual WASM path %q is not accessible: %w", manualPath, statErr)
		}
		if info.IsDir() {
			return nil, fmt.Errorf(
				"manual WASM path %q is a directory, not a file\n"+
					"  Provide the path to the compiled .wasm artifact.",
				manualPath,
			)
		}
		logger.Logger.Info("Manual WASM path provided by user", "path", manualPath)
		return nil, nil
	}

	// 6. Cache the successfully resolved result.
	if r.cache != nil {
		if cacheErr := r.cache.Put(source); cacheErr != nil {
			logger.Logger.Warn("Failed to cache source",
				"contract_id", contractID, "error", cacheErr)
		}
	}

	logger.Logger.Info("Source resolved from registry",
		"contract_id", contractID,
		"repository", source.Repository,
		"file_count", len(source.Files),
	)
	return source, nil
}

// PromptForWasmPath pauses execution and asks the user for a manual WASM path.
// Required by Issue #372: "Please provide path to contract WASM for better mapping".
func (r *Resolver) PromptForWasmPath() (string, error) {
	fmt.Print("Please provide path to contract WASM for better mapping: ")
	reader := bufio.NewReader(os.Stdin)
	path, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(path), nil
}

// AutoDiscoverLocalSymbols scans the project root for local WASM builds.
// If a bytecode hash match is found, it loads and merges DWARF debug symbols.
//
// Validation:
//   - projectRoot must not be empty — returns an error immediately.
//   - expectedHash must not be empty — returns an error immediately.
//   - A missing build directory is treated as a non-fatal debug hint; the
//     caller can continue without local symbols.
func (r *Resolver) AutoDiscoverLocalSymbols(projectRoot string, expectedHash string) error {
	if projectRoot == "" {
		return fmt.Errorf(
			"AutoDiscoverLocalSymbols: projectRoot must not be empty\n" +
				"  Hint: pass the path to your Rust workspace root.",
		)
	}
	if expectedHash == "" {
		return fmt.Errorf(
			"AutoDiscoverLocalSymbols: expectedHash must not be empty\n" +
				"  Hint: provide the SHA-256 hex hash of the on-chain contract bytecode.",
		)
	}

	discovery, discErr := DiscoverLocalSymbols(projectRoot)
	if discErr != nil {
		// A missing build directory is diagnostic, not fatal.
		logger.Logger.Debug("Local symbol discovery: build directory not available",
			"project_root", projectRoot, "reason", discErr.Error())
		return nil
	}

	for _, w := range discovery.Warnings {
		logger.Logger.Warn("Local symbol discovery warning", "detail", w)
	}

	matchedPath, ok := discovery.Found[expectedHash]
	if !ok {
		logger.Logger.Debug("No local WASM matches on-chain hash",
			"expected_hash", expectedHash,
			"candidates_checked", len(discovery.Found),
			"search_dir", discovery.SearchDir,
		)
		return nil
	}

	content, readErr := os.ReadFile(matchedPath)
	if readErr != nil {
		return fmt.Errorf(
			"local WASM match found at %q but could not be read: %w\n"+
				"  Check file permissions and try again.",
			matchedPath, readErr,
		)
	}

	logger.Logger.Info("Found local WASM match",
		"file", filepath.Base(matchedPath), "path", matchedPath)

	parser, parseErr := dwarf.NewParser(content)
	if parseErr != nil {
		return fmt.Errorf(
			"failed to parse DWARF from %q: %w\n"+
				"  Ensure the WASM was compiled with 'debug = true' in [profile.release].",
			matchedPath, parseErr,
		)
	}

	if !parser.HasDebugInfo() {
		logger.Logger.Warn(
			"Local WASM found but contains no DWARF debug symbols — source mapping will be limited",
			"file", filepath.Base(matchedPath),
			"hint", "recompile with 'debug = true' in [profile.release] for accurate source mapping",
		)
		return nil
	}

	subprograms, spErr := parser.GetSubprograms()
	if spErr != nil {
		return fmt.Errorf(
			"failed to extract DWARF subprograms from %q: %w",
			matchedPath, spErr,
		)
	}

	logger.Logger.Info("Automatically merged symbols from local build",
		"file", filepath.Base(matchedPath),
		"count", len(subprograms))
	return nil
}

// ResolveFilePath applies the alias resolver (if configured) to translate a
// workspace-relative source file path to a real filesystem path.
// Returns p unchanged when no alias resolver is set.
func (r *Resolver) ResolveFilePath(p string) string {
	if r.aliasResolver == nil {
		return p
	}
	return r.aliasResolver.Resolve(p)
}

// InvalidateCache removes a specific contract from the cache.
func (r *Resolver) InvalidateCache(contractID string) error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Invalidate(contractID)
}

// ClearCache removes all cached source entries.
func (r *Resolver) ClearCache() error {
	if r.cache == nil {
		return nil
	}
	return r.cache.Clear()
}
