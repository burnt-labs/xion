package cli

import (
	"context"
	"math"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/gogoproto/proto"

	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant/module"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module/testutil"

	aatypes "github.com/burnt-labs/xion/x/abstractaccount/types"
)

// withoutFlag returns a command carrying every flag of cmd except name, so a
// handler can be run against a flag set that is missing one of its inputs.
func withoutFlag(cmd *cobra.Command, name string) *cobra.Command {
	stripped := &cobra.Command{Use: cmd.Use}
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if f.Name != name {
			stripped.Flags().AddFlag(f)
		}
	})
	return stripped
}

// newKeyedClientContext returns a context whose keyring holds one key and
// whose node answers every query with a predicted abstract account address.
func newKeyedClientContext(t *testing.T) (client.Context, string) {
	t.Helper()
	const name = "alice"
	encCfg := testutil.MakeTestEncodingConfig()
	aatypes.RegisterInterfaces(encCfg.InterfaceRegistry)
	kr := keyring.NewInMemory(encCfg.Codec)
	record, _, err := kr.NewMnemonic(name, keyring.English, sdk.FullFundraiserPath, keyring.DefaultBIP39Passphrase, hd.Secp256k1)
	require.NoError(t, err)
	addr, err := record.GetAddress()
	require.NoError(t, err)

	predicted, err := proto.Marshal(&aatypes.QueryAccountAddressResponse{Address: testValidBech32Addr})
	require.NoError(t, err)

	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithClient(clitestutil.NewMockCometRPC(abci.ResponseQuery{Value: predicted})).
		WithKeyring(kr).
		WithFrom(name).
		WithFromName(name).
		WithFromAddress(addr)
	return clientCtx, name
}

func TestTxHandlersReportUnregisteredFlags(t *testing.T) {
	clientCtx, name := newKeyedClientContext(t)
	addr := clientCtx.GetFromAddress().String()

	jwtFlags := func(cmd *cobra.Command) {
		require.NoError(t, cmd.Flags().Set(flagSalt, "salt"))
		require.NoError(t, cmd.Flags().Set(flagAuthenticator, "Jwt"))
	}

	cases := []struct {
		name    string
		newCmd  func() *cobra.Command
		flag    string
		args    []string
		setup   func(*cobra.Command)
		wantErr string
	}{
		{"send without from", NewSendTxCmd, flags.FlagFrom, []string{name, addr, "1stake"}, nil, "no such flag"},
		{"multi-send without from", NewMultiSendTxCmd, flags.FlagFrom, []string{name, addr, addr, "1stake"}, nil, "no such flag"},
		{"multi-send without split", NewMultiSendTxCmd, FlagSplit, []string{name, addr, addr, "1stake"}, nil, "flag accessed but not defined: split"},
		{"register without from", NewRegisterCmd, flags.FlagFrom, []string{"1", name}, nil, "no such flag"},
		{"register without authenticator-id", NewRegisterCmd, flagAuthenticatorID, []string{"1", name}, nil, "flag accessed but not defined: authenticator-id"},
		{"register without salt", NewRegisterCmd, flagSalt, []string{"1", name}, nil, "salt: flag accessed but not defined"},
		{"register without funds", NewRegisterCmd, flagFunds, []string{"1", name}, nil, "amount: flag accessed but not defined"},
		{"register without authenticator", NewRegisterCmd, flagAuthenticator, []string{"1", name}, nil, "authenticator: flag accessed but not defined"},
		{"register jwt without sub", NewRegisterCmd, flagSubject, []string{"1", name}, jwtFlags, "subject: flag accessed but not defined"},
		{"register jwt without aud", NewRegisterCmd, flagAudience, []string{"1", name}, jwtFlags, "audience: flag accessed but not defined"},
		{"register jwt without token", NewRegisterCmd, flagToken, []string{"1", name}, jwtFlags, "token: flag accessed but not defined"},
		{"add-authenticator without authenticator-id", NewAddAuthenticatorCmd, flagAuthenticatorID, []string{addr}, nil, "flag accessed but not defined: authenticator-id"},
		{"sign without from", NewSignCmd, flags.FlagFrom, []string{name, addr, "tx.json"}, nil, "no such flag"},
		{"sign without authenticator-id", NewSignCmd, flagAuthenticatorID, []string{name, addr, "tx.json"}, nil, "flag accessed but not defined: authenticator-id"},
		{"update-configs without local", NewUpdateConfigsCmd, "local", []string{addr, "config.json"}, nil, "failed to parse local flag"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.newCmd()
			stripped := withoutFlag(cmd, tc.flag)
			stripped.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
			if tc.setup != nil {
				tc.setup(stripped)
			}

			err := cmd.RunE(stripped, tc.args)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestQueryAccountAddressRejectsMalformedPrediction(t *testing.T) {
	sender := sdk.AccAddress("sender______________")
	queryClient := new(mockAbstractAccountQueryClient)
	queryClient.On("AccountAddress", mock.Anything, mock.Anything, []grpc.CallOption(nil)).
		Return(&aatypes.QueryAccountAddressResponse{Address: "not-bech32"}, nil).Once()

	_, err := queryAccountAddress(context.Background(), queryClient, sender, []byte("salt"))
	require.ErrorContains(t, err, `returned an invalid address "not-bech32"`)
	queryClient.AssertExpectations(t)
}

func TestConvertJSONToAnyRejectsUnencodableInput(t *testing.T) {
	cdc := testutil.MakeTestEncodingConfig(feegrant.AppModuleBasic{}).Codec
	const basicAllowance = "/cosmos.feegrant.v1beta1.BasicAllowance"

	t.Run("value that JSON cannot encode", func(t *testing.T) {
		_, err := ConvertJSONToAny(cdc, map[string]any{"@type": basicAllowance, "spend_limit": math.NaN()})
		require.ErrorContains(t, err, "failed to marshal JSON input")
	})

	t.Run("field with the wrong shape", func(t *testing.T) {
		_, err := ConvertJSONToAny(cdc, map[string]any{"@type": basicAllowance, "spend_limit": "not-a-list"})
		require.ErrorContains(t, err, "failed to unmarshal JSON into proto.Message")
	})
}
