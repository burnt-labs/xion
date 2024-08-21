package integration_tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v8/conformance"
	"github.com/strangelove-ventures/interchaintest/v8/relayer"
	"github.com/strangelove-ventures/interchaintest/v8/relayer/rly"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	xionImageFrom   = "ghcr.io/burnt-labs/xion/xion"
	xionVersionFrom = "v9.0.0"
	xionImageTo     = "ghcr.io/burnt-labs/xion/heighliner"
	xionVersionTo   = "sha-8fad91a"
	xionUpgradeName = "v10"

	osmosisImage   = "ghcr.io/strangelove-ventures/heighliner/osmosis"
	osmosisVersion = "v25.2.1"

	relayerImage   = "ghcr.io/cosmos/relayer"
	relayerVersion = "main"
	relayerImpl    = ibc.CosmosRly

	authority = "xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc" // Governance authority address
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
	rlyImage := relayer.CustomDockerImage(relayerImage, relayerVersion, rly.RlyDefaultUidGid)
	rf := interchaintest.NewBuiltinRelayerFactory(relayerImpl, zaptest.NewLogger(t), rlyImage)

	// Configure Chains
	chains := ConfigureChains(t, 1, 2)

	// Define Test cases
	testCases := []struct {
		name        string
		setup       func(t *testing.T, path string, dockerClient *client.Client, dockerNetwork string) (ibc.Chain, ibc.Chain, *interchaintest.Interchain, ibc.Relayer)
		conformance func(t *testing.T, ctx context.Context, client *client.Client, network string, srcChain, dstChain ibc.Chain, rf interchaintest.RelayerFactory, rep *testreporter.Reporter, relayerImpl ibc.Relayer, pathNames ...string)
		upgrade     func(t *testing.T, chain *cosmos.CosmosChain, dockerClient *client.Client)
	}{
		{
			name: "xion-osmosis",
			setup: func(t *testing.T, path string, dockerClient *client.Client, dockerNetwork string) (ibc.Chain, ibc.Chain, *interchaintest.Interchain, ibc.Relayer) {
				xion, osmosis := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)
				r := rf.Build(t, dockerClient, dockerNetwork)
				ic := SetupInterchain(t, xion, osmosis, path, r, eRep, dockerClient, dockerNetwork)
				return xion, osmosis, ic, r
			},
			conformance: conformance.TestChainPair,
			upgrade:     SoftwareUpgrade,
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dockerClient, dockerNetwork := interchaintest.DockerSetup(t)
			xion, counterparty, ichain, rlyr := tc.setup(t, tc.name, dockerClient, dockerNetwork)
			defer ichain.Close()
			tc.conformance(t, ctx, dockerClient, dockerNetwork, xion, counterparty, rf, rep, rlyr, tc.name)
			x := xion.(*cosmos.CosmosChain)
			tc.upgrade(t, x, dockerClient)
			tc.conformance(t, ctx, dockerClient, dockerNetwork, xion, counterparty, rf, rep, rlyr, tc.name)
		})
	}
}

// ConfigureChains creates a slice of ibc.Chain with the given number of full nodes and validators.
func ConfigureChains(t *testing.T, numFullNodes, numValidators int) []ibc.Chain {
	// Override default embedded configuredChains.yaml
	f := OverrideConfiguredChainsYaml(t)
	defer os.Remove(f.Name())

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
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
				TrustingPeriod: "336h",
				NoHostMount:    false,
				ModifyGenesis:  ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}),
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "osmosis",
			Version: osmosisVersion,
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: osmosisImage,
						Version:    osmosisVersion,
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

// SoftwareUpgrade submits, votes and performs a software upgrade govprop on the given chain.
func SoftwareUpgrade(
	t *testing.T,
	chain *cosmos.CosmosChain,
	dockerClient *client.Client,
) {
	ctx := context.Background()

	// fund user
	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	// build software upgrade govprop
	height, err := chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height for softwareUpgradeProposal: %v", err)
	haltHeight := height + haltHeightDelta - 3

	// submit and vote on software upgrade
	plan := upgradetypes.Plan{
		Name:   xionUpgradeName,
		Height: haltHeight,
		Info:   fmt.Sprintf("Software Upgrade %s", xionUpgradeName),
	}
	upgrade := upgradetypes.MsgSoftwareUpgrade{
		Authority: authority,
		Plan:      plan,
	}

	address, err := chain.GetAddress(ctx, chainUser.KeyName())
	require.NoError(t, err)

	addrString, err := sdk.Bech32ifyAddressBytes(chain.Config().Bech32Prefix, address)
	require.NoError(t, err)

	proposal, err := chain.BuildProposal(
		[]cosmos.ProtoMessage{&upgrade},
		fmt.Sprintf("Software Upgrade %s", xionUpgradeName),
		"upgrade chain E2E test",
		"",
		fmt.Sprintf("%d%s", 10_000_000, chain.Config().Denom),
		addrString,
		true,
	)
	require.NoError(t, err)

	_, err = chain.SubmitProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err)

	prop, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	err = chain.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoErrorf(t, err, "couldn't submit votes: %v", err)
	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, prop.ProposalId, govv1beta1.StatusPassed)
	require.NoErrorf(t, err, "couldn't poll for softwareUpgradeProposal status: %v", err)
	height, err = chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height: %v", err)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// confirm chain halt
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height), chain)
	height, err = chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height after chain should have halted: %v", err)
	// ERR CONSENSUS FAILURE!!! err="UPGRADE \"v10\" NEEDED at height: 80: Software Upgrade v10" module=consensus
	// INF Timed out dur=2000 height=81 module=consensus round=0 step=RoundStepPropose
	require.GreaterOrEqualf(t, height, haltHeight, "height: %d is not >= to haltHeight: %d", height, haltHeight)

	// upgrade all nodes
	err = chain.StopAllNodes(ctx)
	require.NoErrorf(t, err, "couldn't stop nodes: %v", err)
	chain.UpgradeVersion(ctx, dockerClient, xionImageTo, xionVersionTo)

	// reboot nodes
	err = chain.StartAllNodes(ctx)
	require.NoErrorf(t, err, "couldn't reboot nodes: %v", err)

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")
}
