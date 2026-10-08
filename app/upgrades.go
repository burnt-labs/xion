package app

import (
	"context"
	"encoding/hex"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

const UpgradeName = "v32"

// checksumAccountCode is a code whose accounts were registered, before the
// (sender, salt) registry existed, at addresses derived from the code's
// checksum rather than from the address derivation hash.
type checksumAccountCode struct {
	codeID   uint64
	checksum string
}

// checksumAccountCodes lists those codes per chain. On xion-testnet-2, code
// 1880 registered accounts this way from governance proposal 63 until v31.
var checksumAccountCodes = map[string][]checksumAccountCode{
	"xion-testnet-2": {{codeID: 1880, checksum: "D27A379FF65EB47A9E538E3A3D46101DE2A6C0B86BA3D0BF014C0403849414E6"}},
}

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
	storeUpgrades := nextStoreUpgrades(upgradeInfo.Name)
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

func nextStoreUpgrades(_ string) storetypes.StoreUpgrades {
	storeUpgrades := storetypes.StoreUpgrades{
		Added:   []string{},
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
	return storeUpgrades
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

	// v32
	if err := app.backfillChecksumAccountAddresses(sdkCtx); err != nil {
		return nil, fmt.Errorf("backfill checksum account addresses: %w", err)
	}

	sdkCtx.Logger().Info("upgrade complete", "name", plan.Name)
	return migrations, err
}

// backfillChecksumAccountAddresses registers the (sender, salt) entries of the
// chain's checksum-derived accounts (see checksumAccountCodes). Chains without
// such accounts, including a local chain that only shares the chain ID and has
// no such code, are left unchanged. A code that exists with another checksum
// fails the upgrade.
func (app *WasmApp) backfillChecksumAccountAddresses(ctx sdktypes.Context) error {
	for _, code := range checksumAccountCodes[ctx.ChainID()] {
		if app.WasmKeeper.GetCodeInfo(ctx, code.codeID) == nil {
			ctx.Logger().Info("no checksum-derived abstract accounts to register: code not on chain", "code_id", code.codeID)
			continue
		}
		checksum, err := hex.DecodeString(code.checksum)
		if err != nil {
			return fmt.Errorf("decode checksum of code %d: %w", code.codeID, err)
		}
		result, err := app.AbstractAccountKeeper.BackfillChecksumAccountAddresses(ctx, code.codeID, checksum)
		if err != nil {
			return err
		}
		ctx.Logger().Info(
			"registered checksum-derived abstract account addresses",
			"code_id", code.codeID,
			"registered", result.Registered,
			"skipped", result.Skipped,
		)
	}

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
