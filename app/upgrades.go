package app

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	circuittypes "cosmossdk.io/x/circuit/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const UpgradeName = "v20"

func (app *WasmApp) RegisterUpgradeHandlers() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	// Set UpgradeHandler to NextUpgradeHandler
	app.Logger().Info("setting upgrade handler", "name", UpgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeName, app.NextUpgradeHandler)

	// Set if we see the correct upgrade name on startup
	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)
		app.SetStoreLoader(app.NextStoreLoader(upgradeInfo))
	}
}

// NextStoreLoader is the store loader that is called during the upgrade process.
func (app *WasmApp) NextStoreLoader(upgradeInfo upgradetypes.Plan) (storeLoader baseapp.StoreLoader) {
	storeUpgrades := storetypes.StoreUpgrades{
		Added: []string{
			circuittypes.ModuleName,
		},
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
	if len(storeUpgrades.Added) != 0 {
		app.Logger().Info("upgrade", upgradeInfo.Name, "will add stores", storeUpgrades.Added)
	}
	if len(storeUpgrades.Renamed) != 0 {
		app.Logger().Info("upgrade", upgradeInfo.Name, "will rename stores", storeUpgrades.Renamed)
	}
	if len(storeUpgrades.Deleted) == 0 {
		app.Logger().Info("upgrade", upgradeInfo.Name, "will delete stores", storeUpgrades.Deleted)
	}
	storeLoader = upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades)
	return storeLoader
}

// NextUpgradeHandler is the upgrade handler that is called during the upgrade process.
func (app *WasmApp) NextUpgradeHandler(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("running module migrations", "name", plan.Name)

	// Set the new parameters for mint and staking
	if err := app.V20StakingForceMinimumCommission(ctx); err != nil {
		panic(fmt.Sprintf("failed set minimum commissions: %s", err))
	}

	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}

	return migrations, err
}

// V19StakingParamsChange is a migration function that sets the minimum commission rate for validators to 0.05
func (app *WasmApp) V20StakingForceMinimumCommission(ctx context.Context) (err error) {
	// Set the minimum commission rate to 0.05
	minCommission := math.LegacyMustNewDecFromStr("0.05")

	// Iterate over all validators and update their commission rate if it's less than 0.05
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, validator stakingtypes.ValidatorI) (stop bool) {
		if validator.GetCommission().LT(minCommission) {
			val := validator.(stakingtypes.Validator)
			val.Commission = stakingtypes.NewCommission(minCommission, val.Commission.MaxRate, val.Commission.MaxChangeRate)
			// UpdateValidatorCommission has some sanity checks but does not save the validator
			_, err := app.StakingKeeper.UpdateValidatorCommission(ctx, val, minCommission)
			if err != nil {
				return true
			}
			// SetValidator sets the main record holding validator details
			err = app.StakingKeeper.SetValidator(ctx, val)
			if err != nil {
				return true
			}
		}
		return false
	})
	if err != nil {
		return fmt.Errorf("failed to update validator commission %s", err)
	}
	return nil
}
