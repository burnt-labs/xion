package testlib

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/docker/docker/client"

	"cosmossdk.io/math"
	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/stretchr/testify/require"

	xionapp "github.com/burnt-labs/xion/app"
)

type TestData struct {
	xionChain *cosmos.CosmosChain
	ctx       context.Context
	client    *client.Client
}

var (
	defaultMinGasPrices            = sdk.DecCoins{sdk.NewDecCoin("uxion", math.ZeroInt())}
	defaultIbcClientTrustingPeriod = "336h" // 14 days
	DefaultGenesisKVMods           = []cosmos.GenesisKV{
		// Gov module - short proposals
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.max_deposit_period", "10s"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", "uxion"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "100"),

		// Mint module - inflation parameters
		cosmos.NewGenesisKV("app_state.mint.params.blocks_per_year", "13892511"),
		cosmos.NewGenesisKV("app_state.mint.params.mint_denom", "uxion"),

		// Abstract account module
		cosmos.NewGenesisKV("app_state.abstractaccount.params.allowed_code_ids", []int64{1}),
		cosmos.NewGenesisKV("app_state.abstractaccount.params.allow_all_code_ids", false),

		// Packet forward middleware
		// cosmos.NewGenesisKV("app_state.packetfowardmiddleware.params.fee_percentage", "0.0"),
	}

	// DeployerMnemonic is a test mnemonic used across e2e tests
	DeployerMnemonic = "decorate corn happy degree artist trouble color mountain shadow hazard canal zone hunt unfold deny glove famous area arrow cup under sadness salute item"
)

func BuildXionChain(t *testing.T) *cosmos.CosmosChain {
	return BuildXionChainWithSpec(t, XionLocalChainSpec(t, 3, 1))
}

// BuildXionChainWithSpec builds a Xion chain using a chain spec
func BuildXionChainWithSpec(t *testing.T, spec *interchaintest.ChainSpec) *cosmos.CosmosChain {
	ctx := t.Context()

	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{spec})

	xion := chains[0].(*cosmos.CosmosChain)

	client, network := interchaintest.DockerSetup(t)

	// Prep Interchain
	ic := interchaintest.NewInterchain().
		AddChain(xion)

	// Log location
	f, err := interchaintest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)

	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	require.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
		// Use empty string to disable block database to avoid SQLite lock contention in parallel tests
		BlockDatabaseFile: "",

		SkipPathCreation: false,
	},
	),
	)
	return xion
}

// XionChainSpec returns a chain spec for Xion with configurable validators and full nodes
func XionLocalChainSpec(t *testing.T, numVals int, numFn int) *interchaintest.ChainSpec {
	imageTag, err := GetXionImage()
	if err != nil {
		t.Fatalf("Failed to get XION image: %v", err)
	}
	imageTagComponents := strings.Split(imageTag, ":")
	chainSpec := XionChainSpec(numVals, numFn)
	chainSpec.Version = imageTagComponents[1]
	chainSpec.ChainConfig.EncodingConfig = XionEncodingConfig(t)
	chainSpec.ChainConfig.Images = []ibc.DockerImage{
		{
			Repository: imageTagComponents[0],
			Version:    imageTagComponents[1],
			UIDGID:     "1025:1025",
		},
	}
	chainSpec.ChainConfig.ModifyGenesis = cosmos.ModifyGenesis(DefaultGenesisKVMods)
	chainSpec.ChainConfig.AdditionalStartArgs = []string{
		"--consensus.timeout_commit=1s",
	}
	return chainSpec
}

func XionEncodingConfig(t *testing.T) *moduletestutil.TestEncodingConfig {
	// Get encoding config directly from the Xion app
	appEncoding := xionapp.MakeEncodingConfig(t)

	// Convert to TestEncodingConfig format
	return &moduletestutil.TestEncodingConfig{
		InterfaceRegistry: appEncoding.InterfaceRegistry,
		Codec:             appEncoding.Codec,
		TxConfig:          appEncoding.TxConfig,
		Amino:             appEncoding.Amino,
	}
}

func GetXionImage() (string, error) {
	imageTag, found := os.LookupEnv("XION_IMAGE")
	if found {
		return imageTag, nil
	}
	// Default to local build or public image
	// In CI, XION_IMAGE is set by GitHub Actions
	// For local testing, use the heighliner image
	return "ghcr.io/burnt-labs/xion/heighliner:v22", nil
}

func GetXionImageTagComponents() ([]string, error) {
	imageTag, err := GetXionImage()
	if err != nil {
		return []string{}, err
	}
	return strings.Split(imageTag, ":"), nil
}

// XionChainSpec returns a chain spec for Xion using manually built configuration
func XionChainSpec(numVals int, numFn int) *interchaintest.ChainSpec {
	chainSpec := &interchaintest.ChainSpec{
		Name:          "xion",
		NumValidators: &numVals,
		NumFullNodes:  &numFn,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           "xion",
			ChainID:        "xion-1",
			Bin:            "xiond",
			Bech32Prefix:   "xion",
			Denom:          "uxion",
			GasPrices:      "0.0uxion",
			GasAdjustment:  2.0,
			TrustingPeriod: "336h",
			NoHostMount:    false,
			Images: []ibc.DockerImage{
				{
					Repository: "ghcr.io/burnt-labs/xion/heighliner/xion",
					Version:    "latest",
					UIDGID:     "1025:1025",
				},
			},
		},
	}
	return chainSpec
}

// OsmosisChainSpec returns a chain spec for Osmosis using manually built configuration
func OsmosisChainSpec(numVals int, numFn int) *interchaintest.ChainSpec {
	return &interchaintest.ChainSpec{
		Name:          "osmosis",
		NumValidators: &numVals,
		NumFullNodes:  &numFn,
		ChainConfig: ibc.ChainConfig{
			Type:           "cosmos",
			Name:           "osmosis",
			ChainID:        "localosmo-1",
			Bin:            "osmosisd",
			Bech32Prefix:   "osmo",
			Denom:          "uosmo",
			GasPrices:      "0.0025uosmo",
			GasAdjustment:  1.3,
			TrustingPeriod: "336h",
			NoHostMount:    false,
			Images: []ibc.DockerImage{
				{
					Repository: "ghcr.io/strangelove-ventures/heighliner/osmosis",
					Version:    "latest",
					UIDGID:     "1025:1025",
				},
			},
		},
	}
}
