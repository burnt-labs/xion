package e2e_ibc

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/math"
	transfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	"github.com/cosmos/interchaintest/v11"
	"github.com/cosmos/interchaintest/v11/chain/cosmos"
	"github.com/cosmos/interchaintest/v11/ibc"
	"github.com/cosmos/interchaintest/v11/relayer"
	"github.com/cosmos/interchaintest/v11/testreporter"
	"github.com/cosmos/interchaintest/v11/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
)

// forwardMemo is the packet-forward-middleware memo that asks the receiving
// chain to send the tokens on over the given channel.
type forwardMemo struct {
	Forward forwardMetadata `json:"forward"`
}

type forwardMetadata struct {
	Receiver string `json:"receiver"`
	Port     string `json:"port"`
	Channel  string `json:"channel"`
}

// TestIBCPacketForward sends tokens from one chain to another through Xion
// in a single transfer, using packet-forward-middleware on Xion: osmosis A
// -> xion -> osmosis B. It also checks that a forward Xion cannot deliver is
// refunded to the original sender instead of being stranded on Xion.
func TestIBCPacketForward(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	ctx := t.Context()

	xionChainSpec := testlib.XionLocalChainSpec(t, 1, 0)
	xionChainSpec.ChainConfig.ModifyGenesis = cosmos.ModifyGenesis(append(testlib.DefaultGenesisKVMods,
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices", []map[string]string{{"denom": "uxion", "amount": "0"}}),
	))

	osmosisASpec := testlib.OsmosisChainSpec(1, 0)
	osmosisBSpec := testlib.OsmosisChainSpec(1, 0)
	osmosisBSpec.ChainConfig.ChainID = "localosmo-2"

	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		osmosisASpec,
		xionChainSpec,
		osmosisBSpec,
	})
	osmosisA, xion, osmosisB := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain), chains[2].(*cosmos.CosmosChain)

	client, network := testlib.DockerSetup(t)

	const (
		pathAX      = "osmosis-a-xion"
		pathXB      = "xion-osmosis-b"
		relayerName = "relayer"
	)

	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "1"),
	).Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(osmosisA).
		AddChain(xion).
		AddChain(osmosisB).
		AddRelayer(r, relayerName).
		AddLink(interchaintest.InterchainLink{Chain1: osmosisA, Chain2: xion, Relayer: r, Path: pathAX}).
		AddLink(interchaintest.InterchainLink{Chain1: xion, Chain2: osmosisB, Relayer: r, Path: pathXB})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, osmosisA, xion, osmosisB)
	userA, userX, userB := users[0], users[1], users[2]

	require.NoError(t, r.StartRelayer(ctx, eRep, pathAX, pathXB))
	t.Cleanup(func() {
		_ = r.StopRelayer(context.Background(), eRep)
	})
	require.NoError(t, testutil.WaitForBlocks(ctx, 5, osmosisA, xion, osmosisB))

	chanAX, err := ibc.GetTransferChannel(ctx, r, eRep, osmosisA.Config().ChainID, xion.Config().ChainID)
	require.NoError(t, err)
	chanXB, err := ibc.GetTransferChannel(ctx, r, eRep, xion.Config().ChainID, osmosisB.Config().ChainID)
	require.NoError(t, err)

	// uosmo from A as it appears on Xion, and as it appears on B after the
	// hop through Xion.
	denomOnXion := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(
		chanAX.Counterparty.PortID, chanAX.Counterparty.ChannelID, osmosisA.Config().Denom,
	)).IBCDenom()
	denomOnB := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom(
		chanXB.Counterparty.PortID, chanXB.Counterparty.ChannelID,
		transfertypes.GetPrefixedDenom(chanAX.Counterparty.PortID, chanAX.Counterparty.ChannelID, osmosisA.Config().Denom),
	)).IBCDenom()

	sendThroughXion := func(t *testing.T, amount math.Int, forwardChannel string) {
		t.Helper()
		memo, err := json.Marshal(forwardMemo{Forward: forwardMetadata{
			Receiver: userB.FormattedAddress(),
			Port:     "transfer",
			Channel:  forwardChannel,
		}})
		require.NoError(t, err)

		_, err = osmosisA.SendIBCTransfer(ctx, chanAX.ChannelID, userA.KeyName(), ibc.WalletAmount{
			Address: userX.FormattedAddress(),
			Denom:   osmosisA.Config().Denom,
			Amount:  amount,
		}, ibc.TransferOptions{
			Memo:    string(memo),
			Timeout: &ibc.IBCTimeout{NanoSeconds: uint64(time.Now().Add(10 * time.Minute).UnixNano())},
		})
		require.NoError(t, err)
	}

	t.Run("forwards through xion to the next chain", func(t *testing.T) {
		amount := math.NewInt(1_000_000)
		sendThroughXion(t, amount, chanXB.ChannelID)

		got := waitForBalance(t, ctx, osmosisB, userB.FormattedAddress(), denomOnB, func(b math.Int) bool { return b.Equal(amount) }, osmosisA, xion, osmosisB)
		require.Equal(t, amount, got, "osmosis B receives the full forwarded amount as %s", denomOnB)

		onXion, err := xion.GetBalance(ctx, userX.FormattedAddress(), denomOnXion)
		require.NoError(t, err)
		require.True(t, onXion.IsZero(), "nothing stays with the intermediate receiver on xion, got %s", onXion)
	})

	t.Run("refunds the sender when xion cannot forward", func(t *testing.T) {
		before, err := osmosisA.GetBalance(ctx, userA.FormattedAddress(), osmosisA.Config().Denom)
		require.NoError(t, err)

		amount := math.NewInt(2_000_000)
		sendThroughXion(t, amount, "channel-999")

		// The send itself costs a fee, so the refund is complete once the
		// balance is back within the fee of where it started.
		got := waitForBalance(t, ctx, osmosisA, userA.FormattedAddress(), osmosisA.Config().Denom, func(b math.Int) bool {
			return before.Sub(b).LT(amount)
		}, osmosisA, xion, osmosisB)
		require.True(t, before.Sub(got).LT(math.NewInt(100_000)), "sender refunded: before %s, after %s", before, got)

		onXion, err := xion.GetBalance(ctx, userX.FormattedAddress(), denomOnXion)
		require.NoError(t, err)
		require.True(t, onXion.IsZero(), "a failed forward leaves nothing on xion, got %s", onXion)

		onB, err := osmosisB.GetBalance(ctx, userB.FormattedAddress(), denomOnB)
		require.NoError(t, err)
		require.Equal(t, math.NewInt(1_000_000), onB, "osmosis B only holds the first transfer")
	})
}

// waitForBalance polls address's balance of denom until done accepts it,
// failing the test if that takes more than 60 blocks.
func waitForBalance(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, address, denom string, done func(math.Int) bool, chains ...testutil.ChainHeighter) math.Int {
	t.Helper()
	var balance math.Int
	for i := 0; i < 12; i++ {
		require.NoError(t, testutil.WaitForBlocks(ctx, 5, chains...))
		var err error
		balance, err = chain.GetBalance(ctx, address, denom)
		require.NoError(t, err)
		if done(balance) {
			return balance
		}
	}
	t.Fatalf("balance of %s %s on %s did not settle, last %s", address, denom, chain.Config().ChainID, balance)
	return balance
}
