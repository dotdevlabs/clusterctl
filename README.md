# clusterctl

Cluster lifecycle management CLI tool.

## Installation

### Homebrew

```bash
brew install dotdevlabs/tap/clusterctl
```

### From source

```bash
go install github.com/dotdevlabs/clusterctl@latest
```

## Usage

```bash
# Print version
clusterctl version

# Print version as JSON (machine-stable output)
clusterctl version --json
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

Runs: `gofmt` · `go vet` · `golangci-lint` · `go test` · `go build`

## Release

Releases are handled automatically via [goreleaser](https://goreleaser.com/) on git tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

## License

MIT
