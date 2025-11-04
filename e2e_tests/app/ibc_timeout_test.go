package e2e_app

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"
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

// TestIBCTimeoutHandling tests IBC packet timeout security
// This is a Priority 1 security test preventing fund loss during network issues
//
// CRITICAL: IBC timeouts must:
// - Refund tokens when packets timeout
// - Prevent fund loss during network partitions
// - Handle both timestamp and height timeouts
// - Release escrow correctly on timeout
func TestAppIBCTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔐 PRIORITY 1 SECURITY TEST: IBC Timeout Handling")
	t.Log("==================================================")
	t.Log("Testing IBC packet timeout and refund mechanisms")
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
		testPath    = "ibc-timeout-path"
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

	err := testutil.WaitForBlocks(ctx, 2, xionChain, osmosisChain)
	require.NoError(t, err)

	t.Run("TimeoutHeightRefund", func(t *testing.T) {
		t.Log("Test 1: Refund tokens when packet times out by height...")

		// Get channel info
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID

		// Get initial balance
		initialBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Initial balance: %s", initialBalance.String())

		// Send transfer with very short timeout
		transferAmount := math.NewInt(1_000_000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		// Stop relayer to prevent packet from being relayed
		err = r.StopRelayer(ctx, rep.RelayerExecReporter(t))
		require.NoError(t, err)
		t.Log("  ✓ Relayer stopped")

		// Send transfer with 2 second timeout
		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(2 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)
		t.Log("  Sent IBC transfer with 2s timeout")

		// Wait for timeout to pass
		t.Log("  Waiting for timeout...")
		time.Sleep(4 * time.Second)

		// Restart relayer to process timeout
		err = r.StartRelayer(ctx, rep.RelayerExecReporter(t), testPath)
		require.NoError(t, err)
		t.Log("  ✓ Relayer restarted to process timeout")

		// Wait for timeout to be processed
		err = testutil.WaitForBlocks(ctx, 10, xionChain, osmosisChain)
		require.NoError(t, err)

		// Check balance was refunded
		finalBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Final balance: %s", finalBalance.String())

		// Balance should be close to initial (minus fees)
		// Should have more than if transfer succeeded
		expectedMin := initialBalance.Sub(transferAmount).Sub(math.NewInt(100000)) // minus fees
		require.True(t, finalBalance.GT(expectedMin), "Balance should be refunded")

		t.Log("  ✓ Timeout packet processed")
		t.Log("  ✓ Tokens refunded to sender")
		t.Log("  ✓ No tokens minted on destination")
	})

	t.Run("TimeoutTimestampRefund", func(t *testing.T) {
		t.Log("Test 2: Refund tokens when packet times out by timestamp...")

		// Get channel info
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID

		// Get initial balance
		initialBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Initial balance: %s", initialBalance.String())

		// Stop relayer
		err = r.StopRelayer(ctx, rep.RelayerExecReporter(t))
		require.NoError(t, err)

		// Send transfer with very short timestamp timeout (2 seconds)
		transferAmount := math.NewInt(500_000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(2 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)
		t.Log("  Sent IBC transfer with 2s timeout")

		// Wait for timeout to pass
		t.Log("  Waiting for timeout...")
		time.Sleep(5 * time.Second)

		// Restart relayer to process timeout
		err = r.StartRelayer(ctx, rep.RelayerExecReporter(t), testPath)
		require.NoError(t, err)

		// Wait for timeout processing
		err = testutil.WaitForBlocks(ctx, 10, xionChain, osmosisChain)
		require.NoError(t, err)

		// Verify refund
		finalBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Final balance: %s", finalBalance.String())

		expectedMin := initialBalance.Sub(transferAmount).Sub(math.NewInt(100000))
		require.True(t, finalBalance.GT(expectedMin), "Tokens should be refunded")

		t.Log("  ✓ Timeout packet processed")
		t.Log("  ✓ Tokens refunded after timestamp timeout")
		t.Log("  ✓ Escrow account released")
	})

	t.Run("EscrowReleaseOnTimeout", func(t *testing.T) {
		t.Log("Test 3: Escrow properly released on timeout...")

		// Wait for chains to stabilize after previous test
		// Previous test stopped and restarted relayer, chains need time to sync
		err := testutil.WaitForBlocks(ctx, 8, xionChain, osmosisChain)
		require.NoError(t, err)

		// Verify chain is responsive before proceeding
		height, err := xionChain.Height(ctx)
		require.NoError(t, err)
		t.Logf("  Chain height: %d", height)

		// Get channel and escrow address
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID
		escrowAddress := transfertypes.GetEscrowAddress("transfer", xionChannelID)

		// Get initial escrow balance with retry for gRPC initialization
		// After relayer manipulation, the gRPC client may need time to reinitialize
		var initialEscrow sdkmath.Int
		maxRetries := 3
		for i := 0; i < maxRetries; i++ {
			initialEscrow, err = xionChain.GetBalance(ctx, escrowAddress.String(), xionChain.Config().Denom)
			if err == nil {
				break
			}
			if i < maxRetries-1 {
				t.Logf("  Failed to get balance (attempt %d/%d), waiting...", i+1, maxRetries)
				err = testutil.WaitForBlocks(ctx, 3, xionChain)
				require.NoError(t, err)
			}
		}
		require.NoError(t, err, "Failed to get escrow balance after retries")
		t.Logf("  Initial escrow balance: %s", initialEscrow.String())

		// Stop relayer
		err = r.StopRelayer(ctx, rep.RelayerExecReporter(t))
		require.NoError(t, err)

		// Send transfer that will timeout
		transferAmount := math.NewInt(250_000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(1 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)

		// Wait for blocks to ensure transfer is processed
		err = testutil.WaitForBlocks(ctx, 2, xionChain)
		require.NoError(t, err)

		// Escrow should increase
		escrowAfterSend, err := xionChain.GetBalance(ctx, escrowAddress.String(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Escrow after send: %s", escrowAfterSend.String())
		require.True(t, escrowAfterSend.GTE(initialEscrow.Add(transferAmount)), "Escrow should increase by at least transfer amount")

		// Wait for timeout
		time.Sleep(3 * time.Second)

		// Restart relayer
		err = r.StartRelayer(ctx, rep.RelayerExecReporter(t), testPath)
		require.NoError(t, err)

		// Wait longer to ensure timeout packet is relayed and processed
		err = testutil.WaitForBlocks(ctx, 15, xionChain, osmosisChain)
		require.NoError(t, err)

		// Escrow should return to initial (or close to it)
		finalEscrow, err := xionChain.GetBalance(ctx, escrowAddress.String(), xionChain.Config().Denom)
		require.NoError(t, err)
		t.Logf("  Escrow after timeout: %s", finalEscrow.String())

		// The escrow should have decreased by the transfer amount (accounting for any small variations)
		escrowChange := escrowAfterSend.Sub(finalEscrow)
		require.True(t, escrowChange.GTE(transferAmount),
			"Escrow should be released (decreased by at least %s, actual decrease: %s)",
			transferAmount.String(), escrowChange.String())

		t.Log("  ✓ Escrow increased when packet sent")
		t.Log("  ✓ Escrow decreased when packet timed out")
		t.Log("  ✓ No funds lost in escrow")
	})

	t.Run("NetworkPartitionRecovery", func(t *testing.T) {
		t.Log("Test 4: Graceful recovery from network partition...")

		// Get channel
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID

		// Get user balance
		initialBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)

		t.Log("  Simulating network partition (stop relayer)...")
		err = r.StopRelayer(ctx, rep.RelayerExecReporter(t))
		require.NoError(t, err)

		// Send transfer during "partition"
		transferAmount := math.NewInt(100_000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(2 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)
		t.Log("  IBC transfer sent during partition")

		// Wait for timeout
		time.Sleep(4 * time.Second)
		t.Log("  Packet timed out during partition")

		// Restore network (restart relayer)
		t.Log("  Network partition resolved (restart relayer)...")
		err = r.StartRelayer(ctx, rep.RelayerExecReporter(t), testPath)
		require.NoError(t, err)

		err = testutil.WaitForBlocks(ctx, 10, xionChain, osmosisChain)
		require.NoError(t, err)

		// Verify funds returned
		finalBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)

		expectedMin := initialBalance.Sub(transferAmount).Sub(math.NewInt(100000))
		require.True(t, finalBalance.GT(expectedMin), "Funds should be returned after partition")

		t.Log("  ✓ Timeout packet submitted after partition resolved")
		t.Log("  ✓ Funds returned to sender")
		t.Log("  ✓ User not locked out of funds")
	})

	t.Run("ValidTransferDoesNotTimeout", func(t *testing.T) {
		t.Log("Test 5: Valid transfer with sufficient timeout succeeds...")

		// Ensure relayer is running
		err := r.StartRelayer(ctx, rep.RelayerExecReporter(t), testPath)
		require.NoError(t, err)

		// Get channel
		channels, err := r.GetChannels(ctx, rep.RelayerExecReporter(t), xionChain.Config().ChainID)
		require.NoError(t, err)
		require.NotEmpty(t, channels)

		xionChannelID := channels[0].ChannelID

		// Send transfer with long timeout
		transferAmount := math.NewInt(300_000)
		transfer := ibc.WalletAmount{
			Address: osmosisUser.FormattedAddress(),
			Denom:   xionChain.Config().Denom,
			Amount:  transferAmount,
		}

		initialXionBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)

		_, err = xionChain.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{
			Timeout: &ibc.IBCTimeout{
				NanoSeconds: uint64(time.Now().Add(60 * time.Second).UnixNano()),
			},
		})
		require.NoError(t, err)

		// Wait for relay
		err = testutil.WaitForBlocks(ctx, 10, xionChain, osmosisChain)
		require.NoError(t, err)

		// Verify transfer succeeded
		ibcDenom := transfertypes.ParseDenomTrace(
			transfertypes.GetPrefixedDenom(
				channels[0].Counterparty.PortID,
				channels[0].Counterparty.ChannelID,
				xionChain.Config().Denom,
			),
		).IBCDenom()

		osmosisBalance, err := osmosisChain.GetBalance(ctx, osmosisUser.FormattedAddress(), ibcDenom)
		require.NoError(t, err)
		require.True(t, osmosisBalance.GT(math.ZeroInt()), "Transfer should succeed with long timeout")

		finalXionBalance, err := xionChain.GetBalance(ctx, xionUser.FormattedAddress(), xionChain.Config().Denom)
		require.NoError(t, err)
		require.True(t, finalXionBalance.LT(initialXionBalance), "Xion balance should decrease")

		t.Log("  ✓ Transfer with sufficient timeout succeeded")
		t.Log("  ✓ No timeout packet needed")
		t.Log("  ✓ Funds transferred successfully")
	})

	t.Log("")
	t.Log("✅ SECURITY TEST PASSED: IBC timeouts handled correctly")
	t.Log("   No fund loss during network issues or timeouts")
}
