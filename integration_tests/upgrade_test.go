package integration_tests

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"cosmossdk.io/math"

	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"github.com/burnt-labs/xion/integration_tests/helpers"
	"github.com/strangelove-ventures/interchaintest/v10"
	"github.com/strangelove-ventures/interchaintest/v10/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v10/ibc"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
)

const (
	haltHeightDelta    = int64(10) // will propose upgrade this many blocks in the future
	blocksAfterUpgrade = uint64(10)
	authority          = "xion10d07y265gmmuvt4z0w9aw880jnsr700jctf8qc" // Governance authority address
)

func TestXionUpgradeNetwork(t *testing.T) {
	t.Parallel()

	// Get the "from" image (current version in repo)
	xionFromImage, err := helpers.GetGHCRPackageNameCurrentRepo()
	require.NoError(t, err)

	// Get the "to" from (local image) which is where we want to upgrade from
	xionFromImageParts := strings.SplitN(xionFromImage, ":", 2)
	require.GreaterOrEqual(t, len(xionFromImageParts), 2, "xionFromImage should have repository:tag format")

	// Get the "to" image (local image) which is where we want to upgrade to
	xionToImageParts, err := GetXionImageTagComponents()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(xionToImageParts), 2, "xionToImage should have repository:tag format")

	xionToRepo := xionToImageParts[0]
	xionToVersion := xionToImageParts[1]

	// Use "recent" as upgrade name for local builds, otherwise use version-based name
	upgradeName := "recent"
	if xionToVersion != "local" {
		// For non-local builds, use version as upgrade name (e.g., "v20")
		upgradeName = xionToVersion
	}

	chainSpec := XionChainSpec(3, 1)
	chainSpec.Version = xionFromImageParts[1]
	chainSpec.ChainConfig.Images = []ibc.DockerImage{
		{
			Repository: xionFromImageParts[0],
			Version:    xionFromImageParts[1],
			UidGid:     "1025:1025",
		},
	}

	// Build chain starting with the "from" image
	xion := BuildXionChainWithSpec(t, chainSpec)

	// Upgrade from current version in repo to local image
	CosmosChainUpgradeTest(t, xion, xionToRepo, xionToVersion, upgradeName)
}

func CosmosChainUpgradeTest(t *testing.T, xion *cosmos.CosmosChain, upgradeContainerRepo, upgradeVersion string, upgradeName string) {
	ctx := t.Context()
	// t.Skip("ComosChainUpgradeTest should be run manually, please comment skip and follow instructions when running")
	chain, client := xion, xion.GetNode().DockerClient

	fundAmount := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, chain)
	chainUser := users[0]

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta
	t.Logf("halt height: %d", haltHeight)

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

	// upgrade version on all nodes
	queryRes, _, err := chain.GetNode().ExecQuery(ctx, "mint", "params")
	require.NoError(t, err)
	t.Logf("mint parameters: %+v \n", string(queryRes)) // confirming mint params before the upgrade

	// bring down nodes to prepare for upgrade
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")
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
