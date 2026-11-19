// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package clioutput

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWantsJSON(t *testing.T) {
	if !WantsJSON(true, "") {
		t.Fatal("expected --json to enable JSON")
	}
	if !WantsJSON(false, "json") {
		t.Fatal("expected --format json to enable JSON")
	}
	if WantsJSON(false, "text") {
		t.Fatal("did not expect text format to enable JSON")
	}
}

// TestWantsJSON_CaseInsensitive verifies that --format matching is
// case-insensitive, as documented in docs/json-output.md.
func TestWantsJSON_CaseInsensitive(t *testing.T) {
	cases := []struct {
		format string
		want   bool
	}{
		{"json", true},
		{"JSON", true},
		{"Json", true},
		{"jSoN", true},
		{"text", false},
		{"TEXT", false},
		{"", false},
	}
	for _, c := range cases {
		got := WantsJSON(false, c.format)
		if got != c.want {
			t.Errorf("WantsJSON(false, %q) = %v, want %v", c.format, got, c.want)
		}
	}
}

// TestWantsJSON_WhitespaceTrimmed verifies that leading/trailing whitespace
// in the --format value does not prevent JSON detection.
func TestWantsJSON_WhitespaceTrimmed(t *testing.T) {
	cases := []string{"  json", "json  ", "  json  ", "  JSON  "}
	for _, c := range cases {
		if !WantsJSON(false, c) {
			t.Errorf("WantsJSON(false, %q) = false, want true (whitespace should be trimmed)", c)
		}
	}
}

// TestWantsJSON_FlagTakesPrecedence verifies that jsonFlag=true enables JSON
// output regardless of the format string value.
func TestWantsJSON_FlagTakesPrecedence(t *testing.T) {
	for _, format := range []string{"text", "yaml", "", "not-json"} {
		if !WantsJSON(true, format) {
			t.Errorf("WantsJSON(true, %q) = false; --json flag must take precedence", format)
		}
	}
}

func TestWriteEnvelope(t *testing.T) {
	var buf bytes.Buffer
	payload := map[string]string{"status": "ok"}
	if err := Write(&buf, "test-cmd", payload); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var env Envelope
	if err := json.Unmarshal(buf.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}
	if env.SchemaVersion != SchemaVersion {
		t.Fatalf("schema_version = %q, want %q", env.SchemaVersion, SchemaVersion)
	}
	if env.GlassboxVersion == "" {
		t.Fatal("expected glassbox_version to be set")
	}
	var inner map[string]string
	if err := json.Unmarshal(env.Data, &inner); err != nil {
		t.Fatalf("unmarshal data: %v", err)
	}
	if inner["status"] != "ok" {
		t.Fatalf("data.status = %q", inner["status"])
	}
}
