#!/usr/bin/env bash
set -euo pipefail

VENDORED="docs/api_spec.yaml"

if [ ! -f "$VENDORED" ]; then
  echo "ERROR: Vendored spec not found at $VENDORED"
  exit 1
fi

# Skip if no auth credentials available
if [ -z "${GITHUB_TOKEN:-}" ] && ! command -v gh &>/dev/null; then
  echo "==> skip spec drift check (no GITHUB_TOKEN and no gh CLI)"
  exit 0
fi

if [ -n "${GITHUB_TOKEN:-}" ]; then
  UPSTREAM=$(curl -sSf \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    "https://raw.githubusercontent.com/dotdevlabs/clustercontrol/main/docs/api_spec.yaml")
elif gh auth status &>/dev/null 2>&1; then
  UPSTREAM=$(gh api repos/dotdevlabs/clustercontrol/contents/docs/api_spec.yaml --jq '.content' | base64 -d)
else
  echo "==> skip spec drift check (gh not authenticated)"
  exit 0
fi

if ! diff <(echo "$UPSTREAM") "$VENDORED" > /dev/null; then
  echo "ERROR: Vendored spec (docs/api_spec.yaml) differs from upstream ClusterControl API spec."
  echo "Run: bash scripts/update-api-spec.sh"
  exit 1
fi
echo "==> spec is in sync with upstream"
