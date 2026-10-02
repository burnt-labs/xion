# cosmovisor in the release image

The `release` target of the [Dockerfile](../Dockerfile) ships `/usr/bin/cosmovisor`,
built from pinned upstream source in the `cosmovisor-builder` stage. The
`heighliner` target does not contain cosmovisor.

## Why source, not the release tarball

The upstream `cosmovisor/v1.7.3` release binary is built against Cosmos SDK
0.50, grpc 1.70 and x/crypto 0.32, and carries fixable HIGH/CRITICAL CVEs that
fail the release image's Docker Scout gate. Upstream `main` has moved to
patched dependencies but has not cut a newer cosmovisor release, so the image
builds that source instead (DO-519, option A).

## Inputs

All inputs are Dockerfile `ARG` defaults and are checked before use:

| ARG | Value | Checked by |
| --- | --- | --- |
| `COSMOVISOR_COMMIT` | `642a9c00b69ae3c9cb866a251eb32116de416c55` (cosmos/cosmos-sdk) | source URL |
| `COSMOVISOR_SOURCE_SHA256` | SHA256 of `https://codeload.github.com/cosmos/cosmos-sdk/tar.gz/<commit>` | `sha256sum -c` before extraction |
| `COSMOVISOR_PATCH_SHA256` | SHA256 of `cosmovisor-patches/0001-decode-db-backend-output.patch` | `sha256sum -c` before `git apply` |
| `COSMOVISOR_BUILD_IMAGE` | `golang:1.26.8-bookworm`, pinned by digest | image digest |

The module `tools/cosmovisor` is built with `GOWORK=off GOTOOLCHAIN=local
-mod=readonly`, after `go mod verify`, using upstream's own `go.mod`/`go.sum`
unchanged. Xion's own modules and toolchain are not involved.

The release image records the inputs as labels:
`io.burnt.cosmovisor.revision`, `io.burnt.cosmovisor.source-sha256` and
`io.burnt.cosmovisor.patch-sha256`.

## Version reporting

A source build reports `cosmovisor version: (devel)`; the upstream v1.7.3
tarball reports the same. The exact revision is in the image labels above, and
`go version -m /usr/bin/cosmovisor` shows the module build information.

## The patch

`0001-decode-db-backend-output.patch` changes only `tools/cosmovisor/scanner.go`
and adds `dbbackend.go` with its test. When the daemon exits before cosmovisor
has polled the upgrade file, cosmovisor opens the CometBFT blockstore itself to
compare the current height with the upgrade height. It takes the backend name
from `xiond config get config db_backend`, which prints it TOML-quoted
(`"goleveldb"` plus a newline). Unpatched, that raw output is passed to
`dbm.NewDB`, the blockstore cannot be opened, and the upgrade is missed. The
patch decodes the value strictly and leaves backend validation to `NewDB`.

Twice approved carrying this narrow downstream patch rather than waiting for an
upstream fix (DO-519, 2026-10-02).

## Behaviour changes from v1.7.3

- With `DAEMON_SHUTDOWN_GRACE` set, cosmovisor stops the daemon for an upgrade
  with SIGTERM instead of SIGINT, then kills it when the grace period ends.
- After an exit, cosmovisor reads the blockstore height with the patched
  backend parsing above.

Environment variables, defaults, the `cosmovisor/` layout and the watched
`$DAEMON_HOME/data/upgrade-info.json` are unchanged.

## Tests

`cosmovisor_image_test.go` runs against a locally loaded release image:

```bash
COSMOVISOR_TEST_IMAGE=xion:linux-amd64 COSMOVISOR_TEST_PLATFORM=linux/amd64 \
    go test -count=1 -timeout 20m -v ./scripts -run '^TestCosmovisor'
```

- `TestCosmovisorOperatorImage`: user, labels, build information, required
  environment, `init` layout, config defaults and precedence, `run`,
  `add-upgrade`, binary switch with restart on and off, shutdown grace and the
  download refusal. Uses `testdata/cosmovisor-daemon.sh` as the daemon.
- `TestCosmovisorRealStateFallback`: produces blocks with real xiond, stops it,
  and checks that cosmovisor switches the binary only when the stored height
  has reached the upgrade height, then that the upgraded node produces blocks.
- `TestCosmovisorPatchChecksum` (no image needed) keeps the pinned patch hash in
  step with the patch file and limits the patch to the three reviewed files.

Without `COSMOVISOR_TEST_IMAGE` the two image tests skip.

## Updating

1. Pick the new upstream commit (or, once released, a cosmovisor tag's commit)
   and read the `tools/cosmovisor` diff since the current pin.
2. Download `https://codeload.github.com/cosmos/cosmos-sdk/tar.gz/<commit>`,
   compute its SHA256, and update `COSMOVISOR_COMMIT` and
   `COSMOVISOR_SOURCE_SHA256`.
3. If upstream has fixed the `db_backend` parsing, delete the patch, its
   `COPY`/`git apply` lines and `COSMOVISOR_PATCH_SHA256`, the patch-sha256
   label and `TestCosmovisorPatchChecksum`. Otherwise regenerate the patch
   against the new source, run upstream's tests on it, and update
   `COSMOVISOR_PATCH_SHA256`.
4. If `tools/cosmovisor/go.mod` needs a newer Go, update
   `COSMOVISOR_BUILD_IMAGE` to a matching `golang` image pinned by digest.
5. Build the release image for linux/amd64 and linux/arm64, run the tests
   above on both, and scan both with
   `docker scout cves --only-fixed --only-severity critical,high --exit-code`.
