---
title: CLI bootstrap temporary home must outlive app construction
type: gotcha
scope: cmd/xiond/root.go
confidence: 0.98
sources: [713]
updated: 2026-11-10
---
`NewRootCmd` constructs a full `NewWasmApp` to obtain its encoding config, even for lightweight CLI invocations; `NewMemDB` does not make this bootstrap filesystem-free.
`app/app.go` derives `homePath` from `flags.FlagHome` and passes `<home>/wasm/wasm` to `wasmvm.NewVM` unconditionally, independently of `loadLatest`.
The checked-out `tempDir` helper defers `os.RemoveAll` inside the helper, so cleanup runs before the constructor consumes the returned path; files subsequently created under that home have no matching cleanup.
Keep temporary-home ownership in the calling bootstrap scope: create it before constructing the app and remove it after bootstrap use, rather than deferring removal in a path-returning helper.
PR #713 proposes this lifetime correction after reporting accumulated `wasmd*` directories and inode exhaustion. Review bootstrap changes for cleanup of the constructor's filesystem side effects, including on non-node commands.
