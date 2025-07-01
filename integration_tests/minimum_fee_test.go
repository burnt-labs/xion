package integration_tests

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	"cosmossdk.io/math"

	abcitypes "github.com/cometbft/cometbft/abci/types"
	paramsutils "github.com/cosmos/cosmos-sdk/x/params/client/utils"

	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

// TODO:
// param change test (in the upcoming interchain v8 upgrade)

func TestXionMinimumFeeDefault(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.025uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}, {defaultMinGasPrices.String()}}))

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

	testMinimumFee(t, &td, assertion)
}

func TestXionMinimumFeeZero(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()
	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}, {defaultMinGasPrices.String()}}))

	assertion := func(t *testing.T, ctx context.Context, xion *cosmos.CosmosChain, xionUser ibc.Wallet, recipientAddress string, fundAmount math.Int) {
		toSend := math.NewInt(100)

		_, err := ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"xion", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", toSend.Int64(), xion.Config().Denom),
		)
		require.NoError(t, err)

		balance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, fundAmount.Sub(toSend), balance)

		balance, err = xion.GetBalance(ctx, recipientAddress, xion.Config().Denom)
		require.NoError(t, err)
		require.Equal(t, toSend, balance)
	}

	testMinimumFee(t, &td, assertion)
}

func testMinimumFee(t *testing.T, td *TestData, assert assertionFn) {
	xion, ctx := td.xionChain, td.ctx

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

	setPlatformMinimumsMsg := xiontypes.MsgSetPlatformMinimum{
		Authority: authtypes.NewModuleAddress("gov").String(),
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
	td := BuildXionChain(t, "0.025uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}, {defaultMinGasPrices.String()}}))
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
		hash, err := ExecTx(t, ctx, xion.GetNode(),
			xionUser.KeyName(),
			"bank", "send", xionUser.KeyName(),
			"--chain-id", xion.Config().ChainID,
			recipientAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
		)
		require.NoError(t, err)
		fmt.Printf("we are waiting, this is the hash: %s", hash)
		time.Sleep(10 * time.Minute)
	}

	testMultiDenomFee(t, &td, assertion)
}

func testMultiDenomFee(t *testing.T, td *TestData, assert assertionFn) {
	xion, ctx := td.xionChain, td.ctx

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
	require.Equal(t, fundAmount, xionUserBalInitial)

	cdc := codec.NewProtoCodec(xion.Config().EncodingConfig.InterfaceRegistry)
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	setPlatformMinimumsMsg := xiontypes.MsgSetPlatformMinimum{
		Authority: authtypes.NewModuleAddress("gov").String(),
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

	// verify metadata
	// md, err := xion.GetNode().QueryBankMetadata(ctx, tfDenom)

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

	// mint-to
	// step 3: upgrade minimum through governance
	//
	//
	rawValueBz, err := formatJSON(tfDenom)
	require.NoError(t, err)
	/*
		rawValue := fmt.Sprintf("[{\"denom\":\"%s\",\"amount\":\"0.005000000000000000\"},{\"denom\":\"uxion\",\"amount\":\"0.025000000000000000\"}]", tfDenom)

		paramChange := proposal.ParameterChangeProposal{
			Title:       "add token to globalfee",
			Description: ".",
			Changes: []proposal.ParamChange{
				{
					Subspace: "globalfee",
					Key:      "MinimumGasPricesParam",
					Value:    rawValue,
				},
			},
		}
		msg, err = cdc.MarshalInterfaceJSON(&paramChange)
		require.NoError(t, err)

		prop = cosmos.TxProposalv1{
			Messages: []json.RawMessage{msg},
			Metadata: "",
			Deposit:  "100uxion",
			Title:    "add token to globalfee",
			Summary:  ".",
		}
	*/

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
