package e2e_zk

import (
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v11/chain/cosmos"
)

// TestZKEmailAuthenticator tests the full ZKEmail authenticator flow.
// This test builds its own chain and uses the shared assertion functions.
func TestZKEmailAuthenticator(t *testing.T) {
	ctx := t.Context()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	// Set bech32 prefix before building chain
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	xion := buildZKEmailChain(t, 12)

	t.Parallel()

	// Create a proposal tracker starting at 1 (fresh chain)
	proposalTracker := testlib.NewProposalTracker(1)

	testlib.RunZKEmailAuthenticatorAssertions(t, testlib.ZKEmailAssertionConfig{
		Chain:           xion,
		Ctx:             ctx,
		User:            nil, // Will use DeployerMnemonic for pre-generated proofs
		ProposalTracker: proposalTracker,
	})
}

func buildZKEmailChain(t *testing.T, accountNumber uint64) *cosmos.CosmosChain {
	t.Helper()
	spec := testlib.XionLocalChainSpec(t, 3, 1)
	spec.ModifyGenesis = testlib.WithZKFixtureAccount(spec.ModifyGenesis, accountNumber)
	return testlib.BuildXionChainWithSpec(t, spec)
}
