package app

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/baseapp"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeName = "v33"

// Kept mounted for the v32 globalfee/jwk migrations; delete in the next upgrade.
const legacyParamsStoreKey = "params"

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
	storeUpgrades := nextStoreUpgrades()
	if len(storeUpgrades.Added) != 0 {
		app.Logger().Info("upgrade", upgradeInfo.Name, "will add stores", storeUpgrades.Added)
	}
	if len(storeUpgrades.Renamed) != 0 {
		app.Logger().Info("upgrade", upgradeInfo.Name, "will rename stores", storeUpgrades.Renamed)
	}
	if len(storeUpgrades.Deleted) != 0 {
		app.Logger().Info("upgrade", upgradeInfo.Name, "will delete stores", storeUpgrades.Deleted)
	}
	storeLoader = upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades)
	return storeLoader
}

// nextStoreUpgrades lists the stores the upgrade adds, renames, or deletes. v33
// changes none; v32 already deleted the retired module stores.
func nextStoreUpgrades() storetypes.StoreUpgrades {
	return storetypes.StoreUpgrades{
		Added:   []string{},
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
}

// getExistingStoreNames returns a map of store names that already exist in the database.
func (app *WasmApp) getExistingStoreNames() map[string]bool {
	existingStores := make(map[string]bool)

	cms := app.CommitMultiStore()
	latestVersion := cms.LatestVersion()
	if latestVersion == 0 {
		return existingStores
	}

	if rootStore, ok := cms.(interface {
		GetCommitInfo(ver int64) (*storetypes.CommitInfo, error)
	}); ok {
		commitInfo, err := rootStore.GetCommitInfo(latestVersion)
		if err != nil {
			app.Logger().Error("failed to get commit info", "version", latestVersion, "error", err)
			return existingStores
		}
		for _, storeInfo := range commitInfo.GetStoreInfos() {
			existingStores[storeInfo.Name] = true
		}
	}

	return existingStores
}

// NextUpgradeHandler is the upgrade handler that is called during the upgrade process.
func (app *WasmApp) NextUpgradeHandler(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("running module migrations", "name", plan.Name)

	// Initialize module if not already initialized
	// if !app.isModuleInitialized(ctx, app.<module>Keeper.Params) {
	// 	sdkCtx.Logger().Info("initializing <module> module")
	// 	<module>Genesis := <module>types.DefaultGenesisState()
	// 	app.<module>Keeper.InitGenesis(sdkCtx, <module>Genesis)
	// }

	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}

	// v33
	if err := app.migrateFeeAllowanceQueueValues(sdkCtx); err != nil {
		return nil, fmt.Errorf("migrate feegrant queue values: %w", err)
	}

	sdkCtx.Logger().Info("upgrade complete", "name", plan.Name)
	return migrations, err
}

// migrateFeeAllowanceQueueValues rewrites the fee allowance queue entries that
// cosmossdk.io/x/feegrant stored with an empty value to the BoolValue encoding
// the SDK v0.55 feegrant keeper reads. RemoveExpiredAllowances decodes the value
// of every expired entry, and BoolValue rejects an empty one, so a single legacy
// entry fails the feegrant EndBlocker once it expires; xion-testnet-2 halted
// that way at height 19286361. The module kept consensus version 2, so
// RunMigrations does not cover this. Entries already holding a value are left
// alone, which makes the rewrite idempotent.
func (app *WasmApp) migrateFeeAllowanceQueueValues(ctx sdktypes.Context) error {
	value, err := collections.BoolValue.Encode(true)
	if err != nil {
		return err
	}
	store := ctx.KVStore(app.GetKey(feegrant.StoreKey))
	it := storetypes.KVStorePrefixIterator(store, feegrant.FeeAllowanceQueueKeyPrefix)
	var legacy [][]byte
	for ; it.Valid(); it.Next() {
		if len(it.Value()) == 0 {
			legacy = append(legacy, it.Key())
		}
	}
	if err := it.Close(); err != nil {
		return err
	}
	for _, key := range legacy {
		store.Set(key, value)
	}
	ctx.Logger().Info("migrated feegrant queue values", "entries", len(legacy))
	return nil
}

// isModuleInitialized checks if a module has been initialized by checking if its params exist.
func (app *WasmApp) isModuleInitialized(ctx context.Context, params interface {
	Has(context.Context) (bool, error)
},
) bool {
	has, err := params.Has(ctx)
	if err != nil {
		// If there's an error checking, assume not initialized
		return false
	}
	return has
}
