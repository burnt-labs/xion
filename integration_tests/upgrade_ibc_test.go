package integration_tests

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v10/conformance"
	"github.com/strangelove-ventures/interchaintest/v10/relayer"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"

	"github.com/burnt-labs/xion/integration_tests/helpers"
	"github.com/strangelove-ventures/interchaintest/v10"
	"github.com/strangelove-ventures/interchaintest/v10/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v10/ibc"
	"github.com/strangelove-ventures/interchaintest/v10/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestXionUpgradeIBC tests a Xion software upgrade, ensuring IBC conformance prior-to and after the upgrade.
func TestXionUpgradeIBC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	// Get the "from" image (current version in repo)
	xionFromImage, err := helpers.GetGHCRPackageNameCurrentRepo()
	require.NoError(t, err)

	// Parse the from image to extract repository and version
	xionFromImageParts := strings.SplitN(xionFromImage, ":", 2)
	require.GreaterOrEqual(t, len(xionFromImageParts), 2, "xionFromImage should have repository:tag format")

	xionVersionFrom := xionFromImageParts[1]

	// Get the "to" image (local image) which is where we want to upgrade to
	xionToImageParts, err := GetXionImageTagComponents()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(xionToImageParts), 2, "xionToImage should have repository:tag format")

	xionImageTo := xionToImageParts[0]
	xionVersionTo := xionToImageParts[1]

	// Use "recent" as upgrade name for local builds, otherwise use version-based name
	xionUpgradeName := "recent"
	if xionVersionTo != "local" {
		// For non-local builds, use version as upgrade name (e.g., "v20")
		xionUpgradeName = xionVersionTo
	}

	// Constants for the test
	haltHeightDelta := int64(9) // how many blocks after current height to upgrade
	blocksAfterUpgrade := int64(7)

	// Create chain specs using helper functions
	xionChainSpec := XionChainSpec(3, 1)
	xionChainSpec.Version = xionVersionFrom

	// Add additional genesis modifications for IBC test
	xionChainSpec.ChainConfig.ModifyGenesis = cosmos.ModifyGenesis(append(defaultGenesisKVMods,
		// Globalfee - specific to IBC tests
		cosmos.NewGenesisKV("app_state.globalfee.params.minimum_gas_prices", []map[string]string{{"denom": "uxion", "amount": "0"}}),
	))

	osmosisChainSpec := OsmosisChainSpec(1, 0)

	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		xionChainSpec,
		osmosisChainSpec,
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
