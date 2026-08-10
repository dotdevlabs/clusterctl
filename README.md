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
# Interactive browser login
clusterctl auth login

# Or set token directly
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
clusterctl clusters flux-bootstrap <id>
```

> **Note:** `clusters update` accepts only infrastructure-level fields (`k8s-base-hostname`, `kubeconfig-export-namespace`, `cluster-issuer-name`, `ingress-class-name`, `gitops-repo-url`, `kubeconfig`). Cluster name and type cannot be changed after creation.

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
  [--source-chart <chart>] [--source-tag-pattern <pattern>]
clusterctl packages update <id> [--name <name>] [--description <desc>] \
  [--source-type <type>] [--source-url <url>] [--source-branch <branch>] \
  [--source-path <path>] [--source-chart <chart>] [--source-tag-pattern <pattern>]
clusterctl packages delete <id>
```

### deployments

```bash
clusterctl deployments list
clusterctl deployments get <id>
clusterctl deployments create --project-id <id> --name <name> --namespace <namespace> \
  --package-name <name> --package-version <ver> [--cluster-id <id>] [--values-override <yaml>]
clusterctl deployments update <id> [--project-id <id>] [--cluster-id <id>] [--name <name>] \
  [--namespace <namespace>] [--package-name <name>] [--package-version <ver>] \
  [--values-override <yaml>]
clusterctl deployments delete <id>
```

> **Note:** `--package-version` is required for `deployments create`. `--cluster-id` is optional (the deployment can be created without a cluster assignment).

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
- **Provision a vCluster** — auth, create virtual cluster, health-check
- **Create a package then a deployment** — register a Helm chart, deploy to a cluster
- **Materialize a secret** — create a secret, list it, materialize to clusters

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

Runs: `gofmt` · `go vet` · `golangci-lint` · `go test -race` (≥70% coverage gate) · `go build` · API spec drift check

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
