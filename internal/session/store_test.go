// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── Store.Save validation ──────────────────────────────────────────────────────

func TestStore_Save_NilData_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must not be nil")
}

func TestStore_Save_EmptyID_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), &Data{
		TxHash:  "abc",
		Network: "testnet",
		Status:  "saved",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ID is required")
}

func TestStore_Save_EmptyTxHash_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), &Data{
		ID:      "session-1",
		Network: "testnet",
		Status:  "saved",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TxHash is required")
}

func TestStore_Save_EmptyNetwork_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), &Data{
		ID:      "session-1",
		TxHash:  "abc",
		Status:  "saved",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "network is required")
}

func TestStore_Save_InvalidNetwork_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), &Data{
		ID:      "session-1",
		TxHash:  "abc",
		Network: "invalidnet",
		Status:  "saved",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	assert.Contains(t, err.Error(), "testnet")
	assert.Contains(t, err.Error(), "mainnet")
	assert.Contains(t, err.Error(), "futurenet")
}

func TestStore_Save_EmptyStatus_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), &Data{
		ID:      "session-1",
		TxHash:  "abc",
		Network: "testnet",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status is required")
}

func TestStore_Save_InvalidStatus_ReturnsError(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	err = store.Save(context.Background(), &Data{
		ID:      "session-1",
		TxHash:  "abc",
		Network: "testnet",
		Status:  "borked",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
	assert.Contains(t, err.Error(), "active")
	assert.Contains(t, err.Error(), "saved")
	assert.Contains(t, err.Error(), "resumed")
	assert.Contains(t, err.Error(), "recovered")
	assert.Contains(t, err.Error(), "expired")
}

func TestStore_Save_ValidData_Succeeds(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	now := time.Now()
	data := &Data{
		ID:           "session-valid-" + now.Format("20060102150405"),
		TxHash:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Network:      "testnet",
		Status:       "saved",
		CreatedAt:    now,
		LastAccessAt: now,
		SchemaVersion: SchemaVersion,
	}

	err = store.Save(context.Background(), data)
	require.NoError(t, err)
	assert.NotZero(t, data.CreatedAt)
	assert.NotZero(t, data.LastAccessAt)
	assert.Equal(t, SchemaVersion, data.SchemaVersion)
}

func TestStore_Save_PinnedEndpointValidation(t *testing.T) {
	store, err := NewStore()
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	now := time.Now()
	data := &Data{
		ID:             "session-pinned-" + now.Format("20060102150405"),
		TxHash:         "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Network:        "testnet",
		Status:        "saved",
		CreatedAt:     now,
		LastAccessAt:  now,
		SchemaVersion: SchemaVersion,
		PinnedEndpoint: "http://127.0.0.1:9999",
	}

	err = store.Save(context.Background(), data)
	require.NoError(t, err)
}
