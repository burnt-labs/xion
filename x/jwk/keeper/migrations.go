package keeper

import (
	"fmt"

	corestore "cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

type Migrator struct {
	k            Keeper
	legacyParams corestore.KVStoreService
	amino        *codec.LegacyAmino
}

func NewMigrator(k Keeper, legacyParams corestore.KVStoreService, amino *codec.LegacyAmino) Migrator {
	return Migrator{k: k, legacyParams: legacyParams, amino: amino}
}

func (m Migrator) Migrate3To4(ctx sdk.Context) error {
	params := types.DefaultParams()
	store := m.legacyParams.OpenKVStore(ctx)
	for _, field := range []struct {
		key   string
		value *uint64
	}{
		{"jwk/TimeOffset", &params.TimeOffset},
		{"jwk/DeploymentGas", &params.DeploymentGas},
	} {
		bz, err := store.Get([]byte(field.key))
		if err != nil {
			return fmt.Errorf("read %s: %w", field.key, err)
		}
		if bz != nil {
			if err := m.amino.UnmarshalJSON(bz, field.value); err != nil {
				return fmt.Errorf("decode %s: %w", field.key, err)
			}
		}
	}
	return m.k.SetParams(ctx, params)
}
