# Remove the optional IBC Wasm light client

1. Remove the `08-wasm` keeper, route, module, store registration, lifecycle ordering, and params wiring while preserving CosmWasm `x/wasm`.
2. Delete the legacy `08-wasm` store during the v31 coordinated upgrade.
3. Remove the root and e2e module dependencies and Burnt fork replacements.
4. Add focused upgrade/store-removal coverage and run formatting, dependency, build, and test checks.
