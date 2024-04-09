package integration_tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v7/conformance"
	"github.com/strangelove-ventures/interchaintest/v7/relayer"
	"github.com/strangelove-ventures/interchaintest/v7/relayer/rly"

	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestXionUpgradeIBC tests a Xion software upgrade, ensuring IBC conformance prior-to and after the upgrade.
func TestXionUpgradeIBC(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Setup Docker
	dockerClient, dockerNetwork := interchaintest.DockerSetup(t)

	// Build Relayer
	rlyImage := relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "main", rly.RlyDefaultUidGid)
	rf := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t), rlyImage)
	r := rf.Build(t, dockerClient, dockerNetwork)

	// Setup loggers and reporters
	f, err := interchaintest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Configure Chains
	chains := ConfigureChains(t, 1, 2)

	// Test cases
	testCases := []struct {
		name        string
		setup       func(t *testing.T, path string) (ibc.Chain, ibc.Chain)
		conformance func(t *testing.T, ctx context.Context, client *client.Client, network string, srcChain, dstChain ibc.Chain, rf interchaintest.RelayerFactory, rep *testreporter.Reporter, relayerImpl ibc.Relayer, pathNames ...string)
	}{
		{
			name: "xion-osmosis",
			setup: func(t *testing.T, path string) (ibc.Chain, ibc.Chain) {
				xion, osmosis := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)
				setupInterchain(t, xion, osmosis, path, r, eRep, dockerClient, dockerNetwork)
				return xion, osmosis
			},
			conformance: conformance.TestChainPair,
		},
		{
			name: "xion-axelar",
			setup: func(t *testing.T, path string) (ibc.Chain, ibc.Chain) {
				xion, axelar := chains[0].(*cosmos.CosmosChain), chains[2].(*cosmos.CosmosChain)
				setupInterchain(t, xion, axelar, path, r, eRep, dockerClient, dockerNetwork)
				return xion, axelar
			},
			conformance: conformance.TestChainPair,
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chain1, chain2 := tc.setup(t, tc.name)
			tc.conformance(t, ctx, dockerClient, dockerNetwork, chain1, chain2, nil, rep, r, tc.name)
		})
	}
}

// ConfigureChains creates a list of chains with the given number of full nodes and validators.
func ConfigureChains(t *testing.T, numFullNodes, numValidators int) []ibc.Chain {
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

// setupInterchain builds an Interchain with the given chain pair and relayer path.
func setupInterchain(
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
