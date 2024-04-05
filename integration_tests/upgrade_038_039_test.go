package integration_tests

import (
	"context"
	"fmt"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/conformance"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/relayer/rly"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"os"
	"testing"
	"time"
)

// TestXionUpgrade_038_039 ensures that IBC connections to counterparties are not broken after a software upgrade from v0.3.8 to v0.3.9
func TestXionUpgrade_038_039(t *testing.T) {
	t.Parallel()

	chains := ConfigureChains(t)
	ic := SetupInterchain(t, chains)
	t.Cleanup(func() {
		_ = ic.Close()
	})
}

func ConfigureChains(t *testing.T) []ibc.Chain {

	numValidators := 2
	numFullNodes := 1

	// must override Axelar's default override NoHostMount in yaml
	// otherwise fails on `cp` on heighliner img as it's not available in the container
	f := OverrideConfiguredChainsYaml(t)
	defer os.Remove(f.Name())

	// Chain factory
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:    "xion",
			Version: "v0.3.8",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/burnt-labs/xion/xion",
						Version:    "v0.3.8",
						UidGid:     "1025:1025",
					},
				},
				GasPrices:              "0.0uxion",
				GasAdjustment:          1.3,
				Type:                   "cosmos",
				ChainID:                "xion-1",
				Bin:                    "xiond",
				Bech32Prefix:           "xion",
				Denom:                  "uxion",
				TrustingPeriod:         "336h",
				NoHostMount:            false,
				ModifyGenesis:          ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}),
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
		{
			Name:    "osmosis",
			Version: "v24.0.0-rc0",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/strangelove-ventures/heighliner/osmosis",
						Version:    "v24.0.0-rc0",
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
		{
			Name:    "axelar",
			Version: "v0.35.3",
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "ghcr.io/strangelove-ventures/heighliner/axelar",
						Version:    "v0.35.3",
						UidGid:     "1025:1025",
					},
				},
				Type:           "cosmos",
				Bin:            "axelard",
				Bech32Prefix:   "axelar",
				Denom:          "uaxl",
				GasPrices:      "0.007uaxl",
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

// SetupInterchain sets up the interchain test environment
func SetupInterchain(t *testing.T, chains []ibc.Chain) *interchaintest.Interchain {
	ctx := context.Background()
	const rlyXionOsmosisPath = "xion-osmosis"
	const rlyXionAxelarPath = "xion-axelar"

	xion, osmosis, axelar := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain), chains[2].(*cosmos.CosmosChain)

	// Build relayer instance
	client, network := interchaintest.DockerSetup(t)
	rlyImage := relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main", rly.RlyDefaultUidGid)
	rf := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t), rlyImage)
	r1 := rf.Build(t, client, network)
	r2 := rf.Build(t, client, network)

	// Prep Interchain
	ic := interchaintest.NewInterchain().
		AddChain(xion).
		AddChain(osmosis).
		AddChain(axelar).
		AddRelayer(r1, "r1").
		AddRelayer(r2, "r2").
		AddLink(interchaintest.InterchainLink{
			Chain1:  xion,
			Chain2:  osmosis,
			Relayer: r1,
			Path:    rlyXionOsmosisPath,
		}).
		AddLink(interchaintest.InterchainLink{
			Chain1:  xion,
			Chain2:  axelar,
			Relayer: r2,
			Path:    rlyXionAxelarPath,
		})

	// Setup loggers
	f, err := interchaintest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build Interchain
	err = ic.Build(context.Background(), eRep, interchaintest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation:  false,
	})
	require.NoError(t, err)

	// Fund users on all chains
	fundAmount := int64(10_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion, osmosis, axelar)
	xionUser, osmoUser, axelarUser := users[0], users[1], users[2]
	t.Logf("created xion user %s", xionUser.FormattedAddress())
	t.Logf("created osmosis user %s", osmoUser.FormattedAddress())
	t.Logf("created axelar user %s", axelarUser.FormattedAddress())

	// test IBC conformance
	conformance.TestChainPair(t, ctx, client, network, xion, osmosis, rf, rep, r1, rlyXionOsmosisPath)
	conformance.TestChainPair(t, ctx, client, network, xion, axelar, rf, rep, r2, rlyXionAxelarPath)

	return ic
}
