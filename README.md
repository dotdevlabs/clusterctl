# clusterctl

**clusterctl** is the ClusterControl lifecycle management CLI. It manages clusters, projects, packages, deployments, and secrets against the [ClusterControl](https://clustercontrol.co) API.

## Installation

### Homebrew

```bash
brew install dotdevlabs/tap/clusterctl
```

### go install

```bash
go install github.com/dotdevlabs/clusterctl@latest
```

### Install script

Downloads the latest release binary, verifies the checksum, and installs to `/usr/local/bin`:

```bash
curl -sSfL https://raw.githubusercontent.com/dotdevlabs/clusterctl/main/scripts/install.sh | sh
```

To pin a specific version or change the install directory:

```bash
VERSION=v0.1.0 INSTALL_DIR=~/.local/bin \
  curl -sSfL https://raw.githubusercontent.com/dotdevlabs/clusterctl/main/scripts/install.sh | sh
```

Supports Linux and macOS (amd64/arm64). Windows users should use `go install` or download the `.zip` from the [releases page](https://github.com/dotdevlabs/clusterctl/releases).

### Download a release

Pre-built binaries for Linux, macOS (Intel + Apple Silicon), and Windows are available on the [releases page](https://github.com/dotdevlabs/clusterctl/releases).

## Authentication

```bash
# Store a named context with a bearer token
clusterctl auth login --name prod --url https://api.clustercontrol.co --token <token>

# Or set token directly via environment variable (uses default URL)
export CLUSTERCTL_TOKEN=<your-token>
```

Named contexts are stored in `~/.config/atmt/clusterctl.yaml`. Switch between them with:

```bash
clusterctl context list
clusterctl context select <name>
```

The active context can be overridden per-command with `--context <name>` or `CLUSTERCTL_CONTEXT`.

## Global flags

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON envelope (`{data, pagination}`) |
| `--format` | Go template for custom output |
| `--context` | Named context to use |
| `--dry-run` | Print the request body without making API calls |
| `--verbose` | Verbose HTTP logging |

## Commands

> **Pagination**: All `list` subcommands automatically follow the server's `links.next` pagination links and return the complete result set across all pages. There is no need to pass page parameters manually.

### status

Check the running API server's deploy info (unauthenticated).

```bash
clusterctl status
```

### auth

```bash
clusterctl auth login --name <ctx> --url <base-url> --token <token>  # store a named context
clusterctl auth whoami                                                # verify token and return identity (organization, owner, token)
```

### registrations

Register a new organization and receive an API bearer token (unauthenticated).

```bash
clusterctl registrations create --owner-email <email> --label <token-name>
```

### clusters

```bash
clusterctl clusters list
clusterctl clusters get <id>
clusterctl clusters create --cluster-type <virtual|imported> [--name <name>] \
  [--parent-cluster-id <id>] [--kubeconfig <yaml>] [--gitops-repo-url <url>] \
  [--k8s-base-hostname <hostname>] [--kubeconfig-export-namespace <ns>] \
  [--cluster-issuer-name <name>] [--ingress-class-name <name>]
clusterctl clusters update <id> [--k8s-base-hostname <hostname>] \
  [--kubeconfig-export-namespace <ns>] [--cluster-issuer-name <name>] \
  [--ingress-class-name <name>] [--gitops-repo-url <url>] [--kubeconfig <yaml>]
clusterctl clusters delete <id>
clusterctl clusters health-check <id>
clusterctl clusters flux-bootstrap <id>          # trigger Flux bootstrap (POST)
clusterctl clusters flux-bootstrap-status <id>   # read Flux bootstrap status (GET)
clusterctl clusters provisioning <id>            # read provisioning status for a virtual cluster
clusterctl clusters expose <id>                  # expose a virtual cluster via host ingress
```

> **Note:** `clusters update` accepts only infrastructure-level fields. Cluster name and type cannot be changed after creation.

### projects

```bash
clusterctl projects list
clusterctl projects get <id>
clusterctl projects create --name <name> [--v-cluster-id <id>] [--github-pat-id <id>]
clusterctl projects update <id> [--name <name>] [--v-cluster-id <id>] [--github-pat-id <id>]
clusterctl projects delete <id>
```

### packages

```bash
clusterctl packages list
clusterctl packages get <id>
clusterctl packages create --name <name> [--description <desc>] [--source-type <helm|git>] \
  [--source-url <url>] [--source-branch <branch>] [--source-path <path>] \
  [--source-chart <chart>] [--source-tag-pattern <pattern>] \
  [--tags <comma-list-or-json-array>]
clusterctl packages update <id> [--name <name>] [--description <desc>] \
  [--source-type <type>] [--source-url <url>] [--source-branch <branch>] \
  [--source-path <path>] [--source-chart <chart>] [--source-tag-pattern <pattern>] \
  [--tags <comma-list-or-json-array>]
clusterctl packages delete <id>
```

#### packages releases

```bash
clusterctl packages releases list <package_id>
clusterctl packages releases get <package_id> <release_id>
```

### templates

List and inspect deployment templates available to your organization.

```bash
clusterctl templates list
clusterctl templates get <id>
```

### package-update-policies

Package update policies control automatic update behavior per deployment.

```bash
clusterctl package-update-policies list
clusterctl package-update-policies create --deployment-id <id> \
  [--package-id <id>] [--max-attempts <n>] [--is-blocked]
clusterctl package-update-policies get <id>
clusterctl package-update-policies update <id> \
  [--deployment-id <id>] [--package-id <id>] [--max-attempts <n>] [--is-blocked]
clusterctl package-update-policies delete <id>
```

### deployments

```bash
clusterctl deployments list
clusterctl deployments get <id>    # response includes is_auto_blocked and is_pinned fields
clusterctl deployments create --project-id <id> --name <name> --namespace <namespace> \
  --package-name <name> --package-version <ver> [--cluster-id <id>] \
  [--values-override <yaml>] [--environment-preset <name>] [--is-ai] \
  [--scaling-mode <hpa|keda|manual>] [--scaling-profile <name>] \
  [--min-replicas <n>] [--max-replicas <n>] [--desired-replicas <n>] \
  [--cpu-target-utilization <pct>] [--memory-target-utilization <pct>] \
  [--cpu-request <qty>] [--cpu-limit <qty>] \
  [--memory-request <qty>] [--memory-limit <qty>] \
  [--placement-policy <name>] \
  [--canary-enabled] [--canary-step-weight <pct>] [--canary-interval <dur>] \
  [--canary-max-weight <pct>] [--canary-success-threshold <0-1>] \
  [--canary-error-threshold <0-1>] [--canary-latency-p99-ms <ms>] \
  [--node-selector <json>] [--tolerations <json>] \
  [--template-extra-resources <json>] [--template-values <json>]
clusterctl deployments update <id> [--project-id <id>] [--cluster-id <id>] [--name <name>] \
  [--namespace <namespace>] [--package-name <name>] [--package-version <ver>] \
  [--values-override <yaml>] [--environment-preset <name>] [--is-ai] \
  [--scaling-mode <hpa|keda|manual>] [--scaling-profile <name>] \
  [--min-replicas <n>] [--max-replicas <n>] [--desired-replicas <n>] \
  [--cpu-target-utilization <pct>] [--memory-target-utilization <pct>] \
  [--cpu-request <qty>] [--cpu-limit <qty>] \
  [--memory-request <qty>] [--memory-limit <qty>] \
  [--placement-policy <name>] \
  [--canary-enabled] [--canary-step-weight <pct>] [--canary-interval <dur>] \
  [--canary-max-weight <pct>] [--canary-success-threshold <0-1>] \
  [--canary-error-threshold <0-1>] [--canary-latency-p99-ms <ms>] \
  [--node-selector <json>] [--tolerations <json>] \
  [--template-extra-resources <json>] [--template-values <json>]
clusterctl deployments delete <id>
```

> **Note:** `--package-version` is required for `deployments create`. `--cluster-id` is optional. The `get` response includes `is_auto_blocked` (true when automatic updates are blocked by a package update policy) and `is_pinned`.

**Complex JSON flags** accept inline JSON strings:

| Flag | Type | Example |
|------|------|---------|
| `--node-selector` | JSON object | `'{"kubernetes.io/os":"linux"}'` |
| `--tolerations` | JSON array | `'[{"key":"dedicated","operator":"Equal","value":"gpu","effect":"NoSchedule"}]'` |
| `--template-extra-resources` | JSON object (filename → YAML string) | `'{"extra.yaml":"apiVersion: v1\nkind: ConfigMap\n..."}'` |
| `--template-values` | JSON object (arbitrary key-value inputs) | `'{"image_repository":"ghcr.io/foo/bar","port":8080}'` |

#### deployments update-runs

```bash
clusterctl deployments update-runs list <deployment_id>
clusterctl deployments update-runs get <deployment_id> <run_id>
```

#### deployments pin / unpin

Pin a deployment to its current package version (blocks automatic updates):

```bash
clusterctl deployments pin <deployment_id>
clusterctl deployments unpin <deployment_id>
```

#### deployments package-update

```bash
clusterctl deployments package-update get <deployment_id>    # get latest package update record
clusterctl deployments package-update apply <deployment_id>  # apply a pending package update
```

#### deployments rollout

Trigger a manual rollout for a deployment:

```bash
clusterctl deployments rollout <deployment_id>
```

#### deployments revisions

List the recorded image revisions and their health for a deployment:

```bash
clusterctl deployments revisions <deployment_id>
```

#### deployments rollback

Roll back a deployment to a previous healthy revision. Without `--revision`, the API selects the most recent healthy revision. A successful rollback places a hold on image automation (Flux) until the hold is released.

```bash
clusterctl deployments rollback <deployment_id>
clusterctl deployments rollback <deployment_id> --revision <revision_id>
clusterctl deployments rollback <deployment_id> --reason "latency spike"
```

| Flag | Description |
|------|-------------|
| `--revision` | ID of a specific revision to target (default: previous healthy) |
| `--reason` | Human-readable reason for the rollback |

#### deployments rollback release

Release the image-automation hold placed by a rollback, resuming Flux automation:

```bash
clusterctl deployments rollback release <deployment_id>
```

### secrets

Secrets are scoped to a project. All secrets subcommands require `--project-id`.

```bash
clusterctl secrets list --project-id <id>
clusterctl secrets create --project-id <id> --secret-name <k8s-secret-name> --key <key> [--value <value>]
clusterctl secrets delete --project-id <id> <secret-id>
clusterctl secrets materialize --project-id <id>
```

A ClusterControl project secret maps to a Kubernetes Secret entry: `--secret-name` is the target Kubernetes Secret name and `--key` is the key within its data map. Multiple entries with the same `--secret-name` and different `--key` values are grouped into a single Kubernetes Secret on `materialize`.

```bash
# Add two keys to the same Kubernetes Secret
clusterctl secrets create --project-id <id> --secret-name app-secrets --key DATABASE_URL --value <url>
clusterctl secrets create --project-id <id> --secret-name app-secrets --key SECRET_KEY_BASE --value <key>

# Add a separate secret
clusterctl secrets create --project-id <id> --secret-name cloudflared-token --key TUNNEL_TOKEN --value <token>
```

### ai

```bash
# Print the full AI reference and common workflows
clusterctl ai
```

Common workflows included:
- **Verify identity and service status** — auth whoami, status
- **Provision a vCluster** — auth, create virtual cluster, check provisioning, health-check
- **Create a package then a deployment** — register a Helm chart, browse releases, deploy
- **Materialize a secret** — create a secret, list it, materialize to clusters
- **Check deployment auto-block status and remove a pin** — get deployment (is_auto_blocked/is_pinned), unpin
- **Review and manage package update policies** — list, create, get, update policies
- **Trigger a deployment rollout and inspect update runs** — package-update apply, update-runs list/get, rollout
- **Roll back a deployment and release the hold** — deployments revisions, rollback, rollback release
- **Browse available deployment templates** — templates list/get

### version

```bash
clusterctl version
clusterctl version --json   # machine-stable JSON output
```

## Examples

```bash
# List all clusters as JSON
clusterctl clusters list --json

# Create a virtual cluster nested under a parent
clusterctl clusters create \
  --name my-vcluster \
  --cluster-type virtual \
  --parent-cluster-id 9f8e7d6c-...

# Deploy a package to a cluster (--package-version is required)
clusterctl deployments create \
  --project-id abc123 \
  --cluster-id def456 \
  --name my-deployment \
  --namespace default \
  --package-name promtail \
  --package-version 1.2.0

# Deploy with HPA scaling, resource limits, and extra template resources
clusterctl deployments create \
  --project-id abc123 \
  --cluster-id def456 \
  --name my-app \
  --namespace production \
  --package-name my-app \
  --package-version 2.0.0 \
  --scaling-mode hpa \
  --min-replicas 2 \
  --max-replicas 10 \
  --cpu-request 250m \
  --cpu-limit 500m \
  --memory-request 256Mi \
  --memory-limit 512Mi \
  --node-selector '{"kubernetes.io/os":"linux"}' \
  --template-extra-resources '{"extra-config.yaml":"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: extra\n"}'

# Materialize secrets in a project
clusterctl secrets materialize --project-id abc123

# Dry-run a create to inspect the request body
clusterctl clusters create --name test --cluster-type imported --dry-run
```

## Development

### Prerequisites

- Go 1.21+
- [golangci-lint](https://golangci-lint.run/usage/install/)

### Build

```bash
go build ./...
```

### Test

```bash
go test ./... -race
```

### CI

```bash
bin/ci
```

Runs: `gofmt` · `go vet` · `golangci-lint` · `go test -race` (≥70% coverage gate, includes API spec conformance) · `go build` · API spec drift check

The test suite includes two complementary API spec conformance checks in `cmd/conformance/`:

- **Forward conformance** (`TestOperationCoverage`): every operation published in `docs/api_spec.yaml` must map to a CLI command. Adding a new spec operation without implementing its command fails CI.
- **Reverse conformance** (`TestSecretsDeleteConformance` and similar per-resource tests): no CLI command may call an API operation that the spec does not publish. For example, the ClusterControl API publishes no `GET /projects/{project_id}/secrets/{id}` endpoint; `secrets delete` must send only `DELETE` and must not call a `GET` first.

The drift check compares `docs/api_spec.yaml` (the vendored ClusterControl API spec) against the upstream published spec. It requires a `GITHUB_TOKEN` or `gh` CLI session with read access to `dotdevlabs/clustercontrol`. When cross-repo access is unavailable (e.g. the default `GITHUB_TOKEN` in Actions for a fork), the check skips with a warning rather than failing CI. To enable enforcement, add a fine-grained PAT with `contents: read` on `dotdevlabs/clustercontrol` as a `CLUSTERCONTROL_READ_TOKEN` org secret. To update the vendored spec locally:

```bash
bash scripts/update-api-spec.sh
```

## Release

Releases are handled automatically via [goreleaser](https://goreleaser.com/) on git tags. Static binaries are produced for Linux, macOS (amd64/arm64), and Windows. A Homebrew formula is published to `dotdevlabs/homebrew-tap`.

```bash
git tag v0.1.0
git push origin v0.1.0
```

## License

MIT
