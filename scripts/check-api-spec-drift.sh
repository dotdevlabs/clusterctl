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

UPSTREAM=""
if [ -n "${GITHUB_TOKEN:-}" ]; then
  # The GITHUB_TOKEN in GitHub Actions may not have cross-repo read access to
  # dotdevlabs/clustercontrol (a private repo). Treat any fetch failure as a
  # skip rather than a CI break; the drift check is advisory when cross-repo
  # access is unavailable. Set CLUSTERCONTROL_READ_TOKEN as an org secret to
  # enable the check with a fine-grained PAT that has contents:read on that repo.
  if ! UPSTREAM=$(curl -sSf \
    -H "Authorization: Bearer $GITHUB_TOKEN" \
    "https://raw.githubusercontent.com/dotdevlabs/clustercontrol/main/docs/api_spec.yaml" 2>/dev/null); then
    echo "==> skip spec drift check (upstream fetch failed — GITHUB_TOKEN may lack cross-repo access)"
    exit 0
  fi
elif gh auth status &>/dev/null 2>&1; then
  if ! UPSTREAM=$(gh api repos/dotdevlabs/clustercontrol/contents/docs/api_spec.yaml --jq '.content' 2>/dev/null | base64 -d 2>/dev/null); then
    echo "==> skip spec drift check (upstream fetch via gh CLI failed)"
    exit 0
  fi
else
  echo "==> skip spec drift check (gh not authenticated)"
  exit 0
fi

if [ -z "$UPSTREAM" ]; then
  echo "==> skip spec drift check (empty upstream response)"
  exit 0
fi

if ! diff <(echo "$UPSTREAM") "$VENDORED" > /dev/null; then
  echo "ERROR: Vendored spec (docs/api_spec.yaml) differs from upstream ClusterControl API spec."
  echo "Run: bash scripts/update-api-spec.sh"
  exit 1
fi
echo "==> spec is in sync with upstream"
