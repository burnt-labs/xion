package keeper

import (
	"context"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/errors"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/dkim/types"
	zkkeeper "github.com/burnt-labs/xion/x/zk/keeper"
)

type Keeper struct {
	cdc codec.BinaryCodec

	logger log.Logger

	// state management
	Schema      collections.Schema
	DkimPubKeys collections.Map[collections.Pair[string, string], types.DkimPubKey]
	RevokedKeys collections.Map[string, bool]
	Params      collections.Item[types.Params]
	ZkKeeper    zkkeeper.Keeper

	authority string
}

// NewKeeper creates a new Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	storeService storetypes.KVStoreService,
	logger log.Logger,
	authority string,
	zkKeeper zkkeeper.Keeper,
) Keeper {
	logger = logger.With(log.ModuleKey, "x/"+types.ModuleName)

	sb := collections.NewSchemaBuilder(storeService)

	if authority == "" {
		authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,
		DkimPubKeys: collections.NewMap(
			sb,
			types.DkimPrefix,
			"dkim_pubkeys",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[types.DkimPubKey](cdc),
		),
		RevokedKeys: collections.NewMap(
			sb,
			types.DkimRevokedPrefix,
			"dkim_revoked_pubkeys",
			collections.StringKey,
			collections.BoolValue,
		),
		Params:    collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		authority: authority,
		ZkKeeper:  zkKeeper,
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

	params := data.Params
	if params.MaxPubkeySizeBytes == 0 {
		params.MaxPubkeySizeBytes = types.DefaultMaxPubKeySizeBytes
	}

	if err := k.SetParams(ctx, params); err != nil {
		return err
	}
	for _, dkimPubKey := range data.DkimPubkeys {
		pk := types.DkimPubKey{
			Domain:       dkimPubKey.Domain,
			PubKey:       dkimPubKey.PubKey,
			Selector:     dkimPubKey.Selector,
			PoseidonHash: dkimPubKey.PoseidonHash,
			Version:      dkimPubKey.Version,
			KeyType:      dkimPubKey.KeyType,
		}
		key := collections.Join(pk.Domain, pk.Selector)
		//nolint:govet // copylocks: unavoidable when storing protobuf messages in collections.Map
		if err := k.DkimPubKeys.Set(ctx, key, pk); err != nil {
			return err
		}
	}
	return nil
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	var dkimPubKeys []types.DkimPubKey
	iter, err := k.DkimPubKeys.Iterate(ctx, nil)
	if err != nil {
		panic(err)
	}
	defer iter.Close()
	kvs, err := iter.KeyValues()
	if err != nil {
		panic(err)
	}
	//nolint:govet // copylocks: unavoidable when iterating over collections.Map with protobuf values
	for _, kv := range kvs {
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:       kv.Value.Domain,
			PubKey:       kv.Value.PubKey,
			PoseidonHash: kv.Value.PoseidonHash,
			Selector:     kv.Value.Selector,
			Version:      types.Version(kv.Value.Version),
			KeyType:      types.KeyType(kv.Value.KeyType),
		})
	}
	// this line is used by starport scaffolding # genesis/module/export

	params, err := k.GetParams(ctx)
	if err != nil {
		panic(err)
	}

	return &types.GenesisState{
		DkimPubkeys: dkimPubKeys,
		Params:      params,
	}
}

// GetParams returns the module parameters or defaults when unset.
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

// SetParams validates and stores the module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if params.MaxPubkeySizeBytes == 0 {
		params.MaxPubkeySizeBytes = types.DefaultMaxPubKeySizeBytes
	}

	if err := params.Validate(); err != nil {
		return err
	}

	return k.Params.Set(ctx, params)
}
