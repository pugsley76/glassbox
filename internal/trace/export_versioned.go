// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

// Package trace — versioned JSON export and multi-envelope load helpers.
//
// This file adds three symbols that round out the "versioning and metadata"
// export workflow:
//
//   - ExportJSON      — produce a schema-versioned JSON envelope with a
//                       SHA-256 fingerprinted transaction hash and a
//                       generated_at timestamp.
//   - SaveToFile      — write a plain ExecutionTrace JSON file (legacy format,
//                       no version envelope; suitable for direct round-trips).
//   - LoadExecutionTrace — single entry-point that loads any envelope shape
//                       produced by Glassbox and returns a clear, actionable
//                       error when loading fails.

package trace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// exportJSONEnvelopePayload is the "trace" sub-object in an ExportJSON file.
// It mirrors ExecutionTrace but replaces the transaction hash at marshalling
// time so the raw value never appears in the output.
type exportJSONEnvelopePayload struct {
	TransactionHash  string                `json:"transaction_hash"`
	StartTime        time.Time             `json:"start_time"`
	EndTime          time.Time             `json:"end_time"`
	States           []ExecutionState      `json:"states"`
	Snapshots        []StateSnapshot       `json:"snapshots"`
	DiagnosticEvents interface{}           `json:"diagnostic_events,omitempty"`
	DecodedEvents    interface{}           `json:"decoded_events,omitempty"`
	Annotations      TraceAnnotations      `json:"annotations,omitempty"`
	CurrentStep      int                   `json:"current_step"`
	SnapshotInterval int                   `json:"snapshot_interval"`
}

// exportJSONEnvelope is the top-level object written by ExportJSON.
type exportJSONEnvelope struct {
	SchemaVersion string                    `json:"schema_version"`
	GeneratedAt   time.Time                 `json:"generated_at"`
	Trace         exportJSONEnvelopePayload `json:"trace"`
}

// ExportJSON encodes the trace in the ExportJSON envelope format and returns
// the raw JSON bytes.
//
// Envelope shape:
//
//	{
//	  "schema_version": "<schemaVersion>",
//	  "generated_at":   "<generatedAt truncated to second, UTC>",
//	  "trace": {
//	    "transaction_hash": "sha256:<hex>",
//	    ...
//	  }
//	}
//
// The transaction_hash field is SHA-256 fingerprinted (prefixed with
// "sha256:") so the raw hash never appears in the output.  This makes
// the envelope safe to share without leaking sensitive identifiers.
//
// The method is deterministic: calling it twice with the same receiver
// state and the same (schemaVersion, generatedAt) arguments produces
// identical bytes.
//
// schemaVersion is written verbatim; no validation is performed here so
// callers (e.g. tests) can deliberately write unsupported versions to verify
// that LoadVersionedTrace rejects them properly.
func (t *ExecutionTrace) ExportJSON(schemaVersion string, generatedAt time.Time) ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf(
			"cannot export nil trace — ExportJSON requires a valid *ExecutionTrace\n" +
				"  Fix: ensure the trace object was initialised with NewExecutionTrace()")
	}

	payload := exportJSONEnvelopePayload{
		TransactionHash:  fingerprintTxHash(t.TransactionHash),
		StartTime:        t.StartTime,
		EndTime:          t.EndTime,
		States:           t.States,
		Snapshots:        t.Snapshots,
		DiagnosticEvents: t.DiagnosticEvents,
		DecodedEvents:    t.DecodedEvents,
		Annotations:      t.Annotations,
		CurrentStep:      t.CurrentStep,
		SnapshotInterval: t.SnapshotInterval,
	}

	envelope := exportJSONEnvelope{
		SchemaVersion: schemaVersion,
		// Truncate to second so the timestamp is stable across test runs that
		// supply a fixed time.Time value (sub-second precision is irrelevant for
		// an export timestamp and breaks determinism checks).
		GeneratedAt: generatedAt.UTC().Truncate(time.Second),
		Trace:       payload,
	}

	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, fmt.Errorf(
			"failed to marshal trace as ExportJSON envelope: %w\n"+
				"  This indicates the trace contains non-serialisable data\n"+
				"  Check for circular references or invalid field values in the trace states",
			err)
	}
	return data, nil
}

// fingerprintTxHash returns a privacy-safe SHA-256 digest of txHash in the
// form "sha256:<64 hex chars>".  An empty hash produces "sha256:(empty)"
// rather than the hash of an empty string so the sentinel is unambiguous.
func fingerprintTxHash(txHash string) string {
	if txHash == "" {
		return "sha256:(empty)"
	}
	sum := sha256.Sum256([]byte(txHash))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// SaveToFile serialises the trace as plain ExecutionTrace JSON (the legacy
// format, no version envelope) and writes it to path.
//
// Prefer ExportJSON for new code — it embeds a schema_version and a
// generated_at timestamp that allow LoadVersionedTrace to detect the format
// and validate compatibility automatically.
//
// SaveToFile creates any missing parent directories before writing.
func (t *ExecutionTrace) SaveToFile(path string) error {
	if t == nil {
		return fmt.Errorf(
			"cannot save nil trace to %q — SaveToFile requires a valid *ExecutionTrace\n"+
				"  Fix: ensure the trace object was initialised with NewExecutionTrace()",
			path)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf(
			"failed to marshal trace for SaveToFile: %w\n"+
				"  Check for non-serialisable values in the trace states",
			err)
	}

	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
			return fmt.Errorf(
				"failed to create parent directory for trace file %q: %w\n"+
					"  Fix: ensure you have write permissions for the directory",
				path, mkErr)
		}
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf(
			"failed to write trace file %q: %w\n"+
				"  Fix: check disk space and file permissions",
			path, err)
	}
	return nil
}

// LoadExecutionTrace loads a trace from path and returns the decoded
// *ExecutionTrace.  It accepts all envelope shapes produced by Glassbox:
//
//   - Plain ExecutionTrace JSON (legacy format written by SaveToFile)
//   - VersionedTrace envelope  (written by ExportVersionedTrace)
//   - ExportJSON envelope      (written by ExportJSON / --output-json)
//
// On failure the error message includes path and references "glassbox"
// commands so operators know where to look and how to produce valid files.
//
// This is the recommended single entry-point for loading traces in
// production code; it delegates to LoadVersionedTrace with
// DefaultCompatibilityOptions internally.
func LoadExecutionTrace(path string) (*ExecutionTrace, error) {
	tr, err := LoadVersionedTrace(path, DefaultCompatibilityOptions())
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load execution trace from %q: %w\n"+
				"  Verify the file was produced by a glassbox command such as:\n"+
				"    glassbox debug <tx-hash> --trace-output trace.json\n"+
				"    glassbox trace --output-json trace.json input.json",
			path, err)
	}
	return tr, nil
}
