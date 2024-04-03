package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	interchaintest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

const (
	haltHeightDelta    = uint64(10) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
)

func TestXionUpgrade_037_038(t *testing.T) {
	t.Parallel()

As of Aug 17 2023 this is the necessary process to run this test, this is due to the fact that AWS & docker-hub auto deleting old images, therefore you might lose what the version currently running is image wise
current-testnet: v0.3.4
step between: v0.3.5
upgrade-version: v0.3.6
*/
func TestXionUpgradeIBC(t *testing.T) {
	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesisAAAllowedCodeIDs}, [][]string{{votingPeriod, maxDepositPeriod}, {votingPeriod, maxDepositPeriod}}))
	CosmosChainUpgradeIBCTest(t, &td, "xion", "upgrade", "v4")
}

func CosmosChainUpgradeIBCTest(t *testing.T, td *TestData, upgradeContainerRepo, upgradeVersion string, upgradeName string) {
	// t.Skip("ComosChainUpgradeTest should be run manually, please comment skip and follow instructions when running")
	chain, ctx, client := td.xionChain, td.ctx, td.client

	fundAmount := int64(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta - 3

	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     "500000000" + chain.Config().Denom,
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
	require.Equal(t, haltHeight, height, fmt.Sprintf("height: %d is not equal to halt height: %d", height, haltHeight))

	// bring down nodes to prepare for upgrade
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	chain.UpgradeVersion(ctx, client, upgradeContainerRepo, upgradeVersion)
	err = chain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// check that the upgrade set the params
	paramsModResp, err := ExecQuery(t, ctx, chain.FullNodes[0], "params", "subspace", "jwk", "TimeOffset")
	require.NoError(t, err)
	require.Equal(t, paramsModResp["value"], "\"30000\"")
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
