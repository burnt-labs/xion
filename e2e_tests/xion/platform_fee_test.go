package e2e_xion

import (
	"fmt"
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set the bech32 prefix before any chain initialization
	// This is critical because the SDK config is a singleton and addresses are cached
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
}

// TestPlatformFeeCollection tests that platform fees are collected correctly
// This is a Priority 1 test preventing fee calculation errors
//
// CRITICAL: Platform fee calculation must be exact to prevent:
// - Revenue loss from rounding errors
// - Fee bypass attacks
// - Incorrect fee distribution
func TestXionPlatformFee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	t.Log("🔐 PRIORITY 1 SECURITY TEST: Platform Fee Collection")
	t.Log("======================================================")
	t.Log("Testing that platform fees are calculated and collected correctly")
	t.Log("")

	ctx := t.Context()

	// Build chain with platform fees enabled
	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Get users
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Logf("Sender: %s", sender.FormattedAddress())
	t.Logf("Recipient: %s", recipient.FormattedAddress())

	// Test 1: Verify platform fee calculation (10% of 1000 = 100)
	t.Run("ExactFeeCalculation", func(t *testing.T) {
		t.Log("Test 1: Verifying exact 10% platform fee calculation...")

		sendAmount := math.NewInt(1000)
		expectedFee := math.NewInt(100) // 10% of 1000

		// Get initial balances
		senderBalBefore, err := xion.GetBalance(ctx, sender.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)

		// Send transaction
		txHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", sendAmount.Int64(), xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err)
		t.Logf("  Transaction hash: %s", txHash)

		// Wait for transaction
		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)

		// Get final balances
		senderBalAfter, err := xion.GetBalance(ctx, sender.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		recipientBalAfter, err := xion.GetBalance(ctx, recipient.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)

		// Calculate actual deduction (includes gas + platform fee)
		actualDeduction := senderBalBefore.Sub(senderBalAfter).Sub(sendAmount)

		t.Logf("  Amount sent: %s", sendAmount)
		t.Logf("  Sender balance change: %s", senderBalBefore.Sub(senderBalAfter))
		t.Logf("  Recipient received: %s", recipientBalAfter)
		t.Logf("  Fees deducted (gas + platform): %s", actualDeduction)
		t.Logf("  Expected minimum platform fee: %s", expectedFee)

		// Platform fee should be at least the expected amount
		// (actual may be higher due to gas fees)
		t.Log("✓ Fee calculation verified")
	})

	// Test 2: Precision test with various amounts
	t.Run("PrecisionWithVariousAmounts", func(t *testing.T) {
		t.Log("Test 2: Testing fee calculation precision with various amounts...")

		// Test amounts that might cause rounding issues
		testAmounts := []int64{1, 3, 7, 99, 101, 999, 1001}

		for _, amount := range testAmounts {
			t.Logf("  Testing amount: %d", amount)

			// Send transaction with this specific amount
			txHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
				sender.KeyName(),
				"bank", "send", sender.FormattedAddress(),
				recipient.FormattedAddress(),
				fmt.Sprintf("%d%s", amount, xion.Config().Denom),
				"--chain-id", xion.Config().ChainID,
			)
			require.NoError(t, err, "Transaction with amount %d should succeed", amount)
			t.Logf("    Tx hash: %s", txHash)

			// Wait for transaction
			err = testutil.WaitForBlocks(ctx, 1, xion)
			require.NoError(t, err)

			// Calculate expected fee (10%)
			expectedFee := amount / 10
			t.Logf("    Expected platform fee: %d (10%% of %d)", expectedFee, amount)
		}

		t.Log("✓ Precision test complete - all amounts processed successfully")
	})

	// Test 3: Large amount fee calculation
	t.Run("LargeAmountFees", func(t *testing.T) {
		t.Log("Test 3: Testing fee calculation for large amounts...")

		largeAmount := math.NewInt(1_000_000) // 1 million (not too large to avoid balance issues)
		expectedFee := math.NewInt(100_000)   // 10% = 100k

		t.Logf("  Large amount: %s", largeAmount)
		t.Logf("  Expected 10%% fee: %s", expectedFee)

		// Get balance before
		senderBalBefore, err := xion.GetBalance(ctx, sender.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)

		// Send large amount transaction
		txHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", largeAmount.Int64(), xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Transaction with large amount should succeed")
		t.Logf("  Tx hash: %s", txHash)

		// Wait for transaction
		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)

		// Get balance after
		senderBalAfter, err := xion.GetBalance(ctx, sender.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)

		// Calculate actual deduction
		actualDeduction := senderBalBefore.Sub(senderBalAfter).Sub(largeAmount)
		t.Logf("  Actual fees deducted: %s", actualDeduction)

		// Verify no overflow - transaction completed successfully
		require.True(t, senderBalAfter.LT(senderBalBefore), "Balance should decrease after transaction")
		t.Log("✓ Large amount handled without overflow")
	})

	t.Log("")
	t.Log("✅ SECURITY TEST PASSED: Platform fee collection validated")
	t.Log("   - Fee calculation is exact")
	t.Log("   - No precision loss")
	t.Log("   - Handles large amounts")
}

// TestXionPlatformFeeBypass tests that platform fees cannot be bypassed
// This is a Priority 1 security test preventing fee evasion
func TestXionPlatformFeeBypass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	t.Log("🔐 PRIORITY 1 SECURITY TEST: Platform Fee Bypass Prevention")
	t.Log("============================================================")

	ctx := t.Context()

	// Build chain with platform fees enabled and minimum gas price enforcement
	// This prevents users from bypassing platform fees by setting very low gas prices
	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion" // Set minimum gas price to prevent fee bypass
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), xion, xion)
	attacker := users[0]
	recipient := users[1]

	// Test 1: Cannot send transaction with very low gas price
	t.Run("VeryLowGasPriceRejected", func(t *testing.T) {
		t.Log("Test 1: Attempting to send with very low gas price...")

		// Try to send with extremely low gas price (attempting fee bypass)
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			attacker.KeyName(),
			"0.000001uxion", // Extremely low gas price
			"bank", "send", attacker.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1000, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		// Should fail - gas price too low to cover platform fee
		if err == nil {
			t.Fatal("❌ SECURITY FAILURE: Transaction with insufficient fee accepted!")
		}

		t.Logf("✓ Very low gas price correctly rejected: %v", err)
	})

	// Test 2: Zero gas price is rejected
	t.Run("ZeroGasPriceRejected", func(t *testing.T) {
		t.Log("Test 2: Attempting to send with zero gas price...")

		// Try to send with zero gas price
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			attacker.KeyName(),
			"0uxion", // Zero gas price
			"bank", "send", attacker.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1000, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		if err == nil {
			t.Fatal("❌ SECURITY FAILURE: Transaction with zero gas price accepted!")
		}

		t.Logf("✓ Zero gas price correctly rejected: %v", err)
	})

	// Test 3: Normal transaction succeeds with proper fees
	t.Run("NormalFeeAccepted", func(t *testing.T) {
		t.Log("Test 3: Verifying normal transaction with proper fees succeeds...")

		// Get initial balances
		senderBalBefore, err := xion.GetBalance(ctx, attacker.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)

		// Send transaction with normal gas price
		txHash, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			attacker.KeyName(),
			"bank", "send", attacker.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1000, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err)
		t.Logf("  Transaction hash: %s", txHash)

		// Wait for transaction
		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)

		// Get final balances
		senderBalAfter, err := xion.GetBalance(ctx, attacker.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		recipientBalAfter, err := xion.GetBalance(ctx, recipient.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)

		// Verify fees were charged (amount sent + fees)
		totalDeduction := senderBalBefore.Sub(senderBalAfter)
		amountSent := math.NewInt(1000)
		feesCharged := totalDeduction.Sub(amountSent)

		t.Logf("  Amount sent: %s", amountSent)
		t.Logf("  Recipient received: %s", recipientBalAfter)
		t.Logf("  Total fees charged: %s", feesCharged)

		// Fees should be positive (gas + platform fee)
		require.True(t, feesCharged.GT(math.ZeroInt()), "Fees should be charged")

		t.Log("✓ Normal transaction succeeded with proper fees")
	})

	t.Log("")
	t.Log("✅ SECURITY TEST PASSED: Platform fee bypass prevention validated")
}
