package ante_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/dkim/ante"
	dkimkeeper "github.com/burnt-labs/xion/x/dkim/keeper"
	"github.com/burnt-labs/xion/x/dkim/types"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
	zktypes "github.com/burnt-labs/xion/x/zk/types"
)

const (
	validRSAPubKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv3bzh5rabT+IWegVAoGnS/kRO2kbgr+jls+Gm5S/bsYYCS/MFsWBuegRE8yHwfiyT5Q90KzwZGkeGL609yrgZKJDHv4TM2kmybi4Kr/CsnhjVojMM7iZVu2Ncx/i/PaCEJzo94dcd4nIS+GXrFnRxU/vIilLojJ01W+jwuxrrkNg8zx6a9wWRwdQUYGUIbGkYazPdYUd/8M8rviLwT9qsnJcM4b3Ie/gtcYzsL5LhuvhfbhRVNGXEMADasx++xxfbIpPr5AgpnZo+6rA1UCUfwZT83Q2pAybaOcpjGUEWpP8h30Gi5xiUBR8rLjweG3MtYlnqTHSyiHGUt9JSCXGPQIDAQAB"
)

// mockTx implements sdk.Tx for testing
type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg { return m.msgs }
func (m mockTx) GetMsgsV2() ([]proto.Message, error) {
	var protoMsgs []proto.Message
	for _, msg := range m.msgs {
		if protoMsg, ok := msg.(proto.Message); ok {
			protoMsgs = append(protoMsgs, protoMsg)
		}
	}
	return protoMsgs, nil
}
func (m mockTx) ValidateBasic() error { return nil }

func setupTest(t *testing.T) (sdk.Context, *dkimkeeper.Keeper) {
	t.Helper()

	logger := log.NewTestLogger(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	authtypes.RegisterInterfaces(encCfg.InterfaceRegistry)

	govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	ctx := testCtx.Ctx

	// Setup ZK keeper
	zkKeeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
	defaultZkGenesis := zktypes.DefaultGenesisState()
	zkKeeper.InitGenesis(ctx, defaultZkGenesis)

	// Setup DKIM keeper
	keeper := dkimkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr, zkKeeper)

	return ctx, &keeper
}

func TestDKIMDecorator(t *testing.T) {
	ctx, keeper := setupTest(t)

	// Set params with a specific max pubkey size
	maxSize := uint64(512) // 512 bytes
	params := types.DefaultParams()
	params.MaxPubkeySizeBytes = maxSize
	err := keeper.Params.Set(ctx, params)
	require.NoError(t, err)

	decorator := ante.NewDKIMDecorator(keeper)

	// Mock next handler
	nextCalled := false
	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	t.Run("valid pubkey within size limit", func(t *testing.T) {
		nextCalled = false
		addrs := simtestutil.CreateIncrementalAccounts(1)
		addr := addrs[0]

		validDkimKey := types.DkimPubKey{
			Domain:   "example.com",
			Selector: "default",
			PubKey:   validRSAPubKey,
			Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}

		msg := types.NewMsgAddDkimPubKeys(addr, []types.DkimPubKey{validDkimKey})
		tx := mockTx{msgs: []sdk.Msg{msg}}

		newCtx, err := decorator.AnteHandle(ctx, tx, false, nextHandler)
		require.NoError(t, err)
		require.True(t, nextCalled)
		require.NotNil(t, newCtx)
	})

	t.Run("invalid pubkey within size limit", func(t *testing.T) {
		params := types.DefaultParams()
		params.MaxPubkeySizeBytes = 128
		err := keeper.Params.Set(ctx, params)
		require.NoError(t, err)

		nextCalled = false
		addrs := simtestutil.CreateIncrementalAccounts(1)
		addr := addrs[0]

		validDkimKey := types.DkimPubKey{
			Domain:   "example.com",
			Selector: "default",
			PubKey:   validRSAPubKey,
			Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}

		msg := types.NewMsgAddDkimPubKeys(addr, []types.DkimPubKey{validDkimKey})
		tx := mockTx{msgs: []sdk.Msg{msg}}

		newCtx, err := decorator.AnteHandle(ctx, tx, false, nextHandler)
		require.Error(t, err)
		require.False(t, nextCalled)
		require.NotNil(t, newCtx)
	})

	// Non-DKIM message should pass through
	t.Run("non-dkim message passes through", func(t *testing.T) {
		nextCalled = false
		// Create a mock message that's not a DKIM message
		nonDkimMsg := &mockNonDkimMsg{}
		tx := mockTx{msgs: []sdk.Msg{nonDkimMsg}}

		newCtx, err := decorator.AnteHandle(ctx, tx, false, nextHandler)
		require.NoError(t, err)
		require.True(t, nextCalled)
		require.NotNil(t, newCtx)
	})

	// RecheckTx should bypass validation
	t.Run("recheck tx bypasses validation", func(t *testing.T) {
		nextCalled = false
		addrs := simtestutil.CreateIncrementalAccounts(1)
		addr := addrs[0]

		validDkimKey := types.DkimPubKey{
			Domain:   "example.com",
			Selector: "default",
			PubKey:   validRSAPubKey,
			Version:  types.Version_VERSION_DKIM1_UNSPECIFIED,
			KeyType:  types.KeyType_KEY_TYPE_RSA_UNSPECIFIED,
		}

		msg := types.NewMsgAddDkimPubKeys(addr, []types.DkimPubKey{validDkimKey})
		tx := mockTx{msgs: []sdk.Msg{msg}}

		// Set context to recheck mode
		recheckCtx := ctx.WithIsReCheckTx(true)

		newCtx, err := decorator.AnteHandle(recheckCtx, tx, false, nextHandler)
		require.NoError(t, err)
		require.True(t, nextCalled)
		require.NotNil(t, newCtx)
	})

}

// mockNonDkimMsg is a mock message that's not a DKIM message
type mockNonDkimMsg struct{}

func (m *mockNonDkimMsg) ProtoMessage()                {}
func (m *mockNonDkimMsg) Reset()                       {}
func (m *mockNonDkimMsg) String() string               { return "mockNonDkimMsg" }
func (m *mockNonDkimMsg) ValidateBasic() error         { return nil }
func (m *mockNonDkimMsg) GetSignBytes() []byte         { return nil }
func (m *mockNonDkimMsg) GetSigners() []sdk.AccAddress { return nil }
func (m *mockNonDkimMsg) Route() string                { return "test" }
func (m *mockNonDkimMsg) Type() string                 { return "mock" }
