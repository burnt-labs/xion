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

// TestCodeExecutionAuthorization tests the CodeExecutionAuthorization
// which allows grantees to execute contracts only from specific code IDs.
func TestCodeExecutionAuthorization(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("E2E TEST: CodeExecutionAuthorization")
	t.Log("========================================")
	t.Log("Testing stateful authorization that restricts contract execution by code ID")

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

	// Upload the user_map contract 3 times to get different code IDs
	t.Log("Uploading user_map contract 3 times to generate different code IDs...")

	codeID1, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 1: %s", codeID1)

	codeID2, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 2: %s", codeID2)

	codeID3, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 3: %s", codeID3)

	// Instantiate contracts from each code ID
	t.Log("Instantiating contracts from each code ID...")

	// user_map contract has empty instantiate message
	instantiateMsg := "{}"

	contract1, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID1, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 1 (code_id=%s): %s", codeID1, contract1)

	contract2, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID2, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 2 (code_id=%s): %s", codeID2, contract2)

	contract3, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID3, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 3 (code_id=%s): %s", codeID3, contract3)

	// Create a CodeExecutionAuthorization grant allowing only code IDs 1 and 2 (not 3)
	t.Log("Creating CodeExecutionAuthorization grant for code IDs 1 and 2...")

	// Use the new code-execution authorization type
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "grant",
		grantee.FormattedAddress(),
		"code-execution",
		"--code-ids", fmt.Sprintf("%s,%s", codeID1, codeID2),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("Created CodeExecutionAuthorization grant")

	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Verify the grant was created
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
	require.GreaterOrEqual(t, len(grants), 1, "should have at least 1 grant")
	t.Log("Grant created successfully")

	// Test executing contract via authz - using MsgExec
	t.Log("Testing contract execution via authz...")

	// Execute contract 1 (should succeed with CodeExecutionAuthorization for code IDs 1,2)
	// user_map has an "update" execute message
	execMsg := fmt.Sprintf(`{"update":{"value":"%s"}}`, `{\"key\":\"example\"}`)

	// Generate unsigned transaction using --generate-only
	msgFile := "exec_msg.json"
	stdout, _, err := xion.GetNode().ExecBin(ctx,
		"tx", "wasm", "execute",
		contract1,
		execMsg,
		"--from", granter.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--generate-only",
	)
	require.NoError(t, err)

	// Write the unsigned transaction to a file
	err = xion.GetNode().WriteFile(ctx, stdout, msgFile)
	require.NoError(t, err)

	// Execute via authz exec
	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		grantee.KeyName(),
		"authz", "exec",
		path.Join(xion.GetNode().HomeDir(), msgFile),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("Contract 1 execution via authz succeeded")

	// Verify the value was set by querying contract state
	queryResp, _, err := xion.GetNode().ExecQuery(ctx, "wasm", "contract-state", "all", contract1)
	require.NoError(t, err)
	t.Logf("Contract 1 state after update: %s", string(queryResp))

	// Execute contract 2 (should also succeed with code IDs 1,2 allowed)
	msgFile2 := "exec_msg2.json"
	stdout2, _, err := xion.GetNode().ExecBin(ctx,
		"tx", "wasm", "execute",
		contract2,
		execMsg,
		"--from", granter.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--generate-only",
	)
	require.NoError(t, err)

	err = xion.GetNode().WriteFile(ctx, stdout2, msgFile2)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		grantee.KeyName(),
		"authz", "exec",
		path.Join(xion.GetNode().HomeDir(), msgFile2),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("Contract 2 execution via authz succeeded")

	// Execute contract 3 (should FAIL because code_id=3 is not in allowed list)
	t.Log("Testing that contract 3 execution fails (code ID not allowed)...")
	msgFile3 := "exec_msg3.json"
	stdout3, _, err := xion.GetNode().ExecBin(ctx,
		"tx", "wasm", "execute",
		contract3,
		execMsg,
		"--from", granter.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--generate-only",
	)
	require.NoError(t, err)

	err = xion.GetNode().WriteFile(ctx, stdout3, msgFile3)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		grantee.KeyName(),
		"authz", "exec",
		path.Join(xion.GetNode().HomeDir(), msgFile3),
		"--chain-id", xion.Config().ChainID,
	)
	require.Error(t, err, "Expected error when executing contract from disallowed code ID")
	require.True(t, strings.Contains(err.Error(), "not in allowed list") ||
		strings.Contains(err.Error(), "authorization not found"),
		"Error should indicate code ID not allowed: %v", err)
	t.Log("Contract 3 execution correctly failed (code ID not in allowed list)")

	t.Log("========================================")
	t.Log("CodeExecutionAuthorization E2E test completed")
}

// TestCodeExecutionAuthorizationFailure tests that execution fails
// when trying to execute a contract from a disallowed code ID.
func TestCodeExecutionAuthorizationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("E2E TEST: CodeExecutionAuthorization Failure Case")
	t.Log("====================================================")
	t.Log("Testing that execution fails for contracts not in allowed code IDs")

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

	// Upload contract twice
	t.Log("Uploading contracts...")

	codeID1, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 1: %s", codeID1)

	codeID2, err := xion.StoreContract(ctx, granter.FormattedAddress(),
		testlib.IntegrationTestPath("testdata", "contracts", "user_map.wasm"))
	require.NoError(t, err)
	t.Logf("Code ID 2: %s", codeID2)

	// Instantiate contracts
	instantiateMsg := "{}"

	contract1, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID1, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 1 (code_id=%s): %s", codeID1, contract1)

	contract2, err := xion.InstantiateContract(ctx, granter.FormattedAddress(), codeID2, instantiateMsg, true)
	require.NoError(t, err)
	t.Logf("Contract 2 (code_id=%s): %s", codeID2, contract2)

	// Create a grant allowing ONLY code ID 1 (not code ID 2)
	t.Log("Creating CodeExecutionAuthorization grant for code ID 1 only...")

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		granter.KeyName(),
		"authz", "grant",
		grantee.FormattedAddress(),
		"code-execution",
		"--code-ids", codeID1, // Only allow code ID 1
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("Created CodeExecutionAuthorization grant for code ID 1 only")

	err = testutil.WaitForBlocks(ctx, 3, xion)
	require.NoError(t, err)

	// Test 1: Execute contract 2 (code ID 2) - should FAIL
	t.Log("Testing that contract 2 execution fails (code ID 2 not allowed)...")

	execMsg := fmt.Sprintf(`{"update":{"value":"%s"}}`, `{\"key\":\"example\"}`)

	// Generate unsigned transaction for contract 2
	msgFile2 := "exec_msg2.json"
	stdout2, _, err := xion.GetNode().ExecBin(ctx,
		"tx", "wasm", "execute",
		contract2,
		execMsg,
		"--from", granter.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--generate-only",
	)
	require.NoError(t, err)

	err = xion.GetNode().WriteFile(ctx, stdout2, msgFile2)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		grantee.KeyName(),
		"authz", "exec",
		path.Join(xion.GetNode().HomeDir(), msgFile2),
		"--chain-id", xion.Config().ChainID,
	)
	require.Error(t, err, "Expected error when executing contract from disallowed code ID")
	t.Logf("Contract 2 execution correctly failed: %v", err)

	// Test 2: Execute contract 1 (code ID 1) - should SUCCEED
	t.Log("Testing that contract 1 execution succeeds (code ID 1 is allowed)...")

	msgFile1 := "exec_msg1.json"
	stdout1, _, err := xion.GetNode().ExecBin(ctx,
		"tx", "wasm", "execute",
		contract1,
		execMsg,
		"--from", granter.FormattedAddress(),
		"--chain-id", xion.Config().ChainID,
		"--generate-only",
	)
	require.NoError(t, err)

	err = xion.GetNode().WriteFile(ctx, stdout1, msgFile1)
	require.NoError(t, err)

	_, err = testlib.ExecTx(t, ctx, xion.GetNode(),
		grantee.KeyName(),
		"authz", "exec",
		path.Join(xion.GetNode().HomeDir(), msgFile1),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	t.Log("Contract 1 execution succeeded (code ID 1 is allowed)")

	t.Log("====================================================")
	t.Log("CodeExecutionAuthorization Failure Case test completed")
}
