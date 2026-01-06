package integration_tests

import (
	"context"
	"strconv"
	"testing"

	"cosmossdk.io/math"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/burnt-labs/xion/x/dkim/types"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/icza/dyno"
	"github.com/stretchr/testify/require"
)

// TestDKIMModule tests basic DKIM query operations.
// This test builds its own chain and uses the shared assertion functions.
func TestDKIMModule(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Set bech32 prefix before building chain
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	xion := testlib.BuildXionChain(t)

	// Use proposal tracker starting at 1 for standalone tests
	proposalTracker := testlib.NewProposalTracker(1)

	testlib.RunDKIMModuleAssertions(t, testlib.DKIMAssertionConfig{
		Chain:           xion,
		Ctx:             ctx,
		User:            nil, // Will create and fund a new user
		ProposalTracker: proposalTracker,
		TestData:        testlib.DefaultDKIMTestData(),
	})
}

// TestDKIMGovernance tests adding and removing DKIM records via governance proposals.
// This test builds its own chain and uses the shared assertion functions.
func TestDKIMGovernance(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Set bech32 prefix before building chain
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	xion := testlib.BuildXionChain(t)

	// Use proposal tracker starting at 1 for standalone tests
	proposalTracker := testlib.NewProposalTracker(1)

	testlib.RunDKIMGovernanceAssertions(t, testlib.DKIMAssertionConfig{
		Chain:           xion,
		Ctx:             ctx,
		User:            nil, // Will create and fund a new user
		ProposalTracker: proposalTracker,
		TestData:        testlib.DefaultDKIMTestData(),
	})
}

// TestDKIMPubKeyMaxSize ensures oversized DKIM pubkeys are rejected.
func TestDKIMPubKeyMaxSize(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	xion := testlib.BuildXionChain(t)
	proposalTracker := testlib.NewProposalTracker(1)

	// Fund a user to submit the proposal.
	users := interchaintest.GetAndFundTestUsers(t, ctx, "dkim-max-size", math.NewInt(10_000_000_000), xion)
	chainUser := users[0]

	// Craft an oversized base64 key (decoded length = maxSize + 1).
	basePubKey := []byte("MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB")
	basePubKeySize := uint64(len(basePubKey))
	// set maxSize param to half the basePubKeySize to ensure our key is oversized
	updatedParams := dkimTypes.Params{
		VkeyIdentifier:     uint64(1),
		MaxPubkeySizeBytes: basePubKeySize - 1,
	}
	updateParamsMsg := &dkimTypes.MsgUpdateParams{
		Authority: testlib.GetModuleAddress(t, xion, ctx, govModule.ModuleName),
		Params:    updatedParams,
	}
	proposalID := proposalTracker.NextID()
	err := testlib.SubmitAndPassProposal(t, ctx, xion, chainUser, []cosmos.ProtoMessage{updateParamsMsg}, "Update DKIM Params", "Set max pubkey size smaller to trigger oversize", "", proposalID)
	require.NoError(t, err)
	// make sure params were updated
	paramsRes := queryDkimParams(t, ctx, xion)
	require.Equal(t, updatedParams.MaxPubkeySizeBytes, paramsRes.MaxPubkeySizeBytes)
	govModAddress := testlib.GetModuleAddress(t, xion, ctx, govModule.ModuleName)
	oversizedMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority: govModAddress,
		DkimPubkeys: []dkimTypes.DkimPubKey{{
			Domain:       "oversize.example.com",
			Selector:     "too-big",
			PubKey:       string(basePubKey),
			PoseidonHash: []byte("hash"),
		}},
	}
	require.NoError(t, oversizedMsg.ValidateBasic())

	proposal, err := xion.BuildProposal(
		[]cosmos.ProtoMessage{oversizedMsg},
		"Oversized DKIM key",
		"Oversized DKIM key should be rejected",
		"",
		"500000000"+xion.Config().Denom,
		chainUser.FormattedAddress(),
		false,
	)
	require.NoError(t, err)

	// Submit proposal (should go into voting), then vote yes and confirm it fails execution.
	proposalID = proposalTracker.NextID()
	submitHash, err := xion.SubmitProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err)
	t.Logf("Submitted oversized DKIM proposal tx: %v", submitHash)

	prop, err := xion.GovQueryProposal(ctx, proposalID)
	require.NoError(t, err)
	require.Equal(t, proposalID, prop.ProposalId)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	err = xion.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	startHeight, err := xion.Height(ctx)
	require.NoError(t, err)

	failedProp, err := cosmos.PollForProposalStatus(ctx, xion, startHeight, startHeight+15, proposalID, govv1beta1.StatusFailed)
	require.NoError(t, err, "proposal should fail due to oversized pubkey")
	require.Equal(t, proposalID, failedProp.ProposalId, "polled proposal mismatch")
}

func queryDkimParams(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain) types.Params {
	resp, err := testlib.ExecQuery(t, ctx, chain.GetNode(), "dkim", "params")
	require.NoError(t, err)

	paramsVal, err := dyno.Get(resp, "params")
	require.NoError(t, err)

	maxKeySize, err := dyno.GetString(paramsVal, "max_pubkey_size_bytes")
	require.NoError(t, err)
	maxSize, err := strconv.ParseUint(maxKeySize, 10, 64)
	require.NoError(t, err)

	return types.Params{
		MaxPubkeySizeBytes: maxSize,
	}
}