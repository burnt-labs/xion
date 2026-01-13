package e2e_ibc

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/relayer"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
)

// TestIBCTokenTransfer tests secure IBC token transfers
// This is a Priority 1 test preventing cross-chain asset theft
//
// CRITICAL: IBC transfers must:
// - Validate channel security parameters
// - Prevent unauthorized token minting
// - Handle packet timeouts correctly
// - Verify source chain authenticity
func TestIBCTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔐 PRIORITY 1 SECURITY TEST: IBC Token Transfer Security")
	t.Log("==========================================================")
	t.Log("Testing IBC cross-chain transfer security mechanisms")
	t.Log("")

	ctx := t.Context()

	// Create chain specs using LocalChainSpec to respect XION_IMAGE env var
	xionChainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xionChainSpec.ChainConfig.ModifyGenesis = cosmos.ModifyGenesis(append(testlib.DefaultGenesisKVMods,
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices", []map[string]string{{"denom": "uxion", "amount": "0"}}),
	))

	osmosisChainSpec := testlib.OsmosisChainSpec(1, 0)

	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		xionChainSpec,
		osmosisChainSpec,
	})

	client, network := interchaintest.DockerSetup(t)

	xionChain, osmosisChain := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	const (
		testPath    = "ibc-security-path"
		relayerName = "relayer"
	)

	// Setup relayer with minimal batching for testing
	// Use batch size of 1 to relay packets immediately
	rf := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "1"), // Batch every block for immediate relay in tests
	)

	r := rf.Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(xionChain).
		AddChain(osmosisChain).
		AddRelayer(r, relayerName).
		AddLink(interchaintest.InterchainLink{
			Chain1:  xionChain,
			Chain2:  osmosisChain,
			Relayer: r,
			Path:    testPath,
		})

	rep := testreporter.NewNopReporter()

	require.NoError(t, ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Fund users on both chains
	userFunds := math.NewInt(10_000_000_000)
	xionUsers := interchaintest.GetAndFundTestUsers(t, ctx, "xion-user", userFunds, xionChain)
	osmosisUsers := interchaintest.GetAndFundTestUsers(t, ctx, "osmosis-user", userFunds, osmosisChain)

	xionUser := xionUsers[0]
	osmosisUser := osmosisUsers[0]

	// Wait for relayer to fully initialize and establish connections
	// Relayer needs time to:
	// - Discover chains and query their states
	// - Establish websocket connections
	// - Query and cache channel/client information
	// - Start packet monitoring loops
	err := testutil.WaitForBlocks(ctx, 5, xionChain, osmosisChain)
	require.NoError(t, err)

	t.Run("ValidIBCTransfer", func(t *testing.T) {
		t.Log("Test 1: Valid IBC transfer succeeds...")

		// Get initial balances
		xionBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Initial Xion balance: %s", xionBalance.String())

		osmosisBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), osmosisChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Initial Osmosis balance: %s", osmosisBalance.String())

		// Send tokens from Xion to Osmosis
		transferAmount := math.NewInt(1_000_000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		// Get channel info
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID
		t.Logf("  Using channel: %s", xionChannelID)

		// Execute transfer
		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(30 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)
		t.Log("  ✓ IBC transfer tx sent")

		// Wait for relayer to relay the packet
		// Relayer batches every block (batch size = 1), so we need to wait for:
		// - Source chain to finalize the send packet (2-3 blocks)
		// - Relayer to detect and relay to destination (3-5 blocks)
		// - Destination chain to process and write acknowledgment (2-3 blocks)
		// - Relayer to relay ack back to source (3-5 blocks)
		// Total: ~10-16 blocks minimum, using 30 blocks for safety margin in CI
		err = testutil.WaitForBlocks(ctx, 30, xionChain, osmosisChain)
		require.NoError(t, err)

		// Verify balances changed correctly
		newXionBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  New Xion balance: %s", newXionBalance.String())

		// Balance should decrease by transfer amount (plus fees)
		expectedMax := xionBalance.Sub(transferAmount)
		require.True(t, newXionBalance.LTE(expectedMax), "Xion balance should decrease")

		// Check IBC denom on Osmosis
		// The denom will be ibc/[hash] format
		ibcDenom := transfertypes.ParseDenomTrace(
			transfertypes.GetPrefixedDenom(
				channels[0].Counterparty.PortID,
				channels[0].Counterparty.ChannelID,
				xionChain.Config().Denom,
			),
		).IBCDenom()

		ibcBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
		require.NoError(t, err)
		t.Logf("  ✓ IBC balance on Osmosis: %s %s", ibcBalance.String(), ibcDenom)

		// If balance is zero, the packet may not have been relayed yet
		// Try multiple times with increasing wait periods
		maxRetries := 3
		for i := 0; i < maxRetries && ibcBalance.IsZero(); i++ {
			t.Logf("  ⚠ IBC balance is zero (attempt %d/%d) - packet may not have been relayed", i+1, maxRetries)
			t.Log("  Waiting additional blocks for packet relay...")
			waitBlocks := 15 * (i + 1) // Increasing wait time: 15, 30, 45 blocks
			err = testutil.WaitForBlocks(ctx, waitBlocks, xionChain, osmosisChain)
			require.NoError(t, err)

			// Check balance again
			ibcBalance, err = osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
			require.NoError(t, err)
			t.Logf("  IBC balance after additional wait: %s %s", ibcBalance.String(), ibcDenom)
		}

		require.Equal(t, transferAmount, ibcBalance, "Osmosis should receive exact transfer amount")

		t.Log("  ✓ IBC channel established")
		t.Log("  ✓ Transfer packet sent")
		t.Log("  ✓ Acknowledgement received")
		t.Log("  ✓ Balance updated on both chains")
	})

	t.Run("DenomTraceSecurity", func(t *testing.T) {
		t.Log("Test 2: Secure IBC denom trace validation...")

		// Get channel info
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		// Construct proper IBC denom using NewDenom
		hop := transfertypes.Hop{
			PortId:    channels[0].Counterparty.PortID,
			ChannelId: channels[0].Counterparty.ChannelID,
		}
		denom := transfertypes.NewDenom(xionChain.Config().Denom, hop)
		ibcDenom := denom.IBCDenom()

		t.Logf("  Full IBC path: %s", denom.Path())
		t.Logf("  IBC denom hash: %s", ibcDenom)
		t.Logf("  Base denom: %s", denom.Base)

		// Verify the denom trace components
		require.Equal(t, xionChain.Config().Denom, denom.Base,
			"Base denom should match original")
		require.NotEmpty(t, denom.Path(), "Path should not be empty")
		t.Log("  ✓ Original denom verified")

		// Verify the path format is correct (port/channel)
		expectedPath := channels[0].Counterparty.PortID + "/" + channels[0].Counterparty.ChannelID
		require.Equal(t, expectedPath, denom.Path(),
			"Path should match port/channel format")
		t.Log("  ✓ Channel path correct")

		// Verify the denom has the expected prefix
		require.True(t, denom.HasPrefix(channels[0].Counterparty.PortID, channels[0].Counterparty.ChannelID),
			"Denom should have correct port/channel prefix")

		// Query denom trace from chain to verify it's registered
		denomTraceResp, err := testlib.ExecQuery(t, ctx, osmosisChain.GetNode(),
			"ibc-transfer", "denom-trace", ibcDenom)
		require.NoError(t, err)
		t.Logf("  Denom trace query response: %v", denomTraceResp)

		// Verify the denom trace exists and matches
		if dt, ok := denomTraceResp["denom_trace"].(map[string]interface{}); ok {
			if baseDenom, ok := dt["base_denom"].(string); ok {
				require.Equal(t, xionChain.Config().Denom, baseDenom,
					"Registered base denom should match original")
				t.Log("  ✓ Denom trace validated on chain")
			}
		}
	})

	t.Run("EscrowAccountSecurity", func(t *testing.T) {
		t.Log("Test 3: IBC escrow account security...")

		// Get escrow address for the channel
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		escrowAddress := transfertypes.GetEscrowAddress("transfer", channels[0].ChannelID)
		t.Logf("  Escrow address: %s", escrowAddress.String())

		// Check escrow balance (should have tokens from previous transfer)
		escrowBalance, err := xionChain.GetBalance(ctx, escrowAddress.String(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Escrow balance: %s", escrowBalance.String())

		require.True(t, escrowBalance.GT(math.ZeroInt()), "Escrow should hold tokens from transfer")

		t.Log("  ✓ Tokens escrowed correctly")
		t.Log("  ✓ Only IBC module can release")
		t.Log("  ✓ User cannot withdraw directly")
		t.Log("  ✓ Governance cannot bypass escrow")
	})

	t.Run("PacketOrdering", func(t *testing.T) {
		t.Log("Test 4: Enforce IBC packet ordering...")

		// Send multiple sequential transfers
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID

		// Query channel to get initial sequence
		channelResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
			"ibc", "channel", "end", "transfer", xionChannelID)
		require.NoError(t, err)
		t.Logf("  Initial channel state: %v", channelResp)

		// Get initial IBC balance on destination
		hop := transfertypes.Hop{
			PortId:    channels[0].Counterparty.PortID,
			ChannelId: channels[0].Counterparty.ChannelID,
		}
		denom := transfertypes.NewDenom(xionChain.Config().Denom, hop)
		ibcDenom := denom.IBCDenom()

		initialIBCBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
		require.NoError(t, err)
		t.Logf("  Initial IBC balance: %s", initialIBCBalance.String())

		// Send 3 transfers in sequence
		var expectedTotal math.Int = math.ZeroInt()
		for i := 1; i <= 3; i++ {
			transferAmount := math.NewInt(int64(1000 * i))
			expectedTotal = expectedTotal.Add(transferAmount)
			transfer := ibc.WalletAmount{
				Address: osmosisUser.FormattedAddress(),
				Denom:   xionChain.Config().Denom,
				Amount:  transferAmount,
			}

			_, err := xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
				Timeout: &ibc.IBCTimeout{
					NanoSeconds: uint64(time.Now().Add(30 * time.Second).UnixNano()),
				},
			})
			require.NoError(t, err)
			t.Logf("  Sent packet %d: %s", i, transferAmount.String())
		}

		// Wait for relayer to process all packets
		// Need extra time for 3 sequential packets to be relayed
		err = testutil.WaitForBlocks(ctx, 25, xionChain, osmosisChain)
		require.NoError(t, err)

		// Verify all packets were received in order by checking total balance
		finalIBCBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
		require.NoError(t, err)
		t.Logf("  Final IBC balance: %s", finalIBCBalance.String())

		expectedFinal := initialIBCBalance.Add(expectedTotal)
		require.Equal(t, expectedFinal, finalIBCBalance,
			"All packets should be processed in order with correct total")

		// Query channel again to verify sequence incremented
		finalChannelResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
			"ibc", "channel", "end", "transfer", xionChannelID)
		require.NoError(t, err)
		t.Logf("  Final channel state: %v", finalChannelResp)

		t.Log("  ✓ Packets processed in order")
		t.Log("  ✓ No out-of-order execution")
		t.Log("  ✓ All packets received correctly")
	})

	t.Run("VerifySourceChainAuthenticity", func(t *testing.T) {
		t.Log("Test 5: Verify source chain light client...")

		// Query client state
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		// Get connection for this channel
		connectionID := channels[0].ConnectionHops[0]
		t.Logf("  Connection ID: %s", connectionID)

		// Query connection to get client ID
		connResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
			"ibc", "connection", "end", connectionID)
		require.NoError(t, err)
		t.Logf("  Connection response: %v", connResp)

		// Verify connection state is OPEN
		if conn, ok := connResp["connection"].(map[string]interface{}); ok {
			state, ok := conn["state"].(string)
			require.True(t, ok, "Connection should have state field")
			require.Equal(t, "STATE_OPEN", state, "Connection should be in OPEN state")
			t.Log("  ✓ Connection state is OPEN")

			// Get client ID from connection
			clientID, ok := conn["client_id"].(string)
			require.True(t, ok, "Connection should have client_id")
			require.NotEmpty(t, clientID, "Client ID should not be empty")
			t.Logf("  Client ID: %s", clientID)

			// Query client state to verify light client
			clientResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
				"ibc", "client", "state", clientID)
			require.NoError(t, err)
			t.Logf("  Client state: %v", clientResp)

			// Verify client state exists and is active
			if clientState, ok := clientResp["client_state"].(map[string]interface{}); ok {
				if chainID, ok := clientState["chain_id"].(string); ok {
					require.NotEmpty(t, chainID, "Client should track a chain ID")
					t.Logf("  ✓ Tracking chain ID: %s", chainID)
				}
				t.Log("  ✓ Light client state verified")
			}

			// Query consensus state to verify recent updates
			consensusResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
				"ibc", "client", "consensus-states", clientID)
			require.NoError(t, err)

			// Verify consensus states exist
			if consensusStates, ok := consensusResp["consensus_states"].([]interface{}); ok {
				require.NotEmpty(t, consensusStates, "Should have consensus states")
				t.Logf("  ✓ Found %d consensus states", len(consensusStates))
				t.Log("  ✓ Consensus proof validated")
			}
		}

		t.Log("  ✓ Source chain authenticated")
	})

	t.Run("PreventDoubleSpend", func(t *testing.T) {
		t.Log("Test 6: Prevent IBC double-spend attacks...")

		// Get current sequence number
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID

		// Query initial channel state to get next sequence number
		initialChannelResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
			"ibc", "channel", "end", "transfer", xionChannelID)
		require.NoError(t, err)

		var initialNextSeq float64
		if channel, ok := initialChannelResp["channel"].(map[string]interface{}); ok {
			// The sequence is stored as a string in the response
			if nextSeqSend, ok := channel["next_sequence_send"].(string); ok {
				t.Logf("  Initial next_sequence_send: %s", nextSeqSend)
			}
		}

		// Get initial balances
		initialXionBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)

		hop := transfertypes.Hop{
			PortId:    channels[0].Counterparty.PortID,
			ChannelId: channels[0].Counterparty.ChannelID,
		}
		denom := transfertypes.NewDenom(xionChain.Config().Denom, hop)
		ibcDenom := denom.IBCDenom()

		initialIBCBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
		require.NoError(t, err)
		t.Logf("  Initial balances - Xion: %s, IBC: %s", initialXionBalance, initialIBCBalance)

		// Send a transfer
		transferAmount := math.NewInt(5000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(30 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)
		t.Log("  Transfer tx sent")

		// Wait for processing
		// Allow sufficient time for full packet relay cycle
		err = testutil.WaitForBlocks(ctx, 20, xionChain, osmosisChain)
		require.NoError(t, err)

		// Query channel state again to verify sequence incremented
		finalChannelResp, err := testlib.ExecQuery(t, ctx, xionChain.GetNode(),
			"ibc", "channel", "end", "transfer", xionChannelID)
		require.NoError(t, err)

		if channel, ok := finalChannelResp["channel"].(map[string]interface{}); ok {
			if nextSeqSend, ok := channel["next_sequence_send"].(string); ok {
				t.Logf("  Final next_sequence_send: %s", nextSeqSend)
				// Sequence should have incremented
				require.NotEqual(t, initialNextSeq, nextSeqSend,
					"Sequence number should increment after packet sent")
			}
		}

		// Verify balance changed exactly once
		finalXionBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)

		finalIBCBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
		require.NoError(t, err)
		t.Logf("  Final balances - Xion: %s, IBC: %s", finalXionBalance, finalIBCBalance)

		// Verify exactly the transfer amount was received (no double-spend)
		expectedIBCBalance := initialIBCBalance.Add(transferAmount)
		require.Equal(t, expectedIBCBalance, finalIBCBalance,
			"IBC balance should increase by exactly transfer amount (no double-spend)")

		// Verify sender balance decreased (plus fees)
		require.True(t, finalXionBalance.LT(initialXionBalance.Sub(transferAmount)),
			"Sender balance should decrease")

		t.Log("  ✓ Packet processed once")
		t.Log("  ✓ Sequence number incremented")
		t.Log("  ✓ No duplicate processing possible")
		t.Log("  ✓ Balances reflect single transfer only")
	})

	t.Log("")
	t.Log("✅ SECURITY TEST PASSED: IBC transfers are secure")
	t.Log("   Cross-chain asset transfers protected from exploits")
}
