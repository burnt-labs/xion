package integration_tests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/testutil"
	"github.com/stretchr/testify/require"
)

func TestXionSendPlatformFee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	td := BuildXionChain(t, "0.0uxion", ModifyInterChainGenesis(ModifyInterChainGenesisFn{ModifyGenesisShortProposals}, [][]string{{votingPeriod, maxDepositPeriod}}))
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

	// step 1: send a xion message with default (0%) platform fee
	recipientKeyName := "recipient-key"
	err = xion.CreateKey(ctx, recipientKeyName)
	require.NoError(t, err)
	receipientKeyAddressBytes, err := xion.GetAddress(ctx, recipientKeyName)
	require.NoError(t, err)
	recipientKeyAddress, err := types.Bech32ifyAddressBytes(xion.Config().Bech32Prefix, receipientKeyAddressBytes)
	require.NoError(t, err)

	cdc := codec.NewProtoCodec(xion.Config().EncodingConfig.InterfaceRegistry)
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	// query to make sure minimums are empty

	minimums, err := ExecQuery(t, ctx, xion.GetNode(), "xion", "platform-minimum")
	require.NoError(t, err)
	t.Log(minimums)
	require.Equal(t, []interface{}{}, minimums["minimums"])

	setPlatformMinimumsMsg := xiontypes.MsgSetPlatformMinimum{
		Authority: authtypes.NewModuleAddress("gov").String(),
		Minimums:  types.Coins{types.Coin{Amount: math.NewInt(10), Denom: "uxion"}},
	}

	msg, err := cdc.MarshalInterfaceJSON(&setPlatformMinimumsMsg)
	require.NoError(t, err)

	_, err = xion.GetNode().ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
	)
	// platform minimums unset, so this should fail
	require.Error(t, err)

	prop := cosmos.TxProposalv1{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Set platform minimum to 100uxion",
		Summary:  "Ups the platform minimum to 100uxion for the integration test",
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

	// check that the value has been set

	minimums, err = ExecQuery(t, ctx, xion.GetNode(), "xion", "platform-minimum")
	require.NoError(t, err)
	coins := minimums["minimums"].([]interface{})
	require.Equal(t, 1, len(coins))

	_, err = xion.GetNode().ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
	)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
		require.NoError(t, err)
		t.Logf("expected %d, got %d", 100, balance.Int64())
		return balance.Equal(math.NewInt(100))
	},
		time.Second*20,
		time.Second*6,
		"balance never correctly changed")

	// step 2: update the platform percentage to 5%

	setPlatformPercentageMsg := xiontypes.MsgSetPlatformPercentage{
		Authority:          authtypes.NewModuleAddress("gov").String(),
		PlatformPercentage: 500,
	}
	msg, err = cdc.MarshalInterfaceJSON(&setPlatformPercentageMsg)
	require.NoError(t, err)

	prop = cosmos.TxProposalv1{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Set platform percentage to 5%",
		Summary:  "Ups the platform fee to 5% for the integration test",
	}
	paramChangeTx, err = xion.SubmitProposal(ctx, xionUser.KeyName(), prop)
	require.NoError(t, err)
	t.Logf("Platform percentage change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	proposalID, err = strconv.Atoi(paramChangeTx.ProposalID)
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

	// step 3: transfer and verify platform fees is extracted
	initialSendingBalance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	initialReceivingBalance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(100), initialReceivingBalance)

	_, err = xion.GetNode().ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 200, xion.Config().Denom),
	)

	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, int(currentHeight)+100, xion)
	require.NoError(t, err)

	postSendingBalance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equalf(t, initialSendingBalance.SubRaw(200), postSendingBalance, "Wanted %d, got %d", initialSendingBalance.SubRaw(200), postSendingBalance)
	postReceivingBalance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(290), postReceivingBalance)
}
