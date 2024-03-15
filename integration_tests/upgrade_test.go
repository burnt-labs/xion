package integration_tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	interchainRelayer "github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/relayer/rly"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	haltHeightDelta    = uint64(10) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
)

/*
* CONTRACT: This test requires manual setup before running
* 1- Git checkout to current version of the network
* 2- Build current using heighliner pass in the flag `-t current`. (for instructions on how to build check README.md on the root of the project)
* 3- Git checkout to upgrade-version version of the network
* 4- Build using heighliner pass in the flag `-t upgrade`. (for instructions on how to build check README.md on the root of the project)
* 5- Mark upgrade name as the last parameter of the function
* 6- cd integration_test
* 7- XION_IMAGE=[current version of the network] go test -run TestXionUpgradeIBC ./...

As of Aug 17 2023 this is the necessary process to run this test, this is due to the fact that AWS & docker-hub auto deleting old images, therefore you might lose what the version currently running is image wise
current-testnet: v0.3.4
step between: v0.3.5
upgrade-version: v0.3.6
*/
func TestXionUpgradeIBC(t *testing.T) {
	t.Skip("ComosChainUpgradeTest should be run manually, please comment skip and follow instructions when running")

	t.Parallel()
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	ctx := context.Background()

	var numFullNodes = 1
	var numValidators = 3

	// pulling image from env to foster local dev
	imageTag := os.Getenv("XION_IMAGE")
	println("image tag:", imageTag)
	imageTagComponents := strings.Split(imageTag, ":")

	// disabling seeds in osmosis because it causes intermittent test failures
	osmoConfigFileOverrides := make(map[string]any)
	osmoConfigTomlOverrides := make(testutil.Toml)

	osmoP2POverrides := make(testutil.Toml)
	osmoP2POverrides["seeds"] = ""
	osmoConfigTomlOverrides["p2p"] = osmoP2POverrides

	osmoConfigFileOverrides["config/config.toml"] = osmoConfigTomlOverrides

	// Chain factory
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
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
	client, network := interchaintest.DockerSetup(t)
	relayerImage := interchainRelayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main", rly.RlyDefaultUidGid)
	relayer := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t), relayerImage).Build(
		t, client, network)

	// Prep Interchain
	const ibcPath = "xion-osmo-dungeon-test"
	ic := interchaintest.NewInterchain().
		AddChain(xion).
		AddChain(osmosis).
		AddRelayer(relayer, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  xion,
			Chain2:  osmosis,
			Relayer: relayer,
			Path:    ibcPath,
		})

	// Log location
	f, err := interchaintest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, osmosis)
	xionUser := users[0]
	osmosisUser := users[1]
	t.Logf("created xion user %s", xionUser.FormattedAddress())
	t.Logf("created osmosis user %s", osmosisUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	//CosmosChainUpgradeIBCTest(t, &td, chainUser, "xion", "current", "xion", "upgrade", "v4")
	//chainName := "xion"
	//initialVersion := "current"
	upgradeContainerRepo := "xion"
	upgradeVersion := "upgrade"
	upgradeName := "v4"

	height, err := xion.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta - 3

	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     "500000000" + xion.Config().Denom, // greater than min deposit
		Title:       "Chain Upgrade 1",
		Name:        upgradeName,
		Description: "First chain software upgrade",
		Height:      haltHeight,
	}

	upgradeTx, err := xion.LegacyUpgradeProposal(ctx, xionUser.KeyName(), proposal)
	require.NoError(t, err, "error submitting software upgrade proposal tx")

	err = xion.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	_, err = cosmos.PollForProposalStatus(ctx, xion, height, height+haltHeightDelta, upgradeTx.ProposalID, cosmos.ProposalStatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = xion.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should time out due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height), xion)

	height, err = xion.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, fmt.Sprintf("height: %d is not equal to halt height: %d", height, haltHeight))

	// bring down nodes to prepare for upgrade
	err = xion.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	xion.UpgradeVersion(ctx, client, upgradeContainerRepo, upgradeVersion)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = xion.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), xion)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// after the chain is back up and producing blocks, check that the IBC
	// channel is functional

	// Get Channel ID
	t.Log("getting IBC channel IDs")
	xionChannelInfo, err := relayer.GetChannels(ctx, eRep, xion.Config().ChainID)
	require.NoError(t, err)
	xionChannelID := xionChannelInfo[0].ChannelID

	osmoChannelInfo, err := relayer.GetChannels(ctx, eRep, osmosis.Config().ChainID)
	require.NoError(t, err)
	osmoChannelID := osmoChannelInfo[0].ChannelID

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
}
