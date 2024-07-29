package mint

import (
	"testing"

	"go.uber.org/mock/gomock"

	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	// banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"

	"github.com/burnt-labs/xion/x/mint/keeper"
	minttestutil "github.com/burnt-labs/xion/x/mint/testutil"
	minttypes "github.com/burnt-labs/xion/x/mint/types"
)

type mocks struct {
	minttestutil.MockAccountKeeper
	minttestutil.MockBankKeeper
	minttestutil.MockStakingKeeper
	moduleAccount sdk.ModuleAccountI
}

func createTestBaseKeeperAndContextWithMocks(t *testing.T) (testutil.TestContext, *keeper.Keeper, mocks) {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModuleBasic{})
	key := storetypes.NewKVStoreKey(minttypes.StoreKey)
	store := runtime.NewKVStoreService(key)
	testCtx := testutil.DefaultContextWithDB(t, key, storetypes.NewTransientStoreKey("transient_test"))

	// gomock initializations
	ctrl := gomock.NewController(t)
	accountKeeper := minttestutil.NewMockAccountKeeper(ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := minttestutil.NewMockStakingKeeper(ctrl)

	mintAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName, "fee_collector")
	accountKeeper.EXPECT().GetModuleAddress(minttypes.ModuleName).Return(sdk.AccAddress{})
	accountKeeper.EXPECT().GetModuleAccount(testCtx.Ctx, authtypes.FeeCollectorName).Return(mintAcc)

	keeper := keeper.NewKeeper(
		encCfg.Codec,
		store,
		stakingKeeper,
		accountKeeper,
		bankKeeper,
		authtypes.FeeCollectorName,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	params := minttypes.DefaultParams()
	if err := keeper.SetParams(testCtx.Ctx, params); err != nil {
		t.FailNow()
	}
	// keeper.SetMinter(testCtx.Ctx, minttypes.DefaultInitialMinter()) // TODO: minter needs to be parametrized!!

	return testCtx, &keeper, mocks{*accountKeeper, *bankKeeper, *stakingKeeper, mintAcc}
}
