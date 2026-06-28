// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package watch

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// FileWatcherConfig configures the polling file watcher.
type FileWatcherConfig struct {
	Paths          []string
	PollInterval   time.Duration
	DebounceWindow time.Duration
}

// DefaultFileWatcherConfig provides sensible defaults.
func DefaultFileWatcherConfig(paths []string) FileWatcherConfig {
	return FileWatcherConfig{
		Paths:          paths,
		PollInterval:   500 * time.Millisecond,
		DebounceWindow: 500 * time.Millisecond,
	}
}

// StartFileWatcher starts a background goroutine that polls the specified paths.
// It emits a signal on the returned channel when a file modification is detected and debounced.
func StartFileWatcher(ctx context.Context, cfg FileWatcherConfig) (<-chan struct{}, <-chan error) {
	events := make(chan struct{}, 1)
	errs := make(chan error, 1)

	if len(cfg.Paths) == 0 {
		close(events)
		close(errs)
		return events, errs
	}

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.DebounceWindow <= 0 {
		cfg.DebounceWindow = 500 * time.Millisecond
	}

	go func() {
		defer close(events)
		defer close(errs)

		ticker := time.NewTicker(cfg.PollInterval)
		defer ticker.Stop()

		lastState := make(map[string]time.Time)
		
		// Initial state capture
		err := scanPaths(cfg.Paths, lastState)
		if err != nil {
			select {
			case errs <- err:
			case <-ctx.Done():
				return
			}
		}

		var debounceTimer *time.Timer

		for {
			select {
			case <-ctx.Done():
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				return
			case <-ticker.C:
				currentState := make(map[string]time.Time)
				err := scanPaths(cfg.Paths, currentState)
				if err != nil {
					// Ignore transient scan errors, log or emit if needed
					continue
				}

				changed := false
				for p, modTime := range currentState {
					if lastTime, ok := lastState[p]; !ok || modTime.After(lastTime) {
						changed = true
						break
					}
				}
				
				// Also check for deletions
				if !changed {
					for p := range lastState {
						if _, ok := currentState[p]; !ok {
							changed = true
							break
						}
					}
				}

				if changed {
					lastState = currentState
					if debounceTimer != nil {
						debounceTimer.Stop()
					}
					debounceTimer = time.AfterFunc(cfg.DebounceWindow, func() {
						select {
						case events <- struct{}{}:
						default:
							// Drop if channel is full
						}
					})
				}
			}
		}
	}()

	return events, errs
}

func scanPaths(paths []string, state map[string]time.Time) error {
	for _, p := range paths {
		err := filepath.Walk(p, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors (like permission denied or file deleted during walk)
			}
			if !info.IsDir() {
				state[path] = info.ModTime()
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
