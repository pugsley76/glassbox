// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFileWatcher(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("initial"), 0644)
	require.NoError(t, err)

	cfg := FileWatcherConfig{
		Paths:          []string{tmpDir},
		PollInterval:   50 * time.Millisecond,
		DebounceWindow: 50 * time.Millisecond,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, errs := StartFileWatcher(ctx, cfg)

	// Ensure no errors on start
	select {
	case err := <-errs:
		t.Fatalf("unexpected error: %v", err)
	default:
	}

	// Modify the file
	time.Sleep(100 * time.Millisecond) // Wait for first scan
	err = os.WriteFile(testFile, []byte("changed"), 0644)
	require.NoError(t, err)

	// Wait for event
	select {
	case <-events:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for watch event")
	}

	// Test Debouncing: multiple rapid changes should yield one event
	err = os.WriteFile(testFile, []byte("changed2"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile, []byte("changed3"), 0644)
	require.NoError(t, err)

	eventCount := 0
	timeout := time.After(500 * time.Millisecond)
loop:
	for {
		select {
		case <-events:
			eventCount++
		case <-timeout:
			break loop
		}
	}
	require.Equal(t, 1, eventCount, "expected exactly one debounced event")
}
