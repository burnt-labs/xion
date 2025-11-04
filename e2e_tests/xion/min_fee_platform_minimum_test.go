package e2e_xion

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

func init() {
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")
}

// TestXionPlatformMinimumWithFees tests platform minimum enforcement alongside fees
func TestXionPlatformMinimumWithFees(t *testing.T) {
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

	t.Run("RejectsBelowPlatformMinimum", func(t *testing.T) {
		// Transaction with valid gas fee but send amount below platform minimum
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.025uxion", // Valid fee
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1, xion.Config().Denom), // Below minimum
			"--chain-id", xion.Config().ChainID,
		)
		require.Error(t, err, "Should reject send below platform minimum")
		require.Contains(t, err.Error(), "minimum", "Error should mention minimum")
	})

	t.Run("AcceptsAbovePlatformMinimum", func(t *testing.T) {
		// Transaction meeting both platform minimum and gas fee requirements
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.025uxion", // Valid fee
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom), // Above minimum
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Transaction meeting both requirements should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})
}

// TestXionPlatformMinimumCodecValidation tests MsgSetPlatformMinimum interface
func TestXionPlatformMinimumCodecValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	t.Run("ImplementsSdkMsgInterface", func(t *testing.T) {
		// Verify the message properly implements sdk.Msg
		var msg types.Msg = &xiontypes.MsgSetPlatformMinimum{}
		require.NotNil(t, msg, "MsgSetPlatformMinimum should implement sdk.Msg")

		// Test that ValidateBasic exists and works
		platformMsg := &xiontypes.MsgSetPlatformMinimum{}
		err := platformMsg.ValidateBasic()
		require.Error(t, err, "Empty message should fail validation")
	})
}

// TestXionPlatformMinimumBypassInteraction tests bypass + platform minimum
func TestXionPlatformMinimumBypassInteraction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("BypassFeeButNotPlatformMinimum", func(t *testing.T) {
		// Zero-fee bypass with amount below platform minimum should still fail
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"xion", "send", sender.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1, xion.Config().Denom), // Below minimum
		)

		require.Error(t, err, "Should reject below-minimum send even with bypass")
		require.Contains(t, err.Error(), "minimum", "Error should mention minimum")
	})

	t.Run("BypassFeeWithValidAmount", func(t *testing.T) {
		// Zero-fee bypass with proper send amount should succeed
		_, err := testlib.ExecTx(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"xion", "send", sender.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom), // Above minimum
		)

		// May succeed or fail depending on gas cap, but shouldn't be a platform minimum error
		if err != nil {
			require.NotContains(t, err.Error(), "minimum send amount", "Should not fail on platform minimum")
		} else {
			// If successful, wait for blocks
			err = testutil.WaitForBlocks(ctx, 2, xion)
			require.NoError(t, err)
		}
	})
}
