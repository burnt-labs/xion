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

	"github.com/burnt-labs/xion/x/zk/ante"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
	"github.com/burnt-labs/xion/x/zk/types"
)

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

func setupTest(t *testing.T) (sdk.Context, *zkkeeper.Keeper, []byte) {
	t.Helper()

	logger := log.NewTestLogger(t)
	encCfg := moduletestutil.MakeTestEncodingConfig()
	types.RegisterInterfaces(encCfg.InterfaceRegistry)
	authtypes.RegisterInterfaces(encCfg.InterfaceRegistry)

	key := storetypes.NewKVStoreKey(types.ModuleName)
	storeService := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))
	ctx := testCtx.Ctx

	govModAddr := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	keeper := zkkeeper.NewKeeper(encCfg.Codec, storeService, logger, govModAddr)
	keeper.InitGenesis(ctx, types.DefaultGenesisState())

	vkeyBytes := types.DefaultGenesisState().Vkeys[0].Vkey.KeyBytes

	return ctx, &keeper, vkeyBytes
}

func TestZKDecorator(t *testing.T) {
	ctx, keeper, vkeyBytes := setupTest(t)

	decorator := ante.NewZKDecorator(keeper)

	nextCalled := false
	nextHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		nextCalled = true
		return ctx, nil
	}

	t.Run("valid vkey charges gas and passes", func(t *testing.T) {
		nextCalled = false

		decoded, err := types.DecodeAndValidateVKeyBytes(vkeyBytes, types.DefaultMaxVKeySizeBytes)
		require.NoError(t, err)

		params := types.DefaultParams()
		params.MaxVkeySizeBytes = uint64(len(decoded) + 10)
		require.NoError(t, keeper.Params.Set(ctx, params))

		gasCost, err := params.GasCostForSize(uint64(len(decoded)))
		require.NoError(t, err)

		gasMeter := storetypes.NewInfiniteGasMeter()
		addrs := simtestutil.CreateIncrementalAccounts(1)
		msg := &types.MsgAddVKey{
			Authority:   addrs[0].String(),
			Name:        "test",
			Description: "desc",
			VkeyBytes:   vkeyBytes,
		}
		tx := mockTx{msgs: []sdk.Msg{msg}}

		gasBefore := gasMeter.GasConsumed()

		_, err = decorator.AnteHandle(ctx.WithGasMeter(gasMeter), tx, false, nextHandler)
		require.NoError(t, err)
		require.True(t, nextCalled)
		consumed := gasMeter.GasConsumed() - gasBefore
		require.GreaterOrEqual(t, consumed, gasCost)
		require.Less(t, consumed, gasCost+5_000)
	})

	t.Run("oversized vkey rejected", func(t *testing.T) {
		nextCalled = false

		decoded, err := types.DecodeAndValidateVKeyBytes(vkeyBytes, types.DefaultMaxVKeySizeBytes)
		require.NoError(t, err)

		params := types.DefaultParams()
		params.MaxVkeySizeBytes = uint64(len(decoded) / 2)
		require.NoError(t, keeper.Params.Set(ctx, params))

		addrs := simtestutil.CreateIncrementalAccounts(1)
		msg := &types.MsgAddVKey{
			Authority:   addrs[0].String(),
			Name:        "too-big",
			Description: "desc",
			VkeyBytes:   vkeyBytes,
		}
		tx := mockTx{msgs: []sdk.Msg{msg}}

		_, err = decorator.AnteHandle(ctx, tx, false, nextHandler)
		require.Error(t, err)
		require.False(t, nextCalled)
		require.Contains(t, err.Error(), "size")
	})

	t.Run("non zk message passes through", func(t *testing.T) {
		nextCalled = false
		tx := mockTx{msgs: []sdk.Msg{&mockNonZkMsg{}}}

		_, err := decorator.AnteHandle(ctx, tx, false, nextHandler)
		require.NoError(t, err)
		require.True(t, nextCalled)
	})

	t.Run("recheck bypasses validation", func(t *testing.T) {
		nextCalled = false

		params := types.DefaultParams()
		require.NoError(t, keeper.Params.Set(ctx, params))

		addrs := simtestutil.CreateIncrementalAccounts(1)
		msg := &types.MsgAddVKey{
			Authority:   addrs[0].String(),
			Name:        "recheck",
			Description: "desc",
			VkeyBytes:   vkeyBytes,
		}
		tx := mockTx{msgs: []sdk.Msg{msg}}

		recheckCtx := ctx.WithIsReCheckTx(true)
		_, err := decorator.AnteHandle(recheckCtx, tx, false, nextHandler)
		require.NoError(t, err)
		require.True(t, nextCalled)
	})
}

type mockNonZkMsg struct{}

func (m *mockNonZkMsg) ProtoMessage()                {}
func (m *mockNonZkMsg) Reset()                       {}
func (m *mockNonZkMsg) String() string               { return "mockNonZkMsg" }
func (m *mockNonZkMsg) ValidateBasic() error         { return nil }
func (m *mockNonZkMsg) GetSignBytes() []byte         { return nil }
func (m *mockNonZkMsg) GetSigners() []sdk.AccAddress { return nil }
func (m *mockNonZkMsg) Route() string                { return "test" }
func (m *mockNonZkMsg) Type() string                 { return "mock" }
