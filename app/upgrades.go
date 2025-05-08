package app

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const UpgradeName = "v19"

func (app *WasmApp) RegisterUpgradeHandlers() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}
	app.WrapSetUpgradeHandler(UpgradeName)

	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{}

		app.Logger().Info("setting upgrade store loaders")
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
		app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)
	}
}

func (app *WasmApp) WrapSetUpgradeHandler(upgradeName string) {
	app.Logger().Info("setting upgrade handler", "name", upgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(
		upgradeName,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
			sdkCtx := sdktypes.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Info("running module migrations", "name", plan.Name)

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

			// Retrieve xion mint params
			p, err := app.XionKeeper.XionLegacyMintKeeper.GetParams(sdkCtx)
			if err != nil {
				panic(fmt.Sprintf("failed to retrieve params from legacy Xion Mint: %s", err))
			}
			// Prepare subspace for mint module migration
			mintSubspace := app.GetSubspace(minttypes.ModuleName).WithKeyTable(minttypes.ParamKeyTable()) //nolint:staticcheck
			mintSubspace.SetParamSet(sdkCtx, &p)

			migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
			if err != nil {
				panic(fmt.Sprintf("failed to run migrations: %s", err))
			}

			return migrations, err
		},
	)
}
