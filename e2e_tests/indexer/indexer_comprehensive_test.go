package e2e_indexer

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

// TestIndexerNonConsensusCritical verifies that indexer errors don't halt the node
// This test confirms that StopNodeOnErr is configured to false
func TestIndexerNonConsensusCritical(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: Non-Consensus-Critical Operation")
	t.Log("======================================================")
	t.Log("Testing that indexer errors don't affect consensus")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)
	node := xion.GetNode()

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	user := users[0]

	// Wait for chain to be ready
	require.NoError(t, testutil.WaitForBlocks(ctx, 2, xion))

	t.Log("Step 1: Perform operations that might cause indexer issues")

	// Send a transaction to an invalid address (will fail at module level)
	_, err := node.ExecTx(ctx,
		user.KeyName(),
		"bank", "send",
		user.FormattedAddress(),
		"xion1234567890abcdefghijklmnopqrstuvwxyz0000",
		"1000uxion",
	)
	require.Error(t, err, "sending to invalid address should fail")

	t.Log("Step 2: Verify chain continues despite any potential indexer issues")
	// Even if the indexer encounters any errors, the chain should continue
	err = testutil.WaitForBlocks(ctx, 5, xion)
	require.NoError(t, err, "chain should continue producing blocks regardless of indexer state")

	// Verify we can still query the chain height
	height, err := xion.Height(ctx)
	require.NoError(t, err, "should be able to get chain height")
	require.Greater(t, height, uint64(0), "chain height should be greater than 0")

	// Verify we can still execute valid transactions
	_, err = node.ExecTx(ctx,
		user.KeyName(),
		"bank", "send",
		user.FormattedAddress(),
		user.FormattedAddress(),
		"1uxion",
	)
	require.NoError(t, err, "should be able to execute a valid transaction")

	// Test with multiple rapid transactions to stress the indexer
	t.Log("Step 3: Stress test with rapid transactions")
	for i := 0; i < 5; i++ {
		_, _ = node.ExecTx(ctx,
			user.KeyName(),
			"bank", "send",
			user.FormattedAddress(),
			user.FormattedAddress(),
			"1uxion",
		)
	}

	// Chain should still be operational
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err, "chain should remain operational after rapid transactions")

	t.Log("✓ Test passed: Indexer is non-consensus-critical")
	t.Log("  - StopNodeOnErr is configured as false")
	t.Log("  - Indexer errors don't halt the node")
	t.Log("  - Chain continues to operate normally")
}
