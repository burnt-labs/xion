package integration_tests

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"cosmossdk.io/x/upgrade"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/burnt-labs/xion/x/jwk"
	"github.com/burnt-labs/xion/x/mint"
	"github.com/burnt-labs/xion/x/xion"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/ibc-go/modules/capability"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibccore "github.com/cosmos/ibc-go/v8/modules/core"
	ibcsolomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibclocalhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
	ccvprovider "github.com/cosmos/interchain-security/v5/x/ccv/provider"
	aa "github.com/larry0x/abstract-account/x/abstractaccount"
	"github.com/strangelove-ventures/tokenfactory/x/tokenfactory"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/relayer"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
)

func TestIBCTrasnferTest(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	ctx := context.Background()
	// Create chain factory with Feeabs and Gaia
	numVals := 1
	numFullNodes := 1

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name: "feeabs",
			ChainConfig: ibc.ChainConfig{
				Type:    "cosmos",
				Name:    "feeabs",
				ChainID: "feeabs-2",
				Images: []ibc.DockerImage{
					{
						Repository: FeeabsICTestRepo,
						Version:    "latest",
						UidGid:     "1025:1025",
					},
				},
				Bin:                 "feeappd",
				Bech32Prefix:        "feeabs",
				Denom:               "stake",
				CoinType:            "118",
				GasPrices:           "0.005stake",
				GasAdjustment:       1.5,
				TrustingPeriod:      "112h",
				NoHostMount:         false,
				ModifyGenesis:       modifyGenesisShortProposals(votingPeriod, maxDepositPeriod, queryEpochTime),
				ConfigFileOverrides: nil,
				EncodingConfig:      feeabsEncoding(),
			},
			NumValidators: &numVals,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "xion",
			Version: xionVersionFrom,
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: xionImageFrom,
						Version:    xionVersionFrom,
						UidGid:     "1025:1025",
					},
				},
				GasPrices:      "0.0uxion",
				GasAdjustment:  1.3,
				Type:           "cosmos",
				ChainID:        "xion-1",
				Bin:            "xiond",
				Bech32Prefix:   "xion",
				Denom:          "uxion",
				TrustingPeriod: ibcClientTrustingPeriod,
				EncodingConfig: func() *moduletestutil.TestEncodingConfig {
					cfg := moduletestutil.MakeTestEncodingConfig(
						auth.AppModuleBasic{},
						genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
						bank.AppModuleBasic{},
						capability.AppModuleBasic{},
						staking.AppModuleBasic{},
						mint.AppModuleBasic{},
						distr.AppModuleBasic{},
						gov.NewAppModuleBasic(
							[]govclient.ProposalHandler{
								paramsclient.ProposalHandler,
							},
						),
						params.AppModuleBasic{},
						slashing.AppModuleBasic{},
						upgrade.AppModuleBasic{},
						consensus.AppModuleBasic{},
						transfer.AppModuleBasic{},
						ibccore.AppModuleBasic{},
						ibctm.AppModuleBasic{},
						ibcwasm.AppModuleBasic{},
						ccvprovider.AppModuleBasic{},
						ibcsolomachine.AppModuleBasic{},

						// custom
						wasm.AppModuleBasic{},
						authz.AppModuleBasic{},
						tokenfactory.AppModuleBasic{},
						xion.AppModuleBasic{},
						jwk.AppModuleBasic{},
						aa.AppModuleBasic{},
					)
					// TODO: add encoding types here for the modules you want to use
					ibclocalhost.RegisterInterfaces(cfg.InterfaceRegistry)
					return &cfg
				}(),

				ModifyGenesis: ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}),
			},
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	feeabs, gaia := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	// Create relayer factory to utilize the go-relayer
	client, network := interchaintest.DockerSetup(t)
	r := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.CustomDockerImage(IBCRelayerImage, IBCRelayerVersion, "100:1000"), // TODO: ??
	).Build(t, client, network)

	// Create a new Interchain object which describes the chains, relayers, and IBC connections we want to use
	ic := interchaintest.NewInterchain().
		AddChain(feeabs).
		AddChain(gaia).
		AddRelayer(r, "rly").
		AddLink(interchaintest.InterchainLink{
			Chain1:  feeabs,
			Chain2:  gaia,
			Relayer: r,
			Path:    pathFeeabsXion,
		})

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,

		// This can be used to write to the block database which will index all block data e.g. txs, msgs, events, etc.
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	// Start the relayer
	require.NoError(t, r.StartRelayer(ctx, eRep, pathFeeabsXion))
	t.Cleanup(
		func() {
			err := r.StopRelayer(ctx, eRep)
			if err != nil {
				panic(fmt.Errorf("an error occurred while stopping the relayer: %s", err))
			}
		},
	)

	userFunds := math.NewInt(10_000_000_000)
	amountToSend := math.NewInt(1_000_000_000)

	// Create some user accounts on both chains
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, feeabs, gaia)

	// Wait a few blocks for relayer to start and for user accounts to be created
	err = testutil.WaitForBlocks(ctx, 5, feeabs, gaia)
	require.NoError(t, err)

	// Get our Bech32 encoded user addresses
	feeabsUser, gaiaUser := users[0], users[1]

	feeabsUserAddr := sdk.MustBech32ifyAddressBytes(feeabs.Config().Bech32Prefix, feeabsUser.Address())
	gaiaUserAddr := sdk.MustBech32ifyAddressBytes(gaia.Config().Bech32Prefix, gaiaUser.Address())

	// Get original account balances
	feeabsOrigBal, err := feeabs.GetBalance(ctx, feeabsUserAddr, feeabs.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, userFunds, feeabsOrigBal)

	gaiaOrigBal, err := gaia.GetBalance(ctx, gaiaUserAddr, gaia.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, userFunds, gaiaOrigBal)

	// Compose an IBC transfer and send from feeabs -> Gaia
	transfer := ibc.WalletAmount{
		Address: gaiaUserAddr,
		Denom:   feeabs.Config().Denom,
		Amount:  amountToSend,
	}

	channel, err := ibc.GetTransferChannel(ctx, r, eRep, feeabs.Config().ChainID, gaia.Config().ChainID)
	require.NoError(t, err)

	transferTx, err := feeabs.SendIBCTransfer(ctx, channel.ChannelID, feeabsUserAddr, transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	feeabsHeight, err := feeabs.Height(ctx)
	require.NoError(t, err)

	// Poll for the ack to know the transfer was successful
	_, err = testutil.PollForAck(ctx, feeabs, feeabsHeight, feeabsHeight+10, transferTx.Packet)
	require.NoError(t, err)

	// Get the IBC denom for stake on Gaia
	feeabsTokenDenom := transfertypes.GetPrefixedDenom(channel.Counterparty.PortID, channel.Counterparty.ChannelID, feeabs.Config().Denom)
	feeabsIBCDenom := transfertypes.ParseDenomTrace(feeabsTokenDenom).IBCDenom()

	// Assert that the funds are no longer present in user acc on feeabs and are in the user acc on Gaia
	feeabsUpdateBal, err := feeabs.GetBalance(ctx, feeabsUserAddr, feeabs.Config().Denom)
	require.NoError(t, err)

	// The feeabs account should have the original balance minus the transfer amount and the fee
	require.GreaterOrEqual(t, feeabsOrigBal.Sub(amountToSend).Int64(), feeabsUpdateBal.Int64())

	gaiaUpdateBal, err := gaia.GetBalance(ctx, gaiaUserAddr, feeabsIBCDenom)
	require.NoError(t, err)
	require.Equal(t, amountToSend, gaiaUpdateBal)

	// Compose an IBC transfer and send from Gaia -> Feeabs
	transfer = ibc.WalletAmount{
		Address: feeabsUserAddr,
		Denom:   feeabsIBCDenom,
		Amount:  amountToSend,
	}

	transferTx, err = gaia.SendIBCTransfer(ctx, channel.Counterparty.ChannelID, gaiaUserAddr, transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)

	// Poll for the ack to know the transfer was successful
	_, err = testutil.PollForAck(ctx, gaia, gaiaHeight, gaiaHeight+10, transferTx.Packet)
	require.NoError(t, err)

	// Assert that the funds are now back on feeabs and not on Gaia
	feeabsBalAfterGettingBackToken, err := feeabs.GetBalance(ctx, feeabsUserAddr, feeabs.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, feeabsUpdateBal.Add(amountToSend).Int64(), feeabsBalAfterGettingBackToken.Int64())

	gaiaUpdateBal, err = gaia.GetBalance(ctx, gaiaUserAddr, feeabsIBCDenom)
	require.NoError(t, err)
	require.Equal(t, math.ZeroInt().Int64(), gaiaUpdateBal.Int64())
}
