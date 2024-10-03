package app

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v12"

func (app *WasmApp) RegisterUpgradeHandlers() {
	app.WrapSetUpgradeHandler(UpgradeName)
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}
	app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)

	if upgradeInfo.Name == UpgradeName {
		if !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
			storeUpgrades := storetypes.StoreUpgrades{}

			app.Logger().Info("setting upgrade store loaders")
			app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		}
	}
}

func (app *WasmApp) WrapSetUpgradeHandler(upgradeName string) {
	app.Logger().Info("setting upgrade handler", "name", upgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Info("running module migrations", "name", plan.Name)
			return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		},
	)
}
