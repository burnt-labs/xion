# xion — CLAUDE.md

The main Xion blockchain node repository (Cosmos SDK chain). Contains all chain logic, modules, and release infrastructure.

## Key Commands

```bash
make build                  # Build xiond binary
make test                   # Run unit tests
make lint                   # Run golangci-lint
make proto-gen              # Regenerate protobuf types
```

## GitHub Workflows

### Release Flow (most important)

Releasing is triggered by **manually running** `create-release.yaml` via `workflow_dispatch`, or by **pushing a tag** matching `v[0-9]+\.[0-9]+\.[0-9]+` (stable) or `v[0-9]+\.[0-9]+\.[0-9]+-rc[0-9]+` (release candidate).

A manual run must target an existing release tag, never a branch: `check-ref` rejects any ref that is not a `vX.Y.Z` or `vX.Y.Z-rcN` tag before anything is built or published. For example (do not run this as a test, it builds and pushes a release): `gh workflow run create-release.yaml -R burnt-labs/xion --ref v31.0.2-rc2`.

`create-release.yaml` ends with a **draft** GitHub release. Publishing that draft fires `publish-release.yaml`, and the downstream dispatches below wait for it. The Homebrew update does not: the GoReleaser run inside `create-release.yaml` (`exec-goreleaser.yaml`) already opens the `burnt-labs/homebrew-xion` cask PR for a stable tag (`skip_upload: auto` skips it for an rc).

1. **`create-release.yaml`** — Triggered on tag push. Kicks off the full build pipeline.
2. **`publish-release.yaml`** — Triggered on `release:published`. Runs GoReleaser (Fury packages, homebrew) and triggers downstream repos:
   - → **`burnt-labs/xion-types`** `release.yaml` — regenerates protobuf types for all languages
   - → **`burnt-labs/xion-assets`** — updates chain registry versions (via `repository_dispatch`)
   - → **`burnt-labs/xion-testnet-2`** `create-release.yml` — creates upgrade PR (**rc releases only**)
   - → **`burnt-labs/xion-mainnet-1`** `create-release.yml` — creates upgrade PR (**stable releases only**)

**Homebrew** (`burnt-labs/homebrew-xion`) is updated automatically by GoReleaser via `HOMEBREW_TAP_TOKEN` — it pushes a branch and creates a PR in homebrew-xion.

To republish only the apt/yum/apk packages for a release that is already out (for example after a failed Gemfury upload), dispatch `publish-release.yaml` with `packages_only`; it builds from the release tag itself and skips Homebrew and every downstream dispatch: `gh workflow run publish-release.yaml -R burnt-labs/xion --ref release/v32 -f release_tag=v31.0.1 -f packages_only=true`.

### Reusable Workflows (called by other jobs)

| Workflow | Purpose |
|----------|---------|
| `binaries-darwin.yaml` | Build Darwin binaries |
| `binaries-linux.yaml` | Build Linux binaries |
| `tests.yaml` | Run unit tests |
| `golangci-lint.yaml` | Lint |
| `e2e-tests.yaml` | End-to-end tests |
| `docker-build.yaml` / `docker-push.yaml` | Docker image build/push |
| `exec-goreleaser.yaml` | GoReleaser execution |
| `trigger-types.yaml` | Calls xion-types release workflow |
| `update-swagger.yaml` | Update OpenAPI/Swagger specifications |
| `docker-scout.yaml` | Docker image vulnerability scanning |
| `govulncheck.yaml` | Go dependency scanning, gated on reachable advisories |
| `verify-installers.yaml` | Verify release installers and artifacts |

### CI Workflows

- **`build-test.yaml`** — Triggered on PRs to `main`/`release/*` and `workflow_dispatch`
- **`claude-code-review.yml`** — Claude AI PR review
- **`claude.yml`** — Claude Code agent

## Upstream Triggers

This repo is the **source** of releases — no upstream triggers from other repos.

## Downstream Triggers

On every stable release:
- xion-types regenerates all language types
- xion-assets updates chain registry
- xion-mainnet-1 gets an upgrade PR
- homebrew-xion gets a formula update PR

On every rc release:
- xion-types regenerates all language types
- xion-assets updates chain registry
- xion-testnet-2 gets an upgrade PR

## Secrets Required

| Secret | Purpose |
|--------|---------|
| `GORELEASER_KEY` | GoReleaser Pro license |
| `GPG_PRIVATE_KEY` / `GPG_PASSPHRASE` | Package signing |
| `PEM_PRIVATE_KEY` | Package signing |
| `HOMEBREW_TAP_TOKEN` | Push to homebrew-xion |
| `FURY_TOKEN` | Publish to Gemfury |
| `AWS_OIDC_ROLE` | Docker ECR |
| `DOCKER_HUB_USERNAME` / `DOCKER_HUB_ACCESS_TOKEN` | Docker Hub |
| `BURNT_CLAUDE_API_KEY` | (optional) Used by `claude-code-review.yml` and `claude.yml` Claude Code workflows |
