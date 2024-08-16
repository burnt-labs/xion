package app

import (
	"context"
	"fmt"

	ibcclientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v10"

func (app *WasmApp) RegisterUpgradeHandlers() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}
	app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "expectedName", UpgradeName, "height", upgradeInfo.Height)

	if !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		app.Logger().Info("setting upgrade store loaders")
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storetypes.StoreUpgrades{}))
	}

	app.V10SetUpgradeHandler(upgradeInfo)
}

func (app *WasmApp) V10SetUpgradeHandler(upgradeInfo upgradetypes.Plan) {
	app.UpgradeKeeper.SetUpgradeHandler(upgradeInfo.Name, func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)

		sdkCtx.Logger().Info("running module migrations", "name", plan.Name)
		vm, err = app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		if err != nil {
			return vm, err
		}

		sdkCtx.Logger().Info("running IBC Client migrations", "name", plan.Name)
		migrator := ibcclientkeeper.NewMigrator(app.IBCKeeper.ClientKeeper)
		if err := migrator.MigrateParams(sdkCtx); err != nil {
			sdkCtx.Logger().Error(fmt.Sprintf("failed to migrate IBC Client params: %s", err.Error()))
		}

		return vm, err
	})
}
