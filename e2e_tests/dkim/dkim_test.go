package integration_tests

import (
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
