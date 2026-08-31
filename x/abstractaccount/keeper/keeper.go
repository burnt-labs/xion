package keeper

import (
	"bytes"
	"encoding/binary"
	"errors"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	log "cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/burnt-labs/xion/x/abstractaccount/types"
)

type Keeper struct {
	cdc               codec.BinaryCodec
	storeKey          storetypes.StoreKey
	transientStoreKey storetypes.StoreKey
	ak                authkeeper.AccountKeeperI
	ck                wasmtypes.ContractOpsKeeperWithAddressHash
	vk                wasmtypes.ViewKeeper
	authority         string
}

func NewKeeper(
	cdc codec.BinaryCodec, storeKey storetypes.StoreKey, transientStoreKey storetypes.StoreKey,
	ak authkeeper.AccountKeeperI, ck wasmtypes.ContractOpsKeeperWithAddressHash, vk wasmtypes.ViewKeeper,
	authority string,
) Keeper {
	if ak == nil {
		panic("AccountKeeperI cannot be nil")
	}

	if ck == nil {
		panic("ContractOpsKeeper cannot be nil")
	}

	if vk == nil {
		panic("ViewKeeper cannot be nil")
	}

	return Keeper{
		cdc:               cdc,
		storeKey:          storeKey,
		transientStoreKey: transientStoreKey,
		ak:                ak,
		ck:                ck,
		vk:                vk,
		authority:         authority,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) ContractKeeper() wasmtypes.ContractOpsKeeperWithAddressHash {
	return k.ck
}

// ---------------------------------- Params -----------------------------------

func (k Keeper) GetParams(ctx sdk.Context) (*types.Params, error) {
	store := ctx.KVStore(k.storeKey)

	bz := store.Get(types.KeyParams)
	if bz == nil {
		return nil, sdkerrors.ErrNotFound.Wrap("x/abstractaccount module params")
	}

	var params types.Params
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return nil, types.ErrParsingParams.Wrap(err.Error())
	}

	return &params, nil
}

// SetParams validates and stores intrinsically valid parameters. It does not
// enforce the current-state invariant that the address derivation hash is
// immutable after configuration. Genesis and chain upgrade handlers are
// trusted initialization paths. Runtime governance updates must go through
// MsgUpdateParams, which enforces immutability.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) error {
	store := ctx.KVStore(k.storeKey)

	if err := params.Validate(); err != nil {
		return err
	}

	bz, err := k.cdc.Marshal(params)
	if err != nil {
		return types.ErrParsingParams.Wrap(err.Error())
	}

	store.Set(types.KeyParams, bz)

	return nil
}

// ----------------------------- Account addresses ----------------------------

func accountAddressKey(sender sdk.AccAddress, salt []byte) []byte {
	key := make([]byte, 1+8+len(sender)+len(salt))
	key[0] = types.KeyAccountAddressPrefix[0]
	binary.BigEndian.PutUint64(key[1:9], uint64(len(sender)))
	copy(key[9:], sender)
	copy(key[9+len(sender):], salt)
	return key
}

func parseAccountAddressKey(key []byte) (sdk.AccAddress, []byte, error) {
	if len(key) < 9 || key[0] != types.KeyAccountAddressPrefix[0] {
		return nil, nil, errors.New("invalid account address registry key")
	}

	senderLen := binary.BigEndian.Uint64(key[1:9])
	if senderLen == 0 || senderLen > uint64(len(key)-9) {
		return nil, nil, errors.New("invalid account address registry sender length")
	}

	senderEnd := 9 + int(senderLen)
	return sdk.AccAddress(bytes.Clone(key[9:senderEnd])), bytes.Clone(key[senderEnd:]), nil
}

func (k Keeper) GetAccountAddress(ctx sdk.Context, sender sdk.AccAddress, salt []byte) (sdk.AccAddress, bool) {
	bz := ctx.KVStore(k.storeKey).Get(accountAddressKey(sender, salt))
	if bz == nil {
		return nil, false
	}
	return sdk.AccAddress(bz), true
}

func (k Keeper) SetAccountAddress(ctx sdk.Context, sender sdk.AccAddress, salt []byte, address sdk.AccAddress) {
	ctx.KVStore(k.storeKey).Set(accountAddressKey(sender, salt), address)
}

func (k Keeper) IterateAccountAddresses(
	ctx sdk.Context,
	cb func(sender sdk.AccAddress, salt []byte, address sdk.AccAddress) bool,
) error {
	store := ctx.KVStore(k.storeKey)
	iterator := store.Iterator(types.KeyAccountAddressPrefix, storetypes.PrefixEndBytes(types.KeyAccountAddressPrefix))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		sender, salt, err := parseAccountAddressKey(iterator.Key())
		if err != nil {
			return err
		}
		if cb(sender, salt, sdk.AccAddress(bytes.Clone(iterator.Value()))) {
			return nil
		}
	}

	return nil
}

func (k Keeper) PredictAccountAddress(ctx sdk.Context, sender sdk.AccAddress, salt []byte) (sdk.AccAddress, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if !params.RegistrationConfigured() {
		return nil, types.ErrRegistrationNotConfigured
	}
	if err := wasmtypes.ValidateSalt(salt); err != nil {
		return nil, err
	}

	return wasmkeeper.BuildContractAddressPredictable(params.AddressDerivationHash, sender, salt, nil), nil
}

func (k Keeper) IsAbstractAccount(ctx sdk.Context, address sdk.AccAddress) bool {
	_, ok := k.ak.GetAccount(ctx, address).(*types.AbstractAccount)
	return ok
}

// ------------------------------- NextAccountId -------------------------------

func (k Keeper) GetAndIncrementNextAccountID(ctx sdk.Context) uint64 {
	id := k.GetNextAccountID(ctx)

	k.SetNextAccountID(ctx, id+1)

	return id
}

func (k Keeper) GetNextAccountID(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)

	return sdk.BigEndianToUint64(store.Get(types.KeyNextAccountID))
}

func (k Keeper) SetNextAccountID(ctx sdk.Context, id uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.KeyNextAccountID, sdk.Uint64ToBigEndian(id))
}

// ------------------------------- SignerAddress -------------------------------

// signerAddressKey scopes the signer marker to the transaction currently being
// executed.
//
// The transient store is block-scoped: it is only cleared by Commit, not between
// transactions. BaseApp flushes the AnteHandler's writes as soon as the ante
// chain succeeds, independently of whether message execution later fails, while
// the PostHandler runs against the runMsgs branch that is discarded when it does.
// Under a single fixed key a transaction that passed ante and then failed would
// leave its signer behind, and the next transaction in the same block would read
// it and invoke that account's after_tx hook. Keying by tx bytes keeps the marker
// private to the transaction that wrote it.
func signerAddressKey(ctx sdk.Context) []byte {
	txBytes := ctx.TxBytes()

	key := make([]byte, 0, len(types.KeySignerAddress)+len(txBytes))
	key = append(key, types.KeySignerAddress...)

	return append(key, txBytes...)
}

func (k Keeper) GetSignerAddress(ctx sdk.Context) sdk.AccAddress {
	store := ctx.TransientStore(k.transientStoreKey)

	return sdk.AccAddress(store.Get(signerAddressKey(ctx)))
}

// SetSignerAddress stores the AA signer for the current transaction, so that the
// PostHandler knows whether to call after_tx.
func (k Keeper) SetSignerAddress(ctx sdk.Context, signerAddr sdk.AccAddress) {
	store := ctx.TransientStore(k.transientStoreKey)
	store.Set(signerAddressKey(ctx), signerAddr)
}

func (k Keeper) DeleteSignerAddress(ctx sdk.Context) {
	store := ctx.TransientStore(k.transientStoreKey)
	store.Delete(signerAddressKey(ctx))
}

// ------------------------------- Migration -------------------------------
func (k Keeper) Migrator() Migrator {
	return NewMigrator(k.storeKey, k.cdc)
}
