package e2e_ibc

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/burnt-labs/xion/e2e_tests/testlib"
	abcitypes "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	paramsutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/cosmos/interchaintest/v10/relayer"
	"github.com/cosmos/interchaintest/v10/testreporter"
	"github.com/cosmos/interchaintest/v10/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TestIBCMinFeeMultiDenom tests that IBC-transferred tokens can be used for gas fees
// after being added to the global fee whitelist via governance
func TestIBCMinFeeMultiDenom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		testlib.XionLocalChainSpec(t, 1, 0),
		testlib.OsmosisChainSpec(1, 0),
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

	ctx := t.Context()

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	require.NoError(t, ic.Build(ctx, rep.RelayerExecReporter(t), interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	userFunds := math.NewInt(10_000_000_000)
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, chain, counterpartyChain)
	usersB := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), userFunds, counterpartyChain)

	xionUser := users[0]
	osmoUser := usersB[0]
	currentHeight, _ := chain.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, chain)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := chain.GetBalance(ctx, xionUser.FormattedAddress(), chain.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, userFunds, xionUserBalInitial)

	// Step 2 send funds from chain B to Chain A
	xionChannelInfo, err := r.GetChannels(ctx, eRep, chain.Config().ChainID)
	require.NoError(t, err)
	xionChannelID := xionChannelInfo[0].ChannelID

	osmoUserBalInitial, err := counterpartyChain.GetBalance(ctx, osmoUser.FormattedAddress(), counterpartyChain.Config().Denom)
	require.NoError(t, err)
	require.True(t, osmoUserBalInitial.Equal(userFunds))
	amount := math.NewInt(1_000_000)

	transfer := ibc.WalletAmount{
		Address: xionUser.FormattedAddress(),
		Denom:   counterpartyChain.Config().Denom,
		Amount:  amount,
	}

	tx, err := counterpartyChain.SendIBCTransfer(ctx, xionChannelID, osmoUser.KeyName(), transfer, ibc.TransferOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Validate())
	require.NoError(t, r.Flush(ctx, eRep, testPath, xionChannelID))

	// Trace IBC Denom
	srcDenomTrace := transfertypes.ParseDenomTrace(transfertypes.GetPrefixedDenom("transfer", xionChannelID, counterpartyChain.Config().Denom))
	dstIbcDenom := srcDenomTrace.IBCDenom()

	// Test destination wallet has increased funds
	expectedBal := osmoUserBalInitial.Sub(amount)
	xionUserBalNew, err := chain.GetBalance(ctx, xionUser.FormattedAddress(), dstIbcDenom)
	t.Logf("querying address %s for denom: %s", xionUser.FormattedAddress(), dstIbcDenom)

	require.True(t, xionUserBalNew.Equal(amount), "got: %d, wanted: %d", xionUserBalNew, expectedBal)

	// step 3: upgrade minimum through governance
	rawValueBz, err := formatJSON(dstIbcDenom)
	require.NoError(t, err)

	paramChangeJSON := paramsutils.ParamChangeProposalJSON{
		Title:       "add token to globalfee",
		Description: ".",
		Changes: paramsutils.ParamChangesJSON{
			paramsutils.ParamChangeJSON{
				Subspace: "globalfee",
				Key:      "MinimumGasPricesParam",
				Value:    rawValueBz,
			},
		},
		Deposit: "10000000uxion",
	}

	content, err := json.Marshal(paramChangeJSON)
	require.NoError(t, err)

	hash := sha256.Sum256(content)
	proposalFilename := fmt.Sprintf("%x.json", hash)
	err = chain.GetNode().WriteFile(ctx, content, proposalFilename)
	require.NoError(t, err)

	proposalPath := filepath.Join(chain.GetNode().HomeDir(), proposalFilename)

	command := []string{
		"gov", "submit-legacy-proposal",
		"param-change",
		proposalPath,
		"--gas",
		"2500000",
		"--chain-id",
		chain.Config().ChainID,
	}

	txHash, err := testlib.ExecTx(t, ctx, chain.GetNode(), xionUser.KeyName(), command...)
	require.NoError(t, err)
	t.Logf("Submitted governance proposal with tx Hash: %s", txHash)

	txRes, err := chain.GetTransaction(txHash)
	require.NoError(t, err)

	evtSubmitProp := "submit_proposal"
	paramProposalIDRaw, ok := txProposal(txRes.Events, evtSubmitProp, "proposal_id")
	require.True(t, ok)
	paramProposalID, err := strconv.Atoi(paramProposalIDRaw)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, uint64(paramProposalID))
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

	err = chain.VoteOnProposalAllValidators(ctx, uint64(paramProposalID), cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := chain.GovQueryProposal(ctx, uint64(paramProposalID))
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

	// Wait for a few blocks to ensure the proposal changes take effect
	currentHeight, err = chain.Height(ctx)
	require.NoError(t, err)
	testutil.WaitForBlocks(ctx, int(currentHeight)+3, chain)

	recipientKeyName := "recipient-key"
	err = chain.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := chain.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(chain.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	// Test: paying with uxion should fail after adding IBC denom to global fee
	_, err = testlib.ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		"0.024uxion",
		"xion", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.Error(t, err)

	// Test: paying with IBC denom should succeed
	_, err = testlib.ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		fmt.Sprintf("0.024%s", dstIbcDenom),
		"bank", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.NoError(t, err)

	// Test: paying with native denom should also succeed (at higher amount)
	_, err = testlib.ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		fmt.Sprintf("0.025%s", chain.Config().Denom),
		"bank", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.NoError(t, err)
}

// formatJSON formats the IBC denom for the governance proposal
func formatJSON(ibcDenom string) (json.RawMessage, error) {
	// Note: Cosmos SDK requires denominations to be sorted lexicographically
	// IBC denoms (ibc/...) come before native denoms (uxion) alphabetically
	value := []map[string]string{
		{"denom": ibcDenom, "amount": "0.024"},
		{"denom": "uxion", "amount": "0.025"},
	}
	return json.Marshal(value)
}

// txProposal extracts a value from transaction events
func txProposal(events []abcitypes.Event, eventType, key string) (string, bool) {
	for _, event := range events {
		if event.Type == eventType {
			for _, attr := range event.Attributes {
				if attr.Key == key {
					return attr.Value, true
				}
			}
		}
	}
	return "", false
}
