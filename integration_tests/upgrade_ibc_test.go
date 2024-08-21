package integration_tests

import (
	"context"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"os"
	"strconv"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	"cosmossdk.io/math"

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
	xionVersionTo   = "sha-962b654"
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
			upgrade:     UpgradeXion,
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

// SubmitIBCSoftwareUpgradeProposal submits and passes an IBCSoftwareUpgrade govprop.
func SubmitIBCSoftwareUpgradeProposal(
	t *testing.T,
	chain *cosmos.CosmosChain,
	chainUser ibc.Wallet,
	currentHeight int64,
	haltHeight int64,
) (error error) {
	ctx := context.Background()

	// An UpgradedClientState must be provided to perform an IBC breaking upgrade.
	// This will make the chain commit to the correct upgraded (self) client state
	// before the upgrade occurs, so that connecting chains can verify that the
	// new upgraded client is valid by verifying a proof on the previous version
	// of the chain.

	// The UpgradePlan must specify an upgrade height only (no upgrade time),
	// and the ClientState should only include the fields common to all valid clients
	// (chain-specified parameters) and zero out any client-customizable fields
	// (such as TrustingPeriod).

	upgradedClientState := &ibctm.ClientState{
		ChainId: chain.Config().ChainID,
	}
	upgradedClientStateAny, err := ibcclienttypes.PackClientState(upgradedClientState)
	require.NoError(t, err, "couldn't pack upgraded client state: %v", err)

	// Build IBCSoftwareUpgrade message
	plan := upgradetypes.Plan{
		Name:   xionUpgradeName,
		Height: haltHeight,
		Info:   fmt.Sprintf("Software Upgrade %s", xionUpgradeName),
	}
	upgrade := ibcclienttypes.MsgIBCSoftwareUpgrade{
		Plan:                plan,
		UpgradedClientState: upgradedClientStateAny,
		Signer:              authority,
	}

	// Get proposer addr and keyname
	address, err := chain.GetAddress(ctx, chainUser.KeyName())
	require.NoError(t, err)
	proposerAddr, err := sdk.Bech32ifyAddressBytes(chain.Config().Bech32Prefix, address)
	require.NoError(t, err)
	proposerKeyname := chainUser.KeyName()

	// Build govprop
	proposal, err := chain.BuildProposal(
		[]cosmos.ProtoMessage{&upgrade},
		fmt.Sprintf("Software Upgrade %s", xionUpgradeName),
		"upgrade chain E2E test",
		"",
		fmt.Sprintf("%d%s", 10_000_000, chain.Config().Denom),
		proposerAddr,
		true,
	)
	require.NoError(t, err)

	// Submit govprop
	err = submitGovprop(t, chain, proposerKeyname, proposal, currentHeight)
	require.NoError(t, err, "couldn't submit govprop: %v", err)

	return err
}

// SubmitSoftwareUpgradeProposal submits and passes a SoftwareUpgrade govprop.
func SubmitSoftwareUpgradeProposal(
	t *testing.T,
	chain *cosmos.CosmosChain,
	chainUser ibc.Wallet,
	currentHeight int64,
	haltHeight int64,
) (error error) {
	ctx := context.Background()

	// Get proposer addr and keyname
	proposerKeyname := chainUser.KeyName()
	address, err := chain.GetAddress(ctx, proposerKeyname)
	require.NoError(t, err)
	proposerAddr, err := sdk.Bech32ifyAddressBytes(chain.Config().Bech32Prefix, address)
	require.NoError(t, err)

	// Build SoftwareUpgrade message
	plan := upgradetypes.Plan{
		Name:   xionUpgradeName,
		Height: haltHeight,
		Info:   fmt.Sprintf("Software Upgrade %s", xionUpgradeName),
	}
	upgrade := upgradetypes.MsgSoftwareUpgrade{
		Authority: authority,
		Plan:      plan,
	}

	// Build govprop
	proposal, err := chain.BuildProposal(
		[]cosmos.ProtoMessage{&upgrade},
		fmt.Sprintf("Software Upgrade %s", xionUpgradeName),
		"upgrade chain E2E test",
		"",
		fmt.Sprintf("%d%s", 10_000_000, chain.Config().Denom),
		proposerAddr,
		true,
	)
	require.NoError(t, err)

	// Submit govprop
	err = submitGovprop(t, chain, proposerKeyname, proposal, currentHeight)
	require.NoError(t, err, "couldn't submit govprop: %v", err)

	return err
}

// submitGovprop submits a cosmos.TxProposalv1 and ensures it passes.
func submitGovprop(
	t *testing.T,
	chain *cosmos.CosmosChain,
	proposerKeyname string,
	proposal cosmos.TxProposalv1,
	currentHeight int64,
) (err error) {
	ctx := context.Background()

	// Submit govprop
	tx, err := chain.SubmitProposal(ctx, proposerKeyname, proposal)
	require.NoError(t, err)

	// Ensure prop exists and is vote-able
	propId, err := strconv.Atoi(tx.ProposalID)
	require.NoError(t, err, "couldn't convert proposal ID to int: %v", err)
	prop, err := chain.GovQueryProposal(ctx, uint64(propId))
	require.NoError(t, err, "couldn't query proposal: %v", err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	// Vote on govprop
	err = chain.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoErrorf(t, err, "couldn't submit votes: %v", err)

	// Ensure govprop passed
	_, err = cosmos.PollForProposalStatus(ctx, chain, currentHeight, currentHeight+haltHeightDelta, prop.ProposalId, govv1beta1.StatusPassed)
	require.NoErrorf(t, err, "couldn't poll for proposal status: %v", err)

	return err
}

// UpgradeXion attempts to upgrade a chain, and optionally handle breaking IBC changes.
func UpgradeXion(
	t *testing.T,
	chain *cosmos.CosmosChain,
	dockerClient *client.Client,
) {
	ctx := context.Background()

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// Fund proposer
	fundAmount := math.NewInt(20_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	// determine halt height
	currentHeight, err := chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height: %v", err)
	haltHeight := currentHeight + haltHeightDelta - 3

	// submit IBC upgrade proposal
	err = SubmitIBCSoftwareUpgradeProposal(t, chain, chainUser, haltHeight, currentHeight)
	require.NoErrorf(t, err, "couldn't submit IBC upgrade proposal: %v", err)

	// submit SoftwareUpgrade proposal
	err = SubmitSoftwareUpgradeProposal(t, chain, chainUser, haltHeight, currentHeight)
	require.NoErrorf(t, err, "couldn't submit UpgradeXion proposal: %v", err)

	// confirm chain halt
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-currentHeight), chain)
	currentHeight, err = chain.Height(ctx)
	require.NoErrorf(t, err, "couldn't get chain height after chain should have halted: %v", err)
	// ERR CONSENSUS FAILURE!!! err="UPGRADE \"v10\" NEEDED at height: 80: Software Upgrade v10" module=consensus
	// INF Timed out dur=2000 height=81 module=consensus round=0 step=RoundStepPropose
	require.GreaterOrEqualf(t, currentHeight, haltHeight, "height: %d is not >= to haltHeight: %d", currentHeight, haltHeight)

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
