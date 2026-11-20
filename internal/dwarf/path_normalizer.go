// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package dwarf

// PathDiagnostic represents a warning or error encountered during path
// normalization. It is declared here so the dwarf package can accept a
// normalizer without importing internal/sourcemap; sourcemap aliases this type
// to preserve its public API.
type PathDiagnostic struct {
	Severity       string `json:"severity"`
	Message        string `json:"message"`
	OriginalPath   string `json:"original_path"`
	NormalizedPath string `json:"normalized_path,omitempty"`
	Hint           string `json:"hint,omitempty"`
}

// PathNormalizer is the subset of path normalization behaviour needed by the
// DWARF parser. *sourcemap.PathNormalizer satisfies it via the PathDiagnostic
// alias in that package.
type PathNormalizer interface {
	NormalizePath(rawPath string) (string, *PathDiagnostic)
	GetDiagnostics() []PathDiagnostic
}
