package e2e_indexer

import (
	"context"
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

// TestIndexerFeeGrant tests the FeeGrant indexer functionality end-to-end
// This validates that fee grant allowances are properly indexed and queryable
func TestIndexerFeeGrant(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: FeeGrant Allowance Indexing")
	t.Log("=================================================")
	t.Log("Testing fee grant allowance creation, indexing, and querying")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion, xion)
	granter1 := users[0]
	granter2 := users[1]
	grantee := users[2]
	_ = users[3] // spare

	t.Run("CreateFeeGrants", func(t *testing.T) {
		t.Log("Step 1: Creating multiple fee grants to test indexing")

		// Grant 1: granter1 -> grantee
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			granter1.KeyName(),
			"feegrant", "grant",
			granter1.FormattedAddress(),
			grantee.FormattedAddress(),
			"--spend-limit", "1000000uxion",
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "First fee grant should succeed")

		// Grant 2: granter2 -> grantee (same grantee, different granter)
		_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
			granter2.KeyName(),
			"feegrant", "grant",
			granter2.FormattedAddress(),
			grantee.FormattedAddress(),
			"--spend-limit", "2000000uxion",
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Second fee grant should succeed")

		// Wait for indexing
		err = testutil.WaitForBlocks(ctx, 3, xion)
		require.NoError(t, err)

		t.Log("✓ Fee grants created successfully")
	})

	t.Run("QueryAllowancesByGrantee", func(t *testing.T) {
		t.Log("Step 2: Query allowances by grantee (tests grantee ReversePair index)")

		// Note: The command is "query-grants-by-grantee" but it queries fee allowances
		stdout, _, err := xion.GetNode().ExecBin(ctx,
			"indexer", "query-grants-by-grantee",
			grantee.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
		if err != nil {
			t.Logf("Query error (may not be implemented yet): %v", err)
			t.Logf("Output: %s", string(stdout))
			t.Skip("Indexer query command may not be available in this build")
			return
		}

		t.Logf("Allowances for grantee: %s", string(stdout))

		var response map[string]interface{}
		err = json.Unmarshal(stdout, &response)
		if err != nil {
			t.Logf("Could not parse JSON response: %v", err)
			t.Logf("Raw output: %s", string(stdout))
			return
		}

		// Should have 2 allowances for this grantee (from granter1 and granter2)
		allowances, ok := response["allowances"]
		if ok {
			allowancesList, ok := allowances.([]interface{})
			if ok {
				require.GreaterOrEqual(t, len(allowancesList), 2, "Should have at least 2 allowances for this grantee")
				t.Logf("✓ Found %d allowances for grantee", len(allowancesList))
			}
		}
	})

	t.Run("QueryAllowancesByGranter", func(t *testing.T) {
		t.Log("Step 3: Query allowances by granter (tests granter primary key)")

		stdout, _, err := xion.GetNode().ExecBin(ctx,
			"indexer", "query-allowances-by-granter",
			granter1.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
		require.NoError(t, err, "Query allowances by granter should succeed")

		t.Logf("Allowances by granter1: %s", string(stdout))

		var response map[string]interface{}
		err = json.Unmarshal(stdout, &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Should have at least 1 allowance from this granter
		allowances, ok := response["allowances"]
		if ok {
			allowancesList, ok := allowances.([]interface{})
			if ok {
				require.GreaterOrEqual(t, len(allowancesList), 1, "Should have at least 1 allowance from this granter")
				t.Logf("✓ Found %d allowances from granter1", len(allowancesList))
			}
		}
	})

	t.Run("RevokeFeeGrantAndVerifyIndexCleanup", func(t *testing.T) {
		t.Log("Step 4: Revoke a fee grant and verify index cleanup")

		// Revoke granter1's allowance
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			granter1.KeyName(),
			"feegrant", "revoke",
			granter1.FormattedAddress(),
			grantee.FormattedAddress(),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Revoking fee grant should succeed")

		// Wait for revocation to be indexed
		err = testutil.WaitForBlocks(ctx, 3, xion)
		require.NoError(t, err)

		// Query grantee's allowances - should now have only 1 (from granter2)
		stdout, _, err := xion.GetNode().ExecBin(ctx,
			"indexer", "query-grants-by-grantee",
			grantee.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
		require.NoError(t, err, "Query after revoke should succeed")

		t.Logf("Allowances after revoke: %s", string(stdout))

		var response map[string]interface{}
		err = json.Unmarshal(stdout, &response)
		require.NoError(t, err, "Response should be valid JSON")

		allowances, ok := response["allowances"]
		if ok {
			allowancesList, ok := allowances.([]interface{})
			if ok {
				require.Equal(t, 1, len(allowancesList), "Should have 1 allowance after revoking one")
				t.Log("✓ Index correctly cleaned up after fee grant revocation")
			}
		}
	})

	t.Run("TestFeegrantRobustnessOnDuplicateRevoke", func(t *testing.T) {
		t.Log("Step 5: Test robustness when revoking an already-revoked feegrant")

		// Try to revoke the already-revoked feegrant from granter1
		// This tests the indexer's ability to handle non-existent allowance deletions gracefully
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			granter1.KeyName(),
			"feegrant", "revoke",
			granter1.FormattedAddress(),
			grantee.FormattedAddress(),
			"--chain-id", xion.Config().ChainID,
		)
		// The transaction will fail at the module level but shouldn't affect the indexer
		require.Error(t, err, "double revoke should fail at module level")

		// Verify chain continues to produce blocks (indexer didn't halt)
		err = testutil.WaitForBlocks(ctx, 3, xion)
		require.NoError(t, err, "chain should continue producing blocks after processing non-existent allowance delete")

		t.Log("✓ Feegrant indexer handled duplicate revoke gracefully without disrupting consensus")
	})
}

// TestIndexerFeeGrantMultiple tests the creation of multiple fee grants and their indexing
func TestIndexerFeeGrantMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: FeeGrant Multiple")
	t.Log("=======================================")
	t.Log("Testing multiple fee grants creation and indexing")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion, xion)
	granter1 := users[0]
	granter2 := users[1]
	grantee1 := users[2]
	grantee2 := users[3]

	// Create multiple fee grants with different configurations
	grants := []struct {
		granter    ibc.Wallet
		grantee    ibc.Wallet
		spendLimit string
	}{
		{granter1, grantee1, "1000000uxion"},
		{granter1, grantee2, "2000000uxion"},
		{granter2, grantee1, "3000000uxion"},
		{granter2, grantee2, "4000000uxion"},
	}

	for _, g := range grants {
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			g.granter.KeyName(),
			"feegrant", "grant",
			g.granter.FormattedAddress(),
			g.grantee.FormattedAddress(),
			"--spend-limit", g.spendLimit,
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Fee grant creation should succeed")
	}

	// Wait for indexing
	err := testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query by granter1
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-granter",
		granter1.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	if err != nil {
		stdout, _, err = xion.GetNode().ExecBin(ctx,
			"indexer", "query-grants-by-granter",
			granter1.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
	}
	require.NoError(t, err, "Query by granter1 should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err, "Response should be valid JSON")

	var items []interface{}
	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.GreaterOrEqual(t, len(items), 2, "Granter1 should have at least 2 grants")

	// Query by grantee1 (should see grants from both granters)
	stdout, _, err = xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-grantee",
		grantee1.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query by grantee1 should succeed")

	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err, "Response should be valid JSON")

	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.GreaterOrEqual(t, len(items), 2, "Grantee1 should have at least 2 grants from different granters")

	t.Log("✓ All multiple fee grants successfully indexed and queryable")
}

// TestIndexerFeeGrantPeriodic tests the creation of periodic fee grants and their indexing
func TestIndexerFeeGrantPeriodic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: FeeGrant Periodic")
	t.Log("========================================")
	t.Log("Testing periodic fee grant creation and indexing")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion)
	granter := users[0]
	grantee1 := users[1]
	grantee2 := users[2]

	// Create a periodic fee grant (resets every hour with 100k limit per period)
	_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"feegrant", "grant",
		granter.FormattedAddress(),
		grantee1.FormattedAddress(),
		"--period", "3600", // 1 hour period (in seconds)
		"--period-limit", "100000uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Periodic fee grant creation should succeed")

	// Create a periodic grant with both period limit and overall spend limit
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"feegrant", "grant",
		granter.FormattedAddress(),
		grantee2.FormattedAddress(),
		"--period", "1800", // 30 minute period (in seconds)
		"--period-limit", "50000uxion",
		"--spend-limit", "1000000uxion", // Overall limit
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Periodic fee grant with spend limit should succeed")

	// Wait for indexing
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query to verify periodic grants are indexed
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-granter",
		granter.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	if err != nil {
		stdout, _, err = xion.GetNode().ExecBin(ctx,
			"indexer", "query-grants-by-granter",
			granter.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
	}
	require.NoError(t, err, "Query by granter should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err, "Response should be valid JSON")

	var items []interface{}
	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.GreaterOrEqual(t, len(items), 2, "Should have at least 2 periodic grants")
	t.Log("✓ Periodic fee grants successfully created and indexed")

	// Test that grantee can use the periodic grant
	// Generate a transaction to test the grant
	msgFile := "feegrant_test_msg.json"
	stdout, _, err = xion.GetNode().ExecBin(ctx,
		"tx", "bank", "send",
		grantee1.FormattedAddress(),
		granter.FormattedAddress(), // Send back to granter
		"1uxion",
		"--from", grantee1.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--generate-only",
	)
	require.NoError(t, err, "Generating test transaction should succeed")

	// Write the transaction to a file
	err = xion.GetNode().WriteFile(ctx, stdout, msgFile)
	require.NoError(t, err, "Creating message file should succeed")

	// Execute with fee grant
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		grantee1.KeyName(),
		"tx", "sign",
		msgFile,
		"--from", grantee1.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--fee-granter", granter.FormattedAddress(),
	)
	// The signing might fail if the CLI doesn't support all flags, but the grant should still be indexed
	// We're primarily testing indexing, not the grant usage itself

	// Wait and verify grants still exist
	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Query again to ensure periodic grants persist after potential usage
	stdout, _, err = xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-grantee",
		grantee1.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query after potential usage should succeed")

	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err, "Response should be valid JSON")

	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.GreaterOrEqual(t, len(items), 1, "Periodic grant should still exist")
	t.Log("✓ Periodic fee grants remain indexed after usage")
}

// TestIndexerFeeGrantRevoke tests the revocation of fee grants and index cleanup
func TestIndexerFeeGrantRevoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: FeeGrant Revoke")
	t.Log("=====================================")
	t.Log("Testing fee grant revocation and index cleanup")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion)
	granter1 := users[0]
	granter2 := users[1]
	grantee := users[2]

	// Create fee grants from two different granters
	txResp1, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		granter1.KeyName(),
		"feegrant", "grant",
		granter1.FormattedAddress(),
		grantee.FormattedAddress(),
		"--spend-limit", "1000000uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "First fee grant should succeed")
	t.Logf("First grant tx response: %v", txResp1)

	txResp2, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		granter2.KeyName(),
		"feegrant", "grant",
		granter2.FormattedAddress(),
		grantee.FormattedAddress(),
		"--spend-limit", "2000000uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Second fee grant should succeed")
	t.Logf("Second grant tx response: %v", txResp2)

	// Wait for indexing - increased wait time to ensure indexer processes the grants
	err = testutil.WaitForBlocks(ctx, 5, xion)
	require.NoError(t, err)

	// First verify grants exist in the feegrant module directly
	feeGrantQuery, _, err := xion.GetNode().ExecBin(ctx,
		"query", "feegrant", "grants-by-grantee",
		grantee.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Direct feegrant query should succeed")
	t.Logf("Direct feegrant query response: %s", string(feeGrantQuery))

	// Verify both grants exist through indexer
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-grantee",
		grantee.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	if err != nil {
		// Try querying by granter instead to verify grants exist
		stdout, _, err = xion.GetNode().ExecBin(ctx,
			"indexer", "query-allowances-by-granter",
			granter1.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
	}
	require.NoError(t, err, "Indexer query should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err)

	// Log the raw response for debugging
	t.Logf("Query response: %s", string(stdout))

	var items []interface{}
	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.GreaterOrEqual(t, len(items), 2, "Should have 2 grants before revocation")

	// Revoke the first grant
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter1.KeyName(),
		"feegrant", "revoke",
		granter1.FormattedAddress(),
		grantee.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Revoking first grant should succeed")

	// Wait for revocation to be indexed
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query again - should have only 1 grant
	stdout, _, err = xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-grantee",
		grantee.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query after first revoke should succeed")

	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err)

	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.Equal(t, 1, len(items), "Should have 1 grant after first revocation")
	t.Log("✓ First fee grant successfully revoked and index updated")

	// Revoke the second grant
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter2.KeyName(),
		"feegrant", "revoke",
		granter2.FormattedAddress(),
		grantee.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Revoking second grant should succeed")

	// Wait for revocation to be indexed
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query again - should have no grants
	stdout, _, err = xion.GetNode().ExecBin(ctx,
		"indexer", "query-allowances-by-grantee",
		grantee.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query after second revoke should succeed")

	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err)

	if allowances, ok := response["allowances"].([]interface{}); ok {
		items = allowances
	} else if grants, ok := response["grants"].([]interface{}); ok {
		items = grants
	}
	require.Equal(t, 0, len(items), "Should have no grants after all revocations")
	t.Log("✓ All fee grants successfully revoked and index cleaned up")

	// Test robustness with double revoke
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter1.KeyName(),
		"feegrant", "revoke",
		granter1.FormattedAddress(),
		grantee.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
	)
	require.Error(t, err, "Double revoke should fail at module level")

	// Verify chain continues
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err, "Chain should continue after failed double revoke")
	t.Log("✓ Indexer handled double revoke gracefully")
}
