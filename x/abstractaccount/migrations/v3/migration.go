package v3

import (
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

// MigrateStore adds the fixed address derivation parameters in an unconfigured,
// paused state. The chain-specific hash must be configured by the XION upgrade
// before registration is enabled.
func MigrateStore(ctx sdk.Context, key storetypes.StoreKey, cdc codec.BinaryCodec) error {
	store := ctx.KVStore(key)
	bz := store.Get(types.KeyParams)
	if bz == nil {
		return types.ErrParsingParams.Wrap("x/abstractaccount module params not found")
	}

	var params types.Params
	if err := cdc.Unmarshal(bz, &params); err != nil {
		return types.ErrParsingParams.Wrap(err.Error())
	}
	params.AddressDerivationHash = nil
	params.RegistrationEnabled = false

	bz, err := cdc.Marshal(&params)
	if err != nil {
		return types.ErrParsingParams.Wrap(err.Error())
	}
	store.Set(types.KeyParams, bz)

	return nil
}
