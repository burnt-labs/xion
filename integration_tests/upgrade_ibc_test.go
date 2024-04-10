package integration_tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7/testutil"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v7/conformance"
	"github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/relayer/rly"

	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestXionUpgradeIBC tests a Xion software upgrade, ensuring IBC conformance prior-to and after the upgrade.
func TestXionUpgradeIBC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Setup loggers and reporters
	f, err := interchaintest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build RelayerFactory
	rlyImage := relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main", rly.RlyDefaultUidGid)
	rf := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t), rlyImage)

	// Configure Chains
	chains := ConfigureChains(t, 1, 2)

	// Define Test cases
	testCases := []struct {
		name                string
		setup               func(t *testing.T, path string, dockerClient *client.Client, dockerNetwork string) (ibc.Chain, ibc.Chain, *interchaintest.Interchain, ibc.Relayer)
		conformance         func(t *testing.T, ctx context.Context, client *client.Client, network string, srcChain, dstChain ibc.Chain, rf interchaintest.RelayerFactory, rep *testreporter.Reporter, relayerImpl ibc.Relayer, pathNames ...string)
		upgrade             func(t *testing.T, chain *cosmos.CosmosChain, upgradeName string, dockerClient *client.Client, dockerImageRepo, dockerImageVersion string)
		upgradeName         string
		upgradeImageVersion string
	}{
		{
			name: "xion-osmosis",
			setup: func(t *testing.T, path string, dockerClient *client.Client, dockerNetwork string) (ibc.Chain, ibc.Chain, *interchaintest.Interchain, ibc.Relayer) {
				xion, osmosis := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)
				r := rf.Build(t, dockerClient, dockerNetwork)
				ic := SetupInterchain(t, xion, osmosis, path, r, eRep, dockerClient, dockerNetwork)
				return xion, osmosis, ic, r
			},
			conformance:         conformance.TestChainPair,
			upgrade:             SoftwareUpgrade,
			upgradeName:         "v6",
			upgradeImageVersion: "sha-872993e",
		},
		//{
		//	name: "xion-axelar",
		//	setup: func(t *testing.T, path string, dockerClient *client.Client, dockerNetwork string) (ibc.Chain, ibc.Chain, *interchaintest.Interchain, ibc.Relayer) {
		//		xion, axelar := chains[0].(*cosmos.CosmosChain), chains[2].(*cosmos.CosmosChain)
		//		r := rf.Build(t, dockerClient, dockerNetwork)
		//		ic := SetupInterchain(t, xion, axelar, path, r, eRep, dockerClient, dockerNetwork)
		//		return xion, axelar, ic, r
		//	},
		//	conformance: conformance.TestChainPair,
		//	upgrade:     SoftwareUpgrade,
		//},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dockerClient, dockerNetwork := interchaintest.DockerSetup(t)
			xion, counterparty, ichain, rlyr := tc.setup(t, tc.name, dockerClient, dockerNetwork)
			defer ichain.Close()
			tc.conformance(t, ctx, dockerClient, dockerNetwork, xion, counterparty, rf, rep, rlyr, tc.name)
			x := xion.(*cosmos.CosmosChain)
			tc.upgrade(t, x, tc.upgradeName, dockerClient, "ghcr.io/burnt-labs/xion/xion", tc.upgradeImageVersion)
			tc.conformance(t, ctx, dockerClient, dockerNetwork, xion, counterparty, rf, rep, rlyr, tc.name)
		})
	}
}

// ConfigureChains creates a slice of ibc.Chain with the given number of full nodes and validators.
func ConfigureChains(t *testing.T, numFullNodes, numValidators int) []ibc.Chain {
	// must override Axelar's default override NoHostMount in yaml
	// otherwise fails on `cp` on heighliner img as it's not available in the container
	f := OverrideConfiguredChainsYaml(t)
	defer os.Remove(f.Name())

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:    "xion",
			Version: "v0.3.8",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/burnt-labs/xion/xion",
						Version:    "v0.3.8",
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
				NoHostMount:            false,
				ModifyGenesis:          ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}),
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "osmosis",
			Version: "v24.0.0-rc0",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/strangelove-ventures/heighliner/osmosis",
						Version:    "v24.0.0-rc0",
						UidGid:     "1025:1025",
					},
				},
				Type:           "cosmos",
				Bin:            "osmosisd",
				Bech32Prefix:   "osmo",
				Denom:          "uosmo",
				GasPrices:      "0.025uosmo",
				GasAdjustment:  1.3,
				TrustingPeriod: "336h",
				NoHostMount:    false,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "axelar",
			Version: "v0.35.3",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/strangelove-ventures/heighliner/axelar",
						Version:    "v0.35.3",
						UidGid:     "1025:1025",
					},
				},
				Type:           "cosmos",
				Bin:            "axelard",
				Bech32Prefix:   "axelar",
				Denom:          "uaxl",
				GasPrices:      "0.007uaxl",
				GasAdjustment:  1.3,
				TrustingPeriod: "336h",
				NoHostMount:    false,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err, "error creating chains")

	return chains
}

// SetupInterchain builds an interchaintest.Interchain with the given chain pair and relayer.
func SetupInterchain(
	t *testing.T,
	xion ibc.Chain,
	counterparty ibc.Chain,
	path string,
	r ibc.Relayer,
	eRep *testreporter.RelayerExecReporter,
	dockerClient *client.Client,
	dockerNetwork string,
) *interchaintest.Interchain {
	// Configure Interchain
	ic := interchaintest.NewInterchain().
		AddChain(xion).
		AddChain(counterparty).
		AddRelayer(r, "rly").
		AddLink(interchaintest.InterchainLink{
			Chain1:  xion,
			Chain2:  counterparty,
			Relayer: r,
			Path:    path,
		})

	// Build Interchain
	err := ic.Build(context.Background(), eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            dockerClient,
		NetworkID:         dockerNetwork,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation:  false,
	})

	require.NoError(t, err)
	return ic
}

func SoftwareUpgrade(
	t *testing.T,
	chain *cosmos.CosmosChain,
	upgradeName string,
	dockerClient *client.Client,
	dockerImageRepo, dockerImageVersion string,
) {
	ctx := context.Background()

	// fund user
	fundAmount := int64(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	// build govprop
	height, err := chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height for proposal: %v", err)
	haltHeight := height + haltHeightDelta - 3
	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     fmt.Sprintf("%d%s", 10_000_000, chain.Config().Denom),
		Title:       fmt.Sprintf("Software Upgrade %s", upgradeName),
		Name:        upgradeName,
		Description: fmt.Sprintf("Software Upgrade %s", upgradeName),
		Height:      haltHeight,
	}

	// submit and vote on govprop
	upgradeTx, err := chain.LegacyUpgradeProposal(ctx, chainUser.KeyName(), proposal)
	require.NoErrorf(t, err, "couldn't submit software upgrade proposal tx: %v", err)
	err = chain.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoErrorf(t, err, "couldn't submit votes: %v", err)
	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, upgradeTx.ProposalID, cosmos.ProposalStatusPassed)
	require.NoErrorf(t, err, "couldn't poll for proposal status: %v", err)
	height, err = chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height: %v", err)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// confirm chain halt
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height), chain)
	height, err = chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height after chain should have halted: %v", err)
	require.Equalf(t, haltHeight, height, "height: %d is not equal to halt height: %d", height, haltHeight)

	// upgrade all nodes
	err = chain.StopAllNodes(ctx)
	require.NoErrorf(t, err, "couldn't stop nodes: %v", err)
	chain.UpgradeVersion(ctx, dockerClient, dockerImageRepo, dockerImageVersion)

	// reboot nodes
	err = chain.StartAllNodes(ctx)
	require.NoErrorf(t, err, "couldn't reboot nodes: %v", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")
}
