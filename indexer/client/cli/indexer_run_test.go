package cmd

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	"github.com/burnt-labs/xion/indexer"
	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

var (
	testGranter = sdk.AccAddress([]byte("cli_granter_________"))
	testGrantee = sdk.AccAddress([]byte("cli_grantee_________"))
)

func testCodec() *codec.ProtoCodec {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	std.RegisterInterfaces(interfaceRegistry)
	authz.RegisterInterfaces(interfaceRegistry)
	banktypes.RegisterInterfaces(interfaceRegistry)
	feegrant.RegisterInterfaces(interfaceRegistry)
	return codec.NewProtoCodec(interfaceRegistry)
}

// queryFixture serves the indexer query services over an in-process gRPC
// connection and hands out a client context wired to it.
type queryFixture struct {
	clientCtx client.Context
	service   *indexer.StreamService
	out       *bytes.Buffer
	granter   string
	grantee   string
}

func newQueryFixture(t *testing.T) *queryFixture {
	t.Helper()
	cdc := testCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")

	service := indexer.NewWithDB(dbm.NewMemDB(), cdc, addrCodec, log.NewNopLogger())
	t.Cleanup(func() { _ = service.Close() })

	listener := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(grpc.ForceServerCodec(cdc.GRPCCodec()))
	indexerauthz.RegisterQueryServer(srv, service.AuthzQuerier())
	indexerfeegrant.RegisterQueryServer(srv, service.FeeGrantQuerier())
	go func() { _ = srv.Serve(listener) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///indexer",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(cdc.GRPCCodec())),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	granter, err := addrCodec.BytesToString(testGranter)
	require.NoError(t, err)
	grantee, err := addrCodec.BytesToString(testGrantee)
	require.NoError(t, err)

	out := new(bytes.Buffer)
	clientCtx := client.Context{}.
		WithCodec(cdc).
		WithInterfaceRegistry(cdc.InterfaceRegistry()).
		WithOutput(out).
		WithGRPCClient(conn)

	return &queryFixture{clientCtx: clientCtx, service: service, out: out, granter: granter, grantee: grantee}
}

func (f *queryFixture) seedGrants(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	grant, err := authz.NewGrant(time.Now().Add(time.Hour), sendAuth, nil)
	require.NoError(t, err)
	require.NoError(t, f.service.AuthzHandler().SetGrant(ctx, testGranter, testGrantee, sendAuth.MsgTypeURL(), grant))

	allowance, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
		SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
	})
	require.NoError(t, err)
	require.NoError(t, f.service.FeeGrantHandler().SetGrant(ctx, testGranter, testGrantee, feegrant.Grant{
		Granter:   f.granter,
		Grantee:   f.grantee,
		Allowance: allowance,
	}))
}

func (f *queryFixture) run(t *testing.T, cmdFn func() *cobra.Command, args ...string) error {
	t.Helper()
	cmd := cmdFn()
	cmd.SetOut(f.out)
	cmd.SetErr(f.out)
	cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &f.clientCtx))
	return cmd.RunE(cmd, args)
}

func TestQueryCommands_Output(t *testing.T) {
	cases := []struct {
		name   string
		cmd    func() *cobra.Command
		arg    func(f *queryFixture) string
		expect func(f *queryFixture) string
	}{
		{
			name:   "grants by grantee",
			cmd:    QueryGrantsByGrantee,
			arg:    func(f *queryFixture) string { return f.grantee },
			expect: func(f *queryFixture) string { return f.granter },
		},
		{
			name:   "grants by granter",
			cmd:    QueryGrantsByGranter,
			arg:    func(f *queryFixture) string { return f.granter },
			expect: func(f *queryFixture) string { return f.grantee },
		},
		{
			name:   "allowances by grantee",
			cmd:    QueryAllowancesByGrantee,
			arg:    func(f *queryFixture) string { return f.grantee },
			expect: func(f *queryFixture) string { return f.granter },
		},
		{
			name:   "allowances by granter",
			cmd:    QueryAllowancesByGranter,
			arg:    func(f *queryFixture) string { return f.granter },
			expect: func(f *queryFixture) string { return f.grantee },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newQueryFixture(t)
			f.seedGrants(t)

			require.NoError(t, f.run(t, tc.cmd, tc.arg(f)))
			require.Contains(t, f.out.String(), tc.expect(f))
		})

		t.Run(tc.name+" rejects a malformed address", func(t *testing.T) {
			f := newQueryFixture(t)

			err := f.run(t, tc.cmd, "not-an-address")
			require.Error(t, err)
			require.Empty(t, f.out.String())
		})

		t.Run(tc.name+" fails to build the client context", func(t *testing.T) {
			f := newQueryFixture(t)
			cmd := tc.cmd()
			cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &f.clientCtx))
			// the node address is parsed while the query context is assembled
			require.NoError(t, cmd.Flags().Set(flags.FlagNode, "tcp://local host:26657"))

			err := cmd.RunE(cmd, []string{f.grantee})
			require.Error(t, err)
			require.Empty(t, f.out.String())
		})
	}
}

func newReIndexCommand(t *testing.T, appCreator servertypes.AppCreator, home string, dbBackend string) *cobra.Command {
	t.Helper()
	cmd := ReIndex(appCreator, home)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	serverCtx := server.NewDefaultContext()
	serverCtx.Viper.Set(flags.FlagHome, home)
	serverCtx.Viper.Set(FlagAppDBBackend, dbBackend)
	cmd.SetContext(context.WithValue(context.Background(), server.ServerContextKey, serverCtx))
	return cmd
}

func writeGenesisPlaceholder(t *testing.T, home string) {
	t.Helper()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "genesis.json"), []byte("{}"), 0o600))
}

func TestReIndex_MissingGenesis(t *testing.T) {
	home := t.TempDir()
	cmd := newReIndexCommand(t, mockAppCreator, home, string(dbm.MemDBBackend))

	err := cmd.RunE(cmd, nil)
	require.ErrorIs(t, err, fs.ErrNotExist)
}

func TestReIndex_UnknownDBBackend(t *testing.T) {
	home := t.TempDir()
	writeGenesisPlaceholder(t, home)
	cmd := newReIndexCommand(t, mockAppCreator, home, "no-such-backend")

	err := cmd.RunE(cmd, nil)
	require.Error(t, err)
}

func TestReIndex_RebuildsIndexFromChainState(t *testing.T) {
	home := t.TempDir()
	writeGenesisPlaceholder(t, home)

	sendAuth := banktypes.NewSendAuthorization(sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)), nil)
	allowance := &feegrant.BasicAllowance{SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000))}

	// the app is created by the command itself; populate its check state with
	// one grant and one allowance so the re-index has something to copy
	appCreator := func(logger log.Logger, db dbm.DB, _ io.Writer, appOpts servertypes.AppOptions) servertypes.Application {
		memDB, ok := db.(*dbm.MemDB)
		require.True(t, ok)

		app := xionapp.NewWasmAppWithCustomOptions(t, false, xionapp.SetupOptions{
			Logger:  logger,
			DB:      memDB,
			AppOpts: appOpts,
		})
		ctx := app.NewContext(true)
		app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccountWithAddress(ctx, testGrantee))
		require.NoError(t, app.AuthzKeeper.SaveGrant(ctx, testGrantee, testGranter, sendAuth, nil))
		require.NoError(t, app.FeeGrantKeeper.GrantAllowance(ctx, testGranter, testGrantee, allowance))
		return app
	}

	cmd := newReIndexCommand(t, appCreator, home, string(dbm.MemDBBackend))
	require.NoError(t, cmd.RunE(cmd, nil))

	// the command closed the indexer database; reopen it and check the copies
	cdc := testCodec()
	addrCodec := addresscodec.NewBech32Codec("xion")
	service := indexer.New(home, cdc, addrCodec, log.NewNopLogger())
	defer func() { _ = service.Close() }()

	grant, err := service.AuthzHandler().GetGrant(context.Background(), testGranter, testGrantee, sendAuth.MsgTypeURL())
	require.NoError(t, err)
	require.NotNil(t, grant.Authorization)

	feeGrant, err := service.FeeGrantHandler().GetGrant(context.Background(), testGranter, testGrantee)
	require.NoError(t, err)
	require.NotNil(t, feeGrant.Allowance)
}
