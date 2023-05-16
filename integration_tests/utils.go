package integration_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/icza/dyno"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func BuildXionChain(t *testing.T) (*cosmos.CosmosChain, context.Context) {
	ctx := context.Background()

	var numFullNodes = 1
	var numValidators = 3

	// pulling image from env to foster local dev
	imageTag := os.Getenv("XION_IMAGE")
	imageTagComponents := strings.Split(imageTag, ":")

	// Chain factory
	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:    imageTagComponents[0],
			Version: imageTagComponents[1],
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: imageTagComponents[0],
						Version:    imageTagComponents[1],
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
				ModifyGenesis:          modifyGenesisShortProposals(votingPeriod, maxDepositPeriod),
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

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
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false},
	),
	)
	return xion, ctx
}

const (
	votingPeriod     = "10s"
	maxDepositPeriod = "10s"
)

func modifyGenesisShortProposals(votingPeriod string, maxDepositPeriod string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, "100", "app_state", "gov", "params", "min_deposit", 0, "amount"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, "0", "app_state", "mint", "params", "inflation_min"); err != nil {
			return nil, fmt.Errorf("failed to set inflation in genesis json: %w", err)
		}
		if err := dyno.Set(g, "0", "app_state", "mint", "params", "inflation_max"); err != nil {
			return nil, fmt.Errorf("failed to set inflation in genesis json: %w", err)
		}
		if err := dyno.Set(g, "0", "app_state", "mint", "params", "inflation_rate_change"); err != nil {
			return nil, fmt.Errorf("failed to set rate of inflation change in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
