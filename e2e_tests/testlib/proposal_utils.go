package testlib

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	ibctestutil "github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

// ProposalTracker helps manage proposal IDs across multiple proposal submissions.
// This is necessary when running multiple tests that submit proposals on the same chain,
// such as after an upgrade where the upgrade proposal already used ID 1.
type ProposalTracker struct {
	nextID uint64
}

// NewProposalTracker creates a new tracker starting from the specified ID.
func NewProposalTracker(startingID uint64) *ProposalTracker {
	return &ProposalTracker{nextID: startingID}
}

// NextID returns the next proposal ID and increments the counter.
func (pt *ProposalTracker) NextID() uint64 {
	id := pt.nextID
	pt.nextID++
	return id
}

// CurrentID returns the current next proposal ID without incrementing.
func (pt *ProposalTracker) CurrentID() uint64 {
	return pt.nextID
}

// SubmitAndPassProposal submits a governance proposal and waits for it to pass.
// It handles the full lifecycle: build, submit, vote (all validators), and wait for execution.
func SubmitAndPassProposal(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	proposer ibc.Wallet,
	proposalMsgs []cosmos.ProtoMessage,
	title, summary, metadata string,
	proposalID uint64,
) error {
	proposal, err := chain.BuildProposal(
		proposalMsgs,
		title,
		summary,
		metadata,
		"500000000"+chain.Config().Denom, // greater than min deposit
		proposer.FormattedAddress(),
		false,
	)
	require.NoError(t, err)

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit proposal")

	_, err = chain.SubmitProposal(ctx, proposer.KeyName(), proposal)
	require.NoError(t, err)

	prop, err := chain.GovQueryProposal(ctx, proposalID)
	require.NoError(t, err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	err = chain.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, proposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == govv1beta1.StatusPassed {
				return true
			}
			t.Logf("Waiting for proposal %d to enter voting status PASSED, current status: %s", proposalID, proposalInfo.Status)
		}
		return false
	}, time.Second*15, time.Second, "failed to reach status PASSED after 15s")

	afterVoteHeight, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height after voting on proposal")

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, afterVoteHeight, prop.ProposalId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	// Wait for proposal execution
	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height after proposal passed")
	err = ibctestutil.WaitForBlocks(ctx, int(height+4), chain)
	return err
}

// SubmitAndPassProposalWithEncoding submits a governance proposal using a custom encoding config.
// This is necessary after chain upgrades when the chain object's encoding config is stale.
// It manually marshals messages using the provided encoding config before submission.
func SubmitAndPassProposalWithEncoding(
	t *testing.T,
	ctx context.Context,
	chain *cosmos.CosmosChain,
	proposer ibc.Wallet,
	proposalMsgs []cosmos.ProtoMessage,
	title, summary, metadata string,
	proposalID uint64,
	encodingConfig *moduletestutil.TestEncodingConfig,
) error {
	// Get codec from the encoding config interface registry
	cdc := codec.NewProtoCodec(encodingConfig.InterfaceRegistry)

	// Marshal each message using the fresh encoding config
	messages := make([]json.RawMessage, len(proposalMsgs))
	for i, msg := range proposalMsgs {
		msgBytes, err := cdc.MarshalInterfaceJSON(msg)
		require.NoError(t, err, "failed to marshal proposal message %d", i)
		messages[i] = msgBytes
	}

	// Build proposal using TxProposalv1 format with manually encoded messages
	prop := cosmos.TxProposalv1{
		Messages: messages,
		Metadata: metadata,
		Deposit:  "500000000" + chain.Config().Denom,
		Title:    title,
		Summary:  summary,
	}

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit proposal")

	txResp, err := chain.SubmitProposal(ctx, proposer.KeyName(), prop)
	require.NoError(t, err, "failed to submit proposal")

	// Parse proposal ID from response
	parsedProposalID, err := strconv.ParseUint(txResp.ProposalID, 10, 64)
	require.NoError(t, err, "failed to parse proposal ID")
	require.Equal(t, proposalID, parsedProposalID, "proposal ID mismatch")

	// Wait for proposal to enter voting period
	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, proposalID)
		if err != nil {
			return false
		}
		if proposalInfo.Status == govv1beta1.StatusVotingPeriod {
			return true
		}
		t.Logf("Waiting for proposal %d to enter voting status VOTING, current status: %s", proposalID, proposalInfo.Status)
		return false
	}, time.Second*15, time.Second, "failed to reach status VOTING after 15s")

	// Vote yes from all validators
	err = chain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	// Wait for voting period to complete
	// Voting period is 10s in genesis, which is approximately 20 blocks at ~500ms/block
	// We need to wait for the period to end before the proposal can transition to PASSED
	currentHeight, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height after voting")
	err = ibctestutil.WaitForBlocks(ctx, int(currentHeight)+20, chain)
	require.NoError(t, err, "error waiting for voting period to complete")

	// Wait for proposal to pass
	// After the voting period ends, the proposal should transition to PASSED quickly
	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, proposalID)
		if err != nil {
			return false
		}
		if proposalInfo.Status == govv1beta1.StatusPassed {
			return true
		}
		t.Logf("Waiting for proposal %d to enter voting status PASSED, current status: %s", proposalID, proposalInfo.Status)
		return false
	}, time.Second*10, time.Second, "failed to reach status PASSED after 10s")

	afterVoteHeight, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height after voting on proposal")

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, afterVoteHeight, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	// Wait for proposal execution
	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height after proposal passed")
	err = ibctestutil.WaitForBlocks(ctx, int(height+4), chain)
	return err
}
