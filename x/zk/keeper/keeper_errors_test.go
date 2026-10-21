package keeper_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vocdoni/circom2gnark/parser"

	"github.com/burnt-labs/xion/x/zk/types"
)

func TestInitGenesisStoreFailures(t *testing.T) {
	genesis := func() *types.GenesisState {
		return &types.GenesisState{
			Params: types.DefaultParams(),
			Vkeys: []types.VKeyWithID{{
				Id:   7,
				Vkey: types.VKey{Name: "key7", Description: "seventh", KeyBytes: createTestVKeyBytes("key7")},
			}},
		}
	}

	tests := []struct {
		name string
		arm  func(f *storeFaults)
	}{
		{"params write fails", func(f *storeFaults) { f.failSet(types.ParamsKey.Bytes()) }},
		{"vkey write fails", func(f *storeFaults) { f.failSet(types.VKeyPrefix.Bytes()) }},
		{"name index write fails", func(f *storeFaults) { f.failSet(types.VkeyNameIndexPrefix.Bytes()) }},
		{"sequence read fails", func(f *storeFaults) { f.failGet(types.VkeySeqPrefix.Bytes()) }},
		{"sequence write fails", func(f *storeFaults) { f.failSet(types.VkeySeqPrefix.Bytes()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, faults := setupFaultyFixture(t)
			tc.arm(faults)
			require.Panics(t, func() { f.k.InitGenesis(f.ctx, genesis()) })
		})
	}
}

func TestExportGenesisStoreFailures(t *testing.T) {
	t.Run("vkey iteration fails", func(t *testing.T) {
		f, faults := setupFaultyFixture(t)
		faults.failIterators()
		require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
	})

	t.Run("params read fails", func(t *testing.T) {
		f, faults := setupFaultyFixture(t)
		faults.failGet(types.ParamsKey.Bytes())
		require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
	})

	t.Run("vkey missing from name index", func(t *testing.T) {
		f := SetupTest(t)
		orphan := types.VKey{Name: "orphan", KeyBytes: createTestVKeyBytes("orphan")}
		require.NoError(t, f.k.VKeys.Set(f.ctx, 9, orphan))
		require.Panics(t, func() { f.k.ExportGenesis(f.ctx) })
	})
}

func TestGetParamsStoreFailure(t *testing.T) {
	f, faults := setupFaultyFixture(t)
	faults.failGet(types.ParamsKey.Bytes())

	_, err := f.k.GetParams(f.ctx)
	require.ErrorContains(t, err, errStoreUnavailable.Error())
}

func TestAddVKeyStoreFailures(t *testing.T) {
	tests := []struct {
		name    string
		arm     func(f *storeFaults)
		wantErr string
	}{
		{"name index lookup fails", func(f *storeFaults) { f.failHas(types.VkeyNameIndexPrefix.Bytes()) }, errStoreUnavailable.Error()},
		{"params read fails", func(f *storeFaults) { f.failGet(types.ParamsKey.Bytes()) }, errStoreUnavailable.Error()},
		{"sequence increment fails", func(f *storeFaults) { f.failGet(types.VkeySeqPrefix.Bytes()) }, types.ErrIncreaseSequenceID.Error()},
		{"vkey write fails", func(f *storeFaults) { f.failSet(types.VKeyPrefix.Bytes()) }, errStoreUnavailable.Error()},
		{"name index write fails", func(f *storeFaults) { f.failSet(types.VkeyNameIndexPrefix.Bytes()) }, errStoreUnavailable.Error()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f, faults := setupFaultyFixture(t)
			tc.arm(faults)

			_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key", createTestVKeyBytes("key"), "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestGetVKeyStoreFailures(t *testing.T) {
	f, faults := setupFaultyFixture(t)
	id, err := f.k.AddVKey(f.ctx, f.govModAddr, "key", createTestVKeyBytes("key"), "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	t.Run("by id", func(t *testing.T) {
		faults.failGet(types.VKeyPrefix.Bytes())
		defer faults.clear()

		_, err := f.k.GetVKeyByID(f.ctx, id)
		require.ErrorContains(t, err, errStoreUnavailable.Error())
		require.NotErrorIs(t, err, types.ErrVKeyNotFound)
	})

	t.Run("by name", func(t *testing.T) {
		faults.failGet(types.VkeyNameIndexPrefix.Bytes())
		defer faults.clear()

		_, err := f.k.GetVKeyByName(f.ctx, "key")
		require.ErrorContains(t, err, errStoreUnavailable.Error())
		require.NotErrorIs(t, err, types.ErrVKeyNotFound)
	})
}

func TestUpdateVKeyStoreFailures(t *testing.T) {
	f, faults := setupFaultyFixture(t)
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key", createTestVKeyBytes("key"), "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)
	update := func() error {
		return f.k.UpdateVKey(f.ctx, f.govModAddr, "key", createTestVKeyBytes("key"), "updated", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	}

	tests := []struct {
		name string
		arm  func(f *storeFaults)
	}{
		{"name index read fails", func(f *storeFaults) { f.failGet(types.VkeyNameIndexPrefix.Bytes()) }},
		{"vkey read fails", func(f *storeFaults) { f.failGet(types.VKeyPrefix.Bytes()) }},
		{"params read fails", func(f *storeFaults) { f.failGet(types.ParamsKey.Bytes()) }},
		{"vkey write fails", func(f *storeFaults) { f.failSet(types.VKeyPrefix.Bytes()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.arm(faults)
			defer faults.clear()

			err := update()
			require.ErrorContains(t, err, errStoreUnavailable.Error())
			require.NotErrorIs(t, err, types.ErrVKeyNotFound)
		})
	}

	t.Run("name index points at a missing vkey", func(t *testing.T) {
		require.NoError(t, f.k.VKeyNameIndex.Set(f.ctx, "ghost", 42))

		err := f.k.UpdateVKey(f.ctx, f.govModAddr, "ghost", createTestVKeyBytes("ghost"), "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
		require.ErrorIs(t, err, types.ErrVKeyNotFound)
	})
}

func TestRemoveVKeyStoreFailures(t *testing.T) {
	f, faults := setupFaultyFixture(t)
	_, err := f.k.AddVKey(f.ctx, f.govModAddr, "key", createTestVKeyBytes("key"), "", types.ProofSystem_PROOF_SYSTEM_GROTH16)
	require.NoError(t, err)

	tests := []struct {
		name string
		arm  func(f *storeFaults)
	}{
		{"name index read fails", func(f *storeFaults) { f.failGet(types.VkeyNameIndexPrefix.Bytes()) }},
		{"vkey read fails", func(f *storeFaults) { f.failGet(types.VKeyPrefix.Bytes()) }},
		{"vkey delete fails", func(f *storeFaults) { f.failDel(types.VKeyPrefix.Bytes()) }},
		{"name index delete fails", func(f *storeFaults) { f.failDel(types.VkeyNameIndexPrefix.Bytes()) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.arm(faults)
			defer faults.clear()

			err := f.k.RemoveVKey(f.ctx, f.govModAddr, "key")
			require.ErrorContains(t, err, errStoreUnavailable.Error())
			require.NotErrorIs(t, err, types.ErrVKeyNotFound)
		})
	}

	// The vkey must still be present after the failed attempts.
	has, err := f.k.HasVKey(f.ctx, "key")
	require.NoError(t, err)
	require.True(t, has)

	t.Run("name index points at a missing vkey", func(t *testing.T) {
		require.NoError(t, f.k.VKeyNameIndex.Set(f.ctx, "ghost", 42))

		err := f.k.RemoveVKey(f.ctx, f.govModAddr, "ghost")
		require.ErrorIs(t, err, types.ErrVKeyNotFound)
	})
}

// hexPublicInputs re-encodes decimal public inputs as 0x-prefixed hex strings.
func hexPublicInputs(t *testing.T, inputs []string, prefix string) []string {
	t.Helper()
	out := make([]string, len(inputs))
	for i, in := range inputs {
		n, ok := new(big.Int).SetString(in, 10)
		require.True(t, ok, "input %d is not decimal", i)
		out[i] = prefix + n.Text(16)
	}
	return out
}

func TestVerifyPublicInputEncodings(t *testing.T) {
	f := SetupTest(t)

	vk, err := parser.UnmarshalCircomVerificationKeyJSON(vkeyData)
	require.NoError(t, err)
	proof, err := parser.UnmarshalCircomProofJSON(proofData)
	require.NoError(t, err)

	t.Run("lowercase hex inputs verify", func(t *testing.T) {
		inputs := hexPublicInputs(t, publicInputs, "0x")
		verified, err := f.k.Verify(f.ctx, proof, vk, &inputs)
		require.NoError(t, err)
		require.True(t, verified)
	})

	t.Run("uppercase hex prefix passes the field check but not conversion", func(t *testing.T) {
		inputs := hexPublicInputs(t, publicInputs, "0X")
		verified, err := f.k.Verify(f.ctx, proof, vk, &inputs)
		require.False(t, verified)
		require.NotErrorIs(t, err, types.ErrInvalidRequest)
		require.ErrorContains(t, err, "failed to parse public input 0")
	})

	t.Run("non numeric input is rejected", func(t *testing.T) {
		inputs := []string{"12", "not-a-number"}
		_, err := f.k.Verify(f.ctx, proof, vk, &inputs)
		require.ErrorIs(t, err, types.ErrInvalidRequest)
		require.ErrorContains(t, err, "public input[1]")
	})

	t.Run("negative input is rejected", func(t *testing.T) {
		inputs := []string{"-1"}
		_, err := f.k.Verify(f.ctx, proof, vk, &inputs)
		require.ErrorIs(t, err, types.ErrInvalidRequest)
	})
}

func TestVerifyRecoversFromMalformedProof(t *testing.T) {
	f := SetupTest(t)

	vk, err := parser.UnmarshalCircomVerificationKeyJSON(vkeyData)
	require.NoError(t, err)
	proof, err := parser.UnmarshalCircomProofJSON(proofData)
	require.NoError(t, err)

	// A G2 point whose rows are missing their second component parses as JSON
	// but makes the circom-to-gnark conversion index past the row.
	proof.PiB = [][]string{{"1"}, {"1"}, {"1"}}

	inputs := append([]string(nil), publicInputs...)
	verified, err := f.k.Verify(f.ctx, proof, vk, &inputs)
	require.False(t, verified)
	require.ErrorIs(t, err, types.ErrInvalidRequest)
	require.ErrorContains(t, err, "internal error during proof verification")
}

func TestVerifyGnarkMalformedPublicInputs(t *testing.T) {
	f := SetupTest(t)
	vkeyBytes, proofBytes, _ := loadGnarkTestdata(t)

	verified, err := f.k.VerifyGnark(f.ctx, proofBytes, vkeyBytes, []byte{0x01, 0x02, 0x03})
	require.False(t, verified)
	require.ErrorIs(t, err, types.ErrInvalidRequest)
	require.ErrorContains(t, err, "failed to unmarshal public inputs")
}
