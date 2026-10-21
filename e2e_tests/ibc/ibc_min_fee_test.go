package e2e_ibc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/relayer"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	globalfeetypes "github.com/burnt-labs/xion/x/globalfee/types"
)

// TestIBCMinFeeMultiDenom tests that IBC-transferred tokens can be used for gas fees
// after being added to the global fee whitelist via governance
func TestIBCMinFeeMultiDenom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		testlib.XionLocalChainSpec(t, 1, 0),
		testlib.OsmosisChainSpec(1, 0),
	})

	client, network := interchaintest.DockerSetup(t)

	chain, counterpartyChain := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	const (
		testPath    = "ibc-upgrade-test-testPath"
		relayerName = "relayer"
	)

	// Get a relayer instance
	rf := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "100"),
	)

	r := rf.Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(chain).
		AddChain(counterpartyChain).
		AddRelayer(r, relayerName).
		AddLink(interchaintest.InterchainLink{
			Chain1:  chain,
			Chain2:  counterpartyChain,
			Relayer: r,
			Path:    testPath,
		})

	ctx := t.Context()

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	require.NoError(t, ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, chain, counterpartyChain)
	usersB := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, counterpartyChain)

	xionUser := users[0]
	osmoUser := usersB[0]
	currentHeight, _ := chain.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, chain)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := chain.GetBalance(ctx, xionUser.FormattedAddress(), chain.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, userFunds, xionUserBalInitial)

	// Step 2 send funds from chain B to Chain A
	xionChannelInfo, err := r.GetChannels(ctx, eRep, chain.Config().ChainID)
	require.NoError(t, err)
	xionChannelID := xionChannelInfo[0].ChannelID

	osmoUserBalInitial, err := counterpartyChain.GetBalance(ctx, osmoUser.FormattedAddress(), counterpartyChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, osmoUserBalInitial.Equal(userFunds))
	amount := math.NewInt(1_000_000)

	transfer := ibc.WalletAmount{
		Address: xionUser.FormattedAddress(),
		Denom:   counterpartyChain.Config().Denom,
		Amount:  amount,
	}

	tx, err := counterpartyChain.SendIBCTransfer(ctx, xionChannelID, osmoUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Validate())
	require.NoError(t, r.Flush(ctx, eRep, testPath, xionChannelID))

	// Trace IBC Denom
	srcDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", xionChannelID, counterpartyChain.Config().Denom))
	dstIbcDenom := srcDenomTrace.IBCDenom()

	// Test destination wallet has increased funds
	expectedBal := osmoUserBalInitial.Sub(amount)
	xionUserBalNew, err := chain.GetBalance(ctx, xionUser.FormattedAddress(), dstIbcDenom)
	t.Logf("querying address %s for denom: %s", xionUser.FormattedAddress(), dstIbcDenom)

	require.True(t, xionUserBalNew.Equal(amount), "got: %d, wanted: %d", xionUserBalNew, expectedBal)

	// step 3: upgrade minimum through governance
	err = testlib.SubmitAndPassProposal(t, ctx, chain, xionUser, []cosmos.ProtoMessage{
		&globalfeetypes.MsgUpdateParams{
			Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			Params: globalfeetypes.Params{
				MinimumGasPrices: types.NewDecCoins(
					types.NewDecCoinFromDec(dstIbcDenom, math.LegacyMustNewDecFromStr("0.024")),
					types.NewDecCoinFromDec("uxion", math.LegacyMustNewDecFromStr("0.025")),
				),
				BypassMinFeeMsgTypes:            globalfeetypes.DefaultBypassMinFeeMsgTypes,
				MaxTotalBypassMinFeeMsgGasUsage: globalfeetypes.DefaultmaxTotalBypassMinFeeMsgGasUsage,
			},
		},
	}, "add token to globalfee", ".", "", 1)
	require.NoError(t, err)

	// Wait for a few blocks to ensure the proposal changes take effect
	currentHeight, err = chain.Height(ctx)
	require.NoError(t, err)
	testutil.WaitForBlocks(ctx, int(currentHeight)+3, chain)

	recipientKeyName := "recipient-key"
	err = chain.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := chain.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(chain.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	// Test: paying with uxion should fail after adding IBC denom to global fee
	_, err = testlib.ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		"0.024uxion",
		"xion", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.Error(t, err)

	// Test: paying with IBC denom should succeed
	_, err = testlib.ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		fmt.Sprintf("0.024%s", dstIbcDenom),
		"bank", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.NoError(t, err)

	// Test: paying with native denom should also succeed (at higher amount)
	_, err = testlib.ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		fmt.Sprintf("0.025%s", chain.Config().Denom),
		"bank", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.NoError(t, err)
}
