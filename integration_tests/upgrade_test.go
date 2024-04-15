package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
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
* 7- XION_IMAGE=[current version of the network] go test -run TestXionUpgrade ./...

As of Aug 17 2023 this is the necessary process to run this test, this is due to the fact that AWS & docker-hub auto deleting old images, therefore you might lose what the version currently running is image wise
current-testnet: v0.3.4
step between: v0.3.5
upgrade-version: v0.3.6
*/
func TestXionUpgrade(t *testing.T) {
	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisAAAllowedCodeIDs}, [][]string{{votingPeriod, maxDepositPeriod}, {votingPeriod, maxDepositPeriod}}))
	CosmosChainUpgradeTest(t, &td, "xion", "upgrade", "v4")
}

func CosmosChainUpgradeTest(t *testing.T, td *TestData, upgradeContainerRepo, upgradeVersion string, upgradeName string) {
	// t.Skip("ComosChainUpgradeTest should be run manually, please comment skip and follow instructions when running")
	chain, ctx, client := td.xionChain, td.ctx, td.client

	fundAmount := int64(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta - 3

	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     "500000000" + chain.Config().Denom, // greater than min deposit
		Title:       "Chain Upgrade 1",
		Name:        upgradeName,
		Description: "First chain software upgrade",
		Height:      haltHeight,
	}

	upgradeTx, err := chain.LegacyUpgradeProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err, "error submitting software upgrade proposal tx")

	err = chain.VoteOnProposalAllValidators(ctx, upgradeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, upgradeTx.ProposalID, cosmos.ProposalStatusPassed)
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
	paramsModResp, err := ExecQuery(t, ctx, chain.FullNodes[0],
		"params", "subspace", "jwk", "TimeOffset")
	require.NoError(t, err)
	t.Logf("jwk params response: %v", paramsModResp)

	jwkParams, err := ExecQuery(t, ctx, chain.FullNodes[0],
		"jwk", "params")
	require.NoError(t, err)
	t.Logf("jwk params response: %v", jwkParams)

	tokenFactoryParams, err := ExecQuery(t, ctx, chain.FullNodes[0],
		"tokenfactory", "params")
	require.NoError(t, err)
	t.Logf("tokenfactory params response: %v", tokenFactoryParams)
}
