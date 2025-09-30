package e2e_xion

import (
	"fmt"
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func init() {
	// Set the bech32 prefix before any chain initialization
	// This is critical because the SDK config is a singleton and addresses are cached
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
}

// TestMinimumFeeBypassPrevention tests that minimum fee requirements cannot be bypassed
// This is a Priority 1 security test preventing fee evasion
func TestXionMinFeeBypass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Log("🔐 PRIORITY 1 SECURITY TEST: Minimum Fee Bypass Prevention")
	t.Log("===========================================================")

	t.Parallel()
	ctx := t.Context()

	// Build chain with minimum fee set
	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion" // Minimum 0.025 per gas unit
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("BelowMinimumFeeRejected", func(t *testing.T) {
		t.Log("Test 1: Transaction with fee below minimum should be rejected...")

		// Attempt to send with fee below minimum (0.024 < 0.025)
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.024uxion", // Below minimum!
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		// Should fail
		if err == nil {
			t.Fatal("❌ SECURITY FAILURE: Below-minimum fee transaction accepted!")
		}

		t.Logf("✓ Below-minimum fee correctly rejected: %v", err)
	})

	t.Run("ZeroFeeRejected", func(t *testing.T) {
		t.Log("Test 2: Transaction with zero fee should be rejected...")

		// Attempt with zero fee
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0uxion", // Zero fee!
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		if err == nil {
			t.Fatal("❌ SECURITY FAILURE: Zero fee transaction accepted!")
		}

		t.Logf("✓ Zero fee correctly rejected: %v", err)
	})

	t.Run("CorrectDenominationAccepted", func(t *testing.T) {
		t.Log("Test 3: Transaction with correct denomination should be accepted...")

		// Verify that the correct denomination (uxion) is accepted
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"100uxion", // Correct denomination with sufficient amount
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.NoError(t, err, "Transaction with correct denomination should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)

		t.Logf("✓ Correct denomination (uxion) accepted")
	})

	t.Run("ExactMinimumFeeAccepted", func(t *testing.T) {
		t.Log("Test 4: Transaction with exactly minimum fee should be accepted...")

		// Send with exact minimum fee
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.025uxion", // Exactly minimum
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.NoError(t, err, "Transaction with exact minimum fee should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)

		t.Log("✓ Exact minimum fee accepted")
	})

	t.Log("")
	t.Log("✅ SECURITY TEST PASSED: Minimum fee bypass prevention validated")
}
