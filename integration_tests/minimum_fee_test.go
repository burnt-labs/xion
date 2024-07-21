package integration_tests

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	"testing"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
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
		_, err := ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.NoError(t, err)

		balance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, fundAmount.Sub(math.NewInt(2415)), balance)

		balance, err = xion.GetBalance(ctx, recipientAddress, xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, math.NewInt(100), balance)
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
			recipientAddress, fmt.Sprintf("%d%s", toSend, xion.Config().Denom),
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

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
		&xiontypes.MsgSend{},
	)

	assert(t, ctx, xion, xionUser, recipientKeyAddress, fundAmount)
}

type assertionFn func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, wallet ibc.Wallet, recipientAddress string, fundAmount math.Int)
