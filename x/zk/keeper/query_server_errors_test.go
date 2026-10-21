package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vocdoni/circom2gnark/parser"

	"github.com/burnt-labs/xion/x/zk/types"
)

func TestQueryStoreFailures(t *testing.T) {
	f, faults := setupFaultyFixture(t)
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyData, "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)
	gnarkVKey, gnarkProof, gnarkInputs := loadGnarkTestdata(t)
	_, err = f.k.AddVKey(f.ctx, f.govModAddr, "gnark", gnarkVKey, "", types.ProofSystem_PROOF_SYSTEM_GROTH16_GNARK)
	require.NoError(t, err)

	tests := []struct {
		name string
		arm  func(f *storeFaults)
		call func() error
	}{
		{
			"proof verify params read fails",
			func(f *storeFaults) { f.failGet(types.ParamsKey.Bytes()) },
			func() error {
				_, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{Proof: proofData, PublicInputs: publicInputs, VkeyName: "email_auth"})
				return err
			},
		},
		{
			"ultrahonk params read fails",
			func(f *storeFaults) { f.failGet(types.ParamsKey.Bytes()) },
			func() error {
				_, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{Proof: []byte{0x01}, VkeyName: "email_auth"})
				return err
			},
		},
		{
			"gnark params read fails",
			func(f *storeFaults) { f.failGet(types.ParamsKey.Bytes()) },
			func() error {
				_, err := f.queryServer.ProofVerifyGnark(f.ctx, &types.QueryVerifyGnarkRequest{Proof: gnarkProof, PublicInputs: gnarkInputs, VkeyName: "gnark"})
				return err
			},
		},
		{
			"vkey by name index read fails",
			func(f *storeFaults) { f.failGet(types.VkeyNameIndexPrefix.Bytes()) },
			func() error {
				_, err := f.queryServer.VKeyByName(f.ctx, &types.QueryVKeyByNameRequest{Name: "email_auth"})
				return err
			},
		},
		{
			"vkeys iteration fails",
			func(f *storeFaults) { f.failIterators() },
			func() error {
				_, err := f.queryServer.VKeys(f.ctx, &types.QueryVKeysRequest{})
				return err
			},
		},
		{
			"has vkey index read fails",
			func(f *storeFaults) { f.failGet(types.VkeyNameIndexPrefix.Bytes()) },
			func() error {
				_, err := f.queryServer.HasVKey(f.ctx, &types.QueryHasVKeyRequest{Name: "email_auth"})
				return err
			},
		},
		{
			"next vkey id read fails",
			func(f *storeFaults) { f.failGet(types.VkeySeqPrefix.Bytes()) },
			func() error {
				_, err := f.queryServer.NextVKeyID(f.ctx, &types.QueryNextVKeyIDRequest{})
				return err
			},
		},
		{
			"params read fails",
			func(f *storeFaults) { f.failGet(types.ParamsKey.Bytes()) },
			func() error {
				_, err := f.queryServer.Params(f.ctx, &types.QueryParamsRequest{})
				return err
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.arm(faults)
			defer faults.clear()

			require.ErrorContains(t, tc.call(), errStoreUnavailable.Error())
		})
	}
}

func TestQueryVKeyByNameDanglingIndex(t *testing.T) {
	f := SetupTest(t)
	require.NoError(t, f.k.VKeyNameIndex.Set(f.ctx, "ghost", 99))

	_, err := f.queryServer.VKeyByName(f.ctx, &types.QueryVKeyByNameRequest{Name: "ghost"})
	require.ErrorIs(t, err, types.ErrVKeyNotFound)
}

func TestQueryProofVerifyHexPublicInputs(t *testing.T) {
	f := SetupTest(t)
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyData, "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	t.Run("hex inputs verify like their decimal form", func(t *testing.T) {
		resp, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
			Proof:        proofData,
			PublicInputs: hexPublicInputs(t, publicInputs, "0x"),
			VkeyName:     "email_auth",
		})
		require.NoError(t, err)
		require.True(t, resp.Verified)
	})

	t.Run("malformed hex input is rejected", func(t *testing.T) {
		inputs := append([]string(nil), publicInputs...)
		inputs[0] = "0xnotHex"
		_, err := f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
			Proof:        proofData,
			PublicInputs: inputs,
			VkeyName:     "email_auth",
		})
		require.ErrorIs(t, err, types.ErrInvalidRequest)
		require.ErrorContains(t, err, "public input[0]")
	})
}

func TestQueryProofVerifyMalformedProofPoint(t *testing.T) {
	f := SetupTest(t)
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "email_auth", vkeyData, "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	// pi_b rows with a single component parse as JSON but cannot be converted
	// to a G2 point; the verifier must report an error rather than crash.
	proof, err := parser.UnmarshalCircomProofJSON(proofData)
	require.NoError(t, err)
	proof.PiB = [][]string{{"1"}, {"1"}, {"1"}}
	malformed, err := parser.MarshalCircomProofJSON(proof)
	require.NoError(t, err)

	_, err = f.queryServer.ProofVerify(f.ctx, &types.QueryVerifyRequest{
		Proof:        malformed,
		PublicInputs: publicInputs,
		VkeyName:     "email_auth",
	})
	require.ErrorIs(t, err, types.ErrInvalidRequest)
	require.ErrorContains(t, err, "internal error during proof verification")
}

// storeRawVKey writes a vkey and its name index directly, bypassing AddVKey's
// validation and proof-system defaulting.
func storeRawVKey(t *testing.T, f *TestFixture, vkey types.VKey) uint64 {
	t.Helper()
	id, err := f.k.NextVKeyID.Next(f.ctx)
	require.NoError(t, err)
	require.NoError(t, f.k.VKeys.Set(f.ctx, id, vkey))
	require.NoError(t, f.k.VKeyNameIndex.Set(f.ctx, vkey.Name, id))
	return id
}

func TestQueryProofVerifyUltraHonkStoredKeyShapes(t *testing.T) {
	vkBytes, proofBytes, publicInputsBytes := loadBarretenbergTestdata(t)
	f := SetupTest(t)

	t.Run("legacy key without proof system is reported as groth16", func(t *testing.T) {
		storeRawVKey(t, f, types.VKey{Name: "legacy", KeyBytes: createTestVKeyBytes("legacy")})

		_, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "legacy",
		})
		require.ErrorIs(t, err, types.ErrInvalidRequest)
		require.ErrorContains(t, err, "proof_system=PROOF_SYSTEM_GROTH16")
	})

	t.Run("stored key bytes that barretenberg cannot parse", func(t *testing.T) {
		requireRealBarretenberg(t)
		storeRawVKey(t, f, types.VKey{
			Name:        "broken",
			KeyBytes:    []byte("definitely not an ultrahonk verification key"),
			ProofSystem: types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK,
		})

		_, err := f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        proofBytes,
			PublicInputs: publicInputsBytes,
			VkeyName:     "broken",
		})
		require.ErrorIs(t, err, types.ErrInvalidVKey)
		require.ErrorContains(t, err, "ultrahonk vkey")
	})

	t.Run("proof too short to parse", func(t *testing.T) {
		_, err := f.k.AddVKey(f.ctx, f.govModAddr, "honk", vkBytes, "", types.ProofSystem_PROOF_SYSTEM_ULTRA_HONK_ZK)
		require.NoError(t, err)

		_, err = f.queryServer.ProofVerifyUltraHonk(f.ctx, &types.QueryVerifyUltraHonkRequest{
			Proof:        make([]byte, 100),
			PublicInputs: publicInputsBytes,
			VkeyName:     "honk",
		})
		require.ErrorIs(t, err, types.ErrInvalidRequest)
		require.ErrorContains(t, err, "proof:")
	})
}

func TestQueryProofVerifyGnarkLegacyKey(t *testing.T) {
	_, gnarkProof, gnarkInputs := loadGnarkTestdata(t)
	f := SetupTest(t)
	storeRawVKey(t, f, types.VKey{Name: "legacy", KeyBytes: createTestVKeyBytes("legacy")})

	_, err := f.queryServer.ProofVerifyGnark(f.ctx, &types.QueryVerifyGnarkRequest{
		Proof:        gnarkProof,
		PublicInputs: gnarkInputs,
		VkeyName:     "legacy",
	})
	require.ErrorIs(t, err, types.ErrInvalidRequest)
	require.ErrorContains(t, err, "proof_system=PROOF_SYSTEM_GROTH16")
}
