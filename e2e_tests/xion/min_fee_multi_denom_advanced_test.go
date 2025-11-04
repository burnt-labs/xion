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

// TestXionMinFeeMultiDenomAdvanced tests multi-denomination fee scenarios
func TestXionMinFeeMultiDenomAdvanced(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	ctx := context.Background()

	chainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	chainSpec.GasPrices = "0.025uxion"
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, xion, xion)
	sender := users[0]
	recipient := users[1]

	t.Run("AcceptsPrimaryDenomination", func(t *testing.T) {
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"100uxion",
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 1000, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Transaction with primary denom should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})
}

// TestXionMinFeeExtremeValues tests boundary conditions
func TestXionMinFeeExtremeValues(t *testing.T) {
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

	t.Run("AcceptsHighFeeValue", func(t *testing.T) {
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"1000000uxion",
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Transaction with high fee should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})

	t.Run("AcceptsExactMinimumFee", func(t *testing.T) {
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.025uxion",
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Transaction with exact minimum should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})

	t.Run("RejectsBelowMinimumFee", func(t *testing.T) {
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.024uxion",
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.Error(t, err, "Transaction below minimum should fail")
	})

	t.Run("AcceptsJustAboveMinimumFee", func(t *testing.T) {
		_, err := testlib.ExecTxWithGas(t, ctx, xion.GetNode(),
			sender.KeyName(),
			"0.026uxion",
			"bank", "send", sender.FormattedAddress(),
			recipient.FormattedAddress(),
			fmt.Sprintf("%d%s", 100, xion.Config().Denom),
			"--chain-id", xion.Config().ChainID,
		)
		require.NoError(t, err, "Transaction above minimum should succeed")

		err = testutil.WaitForBlocks(ctx, 2, xion)
		require.NoError(t, err)
	})
}
