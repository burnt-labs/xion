package mint

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	// banktestutil "github.com/cosmos/cosmos-sdk/x/bank/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/golang/mock/gomock"

	"github.com/burnt-labs/xion/x/mint/keeper"
	minttestutil "github.com/burnt-labs/xion/x/mint/testutil"
	"github.com/burnt-labs/xion/x/mint/types"
	minttypes "github.com/burnt-labs/xion/x/mint/types"
)

type mocks struct {
	minttestutil.MockAccountKeeper
	minttestutil.MockBankKeeper
	minttestutil.MockStakingKeeper
	moduleAccount authtypes.ModuleAccountI
}

func createTestBaseKeeperAndContextWithMocks(t *testing.T) (testutil.TestContext, *keeper.Keeper, mocks) {
	encCfg := moduletestutil.MakeTestEncodingConfig(mint.AppModuleBasic{})
	key := sdk.NewKVStoreKey(types.StoreKey)
	testCtx := testutil.DefaultContextWithDB(t, key, sdk.NewTransientStoreKey("transient_test"))

	// gomock initializations
	ctrl := gomock.NewController(t)
	accountKeeper := minttestutil.NewMockAccountKeeper(ctrl)
	bankKeeper := minttestutil.NewMockBankKeeper(ctrl)
	stakingKeeper := minttestutil.NewMockStakingKeeper(ctrl)

	mintAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName, "fee_collector")
	accountKeeper.EXPECT().GetModuleAddress(types.ModuleName).Return(sdk.AccAddress{})
	accountKeeper.EXPECT().GetModuleAccount(testCtx.Ctx, authtypes.FeeCollectorName).Return(mintAcc)

	keeper := keeper.NewKeeper(
		encCfg.Codec,
		key,
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
