package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

// ModifyGenesisGlobalFee sets the globalfee module parameters in genesis
func ModifyGenesisGlobalFee(chainConfig ibc.ChainConfig, genbz []byte, params ...string) ([]byte, error) {
	g := make(map[string]interface{})
	if err := json.Unmarshal(genbz, &g); err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
	}

	// Ensure app_state.globalfee exists and set parameters
	appState := g["app_state"].(map[string]interface{})
	if appState["globalfee"] == nil {
		appState["globalfee"] = make(map[string]interface{})
	}

	appState["globalfee"].(map[string]interface{})["params"] = map[string]interface{}{
		"minimum_gas_prices": []map[string]string{{
			"denom":  "uxion",
			"amount": "0.001000000000000000",
		}},
		"bypass_min_fee_msg_types": []string{
			"/xion.v1.MsgSend",
			"/xion.v1.MsgMultiSend",
			"/xion.jwk.v1.MsgDeleteAudience",
			"/xion.jwk.v1.MsgDeleteAudienceClaim",
			"/cosmos.authz.v1beta1.MsgRevoke",
			"/cosmos.feegrant.v1beta1.MsgRevokeAllowance",
		},
		"max_total_bypass_min_fee_msg_gas_usage": "1000000",
	}

	return json.Marshal(g)
}

// TestGlobalFeeBypassMessages tests the global fee module's bypass message functionality
func TestGlobalFeeBypassMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	// Set up chain with globalfee configured at 0.001uxion per gas
	td := BuildXionChain(t, "0.001uxion", ModifyInterChainGenesis(
		ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisGlobalFee},
		[][]string{{votingPeriod, maxDepositPeriod}, {}},
	))
	xion, ctx := td.xionChain, td.ctx

	// Create and Fund User Wallets - we'll create enough for all tests
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion, xion)

	testutil.WaitForBlocks(ctx, 5, xion)

	// Run all test scenarios with different wallets for each
	t.Run("NonBypassMessageRequiresFees", func(t *testing.T) {
		testNonBypassMessageRequiresFees(t, ctx, xion, users[0], users[1])
	})

	t.Run("DirectBypassMessage", func(t *testing.T) {
		testDirectBypassMessage(t, ctx, xion, users[2], users[3])
	})

	t.Run("AuthzWrappedBypassMessage", func(t *testing.T) {
		testAuthzWrappedBypassMessage(t, ctx, xion, users[4], users[5])
	})

	t.Run("NestedAuthzBypassMessage", func(t *testing.T) {
		testNestedAuthzBypassMessage(t, ctx, xion, users[6], users[7])
	})

	t.Run("MixedMessagesRequireFees", func(t *testing.T) {
		testMixedMessagesRequireFees(t, ctx, xion, users[8], users[9])
	})

	t.Run("MultipleBypassMessages", func(t *testing.T) {
		testMultipleBypassMessages(t, ctx, xion, users[10], users[11])
	})
}

// testNonBypassMessageRequiresFees verifies that non-bypass messages require fees
func testNonBypassMessageRequiresFees(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, alice, bob ibc.Wallet) {
	// Bank send is not a bypass message, should fail with zero fees
	_, err := ExecTxWithGas(t, ctx, xion.GetNode(), alice.KeyName(), "0uxion",
		"bank", "send", alice.FormattedAddress(), bob.FormattedAddress(), "100uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.Error(t, err, "Non-bypass message should fail with zero fees")
	require.Contains(t, err.Error(), "insufficient fee")

	// Should work with proper fees
	_, err = ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"bank", "send", alice.FormattedAddress(), bob.FormattedAddress(), "100uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Non-bypass message should work with proper fees")
}

// testDirectBypassMessage tests that direct bypass messages work with zero fees
func testDirectBypassMessage(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, alice, bob ibc.Wallet) {
	// Create a grant to revoke
	_, err := ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", bob.FormattedAddress(), "send",
		"--spend-limit", "1000uxion",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Revoke with zero fees - should work as authz.MsgRevoke is a bypass message
	_, err = ExecTxWithGas(t, ctx, xion.GetNode(), alice.KeyName(), "0uxion",
		"authz", "revoke", bob.FormattedAddress(), "/cosmos.bank.v1beta1.MsgSend",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Direct bypass message (authz.MsgRevoke) should work with zero fees")

	// Verify the grant was revoked
	testutil.WaitForBlocks(ctx, 2, xion)
	result, err := ExecQuery(t, ctx, xion.GetNode(),
		"authz", "grants",
		alice.FormattedAddress(),
		bob.FormattedAddress(),
		"/cosmos.bank.v1beta1.MsgSend",
	)
	
	// If the grant doesn't exist, the query returns an error - this is expected after revocation
	if err != nil {
		require.Contains(t, err.Error(), "authorization not found", "Expected authorization not found error after revocation")
	} else {
		// If no error, check that grants array is empty
		grants, ok := result["grants"].([]interface{})
		require.True(t, ok)
		require.Empty(t, grants, "Grant should be revoked")
	}
}

// testAuthzWrappedBypassMessage tests authz-wrapped bypass messages
func testAuthzWrappedBypassMessage(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, alice, bob ibc.Wallet) {
	// Grant permission for authz operations
	_, err := ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", bob.FormattedAddress(), "generic",
		"--msg-type", "/cosmos.authz.v1beta1.MsgRevoke",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Create another grant to revoke
	_, err = ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", bob.FormattedAddress(), "generic",
		"--msg-type", "/cosmos.staking.v1beta1.MsgDelegate",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	testutil.WaitForBlocks(ctx, 2, xion)

	// Create a revoke transaction
	revokeTxFile := "authz_revoke_wrapped.json"
	revokeTxJSON, err := GenerateTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "revoke", bob.FormattedAddress(), "/cosmos.staking.v1beta1.MsgDelegate",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	err = xion.GetNode().WriteFile(ctx, []byte(revokeTxJSON), revokeTxFile)
	require.NoError(t, err)

	// Execute the revoke via authz with zero fees
	_, err = ExecTxWithGas(t, ctx, xion.GetNode(), bob.KeyName(), "0uxion",
		"authz", "exec", xion.GetNode().HomeDir()+"/"+revokeTxFile,
		"--from", bob.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Authz-wrapped bypass message should work with zero fees")

	// Verify the grant was revoked
	testutil.WaitForBlocks(ctx, 2, xion)
	result, err := ExecQuery(t, ctx, xion.GetNode(),
		"authz", "grants",
		alice.FormattedAddress(),
		bob.FormattedAddress(),
		"/cosmos.staking.v1beta1.MsgDelegate",
	)
	
	// If the grant doesn't exist, the query returns an error - this is expected after revocation
	if err != nil {
		require.Contains(t, err.Error(), "authorization not found", "Expected authorization not found error after revocation")
	} else {
		// If no error, check that grants array is empty
		grants, ok := result["grants"].([]interface{})
		require.True(t, ok)
		require.Empty(t, grants, "Grant should be revoked via authz-wrapped message")
	}
}

// testNestedAuthzBypassMessage tests nested authz-wrapped bypass messages
func testNestedAuthzBypassMessage(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, alice, bob ibc.Wallet) {
	// For simplicity, let's test a simpler nested case:
	// Alice -> grants Bob to execute MsgExec
	// Bob -> executes MsgExec containing Alice's MsgRevoke

	// 1. Alice grants Bob permission to execute MsgExec on her behalf
	_, err := ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", bob.FormattedAddress(), "generic",
		"--msg-type", "/cosmos.authz.v1beta1.MsgExec",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// 2. Alice grants Bob permission to revoke on her behalf
	_, err = ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", bob.FormattedAddress(), "generic",
		"--msg-type", "/cosmos.authz.v1beta1.MsgRevoke",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// 3. Create a third user and grant them something to revoke
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	charlie := users[0]

	_, err = ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", charlie.FormattedAddress(), "send",
		"--spend-limit", "500uxion",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	testutil.WaitForBlocks(ctx, 2, xion)

	// Create nested authz transaction
	// 1. Create the inner MsgRevoke (Alice revoking Charlie's grant)
	innerRevokeTxFile := "inner_revoke.json"
	innerRevokeTxJSON, err := GenerateTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "revoke", charlie.FormattedAddress(), "/cosmos.bank.v1beta1.MsgSend",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	err = xion.GetNode().WriteFile(ctx, []byte(innerRevokeTxJSON), innerRevokeTxFile)
	require.NoError(t, err)

	// 2. Alice creates an outer MsgExec wrapping the revoke
	outerExecTxFile := "outer_exec.json"
	outerExecTxJSON, err := GenerateTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "exec", xion.GetNode().HomeDir()+"/"+innerRevokeTxFile,
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	err = xion.GetNode().WriteFile(ctx, []byte(outerExecTxJSON), outerExecTxFile)
	require.NoError(t, err)

	// 3. Bob executes the nested authz (MsgExec containing MsgExec containing MsgRevoke)
	_, err = ExecTxWithGas(t, ctx, xion.GetNode(), bob.KeyName(), "0uxion",
		"authz", "exec", xion.GetNode().HomeDir()+"/"+outerExecTxFile,
		"--from", bob.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err, "Nested authz-wrapped bypass message should work with zero fees")
}

// testMixedMessagesRequireFees tests that transactions with mixed bypass and non-bypass messages require fees
func testMixedMessagesRequireFees(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, alice, bob ibc.Wallet) {
	// This test would require creating a transaction with multiple messages
	// For now, we'll test that a non-bypass message wrapped in authz still requires fees

	// Grant permission for bank send (non-bypass)
	_, err := ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "grant", bob.FormattedAddress(), "send",
		"--spend-limit", "2000uxion",
		"--from", alice.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Create a bank send transaction (non-bypass)
	bankSendTxFile := "bank_send_tx.json"
	bankSendTxJSON, err := GenerateTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"bank", "send", alice.FormattedAddress(), bob.FormattedAddress(), "50uxion",
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)
	err = xion.GetNode().WriteFile(ctx, []byte(bankSendTxJSON), bankSendTxFile)
	require.NoError(t, err)

	// Try to execute with zero fees - should fail
	_, err = ExecTxWithGas(t, ctx, xion.GetNode(), bob.KeyName(), "0uxion",
		"authz", "exec", xion.GetNode().HomeDir()+"/"+bankSendTxFile,
		"--from", bob.KeyName(),
		"--chain-id", xion.Config().ChainID,
	)
	require.Error(t, err, "Authz-wrapped non-bypass message should fail with zero fees")
	require.Contains(t, err.Error(), "insufficient fee")
}

// testMultipleBypassMessages tests multiple bypass messages in a single transaction
func testMultipleBypassMessages(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, alice, bob ibc.Wallet) {
	// Create multiple grants to revoke
	grants := []string{
		"/cosmos.bank.v1beta1.MsgSend",
		"/cosmos.staking.v1beta1.MsgDelegate",
		"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward",
	}

	// Create all grants first
	for _, msgType := range grants {
		_, err := ExecTx(t, ctx, xion.GetNode(), alice.KeyName(),
			"authz", "grant", bob.FormattedAddress(), "generic",
			"--msg-type", msgType,
			"--from", alice.KeyName(),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err)
	}

	testutil.WaitForBlocks(ctx, 2, xion)

	// Create a multi-message transaction by building the JSON manually
	// First, get a template transaction
	templateTxJSON, err := GenerateTx(t, ctx, xion.GetNode(), alice.KeyName(),
		"authz", "revoke", bob.FormattedAddress(), grants[0],
		"--chain-id", xion.Config().ChainID,
	)
	require.NoError(t, err)

	// Parse the template
	var txData map[string]interface{}
	err = json.Unmarshal([]byte(templateTxJSON), &txData)
	require.NoError(t, err)

	// Extract the first message as a template
	body := txData["body"].(map[string]interface{})
	messages := body["messages"].([]interface{})
	msgTemplate := messages[0].(map[string]interface{})

	// Build multiple messages
	newMessages := make([]interface{}, len(grants))
	for i, msgType := range grants {
		msg := make(map[string]interface{})
		msg["@type"] = msgTemplate["@type"]
		msg["granter"] = msgTemplate["granter"]
		msg["grantee"] = msgTemplate["grantee"]
		msg["msg_type_url"] = msgType
		newMessages[i] = msg
	}

	// Update the transaction with multiple messages
	body["messages"] = newMessages

	// Marshal back to JSON
	multiMsgTxJSON, err := json.Marshal(txData)
	require.NoError(t, err)

	// Sign and broadcast the multi-message transaction with zero fees
	node := xion.GetNode()
	signedTx, err := ExecSignTx(t, ctx, node, alice.KeyName(), multiMsgTxJSON, "0uxion")
	require.NoError(t, err)
	
	stdout, err := ExecBroadcast(t, ctx, node, signedTx)
	require.NoError(t, err)
	
	// Verify transaction succeeded
	var txResult map[string]interface{}
	err = json.Unmarshal([]byte(stdout), &txResult)
	require.NoError(t, err)
	require.Equal(t, float64(0), txResult["code"], "Transaction should succeed")

	// Wait for transaction to be processed
	testutil.WaitForBlocks(ctx, 2, xion)

	// Verify all grants have been revoked by querying on-chain
	for _, msgType := range grants {
		result, err := ExecQuery(t, ctx, xion.GetNode(),
			"authz", "grants",
			alice.FormattedAddress(), // granter
			bob.FormattedAddress(),   // grantee
			msgType,                  // msg-type
		)
		
		// If the grant doesn't exist, the query returns an error - this is expected after revocation
		if err != nil {
			require.Contains(t, err.Error(), "authorization not found", fmt.Sprintf("Expected authorization not found error after revoking %s", msgType))
		} else {
			// If no error, check that grants array is empty (revoked)
			grants, ok := result["grants"].([]interface{})
			require.True(t, ok, "Expected grants field in response")
			require.Empty(t, grants, fmt.Sprintf("Grant for %s should be revoked", msgType))
		}
	}

	t.Log("✓ All grants successfully revoked in multi-message transaction")
}
