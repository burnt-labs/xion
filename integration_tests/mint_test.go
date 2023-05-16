package integration_tests

import (
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/testutil"
)

func TestMintModuleNoInflationNoFees(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t)

	// Wait for some blocks and check if that supply stays the same
	chainHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(chainHeight)+10, xion)

	// Run test harness
	MintTestHarness(t, xion, ctx)
}
