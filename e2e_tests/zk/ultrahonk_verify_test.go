package e2e_zk

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/icza/dyno"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/stretchr/testify/require"
)

// readBarretenbergTestData loads vk, proof, and public_inputs from e2e_tests/testdata/keys/barretenberg/.
// File format expectations:
//   - vk: UltraHonk verification key bytes (binary) as expected by x/zk/barretenberg.ParseVerificationKey
//   - proof: raw UltraHonk proof bytes
//   - public_inputs: concatenation of 32-byte field elements for public inputs, matching the proof/vkey
//
// A missing or unreadable file causes the test to fail via require.NoError.
func readBarretenbergTestData(t *testing.T) (vkBytes, proofBytes, publicInputsBytes []byte) {
	t.Helper()
	base := testlib.IntegrationTestPath("testdata", "keys", "barretenberg")
	for name, dest := range map[string]*[]byte{
		"vk":             &vkBytes,
		"proof":          &proofBytes,
		"public_inputs":  &publicInputsBytes,
	} {
		path := filepath.Join(base, name)
		data, err := os.ReadFile(path)
		require.NoError(t, err, "read barretenberg testdata %s from %s", name, path)
		*dest = data
	}
	return vkBytes, proofBytes, publicInputsBytes
}

// addUltraHonkVKeyTx sends a MsgAddVKey transaction for an UltraHonk (Barretenberg) verification key
// using the CLI and returns gas used.
func addUltraHonkVKeyTx(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	keyName, vkeyName, description string,
	vkBytes []byte,
) (uint64, error) {
	t.Helper()
	node := chain.GetNode()
	filename := vkeyName + ".bin"
	err := node.WriteFile(ctx, vkBytes, filename)
	require.NoError(t, err)

	vkeyPath := filepath.Join(node.HomeDir(), filename)
	txHash, err := testlib.ExecTx(t, ctx, node, keyName, "zk", "add-vkey", vkeyName, vkeyPath, description, "ultrahonk", "--chain-id", chain.Config().ChainID)
	if err != nil {
		return 0, err
	}

	txResp, err := testlib.ExecQuery(t, ctx, node, "tx", txHash)
	if err != nil {
		return 0, err
	}

	gasUsedStr, err := dyno.GetString(txResp, "gas_used")
	require.NoError(t, err)
	return strconv.ParseUint(gasUsedStr, 10, 64)
}

// TestZKUltraHonkVKeyAndProofVerification is an e2e test that uploads an UltraHonk (Barretenberg)
// verification key via the CLI and then verifies a proof using the stored key through the
// verify-ultrahonk query (by vkey name and by vkey ID). It assumes sample vkey and proofs
// exist in e2e_tests/testdata/keys/barretenberg/ (vk, proof, public_inputs).
func TestZKUltraHonkVKeyAndProofVerification(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	vkBytes, proofBytes, publicInputsBytes := readBarretenbergTestData(t)

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
	chainSpec := testlib.XionLocalChainSpec(t, 3, 1)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	users := interchaintest.GetAndFundTestUsers(t, ctx, "zk-ultrahonk-test", math.NewInt(10_000_000_000), xion)
	chainUser := users[0]
	node := xion.GetNode()

	// Upload UltraHonk vkey via CLI
	_, err := addUltraHonkVKeyTx(t, ctx, xion, chainUser.KeyName(), "ultrahonk_circuit", "UltraHonk test vkey", vkBytes)
	require.NoError(t, err)

	// Assert vkey exists and get ID for verify-by-ID
	hasResp, err := testlib.ExecQuery(t, ctx, node, "zk", "has-vkey", "ultrahonk_circuit")
	require.NoError(t, err)
	existsVal, err := dyno.Get(hasResp, "exists")
	require.NoError(t, err)
	exists, ok := existsVal.(bool)
	require.True(t, ok, "exists should be bool")
	require.True(t, exists, "vkey ultrahonk_circuit should exist after add-vkey")
	vkeyID, err := dyno.Get(hasResp, "id")
	require.NoError(t, err)
	idFloat, ok := vkeyID.(float64)
	require.True(t, ok, "id should be number")
	vkeyIDUint := uint64(idFloat)

	// Write proof and public inputs into node container for verify-ultrahonk query
	err = node.WriteFile(ctx, proofBytes, "proof.bin")
	require.NoError(t, err)
	err = node.WriteFile(ctx, publicInputsBytes, "public_inputs.bin")
	require.NoError(t, err)
	proofPath := filepath.Join(node.HomeDir(), "proof.bin")
	inputsPath := filepath.Join(node.HomeDir(), "public_inputs.bin")

	// Verify proof by vkey name
	respByName, err := testlib.ExecQuery(t, ctx, node, "zk", "verify-ultrahonk", proofPath, "--vkey-name", "ultrahonk_circuit", "--public-inputs-file", inputsPath)
	require.NoError(t, err)
	verifiedByNameVal, err := dyno.Get(respByName, "verified")
	require.NoError(t, err)
	verifiedByName, ok := verifiedByNameVal.(bool)
	require.True(t, ok, "verified should be bool")
	require.True(t, verifiedByName, "proof should verify when using vkey name")

	// Verify proof by vkey ID
	respByID, err := testlib.ExecQuery(t, ctx, node, "zk", "verify-ultrahonk", proofPath, "--vkey-id", strconv.FormatUint(vkeyIDUint, 10), "--public-inputs-file", inputsPath)
	require.NoError(t, err)
	verifiedByIDVal, err := dyno.Get(respByID, "verified")
	require.NoError(t, err)
	verifiedByID, okID := verifiedByIDVal.(bool)
	require.True(t, okID, "verified should be bool")
	require.True(t, verifiedByID, "proof should verify when using vkey ID")

	// Optional negative case: wrong public inputs yield verified == false (not an error)
	t.Run("wrong_public_inputs_returns_false", func(t *testing.T) {
		wrongInputs := make([]byte, len(publicInputsBytes))
		copy(wrongInputs, publicInputsBytes)
		if len(wrongInputs) > 0 {
			wrongInputs[0] ^= 0xff // flip bits so verification fails
		}
		err := node.WriteFile(ctx, wrongInputs, "wrong_inputs.bin")
		require.NoError(t, err)
		wrongInputsPath := filepath.Join(node.HomeDir(), "wrong_inputs.bin")
		resp, err := testlib.ExecQuery(t, ctx, node, "zk", "verify-ultrahonk", proofPath, "--vkey-name", "ultrahonk_circuit", "--public-inputs-file", wrongInputsPath)
		require.NoError(t, err)
		verifiedVal, err := dyno.Get(resp, "verified")
		require.NoError(t, err)
		verified, ok := verifiedVal.(bool)
		require.True(t, ok)
		require.False(t, verified, "proof with wrong public inputs should not verify")
	})
}
