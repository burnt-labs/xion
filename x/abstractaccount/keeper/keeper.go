package keeper

import (
	log "cosmossdk.io/log"

	storetypes "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService storetypes.KVStoreService
	ak           authkeeper.AccountKeeperI
	ck           wasmtypes.ContractOpsKeeper
	authority    string
}

func NewKeeper(
	cdc codec.BinaryCodec, storeService storetypes.KVStoreService,
	ak authkeeper.AccountKeeperI, ck wasmtypes.ContractOpsKeeper,
	authority string,
) Keeper {
	if ak == nil {
		panic("AccountKeeperI cannot be nil")
	}

	if ck == nil {
		panic("ContractOpsKeeper cannot be nil")
	}

	return Keeper{cdc, storeService, ak, ck, authority}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) ContractKeeper() wasmtypes.ContractOpsKeeper {
	return k.ck
}

// ---------------------------------- Params -----------------------------------

func (k Keeper) GetParams(ctx sdk.Context) (*types.Params, error) {
	store := k.storeService.OpenKVStore(ctx)

	bz, err := store.Get(types.KeyParams)
	if err != nil {
		return nil, err
	}
	if bz == nil {
		return nil, sdkerrors.ErrNotFound.Wrap("x/abstractaccount module params")
	}

	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return nil, types.ErrParsingParams.Wrap(err.Error())
	}

	return &params, nil
}

func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) error {
	store := k.storeService.OpenKVStore(ctx)

	// params must be valid before we save it
	// there are two instances where SetParams is called - in Keeper.InitGenesis,
	// and in msgServer.UpdateParams
	// we can either perform the validation in those two functions, or do it
	// together here. doing it here seems cleaner.
	if err := params.Validate(); err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(params)
	if err != nil {
		return types.ErrParsingParams.Wrap(err.Error())
	}

	return store.Set(types.KeyParams, bz)
}

// ------------------------------- NextAccountId -------------------------------

func (k Keeper) GetAndIncrementNextAccountID(ctx sdk.Context) (uint64, error) {
	id, err := k.GetNextAccountID(ctx)
	if err != nil {
		return 0, err
	}

	err = k.SetNextAccountID(ctx, id+1)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (k Keeper) GetNextAccountID(ctx sdk.Context) (uint64, error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.KeyNextAccountID)
	if err != nil {
		return 0, err
	}

	return sdk.BigEndianToUint64(bz), nil
}

func (k Keeper) SetNextAccountID(ctx sdk.Context, id uint64) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.KeyNextAccountID, sdk.Uint64ToBigEndian(id))
}

// ------------------------------- SignerAddress -------------------------------

func (k Keeper) GetSignerAddress(ctx sdk.Context) (sdk.AccAddress, error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.KeySignerAddress)
	if err != nil {
		return nil, err
	}

	return bz, nil
}

func (k Keeper) SetSignerAddress(ctx sdk.Context, signerAddr sdk.AccAddress) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Set(types.KeySignerAddress, signerAddr)
}

func (k Keeper) DeleteSignerAddress(ctx sdk.Context) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Delete(types.KeySignerAddress)
}
