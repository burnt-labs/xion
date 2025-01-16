package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
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

	configFile := "plain_config.json"
	configFileUrl := "https://raw.githubusercontent.com/burnt-labs/xion/6ce7bb89562d5a2964788cb64a623eec170c8748/integration_tests/testdata/unsigned_msgs/plain_config.json"

	// Create temporary JSON files for testing
	configData := []byte(`{"grant_config":[
		{
			"msg_type_url": "/cosmos.bank.v1.MsgSend",
			"grant_config": {
			"description": "Bank grant",
			"authorization": {
				"@type": "/cosmos.authz.v1beta1.GenericAuthorization",
				"msg": "/cosmos.bank.v1beta1.MsgSend"
			},
			"optional": true
			}
		},
		{
			"msg_type_url": "/cosmos.staking.v1beta1.MsgDelegate",
			"grant_config": {
			"description": "Staking grant",
			"authorization": {
				"@type": "/cosmos.authz.v1beta1.GenericAuthorization",
				"msg": "/cosmos.staking.v1beta1.MsgDelegate"
			},
			"optional": false
			}
		}
	], "fee_config":{
			"description": "Fee allowance for user1",
			"allowance": {
				"@type": "/cosmos.feegrant.v1beta1.BasicAllowance",
          		"spend_limit": [
            		{
						"denom": "atom",
						"amount": "1000"
            		}
          		],
          		"expiration": "2025-01-01T00:00:00Z"
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
			"valid execution url",
			func() client.Context {
				return s.baseCtx.WithFromAddress(accounts[0].Address)
			},
			validContractAddress,
			configFileUrl,
			extraArgs,
			false,
		},
		{
			"valid execution",
			func() client.Context {
				return s.baseCtx.WithFromAddress(accounts[0].Address)
			},
			validContractAddress,
			configFile,
			append(extraArgs, fmt.Sprintf("--%s=true", "local")),
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

func (s *CLITestSuite) TestSerializeJSONAllowanceToProto() {
	// Example JSON input for a BasicAllowance
	jsonInput := `{
		"@type": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
		"allowance": {
			"@type": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
			"allowance": {
				"@type": "/cosmos.feegrant.v1beta1.AllowedMsgAllowance",
				"allowance": {
					"@type": "/cosmos.feegrant.v1beta1.BasicAllowance",
					"spend_limit": [{"denom": "atom", "amount": "1000"}],
					"expiration": "2025-01-01T00:00:00Z"
				},
				"allowed_messages": ["/cosmos.bank.v1beta1.MsgSend"]
			},
			"allowed_messages": ["/cosmos.staking.v1beta1.MsgDelegate"]
		},
		"allowed_messages": ["/cosmos.gov.v1beta1.MsgVote"]
	}`

	// Serialize JSON to Protobuf
	var jsonData map[string]interface{}
	require.NoError(s.T(), json.Unmarshal([]byte(jsonInput), &jsonData), "Failed to unmarshal JSON input")

	anyMsg, err := cli.ConvertJSONToAny(s.encCfg.Codec, jsonData)
	require.NoError(s.T(), err, "Failed to serialize JSON to Proto")

	// Assert that the resulting Protobuf message is not nil
	require.NotNil(s.T(), anyMsg, "Protobuf message should not be nil")

	var protoMsg feegrant.FeeAllowanceI
	// Unpack the Any into the top-level AllowedMsgAllowance
	err = s.encCfg.InterfaceRegistry.UnpackAny(&cdctypes.Any{
		TypeUrl: anyMsg.TypeURL,
		Value:   anyMsg.Value,
	}, &protoMsg)
	require.NoError(s.T(), err, "Failed to unpack Any into Protobuf message")

	// Verify first-level AllowedMsgAllowance
	topLevelAllowance, ok := protoMsg.(*feegrant.AllowedMsgAllowance)
	require.True(s.T(), ok, "Top-level Protobuf message is not of type *feegrant.AllowedMsgAllowance")
	require.Equal(s.T(), []string{"/cosmos.gov.v1beta1.MsgVote"}, topLevelAllowance.AllowedMessages)

	// Verify second-level AllowedMsgAllowance
	secondLevelAllowance, ok := topLevelAllowance.Allowance.GetCachedValue().(*feegrant.AllowedMsgAllowance)
	require.True(s.T(), ok, "Second-level Protobuf message is not of type *feegrant.AllowedMsgAllowance")
	require.Equal(s.T(), []string{"/cosmos.staking.v1beta1.MsgDelegate"}, secondLevelAllowance.AllowedMessages)

	// Verify third-level AllowedMsgAllowance
	thirdLevelAllowance, ok := secondLevelAllowance.Allowance.GetCachedValue().(*feegrant.AllowedMsgAllowance)
	require.True(s.T(), ok, "Third-level Protobuf message is not of type *feegrant.AllowedMsgAllowance")
	require.Equal(s.T(), []string{"/cosmos.bank.v1beta1.MsgSend"}, thirdLevelAllowance.AllowedMessages)

	// Verify fourth-level BasicAllowance
	fourthLevelAllowance, ok := thirdLevelAllowance.Allowance.GetCachedValue().(*feegrant.BasicAllowance)
	require.True(s.T(), ok, "Fourth-level Protobuf message is not of type *feegrant.BasicAllowance")
	require.Equal(s.T(), sdk.Coins{{Denom: "atom", Amount: math.NewInt(1000)}}, fourthLevelAllowance.SpendLimit)
	require.NotNil(s.T(), fourthLevelAllowance.Expiration, "Expiration should not be nil")
}
