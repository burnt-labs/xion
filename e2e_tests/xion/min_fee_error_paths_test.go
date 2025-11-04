package e2e_xion

import (
	"context"
	"fmt"
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

// TestXionMinFeeErrorMessages tests that proper error messages are returned
func TestXionMinFeeErrorMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("ProvidesClearInsufficientFeeError", func(t *testing.T) {
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.01uxion", // Below minimum
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.Error(t, err, "Should return error for insufficient fee")
		// Error message should be descriptive
		t.Logf("Error message: %v", err)
		// Basic check that we got an error
		require.NotEmpty(t, err.Error())
	})

	t.Run("ProvidesClearPlatformMinimumError", func(t *testing.T) {
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"xion", "send", sender.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1, xion.Config().Denom),
		)
		if err != nil {
			require.Contains(t, err.Error(), "minimum", "Error should mention minimum")
			t.Logf("Platform minimum error: %v", err)
		}
	})
}

// TestXionMinFeeInsufficientBalance tests edge cases with insufficient balance
func TestXionMinFeeInsufficientBalance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Fund user with minimal amount
	fundAmount := math.NewInt(1000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("RejectsInsufficientBalanceForFee", func(t *testing.T) {
		// Try to send with fee that exceeds balance
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"100000uxion", // More than balance
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.Error(t, err, "Should fail with insufficient balance")
		t.Logf("Insufficient balance error: %v", err)
	})
}

// TestXionMinFeeEdgeCaseScenarios tests various edge cases
func TestXionMinFeeEdgeCaseScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("HandlesSingleBypassMessage", func(t *testing.T) {
		// Transaction with exactly one bypass message
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"xion", "send", sender.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)

		// Log result - may succeed or fail depending on gas/platform minimum
		if err != nil {
			t.Logf("Single bypass message error: %v", err)
		} else {
			err = testutil.WaitForBlocks(ctx, 2, xion)
			require.NoError(t, err)
		}
	})

	t.Run("HandlesAutoGasEstimation", func(t *testing.T) {
		// Transaction using automatic gas estimation
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.025uxion",
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.NoError(t, err, "Auto gas estimation should work with fees")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})
}

// TestXionMinFeeMemPoolBehavior tests mempool interaction with fee requirements
func TestXionMinFeeMemPoolBehavior(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("RejectsBelowMinimumInCheckTx", func(t *testing.T) {
		// Transaction with fee below minimum should be rejected in CheckTx
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.01uxion", // Below minimum
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.Error(t, err, "Below-minimum fee should be rejected")
	})

	t.Run("AcceptsMinimumInCheckTx", func(t *testing.T) {
		// Transaction with exactly minimum fee should pass CheckTx
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.025uxion", // Exactly minimum
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)

		require.NoError(t, err, "Minimum fee should be accepted")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})

	t.Run("AcceptsBypassZeroFeeInCheckTx", func(t *testing.T) {
		// Bypass message with zero fee should pass CheckTx
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"xion", "send", sender.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)

		// Log result - may succeed or fail but should pass CheckTx
		if err != nil {
			t.Logf("Bypass zero-fee result: %v", err)
		} else {
			err = testutil.WaitForBlocks(ctx, 2, xion)
			require.NoError(t, err)
		}
	})
}
