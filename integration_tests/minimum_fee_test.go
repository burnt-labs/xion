package integration_tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	paramsutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"

	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v10"

	"cosmossdk.io/math"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	"github.com/strangelove-ventures/interchaintest/v10/relayer"
	"github.com/strangelove-ventures/interchaintest/v10/testutil"

	"github.com/strangelove-ventures/interchaintest/v10"
	"github.com/strangelove-ventures/interchaintest/v10/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v10/ibc"
	"github.com/strangelove-ventures/interchaintest/v10/testreporter"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// TODO:
// param change test (in the upcoming interchain v10 upgrade)

func TestXionMinimumFeeDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	chainSpec := XionLocalChainSpec(t, 3, 1)
	chainSpec.GasPrices = "0.025uxion"
	xion := BuildXionChainWithSpec(t, chainSpec)

	assertion := func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, xionUser ibc.Wallet, recipientAddress string, fundAmount math.Int) {
		// NOTE: Tx should be rejected inssufficient gas
		amount := 1 // less than minimum send amount
		_, err := ExecTxWithGas(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"0.024uxion",
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", amount, xion.Config().Denom),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "minimum send amount not met")

		// NOTE: Uses default Gas
		_, err = ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.NoError(t, err)
	}

	testMinimumFee(t, xion, assertion)
}

func TestXionMinimumFeeZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	xion := BuildXionChain(t)

	assertion := func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, xionUser ibc.Wallet, recipientAddress string, fundAmount math.Int) {
		toSend := math.NewInt(100)

		// Log initial balances
		userBalBefore, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		recipientBalBefore, err := xion.GetBalance(ctx, recipientAddress, xion.Config().Denom)
		require.NoError(t, err)
		t.Logf("Before transaction - User balance: %s, Recipient balance: %s", userBalBefore, recipientBalBefore)

		txHash, err := ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", toSend.Int64(), xion.Config().Denom),
		)
		require.NoError(t, err)
		t.Logf("Transaction hash: %s", txHash)

		// Get transaction details to verify it succeeded
		txRes, err := xion.GetTransaction(txHash)
		require.NoError(t, err)
		t.Logf("Transaction code: %d, logs: %s", txRes.Code, txRes.RawLog)
		require.Equal(t, uint32(0), txRes.Code, "Transaction should succeed")

		// Wait for a few more blocks to ensure transaction is fully processed
		currentHeight, err := xion.Height(ctx)
		require.NoError(t, err)
		testutil.WaitForBlocks(ctx, int(currentHeight)+2, xion)

		balance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		t.Logf("After transaction - User balance: %s, Expected: %s", balance, fundAmount.Sub(toSend))
		require.Equal(t, fundAmount.Sub(toSend), balance)

		balance, err = xion.GetBalance(ctx, recipientAddress, xion.Config().Denom)
		require.NoError(t, err)
		t.Logf("After transaction - Recipient balance: %s, Expected: %s", balance, toSend)
		require.Equal(t, toSend, balance)
	}

	testMinimumFee(t, xion, assertion)
}

func testMinimumFee(t *testing.T, xion *cosmos.CosmosChain, assert assertionFn) {
	ctx := t.Context()

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	currentHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, xionUserBalInitial)

	cdc := codec.NewProtoCodec(xion.Config().EncodingConfig.InterfaceRegistry)
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// Convert gov module address to the correct bech32 prefix
	govModuleAddr := authtypes.NewModuleAddress("gov")
	authorityAddr, err := types.Bech32ifyAddressBytes("xion", govModuleAddr)
	require.NoError(t, err)

	setPlatformMinimumsMsg := xiontypes.MsgSetPlatformMinimum{
		Authority: authorityAddr,
		Minimums:  types.Coins{types.Coin{Amount: math.NewInt(10), Denom: "uxion"}},
	}

	msg, err := cdc.MarshalInterfaceJSON(&setPlatformMinimumsMsg)
	require.NoError(t, err)

	prop := cosmos.TxProposalv1{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Set platform percentage to 5%",
		Summary:  "Ups the platform fee to 5% for the integration test",
	}
	paramChangeTx, err := xion.SubmitProposal(ctx, xionUser.KeyName(), prop)
	require.NoError(t, err)
	t.Logf("Platform percentage change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	proposalID, err := strconv.Atoi(paramChangeTx.ProposalID)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(proposalID))
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

	err = xion.VoteOnProposalAllValidators(ctx, uint64(proposalID), cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(proposalID))
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

	// Wait for several blocks to ensure the proposal changes take effect
	currentHeight, err = xion.Height(ctx)
	require.NoError(t, err)
	testutil.WaitForBlocks(ctx, int(currentHeight)+5, xion)

	// Verify that the platform minimum has been set correctly
	minimums, err := ExecQuery(t, ctx, xion.GetNode(), "xion", "platform-minimum")
	require.NoError(t, err)
	t.Logf("Platform minimums after proposal: %v", minimums)

	// step 1: send a xion message with default (0%) platform fee
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	assert(t, ctx, xion, xionUser, recipientKeyAddress, fundAmount)
}

type assertionFn func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, wallet ibc.Wallet, recipientAddress string, fundAmount math.Int)

func TestMultiDenomMinGlobalFee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	spec := XionLocalChainSpec(t, 1, 0)
	spec.GasPrices = "0.025uxion"
	xion := BuildXionChainWithSpec(t, spec)
	// Add new denomination

	assertion := func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, xionUser ibc.Wallet, recipientAddress string, fundAmount math.Int) {
		// NOTE: Tx should be rejected inssufficient gas
		_, err := ExecTxWithGas(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"0.024uxion",
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.Error(t, err)

		// NOTE: Uses default Gas
		_, err = ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.NoError(t, err)
		_, err = ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"bank", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.NoError(t, err)
	}

	testMultiDenomFee(t, xion, assertion)
}

func TestMultiDenomMinGlobalFeeIBC(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	chains := interchaintest.CreateChainsWithChainSpecs(t, []*interchaintest.ChainSpec{
		XionLocalChainSpec(t, 1, 0),
		OsmosisChainSpec(1, 0),
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
	eRep := rep.RelayerExecReporter(t)

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
	//
	// test source wallet has decreased funds
	//

	// Tracen IBC Denom
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
	}

	txHash, err := chain.GetNode().ExecTx(ctx, xionUser.KeyName(), command...)
	require.NoError(t, err)
	t.Logf("Failed submitting governance proposal with tx Hash: %s", txHash)

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

	_, err = ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),
		"0.024uxion",
		"xion", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.Error(t, err)

	_, err = ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),

		fmt.Sprintf("0.024%s", dstIbcDenom),
		"bank", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.NoError(t, err)

	_, err = ExecTxWithGas(t, ctx, chain.GetNode(),
		xionUser.KeyName(),

		fmt.Sprintf("0.025%s", chain.Config().Denom),
		"bank", "send", xionUser.KeyName(),
		"--chain-id", chain.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, chain.Config().Denom),
	)
	require.NoError(t, err)
}

func testMultiDenomFee(t *testing.T, xion *cosmos.CosmosChain, assert assertionFn) {
	ctx := t.Context()

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := math.NewInt(100_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
	currentHeight, _ := xion.Height(ctx)
	testutil.WaitForBlocks(ctx, int(currentHeight)+8, xion)
	t.Logf("created xion user %s", xionUser.FormattedAddress())

	xionUserBalInitial, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	log.Printf("Initial balance of user %s: %s", xionUser.FormattedAddress(), xionUserBalInitial.String()+xion.Config().Denom)
	require.Equal(t, fundAmount, xionUserBalInitial)

	cdc := codec.NewProtoCodec(xion.Config().EncodingConfig.InterfaceRegistry)
	config := types.GetConfig()
	config.SetBech32PrefixForAccount(xion.Config().Bech32Prefix, xion.Config().Bech32Prefix+"pub")

	// Convert gov module address to the correct bech32 prefix
	govModuleAddr := authtypes.NewModuleAddress("gov")
	authorityAddr, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, govModuleAddr)
	require.NoError(t, err)

	setPlatformMinimumsMsg := xiontypes.MsgSetPlatformMinimum{
		Authority: authorityAddr,
		Minimums:  types.Coins{types.Coin{Amount: math.NewInt(10), Denom: xion.Config().Denom}},
	}

	msg, err := cdc.MarshalInterfaceJSON(&setPlatformMinimumsMsg)
	require.NoError(t, err)

	prop := cosmos.TxProposalv1{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Set platform percentage to 5%",
		Summary:  "Ups the platform fee to 5% for the integration test",
	}
	paramChangeTx, err := xion.SubmitProposal(ctx, xionUser.KeyName(), prop)
	require.NoError(t, err)
	t.Logf("Platform percentage change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	proposalID, err := strconv.Atoi(paramChangeTx.ProposalID)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(proposalID))
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

	err = xion.VoteOnProposalAllValidators(ctx, uint64(proposalID), cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(proposalID))
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
	currentHeight, err = xion.Height(ctx)
	require.NoError(t, err)
	testutil.WaitForBlocks(ctx, int(currentHeight)+3, xion)

	// step 1: send a xion message with default (0%) platform fee
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	// step 2: create new denomination
	subDenom := "burnt"
	tfDenom, _, err := xion.GetNode().TokenFactoryCreateDenom(ctx, xionUser, subDenom, 2500000)
	require.NoError(t, err)
	require.Equal(t, tfDenom, "factory/"+xionUser.FormattedAddress()+"/"+subDenom)

	// modify metadata
	stdout, err := xion.GetNode().TokenFactoryMetadata(ctx, xionUser.KeyName(), tfDenom, "SYMBOL", "description here", 6)
	t.Log(stdout, err)
	require.NoError(t, err)

	md, _, err := xion.GetNode().ExecQuery(ctx, "bank", "denom-metadata", tfDenom)
	require.NoError(t, err)

	var meta cosmos.BankMetaData
	err = json.Unmarshal(md, &meta)
	require.NoError(t, err)

	require.Equal(t, meta.Metadata.Description, "description here")
	require.Equal(t, meta.Metadata.Symbol, "SYMBOL")
	require.Equal(t, meta.Metadata.DenomUnits[1].Exponent, 6)

	// mint tokens
	_, err = xion.GetNode().TokenFactoryMintDenom(ctx, xionUser.KeyName(), tfDenom, 10)
	require.NoError(t, err)

	balance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), tfDenom)
	require.NoError(t, err)
	require.Equal(t, balance, math.NewInt(10))

	// step 3: upgrade minimum through governance
	rawValueBz, err := formatJSON(tfDenom)
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
	err = xion.GetNode().WriteFile(ctx, content, proposalFilename)
	require.NoError(t, err)

	proposalPath := filepath.Join(xion.GetNode().HomeDir(), proposalFilename)

	command := []string{
		"gov", "submit-legacy-proposal",
		"param-change",
		proposalPath,
		"--gas",
		"2500000",
	}

	txHash, err := xion.GetNode().ExecTx(ctx, xionUser.KeyName(), command...)
	require.NoError(t, err)
	t.Logf("Platform percentage change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	txRes, err := xion.GetTransaction(txHash)
	require.NoError(t, err)

	evtSubmitProp := "submit_proposal"
	paramProposalIDRaw, ok := txProposal(txRes.Events, evtSubmitProp, "proposal_id")
	require.True(t, ok)
	paramProposalID, err := strconv.Atoi(paramProposalIDRaw)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(paramProposalID))
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

	err = xion.VoteOnProposalAllValidators(ctx, uint64(paramProposalID), cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.GovQueryProposal(ctx, uint64(paramProposalID))
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
	currentHeight, err = xion.Height(ctx)
	require.NoError(t, err)
	testutil.WaitForBlocks(ctx, int(currentHeight)+3, xion)

	assert(t, ctx, xion, xionUser, recipientKeyAddress, fundAmount)
}

func txProposal(events []abcitypes.Event, eventType, attrKey string) (string, bool) {
	for _, event := range events {
		if event.Type != eventType {
			continue
		}
		for _, attr := range event.Attributes {
			if attr.Key == attrKey {
				return attr.Value, true
			}

			// tendermint < v0.37-alpha returns base64 encoded strings in events.
			key, err := base64.StdEncoding.DecodeString(attr.Key)
			if err != nil {
				continue
			}
			if string(key) == attrKey {
				value, err := base64.StdEncoding.DecodeString(attr.Value)
				if err != nil {
					continue
				}
				return string(value), true
			}
		}
	}
	return "", false
}

type DenomAmount struct {
	Denom  string `json:"denom"`
	Amount string `json:"amount"`
}

func formatJSON(tfDenom string) ([]byte, error) {
	data := []DenomAmount{
		{Denom: tfDenom, Amount: "0.005000000000000000"},
		{Denom: "uxion", Amount: "0.025000000000000000"},
	}
	return json.Marshal(data)
}
