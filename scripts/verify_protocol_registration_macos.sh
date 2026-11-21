#!/usr/bin/env bash
# Copyright (c) glassbox Authors.
# SPDX-License-Identifier: Apache-2.0
#
# verify_protocol_registration_macos.sh
#
# Smoke-check that the GLASSBOX Protocol app bundle is installed and that macOS
# LaunchServices has registered it for the "Glassbox" URL scheme.
#
# Usage:
#   bash scripts/verify_protocol_registration_macos.sh
#
# Prerequisites:
#   - macOS host with the app bundle installed at:
#       ~/Applications/GLASSBOX Protocol.app
#   - The bundle's Info.plist must exist at:
#       ~/Applications/GLASSBOX Protocol.app/Contents/Info.plist
#     and declare the URL scheme via:
#       <key>CFBundleURLSchemes</key>
#       ...
#       <string>Glassbox</string>
#   - The helper executable must exist and be executable at:
#       ~/Applications/GLASSBOX Protocol.app/Contents/MacOS/glassbox-protocol-handler
#
# Environment:
#   GLASSBOX_BINARY  Optional. When set, the script additionally verifies that
#                    the helper script references this binary path.
#
# Success:
#   Prints "macOS protocol registration verified" and exits 0.
#
#   Example:
#     $ bash scripts/verify_protocol_registration_macos.sh
#     macOS protocol registration verified
#
# Failure:
#   Prints a diagnostic to stderr and exits 1. Possible messages include:
#     - "missing plist: <path>"
#         The Info.plist was not found at the expected location.
#     - "missing helper script: <path>"
#         The helper executable is absent or not executable.
#     - A grep failure (exit 1) when the plist lacks the
#       CFBundleURLSchemes key or the "Glassbox" scheme string, or when the
#       helper script does not reference GLASSBOX_BINARY.
#     - A grep failure (exit 1) when the lsregister dump does not contain the
#       app path or the "glassbox" scheme (only checked when lsregister exists).
#
#   Example (missing plist):
#     $ bash scripts/verify_protocol_registration_macos.sh
#     missing plist: /Users/example/Applications/GLASSBOX Protocol.app/Contents/Info.plist
#     $ echo $?
#     1

set -euo pipefail

app_path="${HOME}/Applications/GLASSBOX Protocol.app"
plist_path="${app_path}/Contents/Info.plist"
helper_script="${app_path}/Contents/MacOS/glassbox-protocol-handler"
lsregister="/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

if [[ ! -f "${plist_path}" ]]; then
  echo "missing plist: ${plist_path}" >&2
  exit 1
fi

if [[ ! -x "${helper_script}" ]]; then
  echo "missing helper script: ${helper_script}" >&2
  exit 1
fi

grep -F '<key>CFBundleURLSchemes</key>' "${plist_path}" >/dev/null
grep -F '<string>Glassbox</string>' "${plist_path}" >/dev/null

if [[ -n "${GLASSBOX_BINARY:-}" ]]; then
  grep -F "${GLASSBOX_BINARY}" "${helper_script}" >/dev/null
fi

if [[ -x "${lsregister}" ]]; then
  "${lsregister}" -dump | grep -F "${app_path}" >/dev/null
  "${lsregister}" -dump | grep -F 'glassbox' >/dev/null
fi

echo "macOS protocol registration verified"