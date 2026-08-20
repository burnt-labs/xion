package app

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"slices"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	aatypes "github.com/burnt-labs/xion/x/abstractaccount/types"
)

const UpgradeName = "v31"

const removedIBCWasmStoreKey = "08-wasm"

const (
	mainnetAddressDerivationHash = "FEFA4D0C57F6CA47A5D89C6F077A176D26027DB4EEFA758A929DD4C4AAF17D1B"
	testnetAddressDerivationHash = "FC06F022C95172F54AD05BC07214F50572CDF684459EADD4F58A765524567DB8"
)

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

func nextStoreUpgrades(upgradeName string) storetypes.StoreUpgrades {
	storeUpgrades := storetypes.StoreUpgrades{
		Added:   []string{},
		Renamed: []storetypes.StoreRename{},
		Deleted: []string{},
	}
	if upgradeName == UpgradeName {
		storeUpgrades.Deleted = []string{removedIBCWasmStoreKey}
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

	// v31
	app.addVeronaDenomMetadataAliases(sdkCtx)

	// Run the migrations for all modules
	migrations, err := app.ModuleManager.RunMigrations(ctx, app.Configurator(), fromVM)
	if err != nil {
		panic(fmt.Sprintf("failed to run migrations: %s", err))
	}
	if err := app.configureAbstractAccountAddressDerivation(sdkCtx); err != nil {
		return nil, fmt.Errorf("configure abstract account address derivation: %w", err)
	}

	sdkCtx.Logger().Info("upgrade complete", "name", plan.Name)
	return migrations, err
}

func (app *WasmApp) configureAbstractAccountAddressDerivation(ctx sdktypes.Context) error {
	hashHex, configuredChain := map[string]string{
		"xion-mainnet-1": mainnetAddressDerivationHash,
		"xion-testnet-2": testnetAddressDerivationHash,
	}[ctx.ChainID()]
	if !configuredChain {
		ctx.Logger().Info(
			"abstract account fixed-hash registration remains disabled on unsupported chain",
			"chain_id", ctx.ChainID(),
		)

		return nil
	}

	addressHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("decode address derivation hash for %s: %w", ctx.ChainID(), err)
	}
	params, err := app.AbstractAccountKeeper.GetParams(ctx)
	if err != nil {
		return err
	}
	if params.RegistrationConfigured() && !bytes.Equal(params.AddressDerivationHash, addressHash) {
		return aatypes.ErrImmutableAddressHash.Wrapf(
			"chain %s expected %s, found %s",
			ctx.ChainID(),
			hashHex,
			hex.EncodeToString(params.AddressDerivationHash),
		)
	}

	params.AddressDerivationHash = addressHash
	params.RegistrationEnabled = true
	if err := app.AbstractAccountKeeper.SetParams(ctx, params); err != nil {
		return err
	}

	ctx.Logger().Info(
		"configured abstract account fixed-hash registration",
		"chain_id", ctx.ChainID(),
		"address_derivation_hash", hashHex,
	)

	return nil
}

func (app *WasmApp) addVeronaDenomMetadataAliases(ctx sdktypes.Context) {
	metadata, found := app.BankKeeper.GetDenomMetaData(ctx, "uxion")
	if !found {
		metadata = banktypes.Metadata{
			Description: "The native staking token of the Xion network.",
			Base:        "uxion",
			Display:     "XION",
			Name:        "xion",
			Symbol:      "XION",
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    "uxion",
					Exponent: 0,
					Aliases:  []string{"microxion"},
				},
				{
					Denom:    "mxion",
					Exponent: 3,
					Aliases:  []string{"millixion"},
				},
				{
					Denom:    "XION",
					Exponent: 6,
					Aliases:  []string{"xion"},
				},
			},
		}
	}
	metadata.DenomUnits = ensureXionDenomUnits(metadata.DenomUnits)

	for _, unit := range metadata.DenomUnits {
		switch unit.Denom {
		case "uxion":
			unit.Aliases = appendMissingAliases(unit.Aliases, "uverona", "microverona")
		case "mxion":
			unit.Aliases = appendMissingAliases(unit.Aliases, "mverona", "milliverona")
		case "XION":
			unit.Aliases = appendMissingAliases(unit.Aliases, "xion", "verona", "VERONA")
		}
	}

	app.BankKeeper.SetDenomMetaData(ctx, metadata)
	ctx.Logger().Info("added Verona aliases to uxion denom metadata")
}

func ensureXionDenomUnits(units []*banktypes.DenomUnit) []*banktypes.DenomUnit {
	required := []*banktypes.DenomUnit{
		{
			Denom:    "uxion",
			Exponent: 0,
			Aliases:  []string{"microxion"},
		},
		{
			Denom:    "mxion",
			Exponent: 3,
			Aliases:  []string{"millixion"},
		},
		{
			Denom:    "XION",
			Exponent: 6,
			Aliases:  []string{"xion"},
		},
	}

	for _, requiredUnit := range required {
		if hasDenomUnit(units, requiredUnit.Denom) {
			continue
		}
		units = append(units, requiredUnit)
	}
	slices.SortStableFunc(units, func(a, b *banktypes.DenomUnit) int {
		return int(a.Exponent) - int(b.Exponent)
	})

	return units
}

func hasDenomUnit(units []*banktypes.DenomUnit, denom string) bool {
	for _, unit := range units {
		if unit.Denom == denom {
			return true
		}
	}
	return false
}

func appendMissingAliases(existing []string, aliases ...string) []string {
	seen := make(map[string]struct{}, len(existing)+len(aliases))
	for _, alias := range existing {
		seen[alias] = struct{}{}
	}
	for _, alias := range aliases {
		if _, ok := seen[alias]; ok {
			continue
		}
		existing = append(existing, alias)
		seen[alias] = struct{}{}
	}
	return existing
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
