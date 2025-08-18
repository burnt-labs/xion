package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/store/types"

	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/core/genesis"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	moduletypes "github.com/cosmos/cosmos-sdk/types/module"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func skipModules(modules []string, toSkip ...string) []string {
	skipMap := make(map[string]bool)
	for _, module := range toSkip {
		skipMap[module] = true
	}

	var result []string
	for _, module := range modules {
		if !skipMap[module] {
			result = append(result, module)
		}
	}
	return result
}

// ExportAppStateAndValidators exports the state of the application for a genesis
// file.
func (app *WasmApp) ExportAppStateAndValidators(forZeroHeight bool, jailAllowedAddrs, modulesToExport []string) (servertypes.ExportedApp, error) {
	// as if they could withdraw from the start of the next block
	ctx := app.NewContextLegacy(true, cmtproto.Header{Height: app.LastBlockHeight()})

	// We export at last height + 1, because that's the height at which
	// CometBFT will start InitChain.
	height := app.LastBlockHeight() + 1
	if forZeroHeight {
		height = 0
		app.prepForZeroHeightGenesis(ctx, jailAllowedAddrs)
	}

	if len(modulesToExport) == 0 {
		// skip authz, feegrant, wasm they are exported in the genextensions module
		modulesToExport = skipModules(app.ModuleManager.OrderExportGenesis, "authz", "feegrant", "wasm")
	}

	genState, err := app.ExportGenesisForModules(ctx, modulesToExport)
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	appState, err := json.MarshalIndent(genState, "", "  ")
	if err != nil {
		return servertypes.ExportedApp{}, err
	}

	validators, err := staking.WriteValidators(ctx, app.StakingKeeper)
	return servertypes.ExportedApp{
		AppState:        appState,
		Validators:      validators,
		Height:          height,
		ConsensusParams: app.GetConsensusParams(ctx),
	}, err
}

// prepare for fresh start at zero height
// NOTE zero height genesis is a temporary feature which will be deprecated
//
//	in favor of export at a block height
func (app *WasmApp) prepForZeroHeightGenesis(ctx sdk.Context, jailAllowedAddrs []string) {
	applyAllowedAddrs := len(jailAllowedAddrs) > 0

	// check if there is a allowed address list

	allowedAddrsMap := make(map[string]bool)

	for _, addr := range jailAllowedAddrs {
		_, err := sdk.ValAddressFromBech32(addr)
		if err != nil {
			log.Fatal(err)
		}
		allowedAddrsMap[addr] = true
	}

	/* Just to be safe, assert the invariants on current state. */
	app.CrisisKeeper.AssertInvariants(ctx)

	/* Handle fee distribution state. */

	// withdraw all validator commission
	err := app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		_, _ = app.DistrKeeper.WithdrawValidatorCommission(ctx, valBz)
		return false
	})
	if err != nil {
		panic(err)
	}

	// withdraw all delegator rewards
	dels, err := app.StakingKeeper.GetAllDelegations(ctx)
	if err != nil {
		panic(err)
	}

	for _, delegation := range dels {
		valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if err != nil {
			panic(err)
		}

		delAddr := sdk.MustAccAddressFromBech32(delegation.DelegatorAddress)

		if _, err = app.DistrKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr); err != nil {
			panic(err)
		}
	}

	// clear validator slash events
	app.DistrKeeper.DeleteAllValidatorSlashEvents(ctx)

	// clear validator historical rewards
	app.DistrKeeper.DeleteAllValidatorHistoricalRewards(ctx)

	// set context height to zero
	height := ctx.BlockHeight()
	ctx = ctx.WithBlockHeight(0)

	// reinitialize all validators
	err = app.StakingKeeper.IterateValidators(ctx, func(_ int64, val stakingtypes.ValidatorI) (stop bool) {
		valBz, err := app.StakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
		if err != nil {
			panic(err)
		}
		// donate any unwithdrawn outstanding reward fraction tokens to the community pool
		scraps, err := app.DistrKeeper.GetValidatorOutstandingRewardsCoins(ctx, valBz)
		if err != nil {
			panic(err)
		}
		feePool, err := app.DistrKeeper.FeePool.Get(ctx)
		if err != nil {
			panic(err)
		}
		feePool.CommunityPool = feePool.CommunityPool.Add(scraps...)
		if err := app.DistrKeeper.FeePool.Set(ctx, feePool); err != nil {
			panic(err)
		}

		if err := app.DistrKeeper.Hooks().AfterValidatorCreated(ctx, valBz); err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// reinitialize all delegations
	for _, del := range dels {
		valAddr, err := sdk.ValAddressFromBech32(del.ValidatorAddress)
		if err != nil {
			panic(err)
		}
		delAddr := sdk.MustAccAddressFromBech32(del.DelegatorAddress)

		if err := app.DistrKeeper.Hooks().BeforeDelegationCreated(ctx, delAddr, valAddr); err != nil {
			// never called as BeforeDelegationCreated always returns nil
			panic(fmt.Errorf("error while incrementing period: %w", err))
		}

		if err := app.DistrKeeper.Hooks().AfterDelegationModified(ctx, delAddr, valAddr); err != nil {
			// never called as AfterDelegationModified always returns nil
			panic(fmt.Errorf("error while creating a new delegation period record: %w", err))
		}
	}

	// reset context height
	ctx = ctx.WithBlockHeight(height)

	/* Handle staking state. */

	// iterate through redelegations, reset creation height
	err = app.StakingKeeper.IterateRedelegations(ctx, func(_ int64, red stakingtypes.Redelegation) (stop bool) {
		for i := range red.Entries {
			red.Entries[i].CreationHeight = 0
		}
		err = app.StakingKeeper.SetRedelegation(ctx, red)
		if err != nil {
			panic(err)
		}
		return false
	})
	if err != nil {
		panic(err)
	}

	// iterate through unbonding delegations, reset creation height
	err = app.StakingKeeper.IterateUnbondingDelegations(ctx, func(_ int64, ubd stakingtypes.UnbondingDelegation) (stop bool) {
		for i := range ubd.Entries {
			ubd.Entries[i].CreationHeight = 0
		}
		err = app.StakingKeeper.SetUnbondingDelegation(ctx, ubd)
		if err != nil {
			panic(err)
		}
		return false
	})

	// Iterate through validators by power descending, reset bond heights, and
	// update bond intra-tx counters.
	store := ctx.KVStore(app.GetKey(stakingtypes.StoreKey))
	iter := storetypes.KVStoreReversePrefixIterator(store, stakingtypes.ValidatorsKey)

	for ; iter.Valid(); iter.Next() {
		addr := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iter.Key()))
		validator, err := app.StakingKeeper.GetValidator(ctx, addr)
		if err != nil {
			if errors.Is(err, collections.ErrNotFound) {
				panic(fmt.Errorf("expected validator, not found: %w", err))
			}
			panic(err)
		}

		validator.UnbondingHeight = 0
		if applyAllowedAddrs && !allowedAddrsMap[addr.String()] {
			validator.Jailed = true
		}

		err = app.StakingKeeper.SetValidator(ctx, validator)
		if err != nil {
			panic(err)
		}
	}

	if err := iter.Close(); err != nil {
		app.Logger().Error("error while closing the key-value store reverse prefix iterator: ", err)
		return
	}

	_, err = app.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	if err != nil {
		log.Fatal(err)
	}

	/* Handle slashing state. */

	// reset start height on signing infos
	err = app.SlashingKeeper.IterateValidatorSigningInfos(
		ctx,
		func(addr sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) (stop bool) {
			info.StartHeight = 0
			if err := app.SlashingKeeper.SetValidatorSigningInfo(ctx, addr, info); err != nil {
				panic(err)
			}
			return false
		},
	)
	if err != nil {
		panic(err)
	}
}

// ExportGenesisForModules performs export genesis functionality for modules
// Sequentially exports genesis data for each module in the order of the module manager
func (app *WasmApp) ExportGenesisForModules(ctx sdk.Context, modulesToExport []string) (map[string]json.RawMessage, error) {
	if len(modulesToExport) == 0 {
		modulesToExport = app.ModuleManager.OrderExportGenesis
	}

	// verify modules exists in app, so that we don't panic in the middle of an export
	if err := app.checkModulesExists(modulesToExport); err != nil {
		return nil, err
	}

	genesisData := make(map[string]json.RawMessage)

	for _, moduleName := range modulesToExport {
		mod := app.ModuleManager.Modules[moduleName]
		fmt.Println("exporting module", moduleName)
		if module, ok := mod.(appmodule.HasGenesis); ok {
			// core API genesis
			ctx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()) // avoid race conditions
			target := genesis.RawJSONTarget{}
			err := module.ExportGenesis(ctx, target.Target())
			if err != nil {
				return nil, err
			}

			rawJSON, err := target.JSON()
			if err != nil {
				return nil, err
			}
			genesisData[moduleName] = rawJSON
		} else if module, ok := mod.(moduletypes.HasGenesis); ok {
			ctx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()) // avoid race conditions
			genesisData[moduleName] = module.ExportGenesis(ctx, app.appCodec)
		} else if module, ok := mod.(moduletypes.HasABCIGenesis); ok {
			ctx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
			genesisData[moduleName] = module.ExportGenesis(ctx, app.appCodec)
		}
	}
	return genesisData, nil
}

// checkModulesExists verifies that all modules in the list exist in the app
func (app *WasmApp) checkModulesExists(moduleName []string) error {
	for _, name := range moduleName {
		if _, ok := app.ModuleManager.Modules[name]; !ok {
			return fmt.Errorf("module %s does not exist", name)
		}
	}
	return nil
}
