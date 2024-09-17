package integration_tests

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

// TODO:
// param change test (in the upcoming interchain v8 upgrade)

func TestXionMinimumFeeDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.025uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}, {defaultMinGasPrices.String()}}))

	assertion := func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, xionUser ibc.Wallet, recipientAddress string, fundAmount math.Int) {
		// NOTE: Tx should be rejected inssufficient gas
		_, err := ExecTxWithGas(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"0.024uxion",
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.Error(t, err)

		// NOTE: Uses default Gas
		_, err = ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.NoError(t, err)
	}

	testMinimumFee(t, &td, assertion)
}

func TestXionMinimumFeeZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}, {defaultMinGasPrices.String()}}))

	assertion := func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, xionUser ibc.Wallet, recipientAddress string, fundAmount math.Int) {
		toSend := math.NewInt(100)

		_, err := ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", toSend.Int64(), xion.Config().Denom),
		)
		require.NoError(t, err)

		balance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, fundAmount.Sub(toSend), balance)

		balance, err = xion.GetBalance(ctx, recipientAddress, xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, toSend, balance)
	}

	testMinimumFee(t, &td, assertion)
}

func testMinimumFee(t *testing.T, td *TestData, assert assertionFn) {
	xion, ctx := td.xionChain, td.ctx

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	currentHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// step 1: send a xion message with default (0%) platform fee
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	assert(t, ctx, xion, xionUser, recipientKeyAddress, fundAmount)
}

type assertionFn func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, wallet ibc.Wallet, recipientAddress string, fundAmount math.Int)
