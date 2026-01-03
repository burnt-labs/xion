package integration_tests

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/icza/dyno"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/burnt-labs/xion/x/zk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"
)

func TestZKParamsAndVKeyUploads(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
	chainSpec := testlib.XionLocalChainSpec(t, 3, 1)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Use proposal tracker starting at 1 for standalone tests
	proposalTracker := testlib.NewProposalTracker(1)

	// Create and fund a user
	users := interchaintest.GetAndFundTestUsers(t, ctx, "zk-params-test", math.NewInt(10_000_000_000), xion)
	chainUser := users[0]

	govModAddress := testlib.GetModuleAddress(t, xion, ctx, govtypes.ModuleName)

	// Verify defaults at chain start
	initialParams := queryZKParams(t, ctx, xion)
	require.Equal(t, types.DefaultMaxVKeySizeBytes, initialParams.MaxVkeySizeBytes)
	require.Equal(t, types.DefaultUploadChunkSize, initialParams.UploadChunkSize)
	require.Equal(t, types.DefaultUploadChunkGas, initialParams.UploadChunkGas)

	// Prepare verification keys with different sizes based on the same valid JSON.
	baseVKey := readTestVKey(t)
	baseSize := uint64(len(baseVKey))
	// update params with enough room for base key
	// Update params to a tighter limit to exercise size checks and gas scaling.
	updatedParams := types.Params{
		MaxVkeySizeBytes: baseSize + 5000,
		UploadChunkSize:  1000,
		UploadChunkGas:   50000,
	}
	updateParamsMsg := &types.MsgUpdateParams{
		Authority: govModAddress,
		Params:    updatedParams,
	}
	proposalID := proposalTracker.NextID()
	_, err := submitAndPassProposalWithGas(t, ctx, xion, chainUser, []cosmos.ProtoMessage{updateParamsMsg}, proposalID)
	require.NoError(t, err)

	currentParams := queryZKParams(t, ctx, xion)
	require.Equal(t, updatedParams.MaxVkeySizeBytes, currentParams.MaxVkeySizeBytes)
	require.Equal(t, updatedParams.UploadChunkSize, currentParams.UploadChunkSize)
	require.Equal(t, updatedParams.UploadChunkGas, currentParams.UploadChunkGas)

	paddedSmall := append(baseVKey, bytes.Repeat([]byte(" "), 2000)...)
	require.Less(t, uint64(len(paddedSmall)), currentParams.MaxVkeySizeBytes)

	paddedTooLarge := append(baseVKey, bytes.Repeat([]byte(" "), 6000)...)
	require.Greater(t, uint64(len(paddedTooLarge)), currentParams.MaxVkeySizeBytes)

	// Upload a small key as a normal transaction and record gas used.
	gasSmall, err := addVKeyTx(t, ctx, xion, chainUser.KeyName(), "zk-small", "small vkey", baseVKey)
	require.NoError(t, err)

	// Upload a larger-but-allowed key as a normal transaction and record gas used.
	gasLarge, err := addVKeyTx(t, ctx, xion, chainUser.KeyName(), "zk-large-ok", "larger vkey within limit", paddedSmall)
	require.NoError(t, err)

	// Gas delta should at least match the additional chunk gas introduced by the larger payload.
	chunksSmall := (baseSize + updatedParams.UploadChunkSize - 1) / updatedParams.UploadChunkSize
	chunksLarge := (uint64(len(paddedSmall)) + updatedParams.UploadChunkSize - 1) / updatedParams.UploadChunkSize
	expectedExtra := (chunksLarge - chunksSmall) * updatedParams.UploadChunkGas
	require.Greater(t, gasLarge, gasSmall)
	require.GreaterOrEqual(t, gasLarge-gasSmall, expectedExtra)

	// Attempt to upload an oversized key and expect failure on submission.
	_, err = addVKeyTx(t, ctx, xion, chainUser.KeyName(), "zk-too-large", "should fail size check", paddedTooLarge)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verification key exceeds maximum size")
}

func queryZKParams(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) types.Params {
	resp, err := testlib.ExecQuery(t, ctx, chain.GetNode(), "zk", "params")
	require.NoError(t, err)

	paramsVal, err := dyno.Get(resp, "params")
	require.NoError(t, err)

	maxSizeStr, err := dyno.GetString(paramsVal, "max_vkey_size_bytes")
	require.NoError(t, err)
	maxSize, err := strconv.ParseUint(maxSizeStr, 10, 64)
	require.NoError(t, err)

	chunkSizeStr, err := dyno.GetString(paramsVal, "upload_chunk_size")
	require.NoError(t, err)
	chunkSize, err := strconv.ParseUint(chunkSizeStr, 10, 64)
	require.NoError(t, err)

	chunkGasStr, err := dyno.GetString(paramsVal, "upload_chunk_gas")
	require.NoError(t, err)
	chunkGas, err := strconv.ParseUint(chunkGasStr, 10, 64)
	require.NoError(t, err)

	return types.Params{
		MaxVkeySizeBytes: maxSize,
		UploadChunkSize:  chunkSize,
		UploadChunkGas:   chunkGas,
	}
}

func readTestVKey(t *testing.T) []byte {
	path := testlib.IntegrationTestPath("testdata", "keys", "vkey.json")
	bz, err := os.ReadFile(path)
	require.NoError(t, err)
	return bz
}

// addVKeyTx sends a MsgAddVKey transaction using the CLI and returns gas used.
func addVKeyTx(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, keyName, vkeyName, description string, vkeyBytes []byte) (uint64, error) {
	node := chain.GetNode()
	filename := vkeyName + ".json"
	err := node.WriteFile(ctx, vkeyBytes, filename)
	require.NoError(t, err)

	vkeyPath := filepath.Join(node.HomeDir(), filename)
	txHash, err := testlib.ExecTx(t, ctx, node, keyName, "zk", "add-vkey", vkeyName, vkeyPath, description, "--chain-id", chain.Config().ChainID)
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

// submitAndPassProposalWithGas submits a proposal, captures the gas used for submission (simulation executes messages),
// votes yes with all validators, and waits for the proposal to pass.
func submitAndPassProposalWithGas(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	proposer ibc.Wallet,
	proposalMsgs []cosmos.ProtoMessage,
	proposalID uint64,
) (uint64, error) {
	prop, err := chain.BuildProposal(
		proposalMsgs,
		"test proposal",
		"test proposal",
		"",
		"500000000"+chain.Config().Denom,
		proposer.FormattedAddress(),
		false,
	)
	require.NoError(t, err)

	_, err = chain.SubmitProposal(ctx, proposer.KeyName(), prop)
	if err != nil {
		return 0, err
	}

	propInfo, err := chain.GovQueryProposal(ctx, proposalID)
	if err != nil {
		return 0, err
	}
	require.Equal(t, propInfo.Status, govv1beta.StatusVotingPeriod)

	err = chain.VoteOnProposalAllValidators(ctx, propInfo.ProposalId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, proposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == govv1beta.StatusPassed {
				return true
			}
			t.Logf("Waiting for proposal %d to reach PASSED, current status: %s", proposalID, proposalInfo.Status)
		}
		return false
	}, time.Second*18, time.Second*6, "proposal did not pass")

	return 0, nil
}
