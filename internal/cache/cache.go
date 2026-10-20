// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/dotandev/glassbox/internal/logger"
	"github.com/dotandev/glassbox/internal/termctx"
)

// Config holds cache configuration
type Config struct {
	// MaxSizeBytes is the maximum cache size in bytes (default 1GB)
	MaxSizeBytes int64
}

// DefaultConfig returns the default cache configuration
func DefaultConfig() Config {
	return Config{
		MaxSizeBytes: 1024 * 1024 * 1024, // 1GB
	}
}

// Manager handles cache operations including cleanup
type Manager struct {
	cacheDir string
	config   Config
}

// NewManager creates a new cache manager
func NewManager(cacheDir string, config Config) *Manager {
	return &Manager{
		cacheDir: cacheDir,
		config:   config,
	}
}

// FileInfo contains information about a cached file
type FileInfo struct {
	Path    string
	Size    int64
	ModTime time.Time
}

// GetCacheDir returns the cache directory path (creates if not exists)
func (m *Manager) GetCacheDir() (string, error) {
	if err := os.MkdirAll(m.cacheDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}
	return m.cacheDir, nil
}

// GetCacheSize returns the current size of the cache in bytes
func (m *Manager) GetCacheSize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return 0, fmt.Errorf("failed to calculate cache size: %w", err)
	}

	return totalSize, nil
}

// ListCachedFiles returns a list of all cached files
func (m *Manager) ListCachedFiles() ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(m.cacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, FileInfo{
				Path:    path,
				Size:    info.Size(),
				ModTime: info.ModTime(),
			})
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to list cache files: %w", err)
	}

	return files, nil
}

// sortFilesByAccessTime sorts files by access time (oldest first)
func sortFilesByAccessTime(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.Before(files[j].ModTime)
	})
}

// CleanupStatus contains information about a cleanup operation
type CleanupStatus struct {
	FilesDeleted int
	SpaceFreed   int64
	OriginalSize int64
	FinalSize    int64
	DeletedFiles []string
}

// CleanLRU performs LRU (Least Recently Used) cleanup to ensure cache size is within limit
// Returns the cleanup status and any errors
func (m *Manager) CleanLRU() (*CleanupStatus, error) {
	// Check if cache directory exists
	if _, err := os.Stat(m.cacheDir); os.IsNotExist(err) {
		return &CleanupStatus{
			OriginalSize: 0,
			FinalSize:    0,
		}, nil
	}

	// Get current cache size
	originalSize, err := m.GetCacheSize()
	if err != nil {
		return nil, err
	}

	status := &CleanupStatus{
		OriginalSize: originalSize,
		DeletedFiles: []string{},
	}

	// Check if cleanup is needed
	if originalSize <= m.config.MaxSizeBytes {
		status.FinalSize = originalSize
		logger.Logger.Info("Cache size within limit", "current", originalSize, "limit", m.config.MaxSizeBytes)
		return status, nil
	}

	// Get list of cached files
	files, err := m.ListCachedFiles()
	if err != nil {
		return nil, err
	}

	// Sort by access time (oldest first)
	sortFilesByAccessTime(files)

	// Delete files until cache size is under limit
	targetSize := m.config.MaxSizeBytes / 2 // Target 50% of max size after cleanup
	currentSize := originalSize

	for _, file := range files {
		if currentSize <= targetSize {
			break
		}

		removeErr := os.Remove(file.Path)
		if removeErr != nil {
			logger.Logger.Warn("Failed to delete cache file", "path", file.Path, "error", removeErr)
			continue
		}

		status.FilesDeleted++
		status.SpaceFreed += file.Size
		status.DeletedFiles = append(status.DeletedFiles, file.Path)
		currentSize -= file.Size

		logger.Logger.Debug("Deleted cache file", "path", file.Path, "size", file.Size)
	}

	status.FinalSize = currentSize

	logger.Logger.Info("Cache cleanup completed",
		"files_deleted", status.FilesDeleted,
		"space_freed", status.SpaceFreed,
		"original_size", status.OriginalSize,
		"final_size", status.FinalSize)

	return status, nil
}

// Clean performs a complete cache cleanup with user confirmation
// It will prompt the user before deleting files
func (m *Manager) Clean(force bool) (*CleanupStatus, error) {
	// Check if cache directory exists
	if _, err := os.Stat(m.cacheDir); os.IsNotExist(err) {
		logger.Logger.Info("Cache directory does not exist")
		return &CleanupStatus{}, nil
	}

	// Get current cache size
	originalSize, err := m.GetCacheSize()
	if err != nil {
		return nil, err
	}

	status := &CleanupStatus{
		OriginalSize: originalSize,
		DeletedFiles: []string{},
	}

	if originalSize == 0 {
		logger.Logger.Info("Cache is empty", "size", "0 B")
		status.FinalSize = 0
		return status, nil
	}

	// Show warning and get confirmation
	logger.Logger.Info("Cache status", "size", formatBytes(originalSize), "max_size", formatBytes(m.config.MaxSizeBytes))

	if !force {
		if termctx.GlobalNonInteractive() {
			return status, fmt.Errorf("non-interactive mode: cache clean requires --force to skip confirmation")
		}
		fmt.Print("\nThis will delete the oldest cached files. Continue? (yes/no): ")
		var response string
		if _, scanErr := fmt.Scanln(&response); scanErr != nil {
			return status, fmt.Errorf("failed to read input: %w", scanErr)
		}
		if response != "yes" && response != "y" {
			logger.Logger.Info("Cache cleanup cancelled")
			status.FinalSize = originalSize
			return status, nil
		}
	}

	logger.Logger.Info("Cleaning cache (Least Recently Used files first)")

	// Get list of cached files
	files, err := m.ListCachedFiles()
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		logger.Logger.Info("No cached files found")
		status.FinalSize = 0
		return status, nil
	}

	// Sort by access time (oldest first)
	sortFilesByAccessTime(files)

	// Delete oldest files
	targetSize := m.config.MaxSizeBytes / 2 // Target 50% of max size
	currentSize := originalSize

	for _, file := range files {
		if currentSize <= targetSize {
			break
		}

		removeErr := os.Remove(file.Path)
		if removeErr != nil {
			logger.Logger.Warn("Failed to delete cache file", "path", file.Path, "error", removeErr)
			continue
		}

		status.FilesDeleted++
		status.SpaceFreed += file.Size
		status.DeletedFiles = append(status.DeletedFiles, filepath.Base(file.Path))
		currentSize -= file.Size

		logger.Logger.Debug("Deleted cache file", "path", file.Path, "size", file.Size)
	}

	status.FinalSize = currentSize

	logger.Logger.Info("Cleanup complete",
		"files_deleted", status.FilesDeleted,
		"space_freed", formatBytes(status.SpaceFreed),
		"final_cache_size", formatBytes(status.FinalSize))

	return status, nil
}

// Troubleshooting guidance for cache size, access, and stale-entry symptoms.
// The first step for every symptom is observational (cache status) before any
// cleanup, and targeted cleanup is preferred over full cache removal.
//
// RPC cache lives in the SQLite cache.db under ~/.glassbox/ and is separate
// from the file-based caches under ~/.glassbox/cache (sourcemap and artifact).
const cacheTroubleshootingDoc = `# Cache Troubleshooting

Glassbox keeps two kinds of local cache:

- **RPC cache** — a SQLite database (cache.db) under ~/.glassbox/. It stores
  RPC responses and is managed with the cache rpc commands.
- **File-based caches** — stored under ~/.glassbox/cache. These include the
  **sourcemap** cache and the **artifact** cache, managed with the cache
  artifact commands.

Because they are separate, a large or stale RPC cache will not be fixed by
clearing the artifact cache, and vice versa. Always identify which cache is
affected before cleaning anything.

## Symptom: unexpected cache size

1. Observe first: run ` + "`Glassbox cache status`" + ` to see the total size and
   per-cache breakdown. For the RPC cache specifically, run
   ` + "`Glassbox cache status --rpc`" + `.
2. Compare the reported size against the configured limit. If only the RPC
   cache is oversized, prefer targeted cleanup:
   - ` + "`Glassbox cache rpc --older-than <duration>`" + ` to drop old entries.
   - ` + "`Glassbox cache rpc --network <network>`" + ` to drop entries for a
     single network.
3. If only the artifact cache is oversized, prefer targeted cleanup:
   - ` + "`Glassbox cache artifact clear --type <type> --network <network>`" + `.
4. Only if targeted cleanup is insufficient should you consider a full
   ` + "`Glassbox cache clear`" + `, which removes all local caches.

## Symptom: permission errors

1. Observe first: run ` + "`Glassbox cache status`" + ` (and
   ` + "`Glassbox cache status --rpc`" + ` for the RPC cache) to confirm which
   cache path is failing.
2. Check ownership and permissions on ~/.glassbox/ and ~/.glassbox/cache.
   The cache directory is created with mode 0700; if it was created by another
   user, fix ownership rather than deleting the cache.
3. If cache.db is locked or unreadable, ensure no other Glassbox process is
   running before retrying.
4. Do not run a full ` + "`Glassbox cache clear`" + ` to work around permissions —
   it will fail for the same reason and may remove data you still need.

## Symptom: stale or missed entries

1. Observe first: run ` + "`Glassbox cache status`" + ` and
   ` + "`Glassbox cache status --rpc`" + ` to confirm the entry is present and
   when it was last updated.
2. For stale RPC responses, target the affected network:
   ` + "`Glassbox cache rpc --network <network>`" + `, or age out old entries with
   ` + "`Glassbox cache rpc --older-than <duration>`" + `.
3. For stale sourcemaps or artifacts, target the affected type and network:
   ` + "`Glassbox cache artifact clear --type <type> --network <network>`" + `.
4. Re-run the original command to repopulate only the entries you removed.
   Avoid a full ` + "`Glassbox cache clear`" + ` unless the problem spans every
   cache.
`

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	size := float64(bytes)
	unitIndex := 0

	for size >= 1024 && unitIndex < len(units)-1 {
		size /= 1024
		unitIndex++
	}

	if unitIndex == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unitIndex])
	}
	return fmt.Sprintf("%.2f %s", size, units[unitIndex])
}
