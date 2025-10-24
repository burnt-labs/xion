package keeper

import (
	"context"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/vocdoni/circom2gnark/parser"

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
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// this line is used by starport scaffolding # genesis/module/init
	if err := data.Validate(); err != nil {
		return err
	}
	return k.Params.Set(ctx, data.Params)
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	// this line is used by starport scaffolding # genesis/module/export

	return &types.GenesisState{
		Params: params,
	}
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
func (k Keeper) AddVKey(ctx context.Context, authority string, name string, keyBytes []byte, description string) (uint64, error) {
	// Check authority
	if authority != k.authority {
		return 0, errors.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", k.authority, authority)
	}

	// Check if name already exists
	has, err := k.VKeyNameIndex.Has(ctx, name)
	if err != nil {
		return 0, err
	}
	if has {
		return 0, errors.Wrapf(types.ErrVKeyExists, "verification key with name '%s' already exists", name)
	}

	// Validate the key bytes
	if err := types.ValidateVKeyBytes(keyBytes); err != nil {
		return 0, errors.Wrap(types.ErrInvalidVKey, err.Error())
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

	k.logger.Info("added verification key", "id", id, "name", name, "authority", authority)
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
func (k Keeper) UpdateVKey(ctx context.Context, authority string, name string, keyBytes []byte, description string) error {
	// Check authority
	if authority != k.authority {
		return errors.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", k.authority, authority)
	}

	// Get existing ID
	id, err := k.VKeyNameIndex.Get(ctx, name)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return errors.Wrapf(types.ErrVKeyNotFound, "verification key '%s' not found", name)
		}
		return err
	}

	// Validate the new key bytes
	if err := types.ValidateVKeyBytes(keyBytes); err != nil {
		return errors.Wrap(types.ErrInvalidVKey, err.Error())
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

	k.logger.Info("updated verification key", "id", id, "name", name, "authority", authority)
	return nil
}

// RemoveVKey removes a verification key by name
func (k Keeper) RemoveVKey(ctx context.Context, authority string, name string) error {
	// Check authority
	if authority != k.authority {
		return errors.Wrapf(types.ErrInvalidAuthority, "expected %s, got %s", k.authority, authority)
	}

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

	k.logger.Info("removed verification key", "id", id, "name", name, "authority", authority)
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
