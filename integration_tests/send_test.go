package integration_tests

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	xiontypes "github.com/burnt-labs/xion/x/xion/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
)

func TestXionSendPlatformFee(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	xion, ctx := BuildXionChain(t)

	// Create and Fund User Wallets
	t.Log("creating and funding user accounts")
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, xion)
	xionUser := users[0]
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

	xion.Config().EncodingConfig.InterfaceRegistry.RegisterImplementations(
		(*types.Msg)(nil),
		&xiontypes.MsgSetPlatformPercentage{},
		&xiontypes.MsgSend{},
	)
	cdc := codec.NewProtoCodec(xion.Config().EncodingConfig.InterfaceRegistry)

	_, err = xion.FullNodes[0].ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 100, xion.Config().Denom),
	)
	require.NoError(t, err)
	balance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(100), uint64(balance))

	// step 2: update the platform percentage to 5%
	config := types.GetConfig()
	config.SetBech32PrefixForAccount("xion", "xionpub")

	setPlatformPercentageMsg := xiontypes.MsgSetPlatformPercentage{
		Authority:          authtypes.NewModuleAddress("gov").String(),
		PlatformPercentage: 500,
	}

	msg, err := cdc.MarshalInterfaceJSON(&setPlatformPercentageMsg)
	require.NoError(t, err)

	prop := cosmos.TxProposalv1{
		Messages: []json.RawMessage{msg},
		Metadata: "",
		Deposit:  "100uxion",
		Title:    "Set platform percentage to 5%",
		Summary:  "Ups the platform fee to 5% for the integration test",
	}
	paramChangeTx, err := xion.SubmitProposal(ctx, xionUser.KeyName(), &prop)
	require.NoError(t, err)
	t.Logf("Platform percentage change proposal submitted with ID %s in transaction %s", paramChangeTx.ProposalID, paramChangeTx.TxHash)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusVotingPeriod {
				return true
			}
			t.Logf("Waiting for proposal to enter voting status VOTING, current status: %s", proposalInfo.Status)
		}
		return false
	}, time.Second*11, time.Second, "failed to reach status VOTING after 11s")

	err = xion.VoteOnProposalAllValidators(ctx, paramChangeTx.ProposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err)

	require.Eventuallyf(t, func() bool {
		proposalInfo, err := xion.QueryProposal(ctx, paramChangeTx.ProposalID)
		if err != nil {
			require.NoError(t, err)
		} else {
			if proposalInfo.Status == cosmos.ProposalStatusPassed {
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
	require.Equal(t, uint64(100), uint64(initialReceivingBalance))

	_, err = xion.FullNodes[0].ExecTx(ctx,
		xionUser.KeyName(),
		"xion", "send", xionUser.KeyName(),
		"--chain-id", xion.Config().ChainID,
		recipientKeyAddress, fmt.Sprintf("%d%s", 200, xion.Config().Denom),
	)
	require.NoError(t, err)

	postSendingBalance, err := xion.GetBalance(ctx, xionUser.FormattedAddress(), xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(initialSendingBalance-200), uint64(postSendingBalance))
	postReceivingBalance, err := xion.GetBalance(ctx, recipientKeyAddress, xion.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, uint64(290), uint64(postReceivingBalance))
}
