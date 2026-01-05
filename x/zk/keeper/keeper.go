package keeper

import (
	"context"

	"github.com/vocdoni/circom2gnark/parser"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/zk/types"
)

type Keeper struct {
	cdc codec.BinaryCodec

	logger log.Logger

	// state management
	Schema        collections.Schema
	VKeys         collections.Map[uint64, types.VKey]
	NextVKeyID    collections.Sequence
	VKeyNameIndex collections.Map[string, uint64]
	Params        collections.Item[types.Params]

	authority string
}

// NewKeeper creates a new Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	logger log.Logger,
	authority string,
) Keeper {
	logger = logger.With(log.ModuleKey, "x/"+types.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	if authority == "" {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,
		VKeys: collections.NewMap(
			sb,
			types.VKeyPrefix,
			"vkeys",
			collections.Uint64Key,
			codec.CollValue[types.VKey](cdc)),

		NextVKeyID: collections.NewSequence(
			sb,
			types.VkeySeqPrefix,
			"vkey_sequence"),
		VKeyNameIndex: collections.NewMap(
			sb,
			types.VkeyNameIndexPrefix, "vkey_name_index",
			collections.StringKey,
			collections.Uint64Value),
		Params: collections.NewItem(
			sb,
			types.ParamsKey,
			"params",
			codec.CollValue[types.Params](cdc),
		),
		authority: authority,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	k.Schema = schema

	return k
}

func (k Keeper) Logger() log.Logger {
	return k.logger
}

// InitGenesis initializes the module's state from a genesis state.
func (k Keeper) InitGenesis(ctx sdk.Context, gs *types.GenesisState) {
	params := gs.Params
	if params == (types.Params{}) {
		params = types.DefaultParams()
	}
	if err := params.Validate(); err != nil {
		panic(err)
	}

	if err := k.SetParams(ctx, params); err != nil {
		panic(err)
	}

	// Import all vkeys
	for _, vkeyWithID := range gs.Vkeys {
		if err := types.ValidateVKeyBytes(vkeyWithID.Vkey.KeyBytes, params.MaxVkeySizeBytes); err != nil {
			panic(err)
		}

		// Set the vkey
		if err := k.VKeys.Set(ctx, vkeyWithID.Id, vkeyWithID.Vkey); err != nil {
			panic(err)
		}

		// Set the name index
		if err := k.VKeyNameIndex.Set(ctx, vkeyWithID.Vkey.Name, vkeyWithID.Id); err != nil {
			panic(err)
		}

		// Update the sequence to be after the highest ID
		currentSeq, err := k.NextVKeyID.Peek(ctx)
		if err != nil {
			panic(err)
		}
		if vkeyWithID.Id >= currentSeq {
			// Set the sequence to be one more than the highest ID
			if err := k.NextVKeyID.Set(ctx, vkeyWithID.Id+1); err != nil {
				panic(err)
			}
		}
	}
}

// ExportGenesis returns the module's exported genesis state.
func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	vkeys, err := k.ListVKeys(ctx)
	if err != nil {
		panic(err)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	// Convert to VKeyWithID format
	vkeysWithID := make([]types.VKeyWithID, 0, len(vkeys))
	for _, vkey := range vkeys {
		// Get the ID from the name index
		id, err := k.VKeyNameIndex.Get(ctx, vkey.Name)
		if err != nil {
			panic(err)
		}

		vkeysWithID = append(vkeysWithID, types.VKeyWithID{
			Id:   id,
			Vkey: vkey,
		})
	}

	return &types.GenesisState{
		Vkeys:  vkeysWithID,
		Params: params,
	}
}

// GetParams returns the current zk module parameters, defaulting when unset.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.DefaultParams(), nil
		}
		return types.Params{}, err
	}

	return params, nil
}

// SetParams validates and persists zk module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	return k.Params.Set(ctx, params)
}

func (k Keeper) ensureVKeySize(params types.Params, size int) (uint64, error) {
	gasCost, err := params.GasCostForSize(uint64(size))
	return gasCost, err
}

func (k *Keeper) Verify(ctx context.Context, proof *parser.CircomProof, vkey *parser.CircomVerificationKey, inputs *[]string) (bool, error) {
	gnarkProof, err := parser.ConvertCircomToGnark(vkey, proof, *inputs)
	if err != nil {
		return false, err
	}
	return parser.VerifyProof(gnarkProof)
}

// AddVKey adds a new verification key to the store
// keyBytes should be the raw JSON from SnarkJS
func (k Keeper) AddVKey(ctx sdk.Context, authority string, name string, keyBytes []byte, description string) (uint64, error) {
	// Check if name already exists
	has, err := k.VKeyNameIndex.Has(ctx, name)
	if err != nil {
		return 0, err
	}
	if has {
		return 0, errors.Wrapf(types.ErrVKeyExists, "verification key with name '%s' already exists", name)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return 0, err
	}

	gasCost, err := k.ensureVKeySize(params, len(keyBytes))
	if err != nil {
		return 0, err
	}
	// charge gas for vkey size
	ctx.GasMeter().ConsumeGas(gasCost, "zk/AddVKey: vkey size cost")

	if err := types.ValidateVKeyBytes(keyBytes, params.MaxVkeySizeBytes); err != nil {
		return 0, err
	}

	// Generate new ID
	id, err := k.NextVKeyID.Next(ctx)
	if err != nil {
		return 0, errors.Wrap(types.ErrIncreaseSequenceID, err.Error())
	}

	// Create VKey
	vkey := &types.VKey{
		KeyBytes:    keyBytes,
		Name:        name,
		Description: description,
	}

	// Store vkey
	if err := k.VKeys.Set(ctx, id, *vkey); err != nil {
		return 0, err
	}

	// Update name index
	if err := k.VKeyNameIndex.Set(ctx, name, id); err != nil {
		return 0, err
	}

	return id, nil
}

// GetVKeyByID retrieves a verification key by its numeric ID
func (k Keeper) GetVKeyByID(ctx context.Context, id uint64) (types.VKey, error) {
	vkey, err := k.VKeys.Get(ctx, id)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.VKey{}, errors.Wrapf(types.ErrVKeyNotFound, "verification key with ID %d not found", id)
		}
		return types.VKey{}, err
	}
	return vkey, nil
}

// GetVKeyByName retrieves a verification key by its name
func (k Keeper) GetVKeyByName(ctx context.Context, name string) (types.VKey, error) {
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return types.VKey{}, errors.Wrapf(types.ErrVKeyNotFound, "verification key with name '%s' not found", name)
		}
		return types.VKey{}, err
	}

	return k.GetVKeyByID(ctx, id)
}

// GetCircomVKeyByName retrieves and unmarshals a verification key for use in verification
// This is the method you'll use in ProofVerify
func (k Keeper) GetCircomVKeyByName(ctx context.Context, name string) (*parser.CircomVerificationKey, error) {
	vkey, err := k.GetVKeyByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return types.UnmarshalVKey(&vkey)
}

// GetCircomVKeyByID retrieves and unmarshals a verification key for use in verification
func (k Keeper) GetCircomVKeyByID(ctx context.Context, id uint64) (*parser.CircomVerificationKey, error) {
	vkey, err := k.GetVKeyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return types.UnmarshalVKey(&vkey)
}

// UpdateVKey updates an existing verification key
func (k Keeper) UpdateVKey(ctx sdk.Context, authority string, name string, keyBytes []byte, description string) error {
	// Get existing ID
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	gasCost, err := k.ensureVKeySize(params, len(keyBytes))
	if err != nil {
		return err
	}
	// charge gas for vkey size
	ctx.GasMeter().ConsumeGas(gasCost, "zk/UpdateVKey: vkey size cost")

	if err := types.ValidateVKeyBytes(keyBytes, params.MaxVkeySizeBytes); err != nil {
		return err
	}

	// Update vkey
	updatedVKey := types.VKey{
		KeyBytes:    keyBytes,
		Name:        name,
		Description: description,
	}

	if err := k.VKeys.Set(ctx, id, updatedVKey); err != nil {
		return err
	}

	return nil
}

// RemoveVKey removes a verification key by name
func (k Keeper) RemoveVKey(ctx context.Context, authority string, name string) error {
	// Get the ID from the name index
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	// Remove from primary storage
	if err := k.VKeys.Remove(ctx, id); err != nil {
		return err
	}

	// Remove from name index
	if err := k.VKeyNameIndex.Remove(ctx, name); err != nil {
		return err
	}

	return nil
}

// ListVKeys returns all verification keys
func (k Keeper) ListVKeys(ctx context.Context) ([]types.VKey, error) {
	var vkeys []types.VKey

	err := k.VKeys.Walk(ctx, nil, func(id uint64, vkey types.VKey) (bool, error) {
		vkeys = append(vkeys, vkey)
		return false, nil
	})

	return vkeys, err
}

// HasVKey checks if a verification key exists by name
// This is a read operation - no authority check needed
func (k Keeper) HasVKey(ctx context.Context, name string) (bool, error) {
	return k.VKeyNameIndex.Has(ctx, name)
}

// IterateVKeys iterates over all verification keys with a callback
// This is a read operation - no authority check needed
func (k Keeper) IterateVKeys(ctx context.Context, cb func(id uint64, vkey types.VKey) (stop bool, err error)) error {
	return k.VKeys.Walk(ctx, nil, cb)
}

// GetAuthority returns the module's authority address
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Add these getter methods to your keeper if they don't exist
func (k Keeper) GetCodec() codec.BinaryCodec {
	return k.cdc
}
