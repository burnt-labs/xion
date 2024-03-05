package integration_tests

import (
	"cosmossdk.io/math"
	"fmt"
	"testing"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestXionMinimumFee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
	xion, ctx := td.xionChain, td.ctx

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(fundAmount), xion)
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
		&xiontypes.MsgUpdateParams{},
		&xiontypes.MsgSend{},
	)

	currentHeight, _ = xion.Height(ctx)
	_, err = ExecTx(t, ctx, xion.FullNodes[0],
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
	)
	require.NoError(t, err)
	balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100), uint64(balance.Int64()))
}
