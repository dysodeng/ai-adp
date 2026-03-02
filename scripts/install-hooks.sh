#!/usr/bin/env bash
# Install Git hooks for the ai-adp project

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOKS_DIR="${REPO_ROOT}/.git/hooks"
SCRIPTS_HOOKS_DIR="${REPO_ROOT}/scripts/hooks"

if [ ! -d "$SCRIPTS_HOOKS_DIR" ]; then
  echo "No hooks directory found at $SCRIPTS_HOOKS_DIR, skipping."
  exit 0
fi

echo "Installing Git hooks..."
for hook in "$SCRIPTS_HOOKS_DIR"/*; do
  hook_name="$(basename "$hook")"
  target="${HOOKS_DIR}/${hook_name}"
  cp "$hook" "$target"
  chmod +x "$target"
  echo "  Installed: $hook_name"
done
echo "Done."
