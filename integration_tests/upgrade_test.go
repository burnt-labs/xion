package integration_tests

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

const (
	haltHeightDelta    = int64(10) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
	authority          = "xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc" // Governance authority address
)

/*
* CONTRACT: This test requires manual setup before running
* 1- Git checkout to current version of the network
* 2- Build current using heighliner pass in the flag `-t current`. (for instructions on how to build check README.md on the root of the project)
* 3- Git checkout to upgrade-version version of the network
* 4- Build using heighliner pass in the flag `-t upgrade`. (for instructions on how to build check README.md on the root of the project)
* 5- Mark upgrade name as the last parameter of the function
* 6- cd integration_test
* 7- XION_IMAGE=[current version of the network] go test -run TestXionUpgrade ./...

As of Aug 17 2023 this is the necessary process to run this test, this is due to the fact that AWS & docker-hub auto deleting old images, therefore you might lose what the version currently running is image wise
current-testnet: v6
upgrade-version: v7
*/

func TestXionUpgradeNetwork(t *testing.T) {
	t.Parallel()

	// pull "recent" version, that is the upgrade target
	imageTag := os.Getenv("XION_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// set "previous" to the value in the test const
	err := os.Setenv("XION_IMAGE", fmt.Sprintf("%s:%s", xionImageFrom, xionVersionFrom))
	require.NoError(t, err)
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisAAAllowedCodeIDs}, [][]string{{votingPeriod, maxDepositPeriod}, {votingPeriod, maxDepositPeriod}}))

	// issue the upgrade with "recent" again
	CosmosChainUpgradeTest(t, &td, imageTagComponents[0], imageTagComponents[1], xionUpgradeName)
}

func CosmosChainUpgradeTest(t *testing.T, td *TestData, upgradeContainerRepo, upgradeVersion string, upgradeName string) {
	// t.Skip("ComosChainUpgradeTest should be run manually, please comment skip and follow instructions when running")
	chain, ctx, client := td.xionChain, td.ctx, td.client

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta

	plan := upgradetypes.Plan{
		Name:   upgradeName,
		Height: haltHeight,
		Info:   fmt.Sprintf("Software Upgrade %s", upgradeName),
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
		"Chain Upgrade 1",
		"First chain software upgrade",
		"",
		"500000000"+chain.Config().Denom, // greater than min deposit
		addrString,
		false,
	)
	require.NoError(t, err)

	_, err = chain.SubmitProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err)

	prop, err := chain.GovQueryProposal(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, govv1beta1.StatusVotingPeriod, prop.Status)

	err = chain.VoteOnProposalAllValidators(ctx, prop.ProposalId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, prop.ProposalId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should timeout due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height), chain)

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, fmt.Sprintf("height: %d is not equal to halt height: %d", height, haltHeight))

	// bring down nodes to prepare for upgrade
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")
	// upgrade version on all nodes
	chain.UpgradeVersion(ctx, client, upgradeContainerRepo, upgradeVersion)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = chain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// check that the upgrade set the params
	paramsModResp, err := ExecQuery(t, ctx, chain.GetNode(),
		"params", "subspace", "jwk", "TimeOffset")
	require.NoError(t, err)
	t.Logf("jwk params response: %v", paramsModResp)

	jwkParams, err := ExecQuery(t, ctx, chain.GetNode(),
		"jwk", "params")
	require.NoError(t, err)
	t.Logf("jwk params response: %v", jwkParams)

	tokenFactoryParams, err := ExecQuery(t, ctx, chain.GetNode(),
		"tokenfactory", "params")
	require.NoError(t, err)
	t.Logf("tokenfactory params response: %v", tokenFactoryParams)
}
