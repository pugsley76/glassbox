#!/usr/bin/env bash
# Copyright (c) glassbox Authors.
# SPDX-License-Identifier: Apache-2.0
#
# verify_protocol_registration_linux.sh
#
# Verifies that the Glassbox custom protocol handler (x-scheme-handler/Glassbox)
# is registered correctly for the current Linux user.
#
# Usage:
#   bash scripts/verify_protocol_registration_linux.sh
#
# Prerequisites:
#   - Desktop entry installed at:
#       ~/.local/share/applications/glassbox-protocol.desktop
#   - Helper script installed and executable at:
#       ~/.local/share/Glassbox/glassbox-protocol-handler
#   - xdg-mime available on PATH (package: xdg-utils)
#
# Environment variables:
#   GLASSBOX_BINARY
#     Optional. When set, the script additionally checks that this path
#     appears in the helper script.
#
# Checks performed:
#   - The desktop file exists.
#   - The helper script exists and is executable.
#   - The desktop file contains "MimeType=x-scheme-handler/Glassbox;".
#   - The desktop file contains "Exec=<helper_script> %u".
#   - If GLASSBOX_BINARY is set, the helper script references it.
#   - "xdg-mime query default x-scheme-handler/Glassbox" returns
#     "glassbox-protocol.desktop".
#
# Success output (exit code 0):
#   linux protocol registration verified
#
# Failure output (exit code 1):
#   missing desktop file: <path>
#   missing helper script: <path>
#   xdg-mime returned <handler>, expected glassbox-protocol.desktop
#   (a missing MimeType/Exec line or GLASSBOX_BINARY reference also exits 1
#    via grep)
#
# Example - success:
#   $ bash scripts/verify_protocol_registration_linux.sh
#   linux protocol registration verified
#
# Example - troubleshooting failure (no default handler registered):
#   $ xdg-mime query default x-scheme-handler/Glassbox
#   (empty output)
#   $ bash scripts/verify_protocol_registration_linux.sh
#   xdg-mime returned , expected glassbox-protocol.desktop
#   $ echo $?
#   1
#
# Example - troubleshooting failure (different handler registered):
#   $ xdg-mime query default x-scheme-handler/Glassbox
#   other-app.desktop
#   $ bash scripts/verify_protocol_registration_linux.sh
#   xdg-mime returned other-app.desktop, expected glassbox-protocol.desktop
#   $ echo $?
#   1
#
# To fix a failing default-handler check, re-register the desktop file:
#   xdg-mime default glassbox-protocol.desktop x-scheme-handler/Glassbox

set -euo pipefail

desktop_file="${HOME}/.local/share/applications/glassbox-protocol.desktop"
helper_script="${HOME}/.local/share/Glassbox/glassbox-protocol-handler"

if [[ ! -f "${desktop_file}" ]]; then
  echo "missing desktop file: ${desktop_file}" >&2
  exit 1
fi

if [[ ! -x "${helper_script}" ]]; then
  echo "missing helper script: ${helper_script}" >&2
  exit 1
fi

grep -F "MimeType=x-scheme-handler/Glassbox;" "${desktop_file}" >/dev/null
grep -F "Exec=${helper_script} %u" "${desktop_file}" >/dev/null

if [[ -n "${GLASSBOX_BINARY:-}" ]]; then
  grep -F "${GLASSBOX_BINARY}" "${helper_script}" >/dev/null
fi

default_handler="$(xdg-mime query default x-scheme-handler/Glassbox)"
if [[ "${default_handler}" != "glassbox-protocol.desktop" ]]; then
  echo "xdg-mime returned ${default_handler}, expected glassbox-protocol.desktop" >&2
  exit 1
fi

echo "linux protocol registration verified"