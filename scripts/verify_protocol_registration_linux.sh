#!/usr/bin/env bash
# Copyright (c) glassbox Authors.
# SPDX-License-Identifier: Apache-2.0
#
# Verifies Linux protocol registration for the Glassbox scheme.
#
# Registration scope and permissions (see docs/protocol-registration.md):
#   - Desktop entry:  ~/.local/share/applications/glassbox-protocol.desktop
#   - Helper script:  ~/.local/share/Glassbox/glassbox-protocol-handler
#   - Registered via: xdg-mime default ... x-scheme-handler/Glassbox
#   - Scope:          per-user (no root / sudo required)
#
# Inspect status afterward with:
#   glassbox protocol:status
#   glassbox protocol:verify
#   glassbox protocol:diagnose
#
# Do NOT run this script with sudo or as root: it inspects files under $HOME
# and the per-user xdg-mime association. Elevated access is unnecessary and
# would inspect the wrong user's registration.

set -euo pipefail

desktop_file="${HOME}/.local/share/applications/glassbox-protocol.desktop"
helper_script="${HOME}/.local/share/Glassbox/glassbox-protocol-handler"

if [[ ! -f "${desktop_file}" ]]; then
  echo "missing desktop file: ${desktop_file}" >&2
  echo "hint: registration is per-user; run 'glassbox protocol:register' as your normal user (no sudo)" >&2
  exit 1
fi

if [[ ! -x "${helper_script}" ]]; then
  echo "missing helper script: ${helper_script}" >&2
  echo "hint: registration is per-user; run 'glassbox protocol:register' as your normal user (no sudo)" >&2
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
  echo "hint: re-run 'glassbox protocol:register' as your normal user (no sudo)" >&2
  exit 1
fi

echo "linux protocol registration verified"