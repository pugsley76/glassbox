// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWASMCache_NilManagerPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for nil manager, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic value, got %T: %v", r, r)
		}
		if msg != "cache: NewWASMCache called with nil manager" {
			t.Errorf("unexpected panic message: %q", msg)
		}
	}()
	NewWASMCache(nil, 0, nil)
}

func TestWASMCache_Get_DetectsCorruption(t *testing.T) {
	cacheDir := t.TempDir()
	manager := NewManager(cacheDir, DefaultConfig())
	wasmCache := NewWASMCache(manager, 24*time.Hour, nil)

	key, err := NewCacheKey(
		[]byte("input wasm"),
		KindCompilation,
		"test-tool-v1",
		map[string]string{"opt": "s"},
		nil,
	)
	require.NoError(t, err)

	payload := []byte("valid payload")
	require.NoError(t, wasmCache.Set(key, payload))

	digest, err := key.Digest()
	require.NoError(t, err)
	path, err := wasmCache.entryPath(key.Kind, digest)
	require.NoError(t, err)

	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var entry WASMEntry
	require.NoError(t, json.Unmarshal(raw, &entry))
	entry.Payload[0] ^= 0xff

	corruptRaw, err := json.Marshal(entry)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, corruptRaw, 0600))

	got, found, err := wasmCache.Get(key)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, got)
}
