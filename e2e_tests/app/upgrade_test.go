package e2e_app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
)

func TestAppUpgradeNetwork(t *testing.T) {
	t.Parallel()

	// Get the "from" image (latest released version from GitHub releases)
	xionFromImageParts, err := testlib.GetLatestReleaseImageComponents()
	require.NoError(t, err)
	require.Len(t, xionFromImageParts, 2, "xionFromImage should have [repository, version] format")

	// Get the "to" image (from XION_IMAGE env var) which is where we want to upgrade to
	xionToImageParts, err := testlib.GetXionImageTagComponents()
	require.NoError(t, err)
	require.Len(t, xionToImageParts, 2, "xionToImage should have [repository, version] format")

	xionToRepo := xionToImageParts[0]
	xionToVersion := xionToImageParts[1]

	// Use the app's UpgradeName constant to ensure consistency with the upgrade handler
	upgradeName := app.UpgradeName

	chainSpec := testlib.XionChainSpec(3, 1)
	chainSpec.Version = xionFromImageParts[1]
	chainSpec.Images = []ibc.DockerImage{
		{
			Repository: xionFromImageParts[0],
			Version:    xionFromImageParts[1],
			UIDGID:     "1000:1000",
		},
	}
	chainSpec.ModifyGenesis = cosmos.ModifyGenesis(testlib.DefaultGenesisKVMods)

	// Build chain starting with the "from" image
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Upgrade from released version to the image specified by XION_IMAGE
	testlib.CosmosChainUpgradeTest(t, xion, xionToRepo, xionToVersion, upgradeName)

	// Post-upgrade validation: verify all expected modules and stores are present
	ctx := t.Context()
	t.Run("PostUpgrade_ModuleValidation", func(t *testing.T) {
		verifyModulesInitialized(t, ctx, xion)
	})
}

// verifyModulesInitialized checks that all expected modules are properly initialized after upgrade.
// It dynamically queries the module versions from the upgrade module and verifies critical modules
// have params that can be queried.
func verifyModulesInitialized(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain) {
	t.Log("Verifying all expected modules are initialized after upgrade")

	// Query module versions dynamically from the upgrade module
	// This returns all registered modules and their consensus versions
	t.Run("module_versions_registered", func(t *testing.T) {
		moduleVersions, err := testlib.ExecQuery(t, ctx, xion.GetNode(), "upgrade", "module_versions")
		require.NoError(t, err, "module versions query should succeed")
		require.NotNil(t, moduleVersions, "module versions should not be nil")

		// Extract the module_versions array from the response
		versions, ok := moduleVersions["module_versions"].([]interface{})
		require.True(t, ok, "module_versions should be an array")
		require.NotEmpty(t, versions, "should have registered modules")

		// Build a map of registered modules for easy lookup
		registeredModules := make(map[string]bool)
		for _, v := range versions {
			if module, ok := v.(map[string]interface{}); ok {
				if name, ok := module["name"].(string); ok {
					registeredModules[name] = true
					t.Logf("Found registered module: %s", name)
				}
			}
		}

		// Verify critical modules are registered (modules that were added in recent upgrades)
		criticalModules := []string{"zk", "dkim"}
		for _, mod := range criticalModules {
			require.True(t, registeredModules[mod], "critical module %s should be registered", mod)
		}

		t.Logf("Total registered modules: %d", len(registeredModules))
	})

	// Modules that expose params queries - we verify these are queryable
	// This is a subset of modules that have params we can query
	modulesWithParams := []string{
		"zk",        // New module added in v26/v27
		"dkim",      // New module added in v26/v27
		"bank",      // Core SDK module
		"staking",   // Core SDK module
		"gov",       // Core SDK module
		"mint",      // Xion-specific module
		"globalfee", // Xion-specific module
		"jwk",       // Xion-specific module
	}

	for _, moduleName := range modulesWithParams {
		// capture range variable
		t.Run(moduleName+"_params_queryable", func(t *testing.T) {
			params, err := testlib.ExecQuery(t, ctx, xion.GetNode(), moduleName, "params")
			require.NoError(t, err, "%s module params query should succeed", moduleName)
			require.NotNil(t, params, "%s module params should not be nil", moduleName)
		})
	}

	t.Log("All module validations passed")
}

// TestAppUpgradeNetworkWithFeatures tests the upgrade and validates new features post-upgrade.
// This is the comprehensive upgrade test that validates DKIM and ZKEmail features after upgrading.
func TestAppUpgradeNetworkWithFeatures(t *testing.T) {
	t.Parallel()

	// Set bech32 prefix before creating encoding config
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Get the "from" image (latest released version from GitHub releases)
	xionFromImageParts, err := testlib.GetLatestReleaseImageComponents()
	require.NoError(t, err)
	require.Len(t, xionFromImageParts, 2, "xionFromImage should have [repository, version] format")

	// Get the "to" image (from XION_IMAGE env var) which is where we want to upgrade to
	xionToImageParts, err := testlib.GetXionImageTagComponents()
	require.NoError(t, err)
	require.Len(t, xionToImageParts, 2, "xionToImage should have [repository, version] format")

	xionToRepo := xionToImageParts[0]
	xionToVersion := xionToImageParts[1]

	// Use the app's UpgradeName constant to ensure consistency with the upgrade handler
	upgradeName := app.UpgradeName

	chainSpec := testlib.XionChainSpec(3, 1)
	chainSpec.Version = xionFromImageParts[1]
	chainSpec.Images = []ibc.DockerImage{
		{
			Repository: xionFromImageParts[0],
			Version:    xionFromImageParts[1],
			UIDGID:     "1000:1000",
		},
	}
	chainSpec.ModifyGenesis = cosmos.ModifyGenesis(testlib.DefaultGenesisKVMods)
	// Set encoding config for proper message serialization (needed for DKIM and ZKEmail assertions)
	chainSpec.EncodingConfig = testlib.XionEncodingConfig(t)
	// Use faster block times to ensure proposals pass within timeout windows
	chainSpec.AdditionalStartArgs = []string{
		"--consensus.timeout_commit=1s",
	}

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
