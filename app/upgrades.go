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
	feeabstypes "github.com/osmosis-labs/fee-abstraction/v8/x/feeabs/types"
)

const UpgradeName = "v19"

func (app *WasmApp) RegisterUpgradeHandlers() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}
	
	app.Logger().Info("setting upgrade handler", "name", UpgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(UpgradeName, app.NextUpgradeHandler)

	if upgradeInfo.Name == UpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)
		app.NextStoreLoader(upgradeInfo)
	}
}

// VersionStoreLoader is the store loader that is called during the upgrade process.
func (app *WasmApp) NextStoreLoader(upgradeInfo upgradetypes.Plan) (err error) {
	app.Logger().Info("setting upgrade store loaders")
	storeUpgrades := storetypes.StoreUpgrades{
		// Added:  []string{""},
		// Deleted: []string{""},
	}
	storeLoader := upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades)
	app.SetStoreLoader(storeLoader)
	return nil
}

// NextUpgradeHandler is the upgrade handler that is called during the upgrade process.
func (app *WasmApp) NextUpgradeHandler(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("running module migrations", "name", plan.Name)

	// Set the new parameters for mint and staking
	if err := app.V19StakingForceMinimumCommission(ctx, plan); err != nil {
		panic(fmt.Sprintf("failed set minimum commissions: %s", err))
	}

	// Set the new parameters for mint
	if err := app.V19MintParamsChange(sdkCtx, plan); err != nil {
		panic(fmt.Sprintf("failed to run mint module migrations: %s", err))
	}

	// Add query and swap epochs to the feeabs module
	if err := app.V19FeeabsEpochAdd(sdkCtx, plan); err != nil {
		panic(fmt.Sprintf("failed to run feeabs module epoch additions: %s", err))
	}

	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}

	return migrations, err
}

// V19StakingParamsChange is a migration function that sets the minimum commission rate for validators to 0.05
func (app *WasmApp) V19StakingForceMinimumCommission(ctx context.Context, plan upgradetypes.Plan) (err error) {
	// Get Staking params
	stakingParams, err := app.StakingKeeper.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get staking params %s", err)
	}

	// Set the minimum commission rate to 0.05
	minCommission := math.LegacyMustNewDecFromStr("0.05")
	stakingParams.MinCommissionRate = minCommission
	err = app.StakingKeeper.SetParams(ctx, stakingParams)
	if err != nil {
		return fmt.Errorf("failed to set staking params %s", err)
	}

	// Iterate over all validators and update their commission rate if it's less than 0.05
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
		return fmt.Errorf("failed to update validator commission %s", err)
	}
	return nil
}

// V19MintUpgrade is a migration function that sets the minting parameters for the xion mint module
func (app *WasmApp) V19MintParamsChange(sdkCtx sdktypes.Context, plan upgradetypes.Plan) (err error) {
	sdkCtx.Logger().Info("running mint module migrations", "name", plan.Name)

	// Retrieve xion mint params
	p, err := app.XionKeeper.XionLegacyMintKeeper.GetParams(sdkCtx)
	if err != nil {
		return fmt.Errorf("failed to retrieve params from legacy Xion Mint: %s", err)
	}
	// Prepare subspace for mint module migration
	mintSubspace := app.GetSubspace(minttypes.ModuleName).WithKeyTable(minttypes.ParamKeyTable()) //nolint:staticcheck
	mintSubspace.SetParamSet(sdkCtx, &p)

	return nil
}

// V19FeeabsEpochAdd adds the query and swap epochs to the feeabs module
func (app *WasmApp) V19FeeabsEpochAdd(sdkCtx sdktypes.Context, plan upgradetypes.Plan) (err error) {
	sdkCtx.Logger().Info("running feeabs module migrations", "name", plan.Name)

	epochs := []feeabstypes.EpochInfo{
		feeabstypes.NewGenesisEpochInfo(feeabstypes.DefaultQueryEpochIdentifier, feeabstypes.DefaultQueryPeriod), 
		feeabstypes.NewGenesisEpochInfo(feeabstypes.DefaultSwapEpochIdentifier, feeabstypes.DefaultSwapPeriod),
	}

	// Set the epochs
	for _, epoch := range epochs {
		err := app.FeeAbsKeeper.AddEpochInfo(sdkCtx, epoch)
		if err != nil {
			return err
		}
	}

	return nil
}
