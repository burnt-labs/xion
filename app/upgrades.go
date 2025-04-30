package app

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const UpgradeName = "v19"

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

			ctx := context.Background()
			minCommission := math.LegacyMustNewDecFromStr("0.05")
			stakingParams, err := app.StakingKeeper.GetParams(ctx)
			if err != nil {
				panic(fmt.Sprintf("failed to get staking params %s", err))
			}
			stakingParams.MinCommissionRate = minCommission
			err = app.StakingKeeper.SetParams(ctx, stakingParams)
			if err != nil {
				panic(fmt.Sprintf("failed to set staking params %s", err))
			}

			err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, validator stakingtypes.ValidatorI) (stop bool) {
				if validator.GetCommission().LT(minCommission) {
					val := validator.(stakingtypes.Validator)
					_, err = app.StakingKeeper.UpdateValidatorCommission(ctx, val, minCommission)
					if err != nil {
						return true
					}
				}
				return false
			})
			if err != nil {
				panic(fmt.Sprintf("failed to update validator commission %s", err))
			}
		}
	}
}

func (app *WasmApp) WrapSetUpgradeHandler(upgradeName string) {
	app.Logger().Info("setting upgrade handler", "name", upgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
			sdkCtx := sdktypes.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Info("running module migrations", "name", plan.Name)
			return app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
		},
	)
}
