// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package sourcemap

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dotandev/glassbox/internal/logger"
)

// WorkspaceSource aggregates source mapping data across multiple source files
// and crates in a contract workspace.
type WorkspaceSource struct {
	// ContractID is the contract address this source belongs to.
	ContractID string `json:"contract_id"`
	// WasmHash is the hash of the WASM this source was verified against.
	WasmHash string `json:"wasm_hash"`
	// Repository is the source repository URL.
	Repository string `json:"repository"`
	// Files maps relative file paths to their source content.
	Files map[string]string `json:"files"`
	// SourceMaps maps source file paths to their DWARF/debug info.
	SourceMaps map[string]SourceMapData `json:"source_maps"`
	// CrateInfo holds metadata about each crate in the workspace.
	CrateInfo map[string]CrateMetadata `json:"crate_info"`
	// FetchedAt is the time this source was retrieved.
	FetchedAt string `json:"fetched_at"`
}

// CrateMetadata holds metadata about a crate in a workspace.
type CrateMetadata struct {
	// Name is the crate name.
	Name string `json:"name"`
	// Version is the crate version.
	Version string `json:"version"`
	// RootFile is the path to the crate root (Cargo.toml).
	RootFile string `json:"root_file"`
	// SourceRoot is the path to the crate source root.
	SourceRoot string `json:"source_root"`
	// Dependencies lists the crate's dependencies.
	Dependencies []string `json:"dependencies"`
	// IsWorkspaceMember indicates if this crate is part of a workspace.
	IsWorkspaceMember bool `json:"is_workspace_member"`
	// WorkspaceMember indicates which workspace it belongs to (if any).
	WorkspaceMember string `json:"workspace_member,omitempty"`
}

// SourceMapData holds source map data for a single source file.
type SourceMapData struct {
	// File is the source file path.
	File string `json:"file"`
	// Lines maps WASM instruction offsets to source line numbers.
	Lines map[uint32]uint32 `json:"lines"`
	// Functions maps function indices to function info.
	Functions map[uint32]FunctionInfo `json:"functions"`
}

// FunctionInfo holds information about a mapped function.
type FunctionInfo struct {
	// Name is the function name.
	Name string `json:"name"`
	// SourceFile is the source file containing this function.
	SourceFile string `json:"source_file"`
	// StartLine is the starting line number.
	StartLine uint32 `json:"start_line"`
	// EndLine is the ending line number.
	EndLine uint32 `json:"end_line"`
}

// WorkspaceAggregator combines source files from multiple crates into a unified
// debug context.
type WorkspaceAggregator struct {
	// sources holds sources from individual crate fetches
	sources []*SourceCode
}

// NewWorkspaceAggregator creates a new workspace aggregator.
func NewWorkspaceAggregator() *WorkspaceAggregator {
	return &WorkspaceAggregator{
		sources: make([]*SourceCode, 0),
	}
}

// AddSource adds a source to the workspace aggregation.
func (wa *WorkspaceAggregator) AddSource(source *SourceCode) {
	wa.sources = append(wa.sources, source)
}

// Aggregate merges all added sources into a single WorkspaceSource.
func (wa *WorkspaceAggregator) Aggregate() *WorkspaceSource {
	if len(wa.sources) == 0 {
		return nil
	}

	// Use the first source as the base
	first := wa.sources[0]

	ws := &WorkspaceSource{
		ContractID:  first.ContractID,
		WasmHash:    first.WasmHash,
		Repository:  first.Repository,
		Files:       make(map[string]string),
		SourceMaps:  make(map[string]SourceMapData),
		CrateInfo:   make(map[string]CrateMetadata),
		FetchedAt:   first.FetchedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	// Collect all files from all sources, avoiding duplicates
	// by prefixing with crate identifier
	for i, src := range wa.sources {
		crateID := fmt.Sprintf("crate_%d", i)

		for filePath, content := range src.Files {
			// Use unique key: crateID + filePath
			uniquePath := filepath.Join(crateID, filePath)
			ws.Files[uniquePath] = content

			// Extract crate info from Cargo.toml files
			if filepath.Base(filePath) == "Cargo.toml" {
				crateMeta := wa.parseCargoToml(filePath, content)
				crateMeta.RootFile = uniquePath
				ws.CrateInfo[crateID] = crateMeta
			}
		}

		// If only one source, use simpler paths
		if len(wa.sources) == 1 {
			ws.Files = src.Files
		}
	}

	// If we have Cargo workspace info, update crate membership
	wa.updateWorkspaceMembership(ws)

	return ws
}

// parseCargoToml extracts crate metadata from a Cargo.toml content.
func (wa *WorkspaceAggregator) parseCargoToml(filePath, content string) CrateMetadata {
	meta := CrateMetadata{
		RootFile: filePath,
	}

	// Simple parsing for common fields
	lines := strings.Split(content, "\n")
	inPackage := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "[package]") {
			inPackage = true
			continue
		}
		if strings.HasPrefix(line, "[") && !strings.HasPrefix(line, "[[") {
			inPackage = false
		}

		if inPackage {
			if strings.HasPrefix(line, "name") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					meta.Name = strings.Trim(parts[1], ` "`)
				}
			}
			if strings.HasPrefix(line, "version") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					meta.Version = strings.Trim(parts[1], ` "`)
				}
			}
		}

		// Check for workspace membership
		if strings.Contains(line, "[workspace]") || strings.HasPrefix(line, "workspace") {
			meta.IsWorkspaceMember = true
		}
	}

	return meta
}

// updateWorkspaceMembership updates crate info with workspace relationships.
func (wa *WorkspaceAggregator) updateWorkspaceMembership(ws *WorkspaceSource) {
	// Find the workspace root (usually has member tables)
	var workspaceRoot string
	var workspaceMembers []string

	for crateID, info := range ws.CrateInfo {
		if info.IsWorkspaceMember {
			workspaceRoot = crateID
		}
		workspaceMembers = append(workspaceMembers, crateID)
	}

	// Update all crates to mark workspace membership
	if workspaceRoot != "" {
		for crateID := range ws.CrateInfo {
			info := ws.CrateInfo[crateID]
			info.WorkspaceMember = workspaceRoot
			ws.CrateInfo[crateID] = info
		}
		rootInfo := ws.CrateInfo[workspaceRoot]
		rootInfo.Dependencies = workspaceMembers
		ws.CrateInfo[workspaceRoot] = rootInfo
	}
}

// FindFile searches for a file by name across all aggregated sources.
// It returns the file path and content if found.
func (ws *WorkspaceSource) FindFile(filename string) (string, string, bool) {
	// Direct match
	if content, ok := ws.Files[filename]; ok {
		return filename, content, true
	}

	// Search by basename
	for filePath, content := range ws.Files {
		if filepath.Base(filePath) == filename {
			return filePath, content, true
		}
	}

	return "", "", false
}

// ListRustFiles returns all Rust source files in the workspace, sorted.
func (ws *WorkspaceSource) ListRustFiles() []string {
	var rustFiles []string

	for filePath := range ws.Files {
		if strings.HasSuffix(filePath, ".rs") {
			rustFiles = append(rustFiles, filePath)
		}
	}

	sort.Strings(rustFiles)
	return rustFiles
}

// GetCrateForFile determines which crate a source file belongs to.
func (ws *WorkspaceSource) GetCrateForFile(filePath string) string {
	// Check if file path starts with any known crate prefix
	for crateID := range ws.CrateInfo {
		cratePrefix := crateID + string(filepath.Separator)
		if strings.HasPrefix(filePath, cratePrefix) {
			return crateID
		}
	}

	// Fallback: try to infer from file path
	parts := strings.Split(filePath, string(filepath.Separator))
	if len(parts) > 0 {
		// Assume first segment might be crate ID
		return parts[0]
	}

	return ""
}

// MergeSourceMaps merges source map data from multiple files into a unified map.
func MergeSourceMaps(maps []SourceMapData) SourceMapData {
	merged := SourceMapData{
		Lines:     make(map[uint32]uint32),
		Functions: make(map[uint32]FunctionInfo),
	}

	for _, sm := range maps {
		for offset, line := range sm.Lines {
			merged.Lines[offset] = line
		}
		for idx, fn := range sm.Functions {
			merged.Functions[idx] = fn
		}
	}

	return merged
}

// ToJSON serializes the WorkspaceSource to JSON with deterministic ordering.
func (ws *WorkspaceSource) ToJSON() (string, error) {
	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling workspace source: %w", err)
	}
	return string(data), nil
}

// LoadWorkspaceSource loads a WorkspaceSource from JSON.
func LoadWorkspaceSource(data []byte) (*WorkspaceSource, error) {
	var ws WorkspaceSource
	if err := json.Unmarshal(data, &ws); err != nil {
		return nil, fmt.Errorf("unmarshaling workspace source: %w", err)
	}
	return &ws, nil
}

// AddSourceMap adds source map data for a specific file.
func (ws *WorkspaceSource) AddSourceMap(filePath string, sm SourceMapData) {
	if ws.SourceMaps == nil {
		ws.SourceMaps = make(map[string]SourceMapData)
	}
	sm.File = filePath
	ws.SourceMaps[filePath] = sm
}

// GetSourceMap returns the source map data for a specific file.
func (ws *WorkspaceSource) GetSourceMap(filePath string) (SourceMapData, bool) {
	sm, ok := ws.SourceMaps[filePath]
	return sm, ok
}

// ResolveSourceLocation resolves a WASM offset to source location.
func (ws *WorkspaceSource) ResolveSourceLocation(wasmOffset uint32) (filePath string, line uint32, fnName string) {
	// Search through all source maps
	for _, sm := range ws.SourceMaps {
		if line, ok := sm.Lines[wasmOffset]; ok {
			fnInfo, fnOk := sm.Functions[wasmOffset]
			if fnOk {
				return sm.File, line, fnInfo.Name
			}
			return sm.File, line, ""
		}
	}

	// Fallback: try to find in functions
	for _, sm := range ws.SourceMaps {
		for offset, fnInfo := range sm.Functions {
			if offset == wasmOffset {
				return sm.File, fnInfo.StartLine, fnInfo.Name
			}
		}
	}

	return "", 0, ""
}

// LogWorkspaceSummary logs a summary of the workspace source.
func (ws *WorkspaceSource) LogWorkspaceSummary() {
	logger.Logger.Info("Workspace source aggregated",
		"contract_id", ws.ContractID,
		"wasm_hash", ws.WasmHash,
		"file_count", len(ws.Files),
		"crate_count", len(ws.CrateInfo),
		"source_map_count", len(ws.SourceMaps),
	)

	for crateID, info := range ws.CrateInfo {
		logger.Logger.Debug("Crate info",
			"crate_id", crateID,
			"name", info.Name,
			"version", info.Version,
			"is_workspace_member", info.IsWorkspaceMember,
		)
	}
}

// workspaceSourceFromMultiFile loads workspace source from multiple source files.
// This is a convenience function for aggregating sources from a workspace directory structure.
func workspaceSourceFromMultiFile(contractID, wasmHash, repoURL string, fileMap map[string]string) *WorkspaceSource {
	ws := &WorkspaceSource{
		ContractID: contractID,
		WasmHash:   wasmHash,
		Repository: repoURL,
		Files:      make(map[string]string),
		CrateInfo:  make(map[string]CrateMetadata),
		SourceMaps: make(map[string]SourceMapData),
	}

	for filePath, content := range fileMap {
		ws.Files[filePath] = content

		// Extract crate info from Cargo.toml
		if filepath.Base(filePath) == "Cargo.toml" {
			aggregator := NewWorkspaceAggregator()
			meta := aggregator.parseCargoToml(filePath, content)
			meta.RootFile = filePath

			// Determine crate ID from path
			crateID := filepath.Dir(filePath)
			ws.CrateInfo[crateID] = meta
		}
	}

	// Update workspace membership
	aggregator := NewWorkspaceAggregator()
	aggregator.updateWorkspaceMembership(ws)

	return ws
}