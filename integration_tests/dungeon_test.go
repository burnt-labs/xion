package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestDungeonTransferBlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	ctx := context.Background()

	var numFullNodes = 1
	var numValidators = 3

	// pulling image from env to foster local dev
	imageTag := os.Getenv("XION_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// disabling seeds in osmosis because it causes intermittent test failures
	osmoConfigFileOverrides := make(map[string]any)
	osmoConfigTomlOverrides := make(testutil.Toml)

	osmoP2POverrides := make(testutil.Toml)
	osmoP2POverrides["seeds"] = ""
	osmoConfigTomlOverrides["p2p"] = osmoP2POverrides

	osmoConfigFileOverrides["config/config.toml"] = osmoConfigTomlOverrides

	// Chain factory
	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		{
			Name:    "osmosis",
			Version: "v14.0.0",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/strangelove-ventures/heighliner/osmosis",
						Version:    "v14.0.0",
						UidGid:     "1025:1025",
					},
				},
				Type:                "cosmos",
				Bin:                 "osmosisd",
				Bech32Prefix:        "osmo",
				Denom:               "uosmo",
				GasPrices:           "0.0uosmo",
				GasAdjustment:       1.3,
				TrustingPeriod:      "336h",
				NoHostMount:         false,
				ConfigFileOverrides: osmoConfigFileOverrides,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    imageTagComponents[0],
			Version: imageTagComponents[1],
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: imageTagComponents[0],
						Version:    imageTagComponents[1],
						UidGid:     "1025:1025",
					},
				},
				GasPrices:              "0.0uxion",
				GasAdjustment:          1.3,
				Type:                   "cosmos",
				ChainID:                "xion-1",
				Bin:                    "xiond",
				Bech32Prefix:           "xion",
				Denom:                  "uxion",
				TrustingPeriod:         "336h",
				ModifyGenesis:          ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}),
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	osmosis, xion := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	// Relayer Factory
	client, network := ibctest.DockerSetup(t)
	relayer := ibctest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t)).Build(
		t, client, network)

	// Prep Interchain
	const ibcPath = "xion-osmo-dungeon-test"
	ic := ibctest.NewInterchain().
		AddChain(xion).
		AddChain(osmosis).
		AddRelayer(relayer, "relayer").
		AddLink(ibctest.InterchainLink{
			Chain1:  xion,
			Chain2:  osmosis,
			Relayer: relayer,
			Path:    ibcPath,
		})

	// Log location
	f, err := ibctest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: ibctest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, osmosis)
	xionUser := users[0]
	osmosisUser := users[1]
	t.Logf("created xion user %s", xionUser.FormattedAddress())
	t.Logf("created osmosis user %s", osmosisUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	// Get Channel ID
	t.Log("getting IBC channel IDs")
	xionChannelInfo, err := relayer.GetChannels(ctx, eRep, xion.Config().ChainID)
	require.NoError(t, err)
	xionChannelID := xionChannelInfo[0].ChannelID

	osmoChannelInfo, err := relayer.GetChannels(ctx, eRep, osmosis.Config().ChainID)
	require.NoError(t, err)
	osmoChannelID := osmoChannelInfo[0].ChannelID

	// Query staking denom
	t.Log("verifying staking denom")
	grpcAddress := xion.GetHostGRPCAddress()
	conn, err := grpc.Dial(grpcAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	defer conn.Close()
	require.NoError(t, err)

	stakingQueryClient := stakingtypes.NewQueryClient(conn)
	paramsResponse, err := stakingQueryClient.Params(ctx, &stakingtypes.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, "uxion", paramsResponse.Params.BondDenom)

	// Disable sends of Xion staking token
	t.Log("disabling sendability of xion staking token")

	sendEnableds := []*banktypes.SendEnabled{
		{
			Denom:   "uxion",
			Enabled: false,
		},
	}

	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	setSendEnabledMsg := banktypes.MsgSetSendEnabled{
		Authority:   authtypes.NewModuleAddress("gov").String(),
		SendEnabled: sendEnableds,
	}

	registry := cdctypes.NewInterfaceRegistry()
	registry.RegisterImplementations(
		(*types.Msg)(nil),
		&banktypes.MsgSetSendEnabled{},
	)
	cdc := codec.NewProtoCodec(registry)

	msg, err := cdc.MarshalInterfaceJSON(&setSendEnabledMsg)

	prop := cosmos.Proposal{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Disable sendability of uxion",
		Summary:  "This proposal prevents uxion from being sent in the bank module",
	}
	paramChangeTx, err := xion.SubmitProposal(ctx, xionUser.KeyName(), &prop)
	require.NoError(t, err)
	t.Logf("Param change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusVotingPeriod {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status VOTING, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status VOTING after 11s")

	err = xion.VoteOnProposalAllValidators(ctx, paramChangeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusPassed {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status PASSED, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status PASSED after 11s")

	// Send Transaction
	t.Log("sending tokens from xion to osmosis")
	amountToSend := int64(1_000_000)
	dstAddress := osmosisUser.FormattedAddress()
	transfer := ibc.WalletAmount{
		Address: dstAddress,
		Denom:   xion.Config().Denom,
		Amount:  amountToSend,
	}
	_, err = xion.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{})
	require.Error(t, err)

	// relay packets and acknowledgments
	require.NoError(t, relayer.Flush(ctx, eRep, ibcPath, osmoChannelID))

	// test source wallet has decreased funds
	xionUserBalNew, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, xionUserBalInitial, xionUserBalNew)

	// Trace IBC Denom
	srcDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", xionChannelID, xion.Config().Denom))
	xionOnOsmoIbcDenom := srcDenomTrace.IBCDenom()

	// Test destination wallet has increased funds
	t.Log("verifying receipt of tokens on osmosis")
	osmosUserBalNew, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), xionOnOsmoIbcDenom)
	require.NoError(t, err)
	require.Equal(t, int64(0), osmosUserBalNew)

	// Create a user without any funds
	emptyKeyName := "xion-empty-key"
	err = xion.CreateKey(ctx, emptyKeyName)
	require.NoError(t, err)
	emptyKeyAddressBytes, err := xion.GetAddress(ctx, emptyKeyName)
	require.NoError(t, err)
	emptyKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, emptyKeyAddressBytes)
	require.NoError(t, err)

	transfer = ibc.WalletAmount{
		Address: emptyKeyAddress,
		Denom:   osmosis.Config().Denom,
		Amount:  int64(1_000_000),
	}
	_, err = osmosis.SendIBCTransfer(ctx, osmoChannelID, osmosisUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)

	// relay packets and acknowledgments
	require.NoError(t, relayer.Flush(ctx, eRep, ibcPath, osmoChannelID))

	osmoUserBalAfterIbcTransfer, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), osmosis.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(9_000_000), osmoUserBalAfterIbcTransfer)

	emptyUserBals, err := xion.AllBalances(ctx, emptyKeyAddress)
	require.NoError(t, err)
	require.Equal(t, 1, len(emptyUserBals))

	osmoDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", osmoChannelID, osmosis.Config().Denom))
	osmoOnXionIbcDenom := osmoDenomTrace.IBCDenom()

	coin := emptyUserBals[0]
	require.Equal(t, osmoOnXionIbcDenom, coin.Denom)
	require.Equal(t, int64(1_000_000), coin.Amount.Int64())

	require.NoError(t, xion.SendFunds(ctx, emptyKeyName, ibc.WalletAmount{
		Address: xionUser.FormattedAddress(),
		Denom:   osmoOnXionIbcDenom,
		Amount:  1_000_000,
	}))

	require.Eventually(t, func() bool {
		xionUserOsmoBal, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), osmoOnXionIbcDenom)
		return err == nil && int64(1_000_000) == xionUserOsmoBal
	}, 5*time.Second, 500*time.Millisecond)

	transfer = ibc.WalletAmount{
		Address: osmosisUser.FormattedAddress(),
		Denom:   osmoOnXionIbcDenom,
		Amount:  int64(1_000_000),
	}
	_, err = xion.SendIBCTransfer(ctx, xionChannelID, xionUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)
	require.NoError(t, relayer.Flush(ctx, eRep, ibcPath, xionChannelID))
	//require.Eventually(t, func() bool {
	//	err := relayer.FlushAcknowledgements(ctx, eRep, ibcPath, osmoChannelID)
	//	return err == nil
	//}, 5*time.Second, 500*time.Millisecond)

	osmoUserBalAfterIbcReturnTransfer, err := osmosis.GetBalance(ctx, osmosisUser.FormattedAddress(), osmosis.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(10_000_000), osmoUserBalAfterIbcReturnTransfer)
}
