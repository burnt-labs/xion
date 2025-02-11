package integration_tests

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/x/upgrade"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/burnt-labs/xion/x/jwk"
	"github.com/burnt-labs/xion/x/mint"
	"github.com/burnt-labs/xion/x/xion"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authz "github.com/cosmos/cosmos-sdk/x/authz/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/consensus"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	"github.com/cosmos/cosmos-sdk/x/gov"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/ibc-go/modules/capability"
	ibcwasm "github.com/cosmos/ibc-go/modules/light-clients/08-wasm"
	"github.com/cosmos/ibc-go/v8/modules/apps/transfer"
	ibccore "github.com/cosmos/ibc-go/v8/modules/core"
	ibcsolomachine "github.com/cosmos/ibc-go/v8/modules/light-clients/06-solomachine"
	ibctm "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
	ibclocalhost "github.com/cosmos/ibc-go/v8/modules/light-clients/09-localhost"
	ccvprovider "github.com/cosmos/interchain-security/v5/x/ccv/provider"
	aa "github.com/larry0x/abstract-account/x/abstractaccount"
	"github.com/strangelove-ventures/tokenfactory/x/tokenfactory"

	"cosmossdk.io/math"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v8/conformance"
	"github.com/strangelove-ventures/interchaintest/v8/relayer"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"

	"github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

const (
	xionImageFrom   = "ghcr.io/burnt-labs/xion/heighliner"
	xionVersionFrom = "12.0.1"
	xionImageTo     = "xion"
	xionVersionTo   = "local"
	xionUpgradeName = "v13"

	osmosisImage   = "ghcr.io/strangelove-ventures/heighliner/osmosis"
	osmosisVersion = "v25.2.1"

	ibcClientTrustingPeriod = "336h"
)

// TestXionUpgradeIBC tests a Xion software upgrade, ensuring IBC conformance prior-to and after the upgrade.
func TestXionUpgradeIBC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		{
			Name:    "xion",
			Version: xionVersionFrom,
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: xionImageFrom,
						Version:    xionVersionFrom,
						UidGid:     "1025:1025",
					},
				},
				GasPrices:      "0.0uxion",
				GasAdjustment:  1.3,
				Type:           "cosmos",
				ChainID:        "xion-1",
				Bin:            "xiond",
				Bech32Prefix:   "xion",
				Denom:          "uxion",
				TrustingPeriod: ibcClientTrustingPeriod,
				EncodingConfig: func() *moduletestutil.TestEncodingConfig {
					cfg := moduletestutil.MakeTestEncodingConfig(
						auth.AppModuleBasic{},
						genutil.NewAppModuleBasic(genutiltypes.DefaultMessageValidator),
						bank.AppModuleBasic{},
						capability.AppModuleBasic{},
						staking.AppModuleBasic{},
						mint.AppModuleBasic{},
						distr.AppModuleBasic{},
						gov.NewAppModuleBasic(
							[]govclient.ProposalHandler{
								paramsclient.ProposalHandler,
							},
						),
						params.AppModuleBasic{},
						slashing.AppModuleBasic{},
						upgrade.AppModuleBasic{},
						consensus.AppModuleBasic{},
						transfer.AppModuleBasic{},
						ibccore.AppModuleBasic{},
						ibctm.AppModuleBasic{},
						ibcwasm.AppModuleBasic{},
						ccvprovider.AppModuleBasic{},
						ibcsolomachine.AppModuleBasic{},

						// custom
						wasm.AppModuleBasic{},
						authz.AppModuleBasic{},
						tokenfactory.AppModuleBasic{},
						xion.AppModuleBasic{},
						jwk.AppModuleBasic{},
						aa.AppModuleBasic{},
					)
					// TODO: add encoding types here for the modules you want to use
					ibclocalhost.RegisterInterfaces(cfg.InterfaceRegistry)
					return &cfg
				}(),

				ModifyGenesis: ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}),
			},
		},
		{
			Name:    "osmosis",
			Version: osmosisVersion,
			ChainConfig: ibc.ChainConfig{
				Images: []ibc.DockerImage{
					{
						Repository: osmosisImage,
						Version:    osmosisVersion,
						UidGid:     "1025:1025",
					},
				},
				Type:           "cosmos",
				Bin:            "osmosisd",
				Bech32Prefix:   "osmo",
				Denom:          "uosmo",
				GasPrices:      "0.025uosmo",
				GasAdjustment:  1.3,
				TrustingPeriod: ibcClientTrustingPeriod,
			},
		},
	})

	client, network := interchaintest.DockerSetup(t)

	chain, counterpartyChain := chains[0].(*cosmos.CosmosChain), chains[1].(*cosmos.CosmosChain)

	const (
		testPath    = "ibc-upgrade-test-testPath"
		relayerName = "relayer"
	)

	// Get a relayer instance
	rf := interchaintest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.StartupFlags("-b", "100"),
	)

	r := rf.Build(t, client, network)

	ic := interchaintest.NewInterchain().
		AddChain(chain).
		AddChain(counterpartyChain).
		AddRelayer(r, relayerName).
		AddLink(interchaintest.InterchainLink{
			Chain1:  chain,
			Chain2:  counterpartyChain,
			Relayer: r,
			Path:    testPath,
		})

	ctx := context.Background()

	rep := testreporter.NewNopReporter()

	require.NoError(t, ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
		TestName:  t.Name(),
		Client:    client,
		NetworkID: network,
		// BlockDatabaseFile: interchaintest.DefaultBlockDatabaseFilepath(),
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, chain)
	chainUser := users[0]

	// deploy the account contract, and pin it
	fp, err := os.Getwd()
	require.NoError(t, err)
	codeIDStr, err := chain.StoreContract(ctx, chainUser.FormattedAddress(),
		path.Join(fp, "integration_tests", "testdata", "contracts", "account_updatable-aarch64.wasm"))
	require.NoError(t, err)

	authority, err := chain.UpgradeQueryAuthority(ctx)
	require.NoError(t, err)
	codeID, err := strconv.Atoi(codeIDStr)
	require.NoError(t, err)

	pinCodeMsg := wasmtypes.MsgPinCodes{
		Authority: authority,
		CodeIDs:   []uint64{uint64(codeID)},
	}
	msg, err := chain.Config().EncodingConfig.Codec.MarshalInterfaceJSON(&pinCodeMsg)
	require.NoError(t, err)

	pinCodeTx, err := chain.SubmitProposal(ctx, chainUser.KeyName(), cosmos.TxProposalv1{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Pin AA Contract Code",
		Summary:  "To verify that the wasm cache doesn't move or change during upgrade",
	})
	require.NoError(t, err)

	proposalID, err := strconv.Atoi(pinCodeTx.ProposalID)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, uint64(proposalID))
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == govv1beta1.StatusVotingPeriod {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status VOTING, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status VOTING after 11s")

	err = chain.VoteOnProposalAllValidators(ctx, uint64(proposalID), cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, uint64(proposalID))
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == govv1beta1.StatusPassed {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status PASSED, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status PASSED after 11s")

	// test IBC conformance before chain upgrade
	conformance.TestChainPair(t, ctx, client, network, chain, counterpartyChain, rf, rep, r, testPath)

	height, err := chain.Height(ctx)
	require.NoError(t, err, "error fetching height before submit upgrade proposal")

	haltHeight := height + haltHeightDelta

	proposal := cosmos.SoftwareUpgradeProposal{
		Deposit:     "500000000" + chain.Config().Denom, // greater than min deposit
		Title:       "Chain Upgrade 1",
		Name:        xionUpgradeName,
		Description: "First chain software upgrade",
		Height:      haltHeight,
	}

	upgradeTx, err := chain.UpgradeProposal(ctx, chainUser.KeyName(), proposal)
	require.NoError(t, err, "error submitting software upgrade proposal tx")

	propId, err := strconv.ParseUint(upgradeTx.ProposalID, 10, 64)
	require.NoError(t, err, "failed to convert proposal ID to uint64")

	err = chain.VoteOnProposalAllValidators(ctx, propId, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	_, err = cosmos.PollForProposalStatus(ctx, chain, height, height+haltHeightDelta, propId, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	// this should timeout due to chain halt at upgrade height.
	_ = testutil.WaitForBlocks(timeoutCtx, int(haltHeight-height)+1, chain)

	height, err = chain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// make sure that chain is halted
	require.Equal(t, haltHeight, height, "height is not equal to halt height")

	// bring down nodes to prepare for upgrade
	err = chain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	chain.UpgradeVersion(ctx, client, xionImageTo, xionVersionTo)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = chain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel = context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), chain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	// test IBC conformance after chain upgrade on same path
	conformance.TestChainPair(t, ctx, client, network, chain, counterpartyChain, rf, rep, r, testPath)
}
