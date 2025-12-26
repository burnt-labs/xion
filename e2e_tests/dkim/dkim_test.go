package integration_tests

import (
	"bytes"
	"encoding/base64"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/icza/dyno"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	dkimTypes "github.com/burnt-labs/xion/x/dkim/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govModule "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
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

	// Fetch current max pubkey size from params.
	paramsResp, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "dkim", "params")
	require.NoError(t, err)
	paramsVal, err := dyno.Get(paramsResp, "params")
	require.NoError(t, err)
	maxSizeStr, err := dyno.GetString(paramsVal, "max_pubkey_size_bytes")
	require.NoError(t, err)
	maxSize, err := strconv.ParseUint(maxSizeStr, 10, 64)
	require.NoError(t, err)

	// Craft an oversized base64 key (decoded length = maxSize + 1).
	rawOversized := bytes.Repeat([]byte{0x42}, int(maxSize+1))
	oversizedPubKey := base64.StdEncoding.EncodeToString(rawOversized)

	govModAddress := testlib.GetModuleAddress(t, xion, ctx, govModule.ModuleName)
	oversizedMsg := &dkimTypes.MsgAddDkimPubKeys{
		Authority: govModAddress,
		DkimPubkeys: []dkimTypes.DkimPubKey{{
			Domain:       "oversize.example.com",
			Selector:     "too-big",
			PubKey:       oversizedPubKey,
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
	proposalID := proposalTracker.NextID()
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
