package e2e_app

import (
	"strings"
	"testing"

	"github.com/burnt-labs/xion/e2e_tests/testlib"
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

	// Build chain starting with the "from" image
	xion := testlib.BuildXionChainWithSpec(t, chainSpec)

	// Upgrade from current version in repo to local image
	testlib.CosmosChainUpgradeTest(t, xion, xionToRepo, xionToVersion, upgradeName)
}
