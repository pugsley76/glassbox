// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package sourcemap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AliasMap maps workspace-relative alias prefixes to real filesystem paths.
// For example: {"my-crate": "/home/user/repo/crates/my-crate/src"}.
type AliasMap map[string]string

// AliasResolver resolves workspace-relative path aliases to real filesystem
// paths. It is populated from Cargo workspace metadata or a user-supplied
// config file.
type AliasResolver struct {
	aliases AliasMap
}

// NewAliasResolver creates an AliasResolver with the given alias map.
func NewAliasResolver(aliases AliasMap) *AliasResolver {
	return &AliasResolver{aliases: aliases}
}

// NewAliasResolverFromWorkspace builds an AliasResolver by scanning the
// Cargo workspace rooted at workspaceDir. It reads each member's Cargo.toml
// and maps the package name to its src/ directory.
func NewAliasResolverFromWorkspace(workspaceDir string) (*AliasResolver, error) {
	rootManifest := filepath.Join(workspaceDir, "Cargo.toml")
	data, err := os.ReadFile(rootManifest) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("alias: read %s: %w", rootManifest, err)
	}

	members := parseWorkspaceMembers(string(data))
	aliases := make(AliasMap, len(members))

	for _, member := range members {
		memberDir := filepath.Join(workspaceDir, member)
		memberManifest := filepath.Join(memberDir, "Cargo.toml")
		mdata, merr := os.ReadFile(memberManifest) //nolint:gosec
		if merr != nil {
			continue
		}
		name := parsePackageName(string(mdata))
		if name == "" {
			name = filepath.Base(memberDir)
		}
		srcDir := filepath.Join(memberDir, "src")
		if _, serr := os.Stat(srcDir); serr == nil {
			aliases[name] = srcDir
		} else {
			aliases[name] = memberDir
		}
	}

	return &AliasResolver{aliases: aliases}, nil
}

// LoadAliasConfig loads an AliasMap from a JSON file.
// The file should be a flat object mapping alias → real path.
func LoadAliasConfig(path string) (AliasMap, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("alias: config path must not be empty")
	}

	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("alias: read config %s: %w", path, err)
	}
	var m AliasMap
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("alias: parse config %s: %w", path, err)
	}

	configDir := filepath.Dir(path)
	for alias, target := range m {
		if target == "" {
			continue
		}
		resolved := target
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Clean(filepath.Join(configDir, resolved))
		}
		info, statErr := os.Stat(resolved)
		if statErr != nil {
			return nil, fmt.Errorf("alias: target for %q must point to an existing directory: %q: %w", alias, resolved, statErr)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("alias: target for %q must point to an existing directory: %q", alias, resolved)
		}
		m[alias] = resolved
	}
	return m, nil
}

// Resolve normalises p by replacing any matching alias prefix with the
// corresponding real path. Returns the original path unchanged if no alias
// matches.
func (r *AliasResolver) Resolve(p string) string {
	if r == nil || len(r.aliases) == 0 {
		return p
	}
	// Normalise separators for comparison.
	norm := filepath.ToSlash(p)
	for alias, real := range r.aliases {
		prefix := filepath.ToSlash(alias)
		if norm == prefix {
			return real
		}
		if strings.HasPrefix(norm, prefix+"/") {
			rel := norm[len(prefix)+1:]
			return filepath.Join(real, filepath.FromSlash(rel))
		}
	}
	return p
}

// ─── minimal Cargo.toml parsers ──────────────────────────────────────────────

// parseWorkspaceMembers extracts the members list from a workspace Cargo.toml.
func parseWorkspaceMembers(content string) []string {
	var members []string
	inWorkspace := false
	inMembers := false

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "[workspace]" {
			inWorkspace = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inWorkspace = false
			inMembers = false
		}
		if !inWorkspace {
			continue
		}
		if strings.HasPrefix(line, "members") {
			inMembers = true
		}
		if inMembers {
			// collect quoted strings
			for _, part := range strings.Split(line, `"`) {
				part = strings.TrimSpace(part)
				if part != "" && !strings.ContainsAny(part, "[]= ,\t") {
					members = append(members, part)
				}
			}
			if strings.Contains(line, "]") {
				inMembers = false
			}
		}
	}
	return members
}

// parsePackageName extracts the package name from a Cargo.toml.
func parsePackageName(content string) string {
	inPackage := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "[package]" {
			inPackage = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPackage = false
		}
		if inPackage && strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`)
			}
		}
	}
	return ""
}
