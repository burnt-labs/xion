package e2e_xion

import (
	"context"
	"fmt"
	"path"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func init() {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
}

// TestCodeExecutionAuthorizationComprehensive tests the CodeExecutionAuthorization
// with multiple code IDs and contract instances to verify:
// 1. Code ID 1: CodeExecutionAuthorization allows executing ALL contracts from this code ID
// 2. Code ID 2: No authorization - executions should fail until granted
// 3. Revocation: After revoking, executions should fail
func TestCodeExecutionAuthorizationComprehensive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("E2E TEST: CodeExecutionAuthorization Comprehensive")
	t.Log("=====================================================")

	t.Parallel()
	ctx := context.Background()

	xion := testlib.BuildXionChain(t)

	// Create and fund test users
	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	granter := users[0]
	grantee := users[1]

	err := testutil.WaitForBlocks(ctx, 5, xion)
	require.NoError(t, err)

	t.Logf("Granter: %s", granter.FormattedAddress())
	t.Logf("Grantee: %s", grantee.FormattedAddress())

	// Upload the user_map contract 2 times to get different code IDs
	t.Log("Step 1: Uploading user_map contract 2 times to generate different code IDs...")

	codeID1, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 1: %s", codeID1)

	codeID2, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 2: %s", codeID2)

	// Instantiate 2 contracts from each code ID
	t.Log("Step 2: Instantiating 2 contracts from each code ID...")
	instantiateMsg := "{}"

	// Code ID 1 contracts
	contract1a, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID1, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 1A (code_id=%s): %s", codeID1, contract1a)

	contract1b, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID1, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 1B (code_id=%s): %s", codeID1, contract1b)

	// Code ID 2 contracts
	contract2a, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID2, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 2A (code_id=%s): %s", codeID2, contract2a)

	contract2b, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID2, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 2B (code_id=%s): %s", codeID2, contract2b)

	// Execute message for user_map contract
	execMsg := fmt.Sprintf(`{"update":{"value":"%s"}}`, `{\"key\":\"example\"}`)

	// Helper function to execute contract via authz
	executeViaAuthz := func(contract, msgFileName string) error {
		stdout, _, err := xion.GetNode().ExecBin(ctx,
			"tx", "wasm", "execute",
			contract,
			execMsg,
			"--from", granter.FormattedAddress(),
			"--chain-id", xion.Config().ChainID,
			"--generate-only",
		)
		if err != nil {
			return fmt.Errorf("generate tx failed: %w", err)
		}

		err = xion.GetNode().WriteFile(ctx, stdout, msgFileName)
		if err != nil {
			return fmt.Errorf("write file failed: %w", err)
		}

		_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
			grantee.KeyName(),
			"authz", "exec",
			path.Join(xion.GetNode().HomeDir(), msgFileName),
			"--chain-id", xion.Config().ChainID,
		)
		return err
	}

	// ============================================================
	// TEST CASE 1: No authorization - execution should fail
	// ============================================================
	t.Log("")
	t.Log("TEST CASE 1: No Authorization")
	t.Log("Expected: Contract execution should fail")
	t.Log("---------------------------------------------------")

	t.Log("Executing contract 1A without authorization (should fail)...")
	err = executeViaAuthz(contract1a, "exec_1a_no_auth.json")
	require.Error(t, err, "Contract 1A execution should fail without authorization")
	require.True(t, strings.Contains(err.Error(), "authorization not found"),
		"Error should indicate no authorization: %v", err)
	t.Log("Contract 1A: FAILED as expected (no authorization)")

	// ============================================================
	// TEST CASE 2: Grant CodeExecutionAuthorization for code ID 1
	// Both contracts from code ID 1 should succeed
	// ============================================================
	t.Log("")
	t.Log("TEST CASE 2: Code ID 1 - CodeExecutionAuthorization")
	t.Log("Expected: Both contracts 1A and 1B should succeed")
	t.Log("---------------------------------------------------")

	// Grant CodeExecutionAuthorization for code ID 1
	t.Logf("Creating CodeExecutionAuthorization for code ID %s...", codeID1)
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"xauthz", "grant",
		grantee.FormattedAddress(),
		codeID1,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("Created CodeExecutionAuthorization for code ID 1")

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	// Verify grant was created
	t.Log("Verifying grant was created...")
	grantsResp, err := testlib.ExecQuery(t, ctx, xion.GetNode(),
		"authz", "grants",
		granter.FormattedAddress(),
		grantee.FormattedAddress(),
	)
	require.NoError(t, err)
	t.Logf("Grants response: %v", grantsResp)

	grants, ok := grantsResp["grants"].([]interface{})
	require.True(t, ok, "grants should be an array")
	require.Equal(t, 1, len(grants), "should have exactly 1 grant")

	t.Log("Executing contract 1A...")
	err = executeViaAuthz(contract1a, "exec_1a.json")
	require.NoError(t, err, "Contract 1A execution should succeed (CodeExecutionAuthorization for code ID 1)")
	t.Log("Contract 1A: SUCCESS")

	t.Log("Executing contract 1B...")
	err = executeViaAuthz(contract1b, "exec_1b.json")
	require.NoError(t, err, "Contract 1B execution should succeed (CodeExecutionAuthorization for code ID 1)")
	t.Log("Contract 1B: SUCCESS")

	// ============================================================
	// TEST CASE 3: Code ID 2 - No authorization for this code ID
	// Contracts from code ID 2 should fail
	// ============================================================
	t.Log("")
	t.Log("TEST CASE 3: Code ID 2 - No Authorization for this code ID")
	t.Log("Expected: Contracts from code ID 2 should fail")
	t.Log("---------------------------------------------------")

	t.Log("Executing contract 2A (should fail - wrong code ID)...")
	err = executeViaAuthz(contract2a, "exec_2a.json")
	require.Error(t, err, "Contract 2A execution should fail (no authorization for code ID 2)")
	require.True(t, strings.Contains(err.Error(), "not in allowed list"),
		"Error should indicate code ID not allowed: %v", err)
	t.Log("Contract 2A: FAILED as expected (code ID not in allowed list)")

	t.Log("Executing contract 2B (should fail - wrong code ID)...")
	err = executeViaAuthz(contract2b, "exec_2b.json")
	require.Error(t, err, "Contract 2B execution should fail (no authorization for code ID 2)")
	require.True(t, strings.Contains(err.Error(), "not in allowed list"),
		"Error should indicate code ID not allowed: %v", err)
	t.Log("Contract 2B: FAILED as expected (code ID not in allowed list)")

	// ============================================================
	// TEST CASE 4: Revoke authorization - execution should fail
	// ============================================================
	t.Log("")
	t.Log("TEST CASE 4: Revoke Authorization")
	t.Log("Expected: Contract execution should fail after revoke")
	t.Log("---------------------------------------------------")

	t.Log("Revoking CodeExecutionAuthorization...")
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "revoke",
		grantee.FormattedAddress(),
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	err = testutil.WaitForBlocks(ctx, 2, xion)
	require.NoError(t, err)

	t.Log("Executing contract 1A after revoke (should fail)...")
	err = executeViaAuthz(contract1a, "exec_1a_after.json")
	require.Error(t, err, "Contract 1A should fail after authorization revoke")
	require.True(t, strings.Contains(err.Error(), "authorization not found"),
		"Error should indicate no authorization: %v", err)
	t.Log("Contract 1A: FAILED as expected (authorization revoked)")

	// ============================================================
	// SUMMARY
	// ============================================================
	t.Log("")
	t.Log("=====================================================")
	t.Log("TEST SUMMARY")
	t.Log("=====================================================")
	t.Log("No Authorization:")
	t.Log("  - Contract 1A: FAILED (no authorization)")
	t.Log("Code ID 1 (with CodeExecutionAuthorization):")
	t.Log("  - Contract 1A: SUCCESS")
	t.Log("  - Contract 1B: SUCCESS")
	t.Log("Code ID 2 (no authorization for this code ID):")
	t.Log("  - Contract 2A: FAILED (code ID not allowed)")
	t.Log("  - Contract 2B: FAILED (code ID not allowed)")
	t.Log("After Revoke:")
	t.Log("  - Contract 1A: FAILED (authorization revoked)")
	t.Log("=====================================================")
	t.Log("All test cases passed!")
}
