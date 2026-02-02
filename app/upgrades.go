package app

import (
	"context"
	"fmt"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	dkimtypes "github.com/burnt-labs/xion/x/dkim/types"
	zktypes "github.com/burnt-labs/xion/x/zk/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const TestnetUpgradeName = "v27-testnet-upgrade"
const MainnetUpgradeName = "v27-mainnet-upgrade"

func (app *WasmApp) RegisterUpgradeHandlers() {
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Sprintf("failed to read upgrade info from disk %s", err))
	}

	// Set UpgradeHandler to NextUpgradeHandler
	app.Logger().Info("setting upgrade handler", "name", TestnetUpgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(TestnetUpgradeName, app.NextUpgradeHandler)
	app.Logger().Info("setting upgrade handler", "name", MainnetUpgradeName)
	app.UpgradeKeeper.SetUpgradeHandler(MainnetUpgradeName, app.NextMainnetV27UpgradeHandler)

	// Set if we see the correct upgrade name on startup
	if upgradeInfo.Name == TestnetUpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)
		app.SetStoreLoader(app.NextStoreLoader(upgradeInfo))
	}
	if upgradeInfo.Name == MainnetUpgradeName && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		app.Logger().Info("upgrade info", "name", upgradeInfo.Name, "height", upgradeInfo.Height)
		app.SetStoreLoader(app.NextMainnetV27StoreLoader(upgradeInfo))
	}
}

// NextStoreLoader is the store loader that is called during the upgrade process.
func (app *WasmApp) NextStoreLoader(upgradeInfo upgradetypes.Plan) (storeLoader baseapp.StoreLoader) {
	storeUpgrades := storetypes.StoreUpgrades{
		Added:   []string{},
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
	LogStoreUpgrades(app.Logger(), upgradeInfo.Name, storeUpgrades)
	storeLoader = upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades)
	return storeLoader
}

// NextUpgradeHandler is the upgrade handler that is called during the upgrade process.
func (app *WasmApp) NextUpgradeHandler(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("running module migrations", "name", plan.Name)

	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}

	sdkCtx.Logger().Info("upgrade complete", "name", plan.Name)
	return migrations, err
}

// NextStoreMainnetV27StoreLoader is the store loader that is called during the upgrade process.
func (app *WasmApp) NextMainnetV27StoreLoader(upgradeInfo upgradetypes.Plan) (storeLoader baseapp.StoreLoader) {
	storeUpgrades := storetypes.StoreUpgrades{
		Added:   []string{dkimtypes.StoreKey, zktypes.StoreKey},
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
	LogStoreUpgrades(app.Logger(), upgradeInfo.Name, storeUpgrades)
	storeLoader = upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades)
	return storeLoader
}

// NextMainnetV27UpgradeHandler is the upgrade handler that is called during the upgrade process.
func (app *WasmApp) NextMainnetV27UpgradeHandler(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (vm module.VersionMap, err error) {
	sdkCtx := sdktypes.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("running module migrations", "name", plan.Name)

	// Initialize new zk module
	zkGenesis := zktypes.DefaultGenesisState()
	app.ZkKeeper.InitGenesis(sdkCtx, zkGenesis)

	// Initialize new dkim module
	dkimGenesis := dkimtypes.DefaultGenesis()
	if err := app.DkimKeeper.InitGenesis(sdkCtx, dkimGenesis); err != nil {
		return nil, err
	}
	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}

	sdkCtx.Logger().Info("upgrade complete", "name", plan.Name)
	return migrations, err
}

func LogStoreUpgrades(logger log.Logger, upgradeName string, storeUpgrades storetypes.StoreUpgrades) {
	if len(storeUpgrades.Added) != 0 {
		logger.Info("upgrade", upgradeName, "will add stores", storeUpgrades.Added)
	}
	if len(storeUpgrades.Renamed) != 0 {
		logger.Info("upgrade", upgradeName, "will rename stores", storeUpgrades.Renamed)
	}
	if len(storeUpgrades.Deleted) != 0 {
		logger.Info("upgrade", upgradeName, "will delete stores", storeUpgrades.Deleted)
	}
}
