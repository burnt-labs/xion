package v10

import (
	"context"
	"fmt"

	ibcclientkeeper "github.com/cosmos/ibc-go/v8/modules/core/02-client/keeper"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

var ibcClientKeeper ibcclientkeeper.Keeper

func SetIBCClientKeeper(k ibcclientkeeper.Keeper) {
	ibcClientKeeper = k
}

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("Starting module migrations...")

		vm, err := mm.RunMigrations(ctx, configurator, vm)
		if err != nil {
			return vm, err
		}

		migrator := ibcclientkeeper.NewMigrator(ibcClientKeeper)
		if err := migrator.MigrateParams(sdkCtx); err != nil {
			sdkCtx.Logger().Error(fmt.Sprintf("failed to migrate IBC Client params: %s", err.Error()))
		}

		sdkCtx.Logger().Info(fmt.Sprintf("Software Upgrade %s complete", UpgradeName))
		return vm, err
	}
}
