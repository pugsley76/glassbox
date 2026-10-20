// Copyright 2026 Glassbox Users
// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dotandev/glassbox/internal/logger"
)

// GlobalConfig holds the global cache configuration
type GlobalConfig struct {
	// MaxSizeBytes is the maximum cache size in bytes
	MaxSizeBytes int64 `json:"max_size_bytes"`
	// AutoClean enables automatic cache cleanup
	AutoClean bool `json:"auto_clean"`
	// AutoCleanThreshold is the size threshold that triggers automatic cleanup
	AutoCleanThreshold int64 `json:"auto_clean_threshold"`
}

// DefaultGlobalConfig returns the default configuration
func DefaultGlobalConfig() GlobalConfig {
	return GlobalConfig{
		MaxSizeBytes:       1024 * 1024 * 1024, // 1GB
		AutoClean:          true,               // Enabled by default
		AutoCleanThreshold: 1024 * 1024 * 1024, // 1GB
	}
}

// Cache directory discovery
//
// The cache location is resolved from the user's home directory as reported by
// os.UserHomeDir. No environment variable override is currently consulted when
// resolving the cache directory; the GLASSBOX_CACHE_DIR variable mentioned in
// some help text is not yet read by the resolver and should not be relied upon.
//
// Verified resolved paths (per platform):
//
//	Platform        File cache path          RPC SQLite cache path
//	Linux           ~/.glassbox/cache        ~/.glassbox/cache.db
//	macOS           ~/.glassbox/cache        ~/.glassbox/cache.db
//	Windows         %USERPROFILE%\.glassbox\cache   %USERPROFILE%\.glassbox\cache.db
//
// The file cache path comes from getCacheDir in internal/cmd/cache.go, which
// joins os.UserHomeDir with ".glassbox" and "cache". The RPC SQLite cache path
// comes from rpc.GetCachePath, which joins os.UserHomeDir with the CacheDirName
// ".Glassbox" and the file "cache.db".
//
// Fallbacks: if os.UserHomeDir fails, callers fall back to their default
// configuration and log a debug message (see LoadConfig below). The cache
// directory itself is not created until a write occurs.
//
// Safe inspection: use `Glassbox cache status` for the file cache and
// `Glassbox cache status --rpc` for the RPC SQLite cache. These commands only
// read metadata and do not modify the cache.
//
// Cleanup advice: only remove files inside the resolved cache directory shown
// by the inspection commands. Never delete the parent directory or paths
// outside the resolved cache location.

// getConfigPath returns the path to the config file
func getConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".Glassbox", "config.json"), nil
}

// LoadConfig loads the cache configuration from disk
func LoadConfig() (GlobalConfig, error) {
	configPath, err := getConfigPath()
	if err != nil {
		logger.Logger.Debug("Failed to get config path, using defaults", "error", err)
		return DefaultGlobalConfig(), nil
	}

	// If config file doesn't exist, return default config
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		logger.Logger.Debug("Config file does not exist, using defaults")
		return DefaultGlobalConfig(), nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		logger.Logger.Warn("Failed to read config file, using defaults", "error", err)
		return DefaultGlobalConfig(), nil
	}

	var config GlobalConfig
	if err = json.Unmarshal(data, &config); err != nil {
		logger.Logger.Warn("Failed to parse config file, using defaults", "error", err)
		return DefaultGlobalConfig(), nil
	}

	return config, nil
}

// SaveConfig saves the cache configuration to disk
func SaveConfig(config GlobalConfig) error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err = os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err = os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Logger.Info("Config saved successfully", "path", configPath)
	return nil
}

// CheckAndCleanup checks if cache exceeds threshold and performs cleanup if needed
func CheckAndCleanup(cacheDir string) error {
	config, err := LoadConfig()
	if err != nil {
		logger.Logger.Warn("Failed to load config", "error", err)
		config = DefaultGlobalConfig()
	}

	if !config.AutoClean {
		return nil
	}

	manager := NewManager(cacheDir, Config{MaxSizeBytes: config.MaxSizeBytes})

	size, err := manager.GetCacheSize()
	if err != nil {
		logger.Logger.Warn("Failed to check cache size", "error", err)
		return nil
	}

	if size > config.AutoCleanThreshold {
		logger.Logger.Info("Cache size exceeds threshold, performing automatic cleanup",
			"current_size", size,
			"threshold", config.AutoCleanThreshold)

		status, err := manager.CleanLRU()
		if err != nil {
			logger.Logger.Error("Automatic cache cleanup failed", "error", err)
			return err
		}

		logger.Logger.Info("Automatic cache cleanup completed",
			"files_deleted", status.FilesDeleted,
			"space_freed", status.SpaceFreed)
	}

	return nil
}
