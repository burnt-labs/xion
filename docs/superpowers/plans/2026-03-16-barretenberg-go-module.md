# barretenberg-go Module Extraction — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract Barretenberg UltraHonk Go bindings from xion into a standalone Go module repo (`github.com/burnt-labs/barretenberg-go`).

**Architecture:** The new repo is a self-contained Go module with committed platform-specific `.a` archives, C++ wrapper shim, and CGo bindings. CI builds the archives from Aztec's pre-built releases. Xion imports it as a normal Go dependency.

**Tech Stack:** Go, CGo, C++20, GitHub Actions, Aztec barretenberg v4.0.4

**Spec:** `docs/superpowers/specs/2026-03-16-barretenberg-go-module-design.md`

---

## Chunk 1: Repository Scaffolding

### Task 1: Create the GitHub repo and initialize Go module

This task creates the `burnt-labs/barretenberg-go` repo on GitHub and initializes it.

**Files:**
- Create: `go.mod`
- Create: `.gitignore`

- [ ] **Step 1: Create GitHub repo**

```bash
gh repo create burnt-labs/barretenberg-go --public --clone --description "Go bindings for Barretenberg UltraHonk proof verification"
cd barretenberg-go
```

- [ ] **Step 2: Initialize Go module**

```bash
go mod init github.com/burnt-labs/barretenberg-go
```

This creates `go.mod`. Edit it to match xion's Go version:
```
module github.com/burnt-labs/barretenberg-go

go 1.25.3
```

No `go.sum` will be created because there are no external dependencies.

- [ ] **Step 3: Create .gitignore**

```gitignore
# Build artifacts
*.o
*.a.tmp

# IDE
.idea/
.vscode/
*.swp

# OS
.DS_Store
```

Note: do NOT gitignore `lib/**/*.a` — those are committed intentionally.

- [ ] **Step 4: Commit**

```bash
git add go.mod .gitignore
git commit -m "feat: initialize Go module"
```

### Task 2: Add checksums.json

**Files:**
- Create: `checksums.json`

- [ ] **Step 1: Create checksums.json**

```json
{
  "aztec_tag": "v4.0.4",
  "aztec_repo": "https://github.com/AztecProtocol/aztec-packages",
  "msgpack_commit": "c0334576ed657fb3b3c49e8e61402989fb84146d",
  "assets": {
    "linux_amd64": {
      "file": "barretenberg-static-amd64-linux.tar.gz",
      "sha256": "7578c9fc80dec89988acd6038ff733f88ff1847eef2e06d16504357e8ad6373a"
    },
    "linux_arm64": {
      "file": "barretenberg-static-arm64-linux.tar.gz",
      "sha256": "1a0b3bd4b2a6dc95c7e15106cd71a161e5f47c40b2bdc06bc155f8c69e44a958"
    },
    "darwin_amd64": {
      "file": "barretenberg-static-amd64-darwin.tar.gz",
      "sha256": "20f77d04b477770f288074e4876ace2044ae265b42a290a7acc164efdb8d9d7e"
    },
    "darwin_arm64": {
      "file": "barretenberg-static-arm64-darwin.tar.gz",
      "sha256": "324249ed62dc266a1d6fea9ee33b37494ea0edc80b0c848bf94f0a1b3ce77a8e"
    }
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add checksums.json
git commit -m "feat: add Aztec release checksums for v4.0.4"
```

### Task 3: Add C++ wrapper and header

Copy the wrapper shim and header from xion. These are the custom C code that bridges barretenberg's C++ API to a C interface for CGo.

**Files:**
- Create: `wrapper/barretenberg_wrapper.cpp` (copy from `xion/x/zk/barretenberg/wrapper/barretenberg_wrapper.cpp`)
- Create: `include/barretenberg_wrapper.h` (copy from `xion/x/zk/barretenberg/include/barretenberg_wrapper.h`)

- [ ] **Step 1: Create directories**

```bash
mkdir -p wrapper include
```

- [ ] **Step 2: Copy wrapper source**

Copy `barretenberg_wrapper.cpp` from xion verbatim. The file is 283 lines. No modifications needed — it uses relative include paths that work with the new repo layout.

Source: `xion/x/zk/barretenberg/wrapper/barretenberg_wrapper.cpp`
Destination: `wrapper/barretenberg_wrapper.cpp`

- [ ] **Step 3: Copy header**

Copy `barretenberg_wrapper.h` from xion verbatim. 126 lines, no changes needed.

Source: `xion/x/zk/barretenberg/include/barretenberg_wrapper.h`
Destination: `include/barretenberg_wrapper.h`

- [ ] **Step 4: Commit**

```bash
git add wrapper/ include/
git commit -m "feat: add C++ wrapper shim and C header"
```

### Task 4: Add Go source files

Copy all Go source files from xion. The only change is `package barretenberg` stays the same (the module path `barretenberg-go` doesn't match, but Go allows explicit package declarations).

**Files:**
- Create: `doc.go` (copy from `xion/x/zk/barretenberg/doc.go`)
- Create: `errors.go` (copy from `xion/x/zk/barretenberg/errors.go`)
- Create: `bindings.go` (copy from `xion/x/zk/barretenberg/bindings.go`)
- Create: `vkey.go` (copy from `xion/x/zk/barretenberg/vkey.go`)
- Create: `proof.go` (copy from `xion/x/zk/barretenberg/proof.go`)
- Create: `verifier.go` (copy from `xion/x/zk/barretenberg/verifier.go`)
- Create: `link_linux_amd64.go` (copy from `xion/x/zk/barretenberg/link_linux_amd64.go`)
- Create: `link_linux_arm64.go` (copy from `xion/x/zk/barretenberg/link_linux_arm64.go`)
- Create: `link_darwin_amd64.go` (copy from `xion/x/zk/barretenberg/link_darwin_amd64.go`)
- Create: `link_darwin_arm64.go` (copy from `xion/x/zk/barretenberg/link_darwin_arm64.go`)

- [ ] **Step 1: Copy all Go source files**

Copy each file from `xion/x/zk/barretenberg/<file>` to `./<file>`. All files declare `package barretenberg` and use only stdlib imports.

**Important:** In each `link_*.go` file, remove the `&& !barretenberg_stub` build constraint since the stub is not carried over. For example, in `link_linux_amd64.go` change:
```go
//go:build linux && amd64 && !barretenberg_stub
```
to:
```go
//go:build linux && amd64
```

Do this for all 4 `link_*.go` files.

- [ ] **Step 2: Update doc.go build instructions**

The `doc.go` file references `wrapper/build.sh` and xion-specific paths. Update the Building section to reference the new repo's `Makefile` and `scripts/build-wrapper.sh`:

Replace the `# Building` section in `doc.go` with:

```go
// # Building
//
// This package includes pre-built static libraries for all supported platforms.
// To rebuild locally for your current platform:
//
//	make build
//
// See scripts/build-wrapper.sh for the full build process.
```

- [ ] **Step 3: Commit**

```bash
git add *.go
git commit -m "feat: add Go bindings source (CGo bridge, verifier, vkey, proof, errors)"
```

### Task 5: Add test files and testdata

**Files:**
- Create: `bindings_test.go` (copy from `xion/x/zk/barretenberg/bindings_test.go`)
- Create: `verifier_test.go` (copy from `xion/x/zk/barretenberg/verifier_test.go`)
- Create: `vkey_test.go` (copy from `xion/x/zk/barretenberg/vkey_test.go`)
- Create: `testdata/statics/vk` (copy from `xion/x/zk/barretenberg/testdata/statics/vk`)
- Create: `testdata/statics/proof` (copy from `xion/x/zk/barretenberg/testdata/statics/proof`)
- Create: `testdata/statics/public_inputs` (copy from `xion/x/zk/barretenberg/testdata/statics/public_inputs`)
- Create: `testdata/README.md`

- [ ] **Step 1: Copy test files**

Copy each `*_test.go` verbatim. No modifications needed — they use relative `testdata/statics/` paths which work in the new repo.

Note: `vkey_test.go` has a linter warning (`unparam` on `loadVkeyTestVector` — the `filename` parameter always receives `vkeyTestFile`). Fix this by inlining the constant:

In `vkey_test.go`, change:
```go
func loadVkeyTestVector(t *testing.T, filename string) []byte {
```
to:
```go
func loadVkeyTestVector(t *testing.T) []byte {
```

And update all call sites from `loadVkeyTestVector(t, vkeyTestFile)` to `loadVkeyTestVector(t)`. Use `vkeyTestFile` constant directly inside the function body.

- [ ] **Step 2: Copy testdata**

```bash
mkdir -p testdata/statics
```

Copy binary test vector files from `xion/x/zk/barretenberg/testdata/statics/`:
- `vk`
- `proof`
- `public_inputs`

- [ ] **Step 3: Create testdata/README.md**

```markdown
# Test Vectors

Binary test vectors for UltraHonk proof verification, generated with Aztec Barretenberg v4.0.4.

## Files

- `statics/vk` — Verification key (binary, UltraHonk format)
- `statics/proof` — Proof (binary)
- `statics/public_inputs` — Concatenated 32-byte field elements (big-endian)

## Regenerating

Requires [Noir](https://noir-lang.org/) (nargo) and [Barretenberg](https://github.com/AztecProtocol/aztec-packages) CLI (bb) v4.0.4.

1. Install nargo and bb CLI at version 4.0.4
2. Create a simple Noir circuit (e.g., `x * x == y`)
3. Generate proof and verification key:

```bash
nargo compile
bb write_vk -b target/circuit.json -o testdata/statics/vk
nargo prove
bb prove -b target/circuit.json -w target/witness.gz -o testdata/statics/proof
# Extract public inputs from the witness
```

See the original generation script for details:
https://github.com/burnt-labs/xion (branch: feature/barrentenberg-go-bindings, file: x/zk/barretenberg/testdata/generate.sh)
```

- [ ] **Step 4: Commit**

```bash
git add *_test.go testdata/
git commit -m "feat: add tests and test vectors"
```

---

## Chunk 2: Build Infrastructure

### Task 6: Add build script

Adapt `xion/x/zk/barretenberg/wrapper/build-wrapper.sh` for the new repo layout. Key differences:
- Reads checksums from `checksums.json` instead of hardcoded shell variables
- Paths are relative to the new repo root (not xion)
- Script lives at `scripts/build-wrapper.sh`

**Files:**
- Create: `scripts/build-wrapper.sh`

- [ ] **Step 1: Create scripts directory**

```bash
mkdir -p scripts
```

- [ ] **Step 2: Write build-wrapper.sh**

Start from the existing script at `xion/x/zk/barretenberg/wrapper/build-wrapper.sh` (301 lines). Copy it to `scripts/build-wrapper.sh` and make these specific changes:

1. **Path resolution** (around line 120-124): Replace:
   ```bash
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   BARRETENBERG_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
   REPO_ROOT="$(cd "$BARRETENBERG_DIR/../../../.." && pwd)"
   LIB_DIR="$BARRETENBERG_DIR/lib/$PLATFORM"
   INCLUDE_DIR="$BARRETENBERG_DIR/include"
   ```
   With:
   ```bash
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
   LIB_DIR="$REPO_ROOT/lib/$PLATFORM"
   INCLUDE_DIR="$REPO_ROOT/include"
   ```

2. **Checksums** (lines 82, 94, 102, 110): Replace hardcoded `EXPECTED_SHA256` per-platform with a read from `checksums.json`:
   ```bash
   EXPECTED_SHA256="$(python3 -c "import json; print(json.load(open('$REPO_ROOT/checksums.json'))['assets']['$PLATFORM']['sha256'])")"
   ```
   Also read `BB_AZTEC_TAG` and `MSGPACK_COMMIT` from `checksums.json` instead of hardcoding them.

3. **Wrapper source path** (line 253): Change `$SCRIPT_DIR/barretenberg_wrapper.cpp` to `$REPO_ROOT/wrapper/barretenberg_wrapper.cpp`.

All other logic (download, checksum verify, sparse-checkout headers, msgpack-c download, Tracy stubs, platform-specific compiler flags, `ar rcs` merge) stays identical.

- [ ] **Step 3: Make executable**

```bash
chmod +x scripts/build-wrapper.sh
```

- [ ] **Step 4: Test locally (current platform only)**

```bash
./scripts/build-wrapper.sh --platform darwin_arm64
```

Expected: Downloads tarball, compiles wrapper, produces `lib/darwin_arm64/libbarretenberg.a`.

- [ ] **Step 5: Commit**

```bash
git add scripts/
git commit -m "feat: add build script for wrapper compilation"
```

### Task 7: Add Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Write Makefile**

```makefile
.PHONY: build test clean

PLATFORM := $(shell go env GOOS)_$(shell go env GOARCH)

# Build libbarretenberg.a for the current platform
build:
	./scripts/build-wrapper.sh --platform $(PLATFORM)

# Build for a specific platform (e.g., make build-linux_amd64)
build-%:
	./scripts/build-wrapper.sh --platform $*

# Build for all platforms (requires cross-compilation toolchains)
build-all: build-linux_amd64 build-linux_arm64 build-darwin_amd64 build-darwin_arm64

# Run tests (requires lib/<platform>/libbarretenberg.a to exist)
test:
	go test -v -count=1 ./...

# Run benchmarks
bench:
	go test -bench=. -benchmem -run=^$$ ./...

# Clean build artifacts (but NOT committed lib/*.a files)
clean:
	rm -rf /tmp/bb-build-*
```

- [ ] **Step 2: Commit**

```bash
git add Makefile
git commit -m "feat: add Makefile for local development"
```

### Task 8: Build and commit platform archive (local platform)

This step builds and commits the `.a` for the current development platform so tests can run locally.

**Files:**
- Create: `lib/darwin_arm64/libbarretenberg.a` (or whichever platform you're on)

- [ ] **Step 1: Build for current platform**

```bash
make build
```

- [ ] **Step 2: Verify the archive**

```bash
ls -lh lib/$(go env GOOS)_$(go env GOARCH)/libbarretenberg.a
ar t lib/$(go env GOOS)_$(go env GOARCH)/libbarretenberg.a | tail -5
```

Expected: File exists, ~50-100 MB, contains `barretenberg_wrapper.o` among other objects.

- [ ] **Step 3: Run tests**

```bash
make test
```

Expected: All tests pass. Tests that require real verification (`TestVerifyValidProof`, `TestVerifyInvalidProof`) should pass with the real library.

- [ ] **Step 4: Commit**

```bash
git add lib/
git commit -m "feat: add pre-built libbarretenberg.a for $(go env GOOS)/$(go env GOARCH)"
```

Note: The remaining 3 platform archives will be built and committed by CI (Task 9). For now, only the local platform archive is committed.

---

## Chunk 3: CI and Release

### Task 9: Add GitHub Actions release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Create workflow directory**

```bash
mkdir -p .github/workflows
```

- [ ] **Step 2: Write release.yml**

The workflow is triggered by manual dispatch or version tag push. It builds all 4 platforms in a matrix, then commits the archives and creates a release.

```yaml
name: Build and Release

on:
  workflow_dispatch:
    inputs:
      create_release:
        description: 'Create a GitHub release after building'
        required: false
        default: 'false'
        type: choice
        options: ['true', 'false']
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  build:
    strategy:
      matrix:
        include:
          - platform: linux_amd64
            runner: ubuntu-latest
            aztec_arch: amd64
            aztec_os: linux
          - platform: linux_arm64
            runner: ubuntu-24.04-arm
            aztec_arch: arm64
            aztec_os: linux
          - platform: darwin_amd64
            runner: macos-13
            aztec_arch: amd64
            aztec_os: darwin
          - platform: darwin_arm64
            runner: macos-latest
            aztec_arch: arm64
            aztec_os: darwin

    runs-on: ${{ matrix.runner }}

    steps:
      - uses: actions/checkout@v4

      - name: Build wrapper
        run: ./scripts/build-wrapper.sh --platform ${{ matrix.platform }}

      - name: Run tests
        run: go test -v -count=1 ./...

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: lib-${{ matrix.platform }}
          path: lib/${{ matrix.platform }}/libbarretenberg.a

  commit-archives:
    needs: build
    runs-on: ubuntu-latest

    steps:
      - uses: actions/checkout@v4
        with:
          ref: main

      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts/

      - name: Place archives
        run: |
          for platform in linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do
            mkdir -p lib/$platform
            cp artifacts/lib-$platform/libbarretenberg.a lib/$platform/
          done

      - name: Commit archives
        run: |
          git config user.name "github-actions[bot]"
          git config user.email "github-actions[bot]@users.noreply.github.com"
          git add lib/
          git diff --staged --quiet || git commit -m "build: update pre-built archives for all platforms"
          git push origin main

      - name: Create release
        if: startsWith(github.ref, 'refs/tags/v')
        run: |
          gh release create ${{ github.ref_name }} \
            --target main \
            --title "${{ github.ref_name }}" \
            --generate-notes
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 3: Commit**

```bash
git add .github/
git commit -m "feat: add CI release workflow for multi-platform builds"
```

### Task 10: Build remaining platform archives via CI

- [ ] **Step 1: Push to GitHub**

```bash
git push -u origin main
```

- [ ] **Step 2: Trigger the workflow**

```bash
gh workflow run release.yml
```

- [ ] **Step 3: Wait for CI to complete and verify**

```bash
gh run list --workflow=release.yml --limit=1
```

Expected: All 4 platform builds succeed, tests pass on each platform.

- [ ] **Step 4: Pull CI commits**

```bash
git pull
```

The CI will have committed the remaining platform archives into `lib/`.

- [ ] **Step 5: Verify all archives exist**

```bash
ls -lh lib/*/libbarretenberg.a
```

Expected: 4 files, one per platform.

### Task 11: Tag initial release

- [ ] **Step 1: Tag v0.1.0**

```bash
git tag v0.1.0
git push origin v0.1.0
```

- [ ] **Step 2: Verify release was created**

```bash
gh release view v0.1.0
```

- [ ] **Step 3: Verify Go module proxy**

```bash
GOPROXY=https://proxy.golang.org go list -m github.com/burnt-labs/barretenberg-go@v0.1.0
```

Expected: Module version is available on the proxy.

---

## Chunk 4: Xion Migration

### Task 12: Update xion to use the external module

This is done in the xion repo, on a new branch off main.

**Files:**
- Modify: `go.mod` (add dependency)
- Modify: `go.sum` (updated by go get)
- Modify: `x/zk/types/vkey.go` (update import)
- Modify: `x/zk/keeper/query_server.go` (update import)
- Modify: `x/zk/keeper/query_server_test.go` (update import)
- Delete: `x/zk/barretenberg/` (entire directory)
- Delete: `make/barretenberg.mk`
- Delete: `.github/workflows/build-barretenberg.yml`

- [ ] **Step 1: Create branch**

```bash
cd /Users/greg/Projects/burnt/xion
git checkout main
git pull
git checkout -b feature/use-barretenberg-go-module
```

- [ ] **Step 2: Add the dependency**

```bash
go get github.com/burnt-labs/barretenberg-go@v0.1.0
```

- [ ] **Step 3: Update imports in x/zk/types/vkey.go**

Change:
```go
"github.com/burnt-labs/xion/x/zk/barretenberg"
```
To:
```go
barretenberg "github.com/burnt-labs/barretenberg-go"
```

- [ ] **Step 4: Update imports in x/zk/keeper/query_server.go**

Same import change as above.

- [ ] **Step 5: Update imports in x/zk/keeper/query_server_test.go**

Same import change as above.

- [ ] **Step 6: Verify imports compile**

```bash
go build ./x/zk/...
```

Expected: Compiles successfully with the external module.

- [ ] **Step 7: Delete the in-tree barretenberg package**

```bash
rm -rf x/zk/barretenberg/
```

- [ ] **Step 8: Delete build infrastructure**

```bash
rm -f make/barretenberg.mk
rm -f .github/workflows/build-barretenberg.yml
```

- [ ] **Step 8a: Clean up root Makefile**

Remove these 3 lines from `Makefile`:
- Line 12: `include make/barretenberg.mk`
- Line 66: `@$(MAKE) --no-print-directory help-barretenberg-brief`
- Line 80: `@$(MAKE) --no-print-directory help-barretenberg`

- [ ] **Step 8b: Clean up Dockerfile**

Remove the libc++ installation line and its comment from `Dockerfile` (around line 27):
```dockerfile
# Install libc++ so the barretenberg wrapper can be compiled and linked against
```
And the associated `apt-get install` for libc++ dev headers.

- [ ] **Step 8c: Fix keeper test testdata paths**

The file `x/zk/keeper/query_server_test.go` has a `loadBarretenbergTestdata()` function that loads test vectors from `x/zk/barretenberg/testdata/statics/`. After deleting `x/zk/barretenberg/`, these paths won't exist.

Copy the test vectors into the keeper's own testdata:

```bash
mkdir -p x/zk/keeper/testdata/barretenberg
cp x/zk/barretenberg/testdata/statics/* x/zk/keeper/testdata/barretenberg/
```

Then update `loadBarretenbergTestdata()` in `query_server_test.go` to look in `testdata/barretenberg/` instead of `../barretenberg/testdata/statics/`.

- [ ] **Step 9: Run go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 10: Run tests**

```bash
go test ./x/zk/...
```

Expected: All tests pass.

- [ ] **Step 11: Run full build**

```bash
make build
```

Expected: Full xion binary builds successfully.

- [ ] **Step 12: Commit**

```bash
git add -A
git commit -m "refactor: use external barretenberg-go module

Replace in-tree barretenberg Go bindings with the standalone
github.com/burnt-labs/barretenberg-go module. This removes:
- x/zk/barretenberg/ (all Go, C++, shell scripts, test data)
- make/barretenberg.mk
- .github/workflows/build-barretenberg.yml

The Go API is identical; only the import path changed."
```

- [ ] **Step 13: Create PR**

```bash
gh pr create \
  --title "Use external barretenberg-go module" \
  --body "$(cat <<'EOF'
## Summary

- Replaces in-tree `x/zk/barretenberg/` with external `github.com/burnt-labs/barretenberg-go` module
- Removes C++ build scripts, shell scripts, and platform-specific archives from this repo
- Import path change only — Go API surface is identical

## Test plan

- [ ] `go test ./x/zk/...` passes
- [ ] `make build` succeeds
- [ ] E2E tests pass
EOF
)"
```
