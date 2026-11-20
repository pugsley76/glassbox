#!/bin/bash
# Copyright 2026 Glassbox Users
# SPDX-License-Identifier: Apache-2.0
# Verify that Glassbox debug produces no ANSI escape characters when piped
# Usage: ./scripts/verify_no_ansi.sh
set -e

cd "$(dirname "$0")/.."
BIN="./glassbox"

# Build if needed
if [ ! -x "$BIN" ] && ! command -v glassbox &>/dev/null; then
  echo "Building glassbox..."
  go build -o glassbox ./cmd/glassbox/
fi
if [ -x "$BIN" ]; then
  : # use ./glassbox
elif command -v glassbox &>/dev/null; then
  BIN="glassbox"
else
  echo "Error: glassbox binary not found. Run: go build -o glassbox ./cmd/glassbox/"
  exit 1
fi

# Use a valid-length hash (64 hex chars) - will fail to fetch but we get initial output
HASH=$(printf '0%.0s' {1..64})

echo "Testing: $BIN debug $HASH 2>&1 | cat"
OUTPUT=$($BIN debug "$HASH" 2>&1 | cat)

if echo "$OUTPUT" | grep -q $'\033'; then
  echo "FAIL: Output contains ANSI escape sequences"
  echo "$OUTPUT" | cat -A
  exit 1
fi

echo "PASS: No ANSI escape characters in piped output"

# Also verify NO_COLOR disables colors
echo ""
echo "Testing: NO_COLOR=1 $BIN debug $HASH 2>&1 | cat"
OUTPUT2=$(NO_COLOR=1 $BIN debug "$HASH" 2>&1 | cat)
if echo "$OUTPUT2" | grep -q $'\033'; then
  echo "FAIL: NO_COLOR=1 still produced ANSI escape sequences"
  exit 1
fi
echo "PASS: NO_COLOR honored"
