package cli_test

import (
	"context"
	"fmt"
	"io"
	"os"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/burnt-labs/xion/x/xion/client/cli"
)

func (s *CLITestSuite) TestSendTxCmd() {
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 1)
	cmd := cli.NewSendTxCmd()
	cmd.SetOutput(io.Discard)

	extraArgs := []string{
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("photon", math.NewInt(10))).String()),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
	}

	testCases := []struct {
		name      string
		ctxGen    func() client.Context
		from, to  sdk.AccAddress
		amount    sdk.Coins
		extraArgs []string
		expectErr bool
	}{
		{
			"valid transaction",
			func() client.Context {
				return s.baseCtx
			},
			accounts[0].Address,
			accounts[0].Address,
			sdk.NewCoins(
				sdk.NewCoin("stake", math.NewInt(10)),
				sdk.NewCoin("photon", math.NewInt(40)),
			),
			extraArgs,
			false,
		},
		{
			"invalid to Address",
			func() client.Context {
				return s.baseCtx
			},
			accounts[0].Address,
			sdk.AccAddress{},
			sdk.NewCoins(
				sdk.NewCoin("stake", math.NewInt(10)),
				sdk.NewCoin("photon", math.NewInt(40)),
			),
			extraArgs,
			true,
		},
		{
			"invalid coins",
			func() client.Context {
				return s.baseCtx
			},
			accounts[0].Address,
			accounts[0].Address,
			nil,
			extraArgs,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := svrcmd.CreateExecuteContext(context.Background())

			cmd.SetContext(ctx)
			cmd.SetArgs(append([]string{tc.from.String(), tc.to.String(), tc.amount.String()}, tc.extraArgs...))

			s.Require().NoError(client.SetCmdClientContextHandler(tc.ctxGen(), cmd))

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *CLITestSuite) TestMultiSendTxCmd() {
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 3)

	cmd := cli.NewMultiSendTxCmd()
	cmd.SetOutput(io.Discard)

	extraArgs := []string{
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("photon", math.NewInt(10))).String()),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
	}

	testCases := []struct {
		name      string
		ctxGen    func() client.Context
		from      string
		to        []string
		amount    sdk.Coins
		extraArgs []string
		expectErr bool
	}{
		{
			"valid transaction",
			func() client.Context {
				return s.baseCtx
			},
			accounts[0].Address.String(),
			[]string{
				accounts[1].Address.String(),
				accounts[2].Address.String(),
			},
			sdk.NewCoins(
				sdk.NewCoin("stake", math.NewInt(10)),
				sdk.NewCoin("photon", math.NewInt(40)),
			),
			extraArgs,
			false,
		},
		{
			"invalid from Address",
			func() client.Context {
				return s.baseCtx
			},
			"foo",
			[]string{
				accounts[1].Address.String(),
				accounts[2].Address.String(),
			},
			sdk.NewCoins(
				sdk.NewCoin("stake", math.NewInt(10)),
				sdk.NewCoin("photon", math.NewInt(40)),
			),
			extraArgs,
			true,
		},
		{
			"invalid recipients",
			func() client.Context {
				return s.baseCtx
			},
			accounts[0].Address.String(),
			[]string{
				accounts[1].Address.String(),
				"bar",
			},
			sdk.NewCoins(
				sdk.NewCoin("stake", math.NewInt(10)),
				sdk.NewCoin("photon", math.NewInt(40)),
			),
			extraArgs,
			true,
		},
		{
			"invalid amount",
			func() client.Context {
				return s.baseCtx
			},
			accounts[0].Address.String(),
			[]string{
				accounts[1].Address.String(),
				accounts[2].Address.String(),
			},
			nil,
			extraArgs,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			ctx := svrcmd.CreateExecuteContext(context.Background())

			var args []string
			args = append(args, tc.from)
			args = append(args, tc.to...)
			args = append(args, tc.amount.String())
			args = append(args, tc.extraArgs...)

			cmd.SetContext(ctx)
			cmd.SetArgs(args)

			s.Require().NoError(client.SetCmdClientContextHandler(tc.ctxGen(), cmd))

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}

func (s *CLITestSuite) TestUpdateConfigsCmd() {
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 1)
	cmd := cli.NewUpdateConfigsCmd()
	cmd.SetOutput(io.Discard)

	configFile := "config.json"

	// Create temporary JSON files for testing
	configData := []byte(`{"grant_config":[
		{
			"msg_type_url": "/cosmos.bank.v1.MsgSend",
			"grant_config": {
			"description": "Bank grant",
			"authorization": {
				"type_url": "/cosmos.authz.v1.GenericAuthorization",
				"value": "CgRQYXk="
			},
			"optional": true
			}
		},
		{
			"msg_type_url": "/cosmos.staking.v1.MsgDelegate",
			"grant_config": {
			"description": "Staking grant",
			"authorization": {
				"type_url": "/cosmos.authz.v1.GenericAuthorization",
				"value": "CgREZWxlZ2F0ZQ=="
			},
			"optional": false
			}
		}
	], "fee_config":{
			"description": "Fee allowance for user1",
			"allowance": {
				"type_url": "/cosmos.feegrant.v1.BasicAllowance",
				"value": "CgQICAI="
			},
			"expiration": 1715151235
		}}`)

	require.NoError(s.T(), os.WriteFile(configFile, configData, 0600))
	defer os.Remove(configFile)

	// Mock valid Bech32 contract address
	validContractAddress := "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"

	extraArgs := []string{
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin("photon", math.NewInt(10))).String()),
		fmt.Sprintf("--%s=test-chain", flags.FlagChainID),
		fmt.Sprintf("--%s=%s", flags.FlagFrom, accounts[0].Name),
		fmt.Sprintf("--%s=true", "local"),
	}

	testCases := []struct {
		name       string
		ctxGen     func() client.Context
		contract   string
		configFile string
		extraArgs  []string
		expectErr  bool
	}{
		{
			"valid execution",
			func() client.Context {
				return s.baseCtx.WithFromAddress(accounts[0].Address)
			},
			validContractAddress,
			configFile,
			extraArgs,
			false,
		},
		{
			"invalid contract address",
			func() client.Context {
				return s.baseCtx.WithFromAddress(accounts[0].Address)
			},
			"invalid-address",
			configFile,
			extraArgs,
			true,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// Validate contract address format before executing the command
			if _, err := sdk.AccAddressFromBech32(tc.contract); tc.contract != "invalid-address" && err != nil {
				s.T().Fatalf("invalid contract address: %s", err)
			}

			ctx := svrcmd.CreateExecuteContext(context.Background())

			cmd.SetContext(ctx)
			cmd.SetArgs(append([]string{tc.contract, tc.configFile}, tc.extraArgs...))

			s.Require().NoError(client.SetCmdClientContextHandler(tc.ctxGen(), cmd))

			err := cmd.Execute()
			if tc.expectErr {
				s.Require().Error(err)
			} else {
				s.Require().NoError(err)
			}
		})
	}
}
