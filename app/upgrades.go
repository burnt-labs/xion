package app

import (
	"context"
	"crypto/sha256"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v29"

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
	// Check which stores already exist (for chains that had v26)
	// existingStores := app.getExistingStoreNames()

	var storesToAdd []string
	// if !existingStores[<module>.StoreKey] {
	// 	storesToAdd = append(storesToAdd, <module>.StoreKey)
	// }

	storeUpgrades := storetypes.StoreUpgrades{
		Added:   storesToAdd,
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
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

	// Remove testnet audience that contains leaked RSA private key material.
	// This audience would fail the stricter ValidateGenesis checks added in v29,
	// blocking any future genesis export/import cycle on testnet-2.
	const leakedAud = "poc-leaked-private-key"
	if _, found := app.JwkKeeper.GetAudience(sdkCtx, leakedAud); found {
		app.JwkKeeper.RemoveAudience(sdkCtx, leakedAud)
		audHash := sha256.Sum256([]byte(leakedAud))
		app.JwkKeeper.RemoveAudienceClaim(sdkCtx, audHash[:])
		sdkCtx.Logger().Info("removed audience with leaked private key material", "aud", leakedAud)
	}

	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}

	sdkCtx.Logger().Info("upgrade complete", "name", plan.Name)
	return migrations, err
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
