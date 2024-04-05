package integration_tests

import (
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"go.uber.org/zap/zaptest"
	"testing"
)

// requires building locally; private ibc-go dependency
func TestXionUpgrade_038_039(t *testing.T) {
	t.Parallel()

	CosmosChainUpgrade_038_039(t, "xion", "v0.3.8", "github.com/burnt-labs/xion/xion", "v0.3.9", "v5")
}

func CosmosChainUpgrade_038_039(t *testing.T, chainName, initialVersion, upgradeContainerRepo, upgradeVersion string, upgradeName string) {

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			Name:    chainName,
			Version: initialVersion,
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: "github.com/burnt-labs/xion/xion",
						Version:    initialVersion,
						UidGid:     "1025:1025",
					},
				},
				GasPrices:              "0.0uxion",
				GasAdjustment:          1.4,
				Type:                   "cosmos",
				ChainID:                "xion-1",
				Bin:                    "xiond",
				Bech32Prefix:           "xion",
				Denom:                  "uxion",
				TrustingPeriod:         "336h",
				ModifyGenesis:          xiontests.ModifyInterChainGenesisFn{ModifyGenesisShortProposals, ModifyGenesispacketForwardMiddleware}, [][]string{{votingPeriod, maxDepositPeriod}, {packetforward}}),
				UsingNewGenesisCommand: true,
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})
}
