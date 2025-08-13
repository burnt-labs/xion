package cli_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/burnt-labs/xion/x/xion/client/cli"
)

func (s *CLITestSuite) TestSendTxCmd() {
	accounts := testutil.CreateKeyringAccounts(s.T(), s.kr, 1)
	cmd := cli.NewSendTxCmd()
	cmd.SetOut(io.Discard)

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
			func() client.Context { return s.baseCtx },
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
			func() client.Context { return s.baseCtx },
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
			func() client.Context { return s.baseCtx },
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
	cmd.SetOut(io.Discard)

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
	cmd.SetOut(io.Discard)

	configFile := "plain_config.json"

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

	require.NoError(s.T(), os.WriteFile(configFile, configData, 0o600))
	defer os.Remove(configFile)

	// local server to avoid network flakiness
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(configData)
	}))
	defer srv.Close()
	configFileURL := srv.URL + "/plain_config.json"

	// Mock valid Bech32 contract address with xion prefix (placeholder length/padding)
	validContractAddress := accounts[0].Address.String()

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
			configFileURL,
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

// TestNewTxCmd tests the main transaction command creation
func (s *CLITestSuite) TestNewTxCmd() {
	cmd := cli.NewTxCmd()
	require.NotNil(s.T(), cmd)
	require.Equal(s.T(), "xion", cmd.Use)
	require.Equal(s.T(), "Xion transaction subcommands", cmd.Short)
	require.True(s.T(), cmd.DisableFlagParsing)
	require.Equal(s.T(), 2, cmd.SuggestionsMinimumDistance)
	require.NotNil(s.T(), cmd.RunE)

	// Test that all expected subcommands are added
	subcommands := cmd.Commands()
	require.Len(s.T(), subcommands, 8) // Should have 8 subcommands

	// Check first few commands exist (order may vary)
	cmdNames := make([]string, len(subcommands))
	for i, subcmd := range subcommands {
		cmdNames[i] = subcmd.Name()
	}
	require.Contains(s.T(), cmdNames, "send")
	require.Contains(s.T(), cmdNames, "multi-send")
	require.Contains(s.T(), cmdNames, "register")

	// Test RunE validation
	err := cmd.RunE(cmd, []string{})
	require.NoError(s.T(), err) // ValidateCmd should pass with no args
}

// Consolidated command metadata & arg validation (reduces multiple redundant tests)
func TestCommandMetadataAndArgs(t *testing.T) {
	cases := []struct {
		name        string
		newCmd      func() *cobra.Command
		useContains string
		short       string
		validArgs   [][]string // arg sets expected to pass
		invalidArgs [][]string // arg sets expected to fail
	}{
		{
			name:        "register",
			newCmd:      cli.NewRegisterCmd,
			useContains: "register",
			short:       "Register an abstract account",
			validArgs:   [][]string{{}, {"1"}, {"1", "key"}},
			invalidArgs: [][]string{{"1", "key", "extra"}},
		},
		{
			name:        "add-authenticator",
			newCmd:      cli.NewAddAuthenticatorCmd,
			useContains: "add-authenticator",
			short:       "Add the signing key as an authenticator to an abstract account",
			validArgs:   [][]string{{"addr"}},
			invalidArgs: [][]string{{}, {"a", "b"}},
		},
		{
			name:        "sign",
			newCmd:      cli.NewSignCmd,
			useContains: "sign",
			short:       "sign a transaction",
			validArgs:   [][]string{{"k", "acct", "file"}},
			invalidArgs: [][]string{{}, {"k"}, {"k", "a"}, {"k", "a", "b", "c"}},
		},
		{
			name:        "emit",
			newCmd:      cli.NewEmitArbitraryDataCmd,
			useContains: "emit",
			short:       "Emit an arbitrary data from the chain",
			validArgs:   [][]string{{"data", "contract"}},
			invalidArgs: [][]string{{}, {"only"}, {"a", "b", "c"}},
		},
		{
			name:        "update-params",
			newCmd:      cli.NewUpdateParamsCmd,
			useContains: "update-params",
			short:       "Update treasury contract parameters",
			validArgs:   [][]string{{"c", "d", "r", "i"}},
			invalidArgs: [][]string{{}, {"c"}, {"c", "d", "r"}, {"c", "d", "r", "i", "x"}},
		},
		{
			name:        "update-configs",
			newCmd:      cli.NewUpdateConfigsCmd,
			useContains: "update-configs",
			short:       "Batch update grant configs and fee config for the treasury",
			validArgs:   [][]string{{"c", "path"}},
			invalidArgs: [][]string{{}, {"c"}, {"c", "p", "x"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.newCmd()
			require.Contains(t, cmd.Use, tc.useContains)
			require.Equal(t, tc.short, cmd.Short)
			for _, a := range tc.validArgs {
				require.NoError(t, cmd.Args(cmd, a), "args should be valid: %v", a)
			}
			for _, a := range tc.invalidArgs {
				require.Error(t, cmd.Args(cmd, a), "args should be invalid: %v", a)
			}
			// smoke help
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{"--help"})
			ctx := svrcmd.CreateExecuteContext(context.Background())
			cmd.SetContext(ctx)
			require.NoError(t, cmd.Execute())
		})
	}
}

// TestConvertJSONToAny tests the ConvertJSONToAny function
func (s *CLITestSuite) TestConvertJSONToAny() {
	testCases := []struct {
		name      string
		jsonInput map[string]interface{}
		expectErr bool
		errMsg    string
	}{
		{
			name: "missing @type field",
			jsonInput: map[string]interface{}{
				"amount": "100",
			},
			expectErr: true,
			errMsg:    "failed to parse type URL from JSON",
		},
		{
			name: "invalid @type value",
			jsonInput: map[string]interface{}{
				"@type":  123, // not a string
				"amount": "100",
			},
			expectErr: true,
			errMsg:    "failed to parse type URL from JSON",
		},
		{
			name: "unknown type URL",
			jsonInput: map[string]interface{}{
				"@type":  "/unknown.Type",
				"amount": "100",
			},
			expectErr: true,
			errMsg:    "failed to resolve type URL",
		},
		{
			name: "valid BasicAllowance type",
			jsonInput: map[string]interface{}{
				"@type": "/cosmos.feegrant.v1beta1.BasicAllowance",
				"spend_limit": []interface{}{
					map[string]interface{}{
						"denom":  "atom",
						"amount": "1000",
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			result, err := cli.ConvertJSONToAny(s.baseCtx.Codec, tc.jsonInput)
			if tc.expectErr {
				s.Require().Error(err)
				if tc.errMsg != "" {
					s.Require().Contains(err.Error(), tc.errMsg)
				}
			} else {
				s.Require().NoError(err)
				s.Require().NotEmpty(result.TypeURL)
				s.Require().NotEmpty(result.Value)
			}
		})
	}
}

// Simplified flag presence test
func TestCommandCommonFlags(t *testing.T) {
	cmds := []struct {
		name string
		cmd  func() *cobra.Command
	}{
		{"register", cli.NewRegisterCmd},
		{"add-authenticator", cli.NewAddAuthenticatorCmd},
		{"sign", cli.NewSignCmd},
		{"emit", cli.NewEmitArbitraryDataCmd},
		{"update-params", cli.NewUpdateParamsCmd},
		{"update-configs", cli.NewUpdateConfigsCmd},
	}
	for _, c := range cmds {
		t.Run(c.name, func(t *testing.T) {
			cmd := c.cmd()
			require.NotNil(t, cmd.Flag(flags.FlagChainID))
			require.NotNil(t, cmd.Flag(flags.FlagFrom))
			require.NotNil(t, cmd.Flag(flags.FlagGas))
		})
	}
}
