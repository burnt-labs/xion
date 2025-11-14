package e2e_indexer

import (
	"context"
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func init() {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
}

// TestIndexerAuthzCreate tests the creation of a single authz grant and its indexing
func TestIndexerAuthzCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: Authz Create")
	t.Log("==================================")
	t.Log("Testing single authz grant creation and indexing")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	granter := users[0]
	grantee := users[1]

	// Create a single authz grant
	_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "grant",
		grantee.FormattedAddress(),
		"send",
		"--spend-limit", "1000000uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Authz grant creation should succeed")

	// Wait for indexing
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query to verify indexing
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"indexer", "query-grants-by-granter",
		granter.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query by granter should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify the grant exists
	grants, ok := response["grants"]
	if ok {
		grantsList, ok := grants.([]interface{})
		if ok {
			require.GreaterOrEqual(t, len(grantsList), 1, "Should have at least 1 grant")
			t.Log("✓ Authz grant successfully created and indexed")
		}
	}
}

// TestIndexerAuthzMultiple tests the creation of multiple authz grants and their indexing
func TestIndexerAuthzMultiple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: Authz Multiple")
	t.Log("====================================")
	t.Log("Testing multiple authz grants creation and indexing")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion, xion)
	granter := users[0]
	grantee1 := users[1]
	grantee2 := users[2]
	grantee3 := users[3]

	// Create multiple authz grants
	grantees := []struct {
		user       ibc.Wallet
		spendLimit string
	}{
		{grantee1, "1000000uxion"},
		{grantee2, "2000000uxion"},
		{grantee3, "3000000uxion"},
	}

	for _, g := range grantees {
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			granter.KeyName(),
			"authz", "grant",
			g.user.FormattedAddress(),
			"send",
			"--spend-limit", g.spendLimit,
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Authz grant creation should succeed")
	}

	// Wait for indexing
	err := testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query by granter to see all grants
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"indexer", "query-grants-by-granter",
		granter.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query by granter should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err, "Response should be valid JSON")

	// Verify all grants exist
	grants, ok := response["grants"]
	if ok {
		grantsList, ok := grants.([]interface{})
		if ok {
			require.GreaterOrEqual(t, len(grantsList), 3, "Should have at least 3 grants")
			t.Logf("✓ Successfully created and indexed %d authz grants", len(grantsList))
		}
	}

	// Test querying by each grantee
	for _, g := range grantees {
		stdout, _, err := xion.GetNode().ExecBin(ctx,
			"indexer", "query-grants-by-grantee",
			g.user.FormattedAddress(),
			"--node", xion.GetRPCAddress(),
			"--output", "json",
		)
		require.NoError(t, err, "Query by grantee should succeed")

		err = json.Unmarshal(stdout, &response)
		require.NoError(t, err, "Response should be valid JSON")

		grants, ok := response["grants"]
		if ok {
			grantsList, ok := grants.([]interface{})
			if ok {
				require.GreaterOrEqual(t, len(grantsList), 1, "Each grantee should have at least 1 grant")
			}
		}
	}

	t.Log("✓ All multiple authz grants successfully indexed and queryable")
}

// TestIndexerAuthzRevoke tests the revocation of authz grants and index cleanup
func TestIndexerAuthzRevoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔍 INDEXER E2E TEST: Authz Revoke")
	t.Log("==================================")
	t.Log("Testing authz grant revocation and index cleanup")

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion)
	granter := users[0]
	grantee := users[1]
	_ = users[2] // recipient - reserved for future use

	// Create an authz grant
	_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "grant",
		grantee.FormattedAddress(),
		"send",
		"--spend-limit", "1000000uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Authz grant creation should succeed")

	// Wait for indexing
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Verify grant exists
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"indexer", "query-grants-by-grantee",
		grantee.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query should succeed")

	var response map[string]interface{}
	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err)

	grants, _ := response["grants"].([]interface{})
	require.GreaterOrEqual(t, len(grants), 1, "Grant should exist before revocation")

	// Revoke the grant
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "revoke",
		grantee.FormattedAddress(),
		"/cosmos.bank.v1beta1.MsgSend",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Revoking grant should succeed")

	// Wait for revocation to be indexed
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Query again - should have no grants
	stdout, _, err = xion.GetNode().ExecBin(ctx,
		"indexer", "query-grants-by-grantee",
		grantee.FormattedAddress(),
		"--node", xion.GetRPCAddress(),
		"--output", "json",
	)
	require.NoError(t, err, "Query after revoke should succeed")

	err = json.Unmarshal(stdout, &response)
	require.NoError(t, err)

	grants, _ = response["grants"].([]interface{})
	require.Equal(t, 0, len(grants), "Should have no grants after revocation")
	t.Log("✓ Authz grant successfully revoked and index cleaned up")

	// Test robustness with double revoke
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "revoke",
		grantee.FormattedAddress(),
		"/cosmos.bank.v1beta1.MsgSend",
		"--chain-id", xion.Config().ChainID,
	)
	require.Error(t, err, "Double revoke should fail at module level")

	// Verify chain continues
	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err, "Chain should continue after failed double revoke")
	t.Log("✓ Indexer handled double revoke gracefully")
}
