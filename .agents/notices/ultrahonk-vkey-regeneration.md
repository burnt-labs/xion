# UltraHonk verification key regeneration — v31 testnet upgrade

Three owners hold 11 `PROOF_SYSTEM_ULTRA_HONK_ZK` keys on `xion-testnet-2`. Ten
of them stop verifying when testnet moves to v31. Mainnet is unaffected — its
only registered key is Groth16.

Send each owner their own section. The shared background applies to all three.

---

## Shared background

v31 moves the node's Barretenberg from **4.0.4** to **5.2.0**. That changes the
UltraHonk proof transcript, so keys and proofs minted under the old toolchain
stop verifying. Tested against a v5.2.0-linked verifier:

| Minted with | Result | Detail |
|---|---|---|
| bb 4.0.4 (what testnet runs today) | **fails** | `barretenberg: internal error`; proof is 16000 B where 5.x is 14656 B |
| bb 5.0.0-rc.1 | **fails** | verification fails at the reduction step |
| bb 5.0.0 final | **passes** | vk is byte-identical to 5.2.0 |
| bb 5.2.0 | passes | |

Two consequences worth reading twice:

1. **Anything minted at bb 5.0.0 or later already works.** You do not need to
   redo it for 5.2.0.
2. **It is a toolchain pair, not just a `bb` bump.** bb 4.0.4 cannot parse ACIR
   from nargo beta.26 at all (`error converting into field
   Circuit::current_witness_index`). Move nargo and bb together.

### Regenerating a key

Use **nargo 1.0.0-beta.26** and the **bb 5.2.0** CLI, which ships as
`barretenberg-<arch>-<os>.tar.gz` on the
[aztec-packages v5.2.0 release](https://github.com/AztecProtocol/aztec-packages/releases/tag/v5.2.0).

```bash
nargo compile
bb write_vk -b target/<package>.json -o out
# out/vk is the file to register
```

`bb write_vk` builds the circuit from its own throwaway witness, so `nargo
compile` is enough — you do not need `nargo execute`, a witness, or a populated
`Prover.toml` to reissue a key. We confirmed the vk is byte-identical either
way, and identical again to what `bb prove --write_vk` emits.

Do **not** pass `--verifier_target`. The node verifies with `bb::UltraZKFlavor`,
which is the CLI default; every other target changes the transcript hash and
produces keys the chain will reject.

### Registering

Registration is permissionless — **no governance proposal is required**, for any
of these keys. `add-vkey` accepts any signer. `update-vkey` and `remove-vkey`
are restricted to the account that originally registered the key, which is your
account for every key listed below.

> Note: the CLI help on `update-vkey` says "Any account can update verification
> keys." That is wrong — the keeper enforces owner-only. Only `add-vkey` is open.

### Two ways to sequence it, pick one

**A — new names, no downtime (recommended).** Register the 5.x keys under new
names *before* the upgrade. Your existing keys keep working until the upgrade,
the new ones start working after it, and consumers switch names when ready.
Nothing breaks at any point, and it is trivially reversible.

```bash
xiond tx zk add-vkey <new-name> ./out/vk "<description>" ultrahonk \
    --from <your-key> \
    --chain-id xion-testnet-2 --node https://rpc.xion-testnet-2.burnt.com:443
```

**B — update in place.** Keeps the existing id and name, so consumers need no
change, but there is a window where verification is broken: the old key dies at
the upgrade and the new one cannot be accepted until the node is on v31.

```bash
xiond tx zk update-vkey <name> ./out/vk "<description>" ultrahonk \
    --from <your-key> \
    --chain-id xion-testnet-2 --node https://rpc.xion-testnet-2.burnt.com:443
```

Use B only where a consumer cannot switch names.

---

## Notice 1 — `xion1uk6g4hjtf477zf8arl6qrq4v29k89xkjuftql3` (zkDCAP)

You hold 5 UltraHonk keys. **Four need regenerating, one is already fine.**

| id | name | action |
|---|---|---|
| 15 | `dcap-ultrahonk-v1` | **regenerate** — the production key, treat as priority |
| 22 | `zkdcap-scratch-bb500-2026-08-13` | **none** — already nargo beta.26 + bb 5.0.0, verifies under 5.2.0 unchanged |
| 23 | `zkdcap-scratch-bb404-2026-08-13` | regenerate or retire — it is the 4.0.4 baseline probe, so it is expected to die |
| 24 | `zkdcap-scratch-phala-resized-2026-08-13` | regenerate — version not recorded, assume 4.0.4 |
| 25 | `zkdcap-scratch-hardened-2026-08-13` | regenerate — version not recorded, assume 4.0.4 |

Your `bb500` probe from 2026-08-13 is what confirms the 5.x line is stable: we
verified a bb 5.0.0 vk is byte-identical to a 5.2.0 one, so that work carries
over as-is. If 24 and 25 were also cut with beta.26 + bb 5.0.0, they are fine
too — worth checking your build logs before redoing them.

`dcap-ultrahonk-v1` is the only one here with consumers, so it is the one that
wants option A.

---

## Notice 2 — `xion13m6uhc6ezv7qka4lnxsp8vf50z0juslld4ftfh` (Juodzekas, UltraHonk)

All 3 of your keys need regenerating. Their descriptions record **bb 4.0.4**, so
they are confirmed affected.

| id | name |
|---|---|
| 16 | `juodzekas_decrypt_uh_v1` |
| 17 | `juodzekas_shuffle_uh_v1` |
| 18 | `juodzekas_peek_uh_v1` |

Descriptions mention "optimized (windowed muls)" — regenerate from the same
circuit sources, only the toolchain changes. These relate to the zk-shuffle work
in xion PR #451, so worth coordinating with that branch before it merges.

---

## Notice 3 — `xion1pulr6verhzcepjmalzv3l3vn8pt6alch5wl6xk` (Juodzekas, Grumpkin)

All 3 of your keys need regenerating. The descriptions do not record a bb
version; assume 4.0.4 unless your build logs say otherwise.

| id | name |
|---|---|
| 19 | `juodzekas_shuffle_gk_v1` |
| 20 | `juodzekas_decrypt_gk_v1` |
| 21 | `juodzekas_peek_gk_v1` |

If these were cut with bb 5.0.0 or later, no action is needed — confirm before
spending the effort.

---

## Checking your own keys

To read what is currently registered:

```bash
xiond query zk vkeys \
    --node https://rpc.xion-testnet-2.burnt.com:443 --output json
```

`--chain-id` only sets the chain ID a transaction is signed for; it does not
route anything. Without `--node`, the CLI talks to `tcp://localhost:26657`.

The authoritative check is your own build records: which nargo and `bb` produced
each key. Failing that, regenerate the circuit with the 5.x toolchain and verify
a fresh proof against the existing registered key — if it verifies, the key is
already 5.x.

As a quick triage, a **proof** artifact's size distinguishes the two toolchains:

| Toolchain | Proof size |
|---|---|
| bb 4.0.4 | 16000 bytes |
| bb 5.2.0 | 14656 bytes |

We checked this holds across a roughly 16,000x range of circuit sizes and for
both 1 and 21 public inputs, on both versions — public inputs are serialized
into a separate file, so they do not change the proof length. Treat it as a
signal for these two versions only, not a general version oracle.

**The verification key does not distinguish them.** A vk is 3680 bytes under
both 4.0.4 and 5.2.0, so checking the file you actually registered tells you
nothing. Only a proof works for this.
