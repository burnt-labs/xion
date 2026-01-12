# Release Flow Documentation

This document describes the release workflow between the `xion` repository, `xion-types` package publishing, and `homebrew-xion` formula updates.

## Overview

When releases are created in the `xion` repository, they trigger workflows that:
- Publish type packages to various registries (npm, PyPI, crates.io, RubyGems, CocoaPods)
- Update Homebrew formulas (only for releases marked as "latest")
- Update xion-assets chain registry (mainnet only for "latest", testnet for any release)

## GitHub Release Event Types

GitHub provides three relevant release event types:

| Event Type | When It Fires |
|------------|---------------|
| `prereleased` | A pre-release is created (e.g., `v1.0.0-rc.1`) |
| `released` | A release is published, OR a pre-release is converted to a release |
| `published` | A release is marked as "latest" |

**Reference:** https://docs.github.com/en/actions/reference/workflows-and-actions/events-that-trigger-workflows#release

## Workflow Chain

```
┌───────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       xion repository                                          │
│                                                                                               │
│  GitHub Release Event (prereleased / released / published)                                    │
│                                         │                                                     │
│                                         ▼                                                     │
│                            publish-release.yaml                                               │
│                                         │                                                     │
│         ┌───────────────┬───────────────┼───────────────┬───────────────┐                    │
│         ▼               ▼               ▼               ▼               ▼                    │
│  trigger-types    trigger-homebrew   trigger-assets-mainnet   trigger-assets-testnet         │
│  (all events)     (published only)   (published only)         (released only)                │
│         │               │               │                       │                            │
└─────────┼───────────────┼───────────────┼───────────────────────┼────────────────────────────┘
          │               │               │                       │
          │ repo_dispatch │ repo_dispatch │ repo_dispatch         │ repo_dispatch
          │               │               │                       │
          ▼               ▼               ▼                       ▼
┌──────────────────┐ ┌──────────────┐ ┌─────────────────────────────────────────┐
│  xion-types      │ │ homebrew-xion│ │              xion-assets                 │
│                  │ │              │ │                                          │
│  ts/rust/python/ │ │ update-      │ │  update-mainnet-chain-registry.yaml     │
│  ruby/swift-     │ │ release.yaml │ │  (published only = "latest")            │
│  protobuf.yaml   │ │      │       │ │                  │                      │
│       │          │ │      ▼       │ │                  ▼                      │
│       ▼          │ │  Homebrew    │ │  update-testnet-chain-registry.yaml     │
│  npm, crates.io  │ │  formulas    │ │  (released = any non-prerelease)        │
│  PyPI, RubyGems  │ │              │ │                  │                      │
│  CocoaPods       │ │              │ │                  ▼                      │
│                  │ │              │ │  Chain registry updates + PRs           │
└──────────────────┘ └──────────────┘ └─────────────────────────────────────────┘
```

## Release Scenarios

### 1. Pre-release Created (e.g., `v1.0.0-rc.1`)

```
User creates pre-release on GitHub
        │
        ▼
GitHub fires: prereleased event
        │
        ▼
publish-release.yaml triggers
        │
        ▼
trigger-types.yaml sends:
  release_type: "prereleased"
        │
        ▼
xion-types workflows match: ✅
        │
        ▼
RC packages published:
  - @burnt-labs/xion-types@1.0.0-rc.1 (npm)
  - xion-types 1.0.0-rc.1 (crates.io)
  - xion-types 1.0.0rc1 (PyPI)
  - etc.
```

### 2. Pre-release Converted to Release (NOT marked as latest)

```
User converts pre-release to release (without marking as latest)
        │
        ▼
GitHub fires: released event
        │
        ▼
publish-release.yaml triggers
        │
        ├─► trigger-types.yaml sends: release_type: "released"
        │         │
        │         ▼
        │   xion-types workflows match: ❌
        │   No packages published (RC already exists)
        │
        └─► trigger-assets-testnet.yaml sends to xion-assets
                  │
                  ▼
            Testnet chain registry updated ✅
            (Mainnet NOT updated)
```

### 3. Release Marked as Latest (e.g., `v1.0.0`)

```
User creates/marks release as "latest"
        │
        ▼
GitHub fires: published event
        │
        ▼
publish-release.yaml triggers
        │
        ├─► trigger-types.yaml sends: release_type: "published"
        │         │
        │         ▼
        │   xion-types workflows match: ✅
        │   Stable packages published:
        │     - @burnt-labs/xion-types@1.0.0 (npm, tagged latest)
        │     - xion-types 1.0.0 (crates.io, PyPI, etc.)
        │
        ├─► trigger-homebrew.yaml
        │         │
        │         ▼
        │   Homebrew formula updated ✅
        │
        └─► trigger-assets-mainnet.yaml sends to xion-assets
                  │
                  ▼
            Mainnet chain registry updated ✅
            (Testnet NOT updated)
```

---

## Homebrew Flow

The Homebrew formula is **only updated when a release is marked as latest**.

### Workflow Chain

```
┌─────────────────────────────────────────────────────────────────┐
│                    xion repository                               │
│                                                                  │
│  GitHub Release Event: published (marked as latest)             │
│                           │                                      │
│                           ▼                                      │
│              publish-release.yaml                                │
│                           │                                      │
│                           ▼                                      │
│   trigger-homebrew job (if: github.event.action == 'published') │
│                           │                                      │
│                           ▼                                      │
│              trigger-homebrew.yaml                               │
│                           │                                      │
└───────────────────────────┼─────────────────────────────────────┘
                            │
                            │  repository_dispatch
                            │  (homebrew-release-trigger)
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────┐
│                   homebrew-xion repository                       │
│                                                                  │
│  Receives: { tag_name, release_name }                           │
│                           │                                      │
│                           ▼                                      │
│              update-release.yaml                                 │
│                           │                                      │
│                           ▼                                      │
│  - Downloads checksums from xion release                        │
│  - Updates Formula/xiond.rb (main formula)                      │
│  - Updates Formula/xiond@{major}.rb (versioned formula)         │
│  - Updates Formula/xiond@{version}.rb (specific version)        │
│  - Creates PR to homebrew-xion                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Homebrew Trigger Conditions

| Scenario | GitHub Event | Homebrew Triggered? |
|----------|--------------|---------------------|
| Pre-release created | `prereleased` | ❌ No |
| Release (not latest) | `released` | ❌ No |
| Release marked as latest | `published` | ✅ Yes |

### Why Homebrew Only Updates on "Latest"

1. **User expectation** - `brew install xiond` should install the stable, latest version
2. **Avoid RC in Homebrew** - Pre-releases should not be distributed via Homebrew
3. **Single source of truth** - The "latest" release is the official stable version

### Homebrew Update Process

When triggered, the `update-release.yaml` workflow:

1. Extracts version from tag (e.g., `v1.0.0` → `1.0.0`)
2. Downloads checksums file from the xion release
3. Updates three formula files:
   - `Formula/xiond.rb` - Main formula (always latest)
   - `Formula/xiond@{major}.rb` - Major version formula (e.g., `xiond@21`)
   - `Formula/xiond@{version}.rb` - Specific version formula (e.g., `xiond@21.0.0`)
4. Creates a PR to the `homebrew-xion` repository

---

## xion-assets Flow

The xion-assets repository maintains chain registry files for both mainnet and testnet. **Mainnet updates only when a release is marked as latest**, while **testnet updates on any non-prerelease**.

### Workflow Chain

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           xion repository                                    │
│                                                                             │
│  GitHub Release Event                                                       │
│         │                                                                   │
│         ├── published (latest) ──► trigger-assets-mainnet.yaml             │
│         │                                   │                               │
│         └── released (any) ──────► trigger-assets-testnet.yaml             │
│                                             │                               │
└─────────────────────────────────────────────┼───────────────────────────────┘
                                              │
                                              │  repository_dispatch
                                              │
                                              ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        xion-assets repository                                │
│                                                                             │
│  Mainnet: xion-assets-mainnet-release-trigger                               │
│         │                                                                   │
│         ▼                                                                   │
│  update-mainnet-chain-registry.yaml                                         │
│  - Updates public/chain-registry/xion/chain.json                           │
│  - Updates public/chain-registry/xion/versions.json                        │
│  - Creates PR to xion-assets                                               │
│                                                                             │
│  Testnet: xion-assets-testnet-release-trigger                               │
│         │                                                                   │
│         ▼                                                                   │
│  update-testnet-chain-registry.yaml                                         │
│  - Updates public/chain-registry/testnets/xiontestnet2/chain.json          │
│  - Updates public/chain-registry/testnets/xiontestnet2/versions.json       │
│  - Creates PR to xion-assets                                               │
└─────────────────────────────────────────────────────────────────────────────┘
```

### xion-assets Trigger Conditions

| Scenario | GitHub Event | Mainnet Updated? | Testnet Updated? |
|----------|--------------|------------------|------------------|
| Pre-release created | `prereleased` | ❌ No | ❌ No |
| Release (not latest) | `released` | ❌ No | ✅ Yes |
| Release marked as latest | `published` | ✅ Yes | ❌ No |

### Why Mainnet Only Updates on "Latest"

1. **Production stability** - Mainnet chain registry should only reflect stable, official releases
2. **Prevents accidental updates** - Non-latest releases should not update mainnet infrastructure
3. **Clear separation** - Testnet can be updated more frequently for testing purposes

---

## Summary Tables

### xion-types Publishing

| Scenario | GitHub Event | `release_type` | Packages Published? |
|----------|--------------|----------------|---------------------|
| Pre-release created | `prereleased` | `"prereleased"` | ✅ RC packages |
| Pre-release → release (not latest) | `released` | `"released"` | ❌ Skipped |
| Release marked as latest | `published` | `"published"` | ✅ Stable packages |

### Homebrew Updates

| Scenario | GitHub Event | Homebrew Updated? |
|----------|--------------|-------------------|
| Pre-release created | `prereleased` | ❌ No |
| Pre-release → release (not latest) | `released` | ❌ No |
| Release marked as latest | `published` | ✅ Yes (PR created) |

### xion-assets Updates

| Scenario | GitHub Event | Mainnet Updated? | Testnet Updated? |
|----------|--------------|------------------|------------------|
| Pre-release created | `prereleased` | ❌ No | ❌ No |
| Pre-release → release (not latest) | `released` | ❌ No | ✅ Yes (PR created) |
| Release marked as latest | `published` | ✅ Yes (PR created) | ❌ No |

## Rationale

### Why skip "released but not latest"?

1. **RC package already exists** - Users testing can use the pre-release package
2. **Avoids confusion** - Only "latest" releases should update the `latest` tag in registries
3. **Prevents accidental publishes** - Releases not marked as latest are typically maintenance or cleanup

### Typical Release Flow

```
v1.0.0-rc.1  →  prereleased  →  RC package published
v1.0.0-rc.2  →  prereleased  →  RC package published (users test)
v1.0.0       →  published    →  Stable package published (marked latest)
```

## Other Triggered Workflows

| Workflow | Triggers On | Purpose |
|----------|-------------|---------|
| `trigger-types.yaml` | All release events | Publish type packages |
| `trigger-assets-mainnet.yaml` | `published` only | Update mainnet chain registry (latest releases) |
| `trigger-assets-testnet.yaml` | `released` only | Update testnet chain registry (any non-prerelease) |
| `trigger-homebrew.yaml` | `published` only | Update Homebrew formula |

## Configuration Files

### xion repository
- `xion/.github/workflows/publish-release.yaml` - Main release workflow
- `xion/.github/workflows/trigger-types.yaml` - Dispatches to xion-types
- `xion/.github/workflows/trigger-homebrew.yaml` - Dispatches to homebrew-xion
- `xion/.github/workflows/trigger-assets-mainnet.yaml` - Dispatches to xion-assets (mainnet, latest only)
- `xion/.github/workflows/trigger-assets-testnet.yaml` - Dispatches to xion-assets (testnet, any release)

### xion-types repository
- `xion-types/.github/workflows/ts-protobuf.yaml` - TypeScript/npm publishing
- `xion-types/.github/workflows/rust-protobuf.yaml` - Rust/crates.io publishing
- `xion-types/.github/workflows/python-protobuf.yaml` - Python/PyPI publishing
- `xion-types/.github/workflows/ruby-protobuf.yaml` - Ruby/RubyGems publishing
- `xion-types/.github/workflows/swift-protobuf.yaml` - Swift/CocoaPods publishing

### xion-assets repository
- `xion-assets/.github/workflows/update-mainnet-chain-registry.yaml` - Updates mainnet chain registry
- `xion-assets/.github/workflows/update-testnet-chain-registry.yaml` - Updates testnet chain registry

### homebrew-xion repository
- `homebrew-xion/.github/workflows/update-release.yaml` - Updates Homebrew formulas

