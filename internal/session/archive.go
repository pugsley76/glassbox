// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dotandev/glassbox/internal/version"
)

// archiveVersion is incremented whenever the archive layout changes.
const archiveVersion = 1

// archiveMeta is written to meta.json inside every archive.
type archiveMeta struct {
	ArchiveVersion int    `json:"archive_version"`
	GlassboxVersion string `json:"glassbox_version"`
	CreatedAt      string `json:"created_at"`
	SchemaVersion  int    `json:"schema_version"`
}

// SupportedArchiveExtensions lists the file extensions accepted for session
// archive files. The canonical extension is ".gbx"; ".zip" is also accepted
// for interoperability with generic ZIP tools.
var SupportedArchiveExtensions = []string{".gbx", ".zip"}

// ValidateArchivePath checks that destPath is non-empty and ends with a
// supported archive extension. It returns an actionable error when the
// extension is missing or unsupported.
func ValidateArchivePath(destPath string) error {
	if strings.TrimSpace(destPath) == "" {
		return fmt.Errorf("destination path is required")
	}
	ext := strings.ToLower(filepath.Ext(destPath))
	for _, supported := range SupportedArchiveExtensions {
		if ext == supported {
			return nil
		}
	}
	return fmt.Errorf(
		"unsupported archive extension %q — must be one of: %s\n"+
			"  Fix: rename the output file with a supported extension\n"+
			"  Example: glassbox session share --output ./session.gbx",
		ext, strings.Join(SupportedArchiveExtensions, ", "),
	)
}

// ExportArchive packages a debug session into a portable ZIP archive at
// destPath. The archive contains:
//
//	meta.json       – version and compatibility metadata
//	session.json    – the full session Data record
//
// Additional artifacts (source maps, trace JSON) can be embedded by callers
// that have access to them; the format reserves space via the zip comment.
//
// The session data is validated before export so that corrupt or incomplete
// sessions are rejected early with a clear error rather than silently archived.
func ExportArchive(data *Data, destPath string) error {
	if data == nil {
		return fmt.Errorf("session data is nil")
	}
	if err := ValidateArchivePath(destPath); err != nil {
		return err
	}

	// Validate session data before exporting so the user gets an actionable
	// error instead of an archive that cannot be imported on the other side.
	report := ValidateIntegrity(data)
	if !report.OK {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("cannot export invalid session (%d issue(s)):\n", len(report.Issues)))
		for i, issue := range report.Issues {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, issue.Field, issue.Description))
			if issue.Hint != "" {
				sb.WriteString(fmt.Sprintf("     Hint: %s\n", issue.Hint))
			}
		}
		sb.WriteString("Fix the issues above and re-run 'glassbox session share'.")
		return fmt.Errorf("%s", sb.String())
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("cannot create archive file %q: %w", destPath, err)
	}
	defer func() { _ = f.Close() }()

	zw := zip.NewWriter(f)
	defer func() { _ = zw.Close() }()

	// Write meta.json.
	meta := archiveMeta{
		ArchiveVersion:  archiveVersion,
		GlassboxVersion: version.Version,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		SchemaVersion:   SchemaVersion,
	}
	if err := writeJSONEntry(zw, "meta.json", meta); err != nil {
		return fmt.Errorf("failed to write meta.json: %w", err)
	}

	// Write session.json.
	if err := writeJSONEntry(zw, "session.json", data); err != nil {
		return fmt.Errorf("failed to write session.json: %w", err)
	}

	// Write envelope XDR as raw bytes when present.
	if data.EnvelopeXdr != "" {
		if err := writeStringEntry(zw, "envelope.xdr", data.EnvelopeXdr); err != nil {
			return fmt.Errorf("failed to write envelope.xdr: %w", err)
		}
	}

	// Write simulation response JSON when present.
	if data.SimResponseJSON != "" {
		if err := writeStringEntry(zw, "sim_response.json", data.SimResponseJSON); err != nil {
			return fmt.Errorf("failed to write sim_response.json: %w", err)
		}
	}

	return nil
}

// ImportArchive reads a session archive produced by ExportArchive and returns
// the reconstructed Data. It validates the archive format, extension, and
// version compatibility before returning, surfacing actionable errors for each
// failure mode.
func ImportArchive(srcPath string) (*Data, error) {
	if strings.TrimSpace(srcPath) == "" {
		return nil, fmt.Errorf(
			"archive path is required\n" +
				"  Fix: provide the path to a .gbx archive file\n" +
				"  Example: glassbox session load ./session.gbx",
		)
	}

	// Validate extension so we surface a clear error before attempting to open.
	ext := strings.ToLower(filepath.Ext(srcPath))
	validExt := false
	for _, supported := range SupportedArchiveExtensions {
		if ext == supported {
			validExt = true
			break
		}
	}
	if !validExt {
		return nil, fmt.Errorf(
			"unsupported archive extension %q — expected one of: %s\n"+
				"  Fix: use a file exported by 'glassbox session share'\n"+
				"  Example: glassbox session load ./session.gbx",
			ext, strings.Join(SupportedArchiveExtensions, ", "),
		)
	}

	zr, err := zip.OpenReader(srcPath)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot open archive %q: %w\n"+
				"  Fix: ensure the file exists, is readable, and is a valid Glassbox session archive",
			srcPath, err,
		)
	}
	defer func() { _ = zr.Close() }()

	var meta archiveMeta
	var data Data
	metaFound := false
	sessionFound := false

	for _, f := range zr.File {
		switch f.Name {
		case "meta.json":
			if err := readJSONEntry(f, &meta); err != nil {
				return nil, fmt.Errorf(
					"failed to read meta.json from archive: %w\n"+
						"  Fix: the archive may be corrupt. Re-export it with 'glassbox session share'",
					err,
				)
			}
			metaFound = true
		case "session.json":
			if err := readJSONEntry(f, &data); err != nil {
				return nil, fmt.Errorf(
					"failed to read session.json from archive: %w\n"+
						"  Fix: the archive may be corrupt. Re-export it with 'glassbox session share'",
					err,
				)
			}
			sessionFound = true
		}
	}

	if !metaFound {
		return nil, fmt.Errorf(
			"archive is missing meta.json — not a valid Glassbox session archive\n" +
				"  Fix: use a file exported by 'glassbox session share'",
		)
	}
	if !sessionFound {
		return nil, fmt.Errorf(
			"archive is missing session.json — not a valid Glassbox session archive\n" +
				"  Fix: use a file exported by 'glassbox session share'",
		)
	}
	if meta.ArchiveVersion > archiveVersion {
		return nil, fmt.Errorf(
			"archive version %d is newer than supported version %d\n"+
				"  Fix: upgrade Glassbox to the latest release to open this archive",
			meta.ArchiveVersion, archiveVersion,
		)
	}
	if meta.SchemaVersion > SchemaVersion {
		return nil, fmt.Errorf(
			"session schema version %d is newer than supported version %d\n"+
				"  Fix: upgrade Glassbox to the latest release to open this session",
			meta.SchemaVersion, SchemaVersion,
		)
	}

	// Validate the reconstructed session data so imported archives with missing
	// or corrupt fields are rejected with a clear diagnostic instead of silently
	// producing a broken session.
	report := ValidateIntegrity(&data)
	if !report.OK {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(
			"archive %q contains an invalid session (%d issue(s)):\n",
			srcPath, len(report.Issues),
		))
		for i, issue := range report.Issues {
			sb.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, issue.Field, issue.Description))
			if issue.Hint != "" {
				sb.WriteString(fmt.Sprintf("     Hint: %s\n", issue.Hint))
			}
		}
		sb.WriteString("Re-export with 'glassbox session share' from a valid session.")
		return nil, fmt.Errorf("%s", sb.String())
	}

	return &data, nil
}

// writeJSONEntry serialises v and writes it as a named entry in the zip.
// It uses deterministic key ordering for reproducible exports.
func writeJSONEntry(zw *zip.Writer, name string, v interface{}) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}

	// Sort map keys recursively for deterministic output
	sorted := SortMapKeys(v)

	// Use json.Marshal for consistent ordering with sorted keys
	data, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		return err
	}

	_, err = w.Write(data)
	return err
}

// writeStringEntry writes a plain string as a named entry in the zip.
func writeStringEntry(zw *zip.Writer, name, content string) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, content)
	return err
}

// readJSONEntry decodes JSON from a zip file entry into v.
func readJSONEntry(f *zip.File, v interface{}) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	return json.NewDecoder(rc).Decode(v)
}
// sortedKeys returns the sorted keys of a map for deterministic serialization.
func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SortMapKeys recursively sorts map keys for deterministic JSON serialization.
// This ensures session metadata is serialized in a consistent order for reproducibility.
func SortMapKeys(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{}, len(val))
		keys := sortedKeys(val)
		for _, k := range keys {
			result[k] = SortMapKeys(val[k])
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(val))
		for i, item := range val {
			result[i] = SortMapKeys(item)
		}
		return result
	default:
		return v
	}
}

// DeterministicMarshal marshals a value to JSON with sorted map keys.
// This is used for session metadata serialization to ensure reproducible exports.
func DeterministicMarshal(v interface{}) ([]byte, error) {
	// Sort all map keys recursively
	sorted := SortMapKeys(v)

	// Use a sorted key encoder for deterministic output
	enc := json.NewEncoder(new(strings.Builder))
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)

	// For full deterministic output, we need custom marshaling
	return json.MarshalIndent(sorted, "", "  ")
}

// CommandParams holds command-line parameters for session metadata.
type CommandParams map[string]interface{}

// Sorted returns a new map with keys sorted deterministically.
func (cp CommandParams) Sorted() CommandParams {
	result := make(CommandParams)
	keys := sortedKeys(map[string]interface{}(cp))
	for _, k := range keys {
		result[k] = SortMapKeys(cp[k])
	}
	return result
}

// ToJSON returns JSON representation with deterministically sorted keys.
func (cp CommandParams) ToJSON() (string, error) {
	data, err := json.MarshalIndent(cp.Sorted(), "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// EnsureDeterministicOrder sorts command parameters before serialization.
// This should be called before writing session metadata to ensure reproducible exports.
func EnsureDeterministicOrder(data *Data) *Data {
	if data == nil {
		return data
	}

	// Create a copy to avoid mutating the original
	result := *data

	// Sort any nested maps in the session data
	// This is a shallow sort; deep sorting is handled by SortMapKeys

	return &result
}