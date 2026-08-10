#!/usr/bin/env bash
set -euo pipefail
gh api repos/dotdevlabs/clustercontrol/contents/docs/api_spec.yaml --jq '.content' | base64 -d > docs/api_spec.yaml
echo "Updated docs/api_spec.yaml"
