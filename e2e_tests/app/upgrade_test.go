package e2e_app

import (
	"strings"
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/require"
)

func TestAppUpgradeNetwork(t *testing.T) {
	t.Parallel()

	// Get the "from" image (current version in repo)
	xionFromImage, err := testlib.GetGHCRPackageNameCurrentRepo()
	require.NoError(t, err)

	// Get the "to" from (local image) which is where we want to upgrade from
	xionFromImageParts := strings.SplitN(xionFromImage, ":", 2)
	require.GreaterOrEqual(t, len(xionFromImageParts), 2, "xionFromImage should have repository:tag format")

	// Get the "to" image (local image) which is where we want to upgrade to
	xionToImageParts, err := testlib.GetXionImageTagComponents()
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

	chainSpec := testlib.XionChainSpec(3, 1)
	chainSpec.Version = xionFromImageParts[1]
	chainSpec.ChainConfig.Images = []ibc.DockerImage{
		{
			Repository: xionFromImageParts[0],
			Version:    xionFromImageParts[1],
			UIDGID:     "1025:1025",
		},
	}
	chainSpec.ChainConfig.ModifyGenesis = cosmos.ModifyGenesis(testlib.DefaultGenesisKVMods)

	// Build chain starting with the "from" image
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Upgrade from current version in repo to local image
	testlib.CosmosChainUpgradeTest(t, xion, xionToRepo, xionToVersion, upgradeName)
}

// TestAppUpgradeNetworkWithFeatures tests the upgrade and validates new features post-upgrade.
// This is the comprehensive upgrade test that validates DKIM and ZKEmail features after upgrading.
func TestAppUpgradeNetworkWithFeatures(t *testing.T) {
	t.Parallel()

	// Set bech32 prefix before creating encoding config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Get the "from" image (current version in repo)
	xionFromImage, err := testlib.GetGHCRPackageNameCurrentRepo()
	require.NoError(t, err)

	// Get the "to" from (local image) which is where we want to upgrade from
	xionFromImageParts := strings.SplitN(xionFromImage, ":", 2)
	require.GreaterOrEqual(t, len(xionFromImageParts), 2, "xionFromImage should have repository:tag format")

	// Get the "to" image (local image) which is where we want to upgrade to
	xionToImageParts, err := testlib.GetXionImageTagComponents()
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

	chainSpec := testlib.XionChainSpec(3, 1)
	chainSpec.Version = xionFromImageParts[1]
	chainSpec.ChainConfig.Images = []ibc.DockerImage{
		{
			Repository: xionFromImageParts[0],
			Version:    xionFromImageParts[1],
			UIDGID:     "1025:1025",
		},
	}
	chainSpec.ChainConfig.ModifyGenesis = cosmos.ModifyGenesis(testlib.DefaultGenesisKVMods)
	// Set encoding config for proper message serialization (needed for DKIM and ZKEmail assertions)
	chainSpec.ChainConfig.EncodingConfig = testlib.XionEncodingConfig(t)

	// Build chain starting with the "from" image
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Upgrade from current version in repo to local image
	// NOTE: CosmosChainUpgradeTest uses proposal ID 1 for the upgrade
	// It uses a validator to submit the proposal, avoiding creation of a new user account
	// which would shift account numbers and break ZK proof verification.
	testlib.CosmosChainUpgradeTest(t, xion, xionToRepo, xionToVersion, upgradeName)

	// Run post-upgrade feature validations
	// Create a proposal tracker starting at 2 (since upgrade used proposal 1)
	proposalTracker := testlib.NewProposalTracker(2)
	ctx := t.Context()

	// Run ZKEmail authenticator assertions
	// NOTE: ZKEmail now seeds its own DKIM record using the proposal tracker
	// User is nil so it will create the "zkemail-test" user with DeployerMnemonic,
	// ensuring the AA contract address matches the pre-generated ZK proofs.
	t.Run("PostUpgrade_ZKEmail", func(t *testing.T) {
		testlib.RunZKEmailAuthenticatorAssertions(t, testlib.ZKEmailAssertionConfig{
			Chain:           xion,
			Ctx:             ctx,
			User:            nil, // Will use DeployerMnemonic for pre-generated proofs
			ProposalTracker: proposalTracker,
		})
	})
	// Run DKIM module assertions
	t.Run("PostUpgrade_DKIM_Module", func(t *testing.T) {
		testlib.RunDKIMModuleAssertions(t, testlib.DKIMAssertionConfig{
			Chain:           xion,
			Ctx:             ctx,
			User:            nil, // Will create and fund a new user
			ProposalTracker: proposalTracker,
			TestData:        testlib.DefaultDKIMTestData(),
		})
	})

	// Run DKIM governance assertions
	t.Run("PostUpgrade_DKIM_Governance", func(t *testing.T) {
		testlib.RunDKIMGovernanceAssertions(t, testlib.DKIMAssertionConfig{
			Chain:           xion,
			Ctx:             ctx,
			User:            nil, // Will create and fund a new user
			ProposalTracker: proposalTracker,
			TestData:        testlib.DefaultDKIMTestData(),
		})
	})
}
