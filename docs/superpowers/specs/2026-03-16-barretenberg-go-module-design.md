# barretenberg-go: Standalone Go Module for Barretenberg UltraHonk Verification

## Problem

The `feature/barrentenberg-go-bindings` branch integrates Barretenberg's UltraHonk proof verification into the xion codebase, but violates several best practices:

- Shell scripts embedded in module code (`wrapper/build-wrapper.sh`, `wrapper/build.sh`)
- C/C++ source mixed with Go code (`wrapper/barretenberg_wrapper.cpp`)
- C++ build process coupled to the Go application build
- Platform-specific static libraries committed into the application repo

## Solution

Extract the Barretenberg Go bindings into a standalone Go module (`github.com/burnt-labs/barretenberg-go`) that:

1. Contains all Go source, the C++ wrapper shim, and the C header
2. Commits pre-built platform-specific `.a` archives (built by CI)
3. Is imported by xion as a normal Go dependency via `go.mod`
4. Builds releases by downloading Aztec's pre-built `libbb-external.a` — no upstream source fork required

## Architecture

### Repository Structure

```
barretenberg-go/
├── go.mod                          # module github.com/burnt-labs/barretenberg-go
├── go.sum                          # no external deps — stdlib only
│
├── bindings.go                     # CGo bridge (vkeyHandle, verifyProof, etc.)
├── verifier.go                     # High-level Verifier type
├── vkey.go                         # VerificationKey parsing and validation
├── proof.go                        # Proof and PublicInputs types
├── errors.go                       # Sentinel errors and error mapping
├── doc.go                          # Package documentation
│
├── link_linux_amd64.go             # #cgo LDFLAGS for linux/amd64 (libstdc++)
├── link_linux_arm64.go             # #cgo LDFLAGS for linux/arm64 (libc++)
├── link_darwin_amd64.go            # #cgo LDFLAGS for darwin/amd64 (libc++)
├── link_darwin_arm64.go            # #cgo LDFLAGS for darwin/arm64 (libc++)
│
├── include/
│   └── barretenberg_wrapper.h      # C API header for CGo
│
├── wrapper/
│   └── barretenberg_wrapper.cpp    # C++ wrapper shim (extern "C" functions)
│
├── lib/
│   ├── linux_amd64/
│   │   └── libbarretenberg.a       # Pre-built: libbb-external.a + wrapper.o
│   ├── linux_arm64/
│   │   └── libbarretenberg.a
│   ├── darwin_amd64/
│   │   └── libbarretenberg.a
│   └── darwin_arm64/
│       └── libbarretenberg.a
│
├── testdata/
│   └── statics/
│       ├── vk                      # Test verification key (binary, no extension)
│       ├── proof                   # Test proof
│       └── public_inputs           # Test public inputs
├── testdata/
│   └── README.md                   # How to regenerate test vectors (requires nargo + bb CLI)
│
├── *_test.go                       # Unit and integration tests
│
├── Makefile                        # Local dev: build wrapper for current platform
│
├── .github/
│   └── workflows/
│       └── release.yml             # CI: build all platforms, commit .a files, tag
│
├── scripts/
│   └── build-wrapper.sh            # Build script (used by CI and local dev)
│
└── checksums.json                  # SHA256 checksums for Aztec release assets, per platform
```

### Files NOT included (vs. current branch)

- `wrapper/build-wrapper.sh` — replaced by `scripts/build-wrapper.sh` + CI workflow
- `wrapper/build.sh` — legacy, removed
- `wrapper/test_verify.cpp` — standalone C++ test; replaced by Go tests
- `link_stub.go` — development stub for building without real archives. Removed because archives are always committed. If needed for linting on unsupported platforms, a stub can be reintroduced later.
- `make/barretenberg.mk` — build targets live in the new repo's Makefile
- `testdata/generate.sh` — replaced by `testdata/README.md` containing the exact `nargo` + `bb` CLI commands to regenerate vectors

## Go API Surface

The public API is identical to the current `x/zk/barretenberg` package — every exported type, function, method, constant, and error is carried over unchanged. The only difference is the import path.

The Go files declare `package barretenberg` despite the module path ending in `barretenberg-go`. Consumers import with an alias:
```go
import barretenberg "github.com/burnt-labs/barretenberg-go"
```

### Dependencies

The module has **no external Go dependencies** — only stdlib packages (`encoding/hex`, `errors`, `fmt`, `math/big`, `runtime`, `sync`, `unsafe`). `go.mod` will contain only the module declaration.

### Types and Key Constructors (illustrative, not exhaustive)

| Type | Description |
|------|-------------|
| `Verifier` | Thread-safe proof verifier wrapping a verification key |
| `VerificationKey` | Parsed UltraHonk verification key (wraps native C handle) |
| `Proof` | Parsed UltraHonk proof |
| `PublicInputs` | Set of field elements for verification |

| Function | Signature |
|----------|-----------|
| `NewVerifier` | `(vkey *VerificationKey) (*Verifier, error)` |
| `NewVerifierFromBytes` | `(vkeyData []byte) (*Verifier, error)` |
| `NewVerifierFromHex` | `(vkeyHex string) (*Verifier, error)` |
| `ParseVerificationKey` | `(data []byte) (*VerificationKey, error)` |
| `ParseVerificationKeyHex` | `(hexStr string) (*VerificationKey, error)` |
| `ParseProof` | `(data []byte) (*Proof, error)` |
| `ParseProofHex` | `(hexStr string) (*Proof, error)` |
| `NewPublicInputs` | `(elements [][]byte) (*PublicInputs, error)` |
| `ParsePublicInputsFromStrings` | `(inputs []string) (*PublicInputs, error)` |
| `ParsePublicInputsFromHex` | `(hexInputs []string) (*PublicInputs, error)` |
| `VerifyProofBytes` | `(vkeyData, proofData []byte, publicInputs []string) (bool, error)` |
| `ValidateVerificationKeyBytes` | `(data []byte, maxSizeBytes uint64) error` |
| `Version` | `() string` |
| `SupportsUltraHonk` | `() bool` |

### Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `FieldElementSize` | `32` | Size of a BN254 field element in bytes |
| `MinVKeySizeBytes` | `256` | Minimum vkey size for pre-validation |
| `MinCircuitSize` | `1` | Minimum valid circuit size |

### Sentinel Errors

`ErrInvalidVKey`, `ErrInvalidProof`, `ErrInvalidPublicInputs`, `ErrVerificationFailed`, `ErrInternal`, `ErrNullPointer`, `ErrAllocationFailed`, `ErrDeserializationFailed`, `ErrClosed`, `ErrInvalidFieldElement`

## Build Process

### CI Workflow (`release.yml`)

Triggered by: manual dispatch or push of a version tag (`v*`).

**Inputs:**
- `aztec_version`: Aztec release tag (default: `v4.0.4`)

**Matrix:** 4 jobs, one per platform. Note: the "Stdlib" column refers to the stdlib used to compile *our wrapper* — it must match the stdlib that *Aztec* used to build `libbb-external.a` (see Platform-Specific Linking section).

| Platform | Runner | Wrapper Compiler | Stdlib for Wrapper |
|----------|--------|------------------|--------------------|
| `linux_amd64` | `ubuntu-latest` | `clang++` | libstdc++ (omit flag; matches Aztec's g++ build) |
| `linux_arm64` | `ubuntu-24.04-arm` | `clang++` | `-stdlib=libc++` (matches Aztec's Zig build) |
| `darwin_amd64` | `macos-13` | `clang++` | libc++ (Apple default) |
| `darwin_arm64` | `macos-latest` | `clang++` | libc++ (Apple default) |

**Each job:**

1. Download `barretenberg-static-{arch}-{os}.tar.gz` from the Aztec release
2. Verify SHA256 checksum against `checksums.json` (see below)
3. Sparse-checkout barretenberg headers from `aztec-packages` at the pinned tag
4. Download msgpack-c headers (pinned commit)
5. Create Tracy stub header
6. Compile `wrapper/barretenberg_wrapper.cpp` with platform-appropriate flags:
   - `-std=c++20 -fPIC -O2 -fvisibility=hidden`
   - Include paths: msgpack, stubs, barretenberg headers, `include/`
   - `-DBB_VERSION="<tag>"`
7. Merge wrapper `.o` into `libbb-external.a` → `lib/{platform}/libbarretenberg.a`
8. Upload `lib/{platform}/libbarretenberg.a` as build artifact

**After matrix completes:**
1. Download all 4 artifacts
2. Commit to repo: `lib/{platform}/libbarretenberg.a` for each platform
3. Create git tag and GitHub release

### Local Development (`Makefile`)

For contributors who need to rebuild locally:

```makefile
# Detect current platform
PLATFORM := $(shell go env GOOS)_$(shell go env GOARCH)

# Build for current platform only
build:
    ./scripts/build-wrapper.sh --platform $(PLATFORM)

# Run tests
test:
    go test -v ./...
```

The `scripts/build-wrapper.sh` is a simplified version of the current `build-wrapper.sh`, restructured for the new repo layout.

## Consumption from xion

### Changes to xion

1. **Add dependency:**
   ```
   go get github.com/burnt-labs/barretenberg-go@v0.1.0
   ```

2. **Delete `x/zk/barretenberg/` entirely** — all Go files, wrapper/, include/, lib/, testdata/

3. **Update imports** in `x/zk/types/` and `x/zk/keeper/`:
   ```go
   // Before
   "github.com/burnt-labs/xion/x/zk/barretenberg"
   // After
   barretenberg "github.com/burnt-labs/barretenberg-go"
   ```

4. **Remove build infrastructure:**
   - `make/barretenberg.mk`
   - `.github/workflows/build-barretenberg.yml`
   - Barretenberg-related targets in the root Makefile

5. **Remove Dockerfile changes** related to barretenberg build dependencies (libc++ dev headers, etc.)

### No code changes needed

The Go API surface is identical. All call sites (`NewVerifier`, `ParseVerificationKey`, `Verify`, `ValidateVerificationKeyBytes`, etc.) work unchanged — only the import path changes.

## Version Pinning Strategy

| Entity | Version | Tracked Where |
|--------|---------|---------------|
| Aztec `libbb-external.a` | `v4.0.4` | `BB_AZTEC_TAG` in build script + `checksums.json` |
| Barretenberg headers | Same tag as above | Sparse-checkout at `BB_AZTEC_TAG` |
| msgpack-c | Pinned commit hash | Build script constant |
| barretenberg-go module | Semver tags (`v0.1.0`, ...) | Git tags, `go.mod` in xion |

**Upgrading the Aztec version:**
1. Update `BB_AZTEC_TAG` and checksums in the repo
2. Verify `barretenberg_wrapper.cpp` compiles against new headers
3. Run tests with new test vectors if the proof format changed
4. Tag a new release
5. In xion: `go get github.com/burnt-labs/barretenberg-go@<new-tag>`

## Platform-Specific Linking

The C++ stdlib linking must match how Aztec built `libbb-external.a`:

| Platform | Aztec Build Toolchain | Stdlib ABI | CGo Link Flag |
|----------|-----------------------|------------|---------------|
| linux/amd64 | g++ | libstdc++ (`std::`) | `-lstdc++ -lm -lpthread` |
| linux/arm64 | Zig + libc++ | libc++ (`std::__1`) | `-lc++ -lm -lpthread` |
| darwin/amd64 | Apple clang | libc++ (`std::__1`) | `-lc++ -lm` |
| darwin/arm64 | Apple clang | libc++ (`std::__1`) | `-lc++ -lm` |

This is encoded in the `link_*.go` files and must stay in sync with the Aztec release being consumed.

## Binary Size and Git

Each `libbarretenberg.a` is approximately 50-100 MB. With 4 platforms, the repo will carry ~200-400 MB of archives.

**Approach:** Commit directly (no Git LFS). The Go module proxy (`proxy.golang.org`) downloads module source as zip archives, which does **not** resolve LFS pointers — so LFS would break `go get`. Direct commits work fine; the Go module proxy caches each version, so most consumers never clone the repo. Only contributors who `git clone` pay the full size cost.

If repo size becomes a problem, consider shallow clones (`--depth=1`) for CI, or hosting archives in GitHub Releases with a download script instead of committing them.

## Testing

Tests move to the new repo with the Go source. They use the same `testdata/statics/` fixtures (`vk`, `proof`, `public_inputs`).

CI runs `go test ./...` on the matrix runners after building, ensuring the archives work on each platform.

## Checksums Format

`checksums.json` in the repo root, keyed by Aztec tag and platform:

```json
{
  "aztec_tag": "v4.0.4",
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

The build script reads this file to determine download URLs and expected checksums.

## Migration Plan

The cutover from xion's in-tree barretenberg to the external module:

1. **Create the `barretenberg-go` repo** with all Go source, wrapper, header, and CI workflow
2. **CI builds and commits** the 4 platform `.a` files; tag `v0.1.0`
3. **Verify independently**: `go test ./...` passes in the new repo on all platforms
4. **In xion** (single PR on a new branch off main):
   - `go get github.com/burnt-labs/barretenberg-go@v0.1.0`
   - Delete `x/zk/barretenberg/` entirely
   - Update imports in `x/zk/types/` and `x/zk/keeper/`
   - Remove `make/barretenberg.mk`, `.github/workflows/build-barretenberg.yml`
   - Run full test suite to confirm no regressions
5. **Rollback path**: If a bug is found in the extracted module, xion can pin `go.mod` to a prior commit or temporarily revert the PR. The old in-tree code remains in git history.

Steps 1-3 happen in the new repo. Step 4 is an atomic PR in xion. The two repos are never in a half-migrated state.

## Out of Scope

- Building barretenberg from source (use Aztec's pre-built releases)
- Forking or tracking the upstream barretenberg C++ source
- Supporting proof systems other than UltraHonk
- Android/iOS/WASM targets (can be added later by extending the matrix)
