package keeper

import (
	"context"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	apiv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	"github.com/burnt-labs/xion/x/dkim/types"
)

type Keeper struct {
	cdc codec.BinaryCodec

	logger log.Logger

	// state management
	Schema      collections.Schema
	Params      collections.Item[types.Params]
	DkimPubKeys collections.Map[collections.Pair[string, string], apiv1.DkimPubKey]

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
		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		DkimPubKeys: collections.NewMap(
			sb,
			collections.NewPrefix(uint8(1)), // NOTE: add an actual prefix
			"dkim_pubkeys",
			collections.PairKeyCodec(collections.StringKey, collections.StringKey),
			codec.CollValue[apiv1.DkimPubKey](cdc),
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
func (k *Keeper) InitGenesis(ctx context.Context, data *types.GenesisState) error {
	// this line is used by starport scaffolding # genesis/module/init
	if err := data.Validate(); err != nil {
		return err
	}
	for _, dkimPubKey := range data.DkimPubkeys {
		pk := &apiv1.DkimPubKey{
			Domain:       dkimPubKey.Domain,
			PubKey:       dkimPubKey.PubKey,
			Selector:     dkimPubKey.Selector,
			PoseidonHash: dkimPubKey.PoseidonHash,
		}
		key := collections.Join(pk.Domain, pk.Selector)
		if err := k.DkimPubKeys.Set(ctx, key, pk); err != nil {
			return err
		}
	}
	return k.Params.Set(ctx, data.Params)
}

// ExportGenesis exports the module's state to a genesis state.
func (k *Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	params, err := k.Params.Get(ctx)
	if err != nil {
		panic(err)
	}

	var dkimPubKeys []types.DkimPubKey
	iter, err := k.DkimPubKeys.Iterate(ctx, collections.RangeFull())
	if err != nil {
		panic(err)
	}
	defer iter.Close()
	kvs, err := iter.KeyValues()
	if err != nil {
		panic(err)
	}
	for _, kv := range kvs {
		dkimPubKey := kv.Value
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:       dkimPubKey.Domain,
			PubKey:       dkimPubKey.PubKey,
			PoseidonHash: dkimPubKey.PoseidonHash,
			Selector:     dkimPubKey.Selector,
			Version:      types.Version(dkimPubKey.Version),
			KeyType:      types.KeyType(dkimPubKey.KeyType),
		})
	}
	// this line is used by starport scaffolding # genesis/module/export

	return &types.GenesisState{
		Params:      params,
		DkimPubkeys: dkimPubKeys,
	}
}
