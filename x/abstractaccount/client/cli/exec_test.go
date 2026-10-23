package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/x/abstractaccount/client/cli"
	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

const (
	flagSalt  = "salt"
	flagFunds = "funds"
)

// subcommand returns the named subcommand of root.
func subcommand(t *testing.T, root *cobra.Command, name string) *cobra.Command {
	t.Helper()
	cmd, _, err := root.Find([]string{name})
	require.NoError(t, err)
	require.Equal(t, name, cmd.Name())
	return cmd
}

// The tests in this file run the query and tx commands end to end: queries
// against an in-process gRPC server, transactions in generate-only mode.

// stubQueryServer answers the module queries with canned responses and
// records what it was asked.
type stubQueryServer struct {
	params  *types.Params
	address *types.QueryAccountAddressResponse

	accountAddressReq *types.QueryAccountAddressRequest
}

func (s *stubQueryServer) Params(context.Context, *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	return &types.QueryParamsResponse{Params: s.params}, nil
}

func (s *stubQueryServer) AccountAddress(_ context.Context, req *types.QueryAccountAddressRequest) (*types.QueryAccountAddressResponse, error) {
	s.accountAddressReq = req
	return s.address, nil
}

// newQueryClientContext serves srv over an in-memory connection and returns
// a client context that queries it, printing JSON into the returned buffer.
func newQueryClientContext(t *testing.T, srv types.QueryServer) (client.Context, *bytes.Buffer) {
	t.Helper()
	encCfg := xionapp.MakeEncodingConfig(t)
	grpcCodec := codec.NewProtoCodec(encCfg.InterfaceRegistry).GRPCCodec()

	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer(grpc.ForceServerCodec(grpcCodec))
	types.RegisterQueryServer(server, srv)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(grpcCodec)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	out := &bytes.Buffer{}
	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithGRPCClient(conn).
		WithOutput(out).
		WithOutputFormat(flags.OutputFormatJSON)

	return clientCtx, out
}

func withClientContext(cmd *cobra.Command, clientCtx client.Context) {
	cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
}

func TestQueryParamsPrintsResponse(t *testing.T) {
	params, err := types.NewParams(false, []uint64{4, 9}, 1234, 5678)
	require.NoError(t, err)
	clientCtx, out := newQueryClientContext(t, &stubQueryServer{params: params})

	cmd := subcommand(t, cli.GetQueryCmd(), "params")
	withClientContext(cmd, clientCtx)

	require.NoError(t, cmd.RunE(cmd, nil))

	var printed types.Params
	require.NoError(t, clientCtx.Codec.UnmarshalJSON(out.Bytes(), &printed))
	require.Equal(t, *params, printed)
}

func TestQueryAccountAddressPrintsResponse(t *testing.T) {
	sender := sdk.AccAddress("account-address-sender")
	address := sdk.AccAddress("predicted-account-addr")
	srv := &stubQueryServer{address: &types.QueryAccountAddressResponse{Address: address.String(), Registered: true}}
	clientCtx, out := newQueryClientContext(t, srv)

	cmd := subcommand(t, cli.GetQueryCmd(), "account-address")
	withClientContext(cmd, clientCtx)
	require.NoError(t, cmd.Flags().Set(flagSalt, "hello"))

	require.NoError(t, cmd.RunE(cmd, []string{sender.String()}))

	require.Equal(t, &types.QueryAccountAddressRequest{Sender: sender.String(), Salt: []byte("hello")}, srv.accountAddressReq)
	var printed types.QueryAccountAddressResponse
	require.NoError(t, clientCtx.Codec.UnmarshalJSON(out.Bytes(), &printed))
	require.Equal(t, address.String(), printed.Address)
	require.True(t, printed.Registered)
}

func TestQueryCommandsRejectMalformedNode(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "params"},
		{name: "account-address", args: []string{"sender"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := subcommand(t, cli.GetQueryCmd(), tc.name)
			withClientContext(cmd, client.Context{})
			require.NoError(t, cmd.Flags().Set(flags.FlagNode, "://bad"))

			err := cmd.RunE(cmd, tc.args)
			require.ErrorContains(t, err, "missing protocol scheme")
		})
	}
}

// newGenerateOnlyClientContext returns a client context that prints unsigned
// transactions instead of broadcasting them.
func newGenerateOnlyClientContext(t *testing.T) (client.Context, *bytes.Buffer) {
	t.Helper()
	encCfg := xionapp.MakeEncodingConfig(t)
	out := &bytes.Buffer{}

	clientCtx := client.Context{}.
		WithCodec(encCfg.Codec).
		WithInterfaceRegistry(encCfg.InterfaceRegistry).
		WithTxConfig(encCfg.TxConfig).
		WithLegacyAmino(encCfg.Amino).
		WithFromAddress(sdk.AccAddress("generate-only-sender")).
		WithChainID("test-chain").
		WithGenerateOnly(true).
		WithOutput(out)

	return clientCtx, out
}

// decodeGeneratedTx decodes the single message of the unsigned transaction
// printed by a generate-only run.
func decodeGeneratedTx(t *testing.T, clientCtx client.Context, printed []byte) sdk.Msg {
	t.Helper()
	tx, err := clientCtx.TxConfig.TxJSONDecoder()(bytes.TrimSpace(printed))
	require.NoError(t, err)
	msgs := tx.GetMsgs()
	require.Len(t, msgs, 1)
	return msgs[0]
}

func TestRegisterAccountGeneratesTx(t *testing.T) {
	clientCtx, out := newGenerateOnlyClientContext(t)
	cmd := subcommand(t, cli.GetTxCmd(), "register")
	require.NoError(t, cmd.Flags().Set(flagSalt, "hello"))
	require.NoError(t, cmd.Flags().Set(flagFunds, "25uxion"))

	require.NoError(t, cli.RegisterAccount(clientCtx, cmd.Flags(), []string{"7", `{"pubkey":"AQID"}`}))

	msg, ok := decodeGeneratedTx(t, clientCtx, out.Bytes()).(*types.MsgRegisterAccount)
	require.True(t, ok)
	require.Equal(t, clientCtx.GetFromAddress().String(), msg.Sender)
	require.Equal(t, uint64(7), msg.CodeID)
	require.JSONEq(t, `{"pubkey":"AQID"}`, string(msg.Msg))
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uxion", 25)), msg.Funds)
	require.Equal(t, []byte("hello"), msg.Salt)
}

func TestUpdateParamsGeneratesTx(t *testing.T) {
	clientCtx, out := newGenerateOnlyClientContext(t)
	cmd := subcommand(t, cli.GetTxCmd(), "update-params")
	params := types.Params{AllowAllCodeIDs: false, AllowedCodeIDs: []uint64{3}, MaxGasBefore: 111, MaxGasAfter: 222}
	paramsJSON, err := json.Marshal(params)
	require.NoError(t, err)

	require.NoError(t, cli.UpdateParams(clientCtx, cmd.Flags(), []string{string(paramsJSON)}))

	msg, ok := decodeGeneratedTx(t, clientCtx, out.Bytes()).(*types.MsgUpdateParams)
	require.True(t, ok)
	require.Equal(t, clientCtx.GetFromAddress().String(), msg.Sender)
	require.Equal(t, &params, msg.Params)
}

func TestTxCommandsRejectMalformedNode(t *testing.T) {
	encCfg := xionapp.MakeEncodingConfig(t)
	clientCtx := client.Context{}.WithCodec(encCfg.Codec).WithHomeDir(t.TempDir())

	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "register", args: []string{"1", "{}"}},
		{name: "update-params", args: []string{"{}"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := subcommand(t, cli.GetTxCmd(), tc.name)
			withClientContext(cmd, clientCtx)
			require.NoError(t, cmd.Flags().Set(flags.FlagKeyringBackend, "test"))
			require.NoError(t, cmd.Flags().Set(flags.FlagNode, "://bad"))

			err := cmd.RunE(cmd, tc.args)
			require.ErrorContains(t, err, "missing protocol scheme")
		})
	}
}
