# Agent guide

This repository is the OpenTofu/Terraform provider for the native **Polyfin**
administration API. The provider is security-first: it manages server-scoped
addons and settings only, never user data, and it never reads or writes a
secret in the state.

## Toolchain

| Tool | Version | Source of truth |
|------|---------|-----------------|
| Go | `go 1.24.0` | the `go` directive of `go.mod` |

`go.mod` is the authority for the Go version. `mise.lock` pins the exact
archives and their sha256 checksums (macos-arm64, linux-x64, linux-arm64),
and `mise.toml` holds the toolchain contract: `mise install --locked`
installs the pinned Go and `GOTOOLCHAIN=local` forbids any implicit
toolchain download. A Go version change updates `go.mod` **and** `mise.lock`
(`mise lock`), never one alone.

## Pod setup (Optio agent)

The Optio agent pod image ships Node.js only: `go` is missing there, so the
provider tests cannot run until the toolchain is installed. `.optio/setup.sh`
is the setup hook Optio runs once per pod, from the repository checkout, at
pod initialisation (`/opt/optio/repo-init.sh`).

The script:

* reads the Go version from `go.mod` and cross-checks it with `mise.lock`;
* downloads the official `dl.google.com` archive named in `mise.lock`;
* verifies the sha256 recorded in `mise.lock` **before** extracting it;
* installs it in `$HOME/.local/optio/go`, without root and without any system
  package manager, and exposes `go` on the `PATH` through
  `$HOME/.local/bin/go`;
* is idempotent: an up-to-date installation is left untouched and nothing is
  downloaded again.

Re-running the script is safe. Its exit status is always `0` in pod mode,
because the Optio init script runs with `set -e` and a failure there would
keep the pod from starting. Set `OPTIO_SETUP_STRICT=1` to propagate the
failures instead (this is what the tests use).

In a pod, `go version` reports `go version go1.24.0 linux/amd64` once the
setup has run.

## Commands

The CI (`.github/workflows/test.yml`) runs:

```sh
go test -race ./...
go vet ./...
```

`mise run test` runs the same commands. The acceptance target under
`testdata/acceptance` needs Docker and is ignored by the go tool: it never
runs with `go test ./...`.

## Repository rules

* every registered resource and data source needs a page under `docs/` and an
  example under `examples/` (`internal/provider/docs_coverage_test.go`);
* no cookie, token, key or manifest URL in the state, in a log or in the
  repository: credentials come from protected runtime files only;
* the provider contract is described in `docs/architecture.md`, the Registry
  documentation in `docs/index.md`.
