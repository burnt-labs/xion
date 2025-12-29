package testlib

import (
	"context"
	"testing"
	"time"

	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
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
	err = testutil.WaitForBlocks(ctx, int(height+4), chain)
	return err
}
