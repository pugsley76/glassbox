#!/bin/bash
set -e

# Load configuration if it exists
CONFIG_FILE="size_thresholds.conf"
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
else
    echo "Configuration file $CONFIG_FILE not found. Using defaults."
    MAX_GO_SIZE=$((30 * 1024 * 1024))
    MAX_RUST_SIZE=$((20 * 1024 * 1024))
fi

EXIT_CODE=0

check_size() {
    local file=$1
    local max_size=$2

    if [ ! -f "$file" ]; then
        echo "Warning: File $file not found, skipping size check."
        return 0
    fi

    # Cross-platform stat (Linux/macOS)
    if stat -c%s "$file" >/dev/null 2>&1; then
        local size=$(stat -c%s "$file")
    else
        local size=$(stat -f%z "$file")
    fi

    echo "Size of $file: $size bytes (Max: $max_size bytes)"

    if [ "$size" -gt "$max_size" ]; then
        echo "ERROR: $file exceeds maximum allowed size of $max_size bytes!"
        return 1
    fi
    return 0
}

# Go artifacts
check_size "bin/glassbox" "$MAX_GO_SIZE" || EXIT_CODE=1
check_size "dist/release/glassbox-linux-amd64" "$MAX_GO_SIZE" || EXIT_CODE=1
check_size "dist/release/glassbox-linux-arm64" "$MAX_GO_SIZE" || EXIT_CODE=1
check_size "dist/release/glassbox-darwin-amd64" "$MAX_GO_SIZE" || EXIT_CODE=1
check_size "dist/release/glassbox-darwin-arm64" "$MAX_GO_SIZE" || EXIT_CODE=1
check_size "dist/release/glassbox-windows-amd64.exe" "$MAX_GO_SIZE" || EXIT_CODE=1

# Rust artifact
check_size "simulator/target/release/simulator" "$MAX_RUST_SIZE" || EXIT_CODE=1

if [ "$EXIT_CODE" -eq 0 ]; then
    echo "All binary sizes are within thresholds."
else
    echo "One or more binaries exceeded size thresholds."
fi

exit $EXIT_CODE
