package keeper

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"cosmossdk.io/collections"
	storetypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/orm/model/ormdb"

	apiv1 "github.com/burnt-labs/xion/api/xion/dkim/v1"
	"github.com/burnt-labs/xion/x/dkim/types"
)

type Keeper struct {
	cdc codec.BinaryCodec

	logger log.Logger

	// state management
	Schema collections.Schema
	Params collections.Item[types.Params]
	OrmDB  apiv1.StateStore

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

	db, err := ormdb.NewModuleDB(&types.ORMModuleSchema, ormdb.ModuleDBOptions{KVStoreService: storeService})
	if err != nil {
		panic(err)
	}

	store, err := apiv1.NewStateStore(db)
	if err != nil {
		panic(err)
	}

	k := Keeper{
		cdc:    cdc,
		logger: logger,

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		OrmDB:  store,

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
	if err := data.Params.Validate(); err != nil {
		return err
	}

	for _, dkimPubKey := range data.DkimPubkeys {
		if err := k.OrmDB.DkimPubKeyTable().Save(ctx, &apiv1.DkimPubKey{
			Domain:   dkimPubKey.Domain,
			PubKey:   dkimPubKey.PubKey,
			Selector: dkimPubKey.Selector,
		}); err != nil {
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

	allDkimPubKeys, err := k.OrmDB.DkimPubKeyTable().List(ctx, apiv1.DkimPubKeyDomainSelectorIndexKey{})
	if err != nil {
		panic(err)
	}
	var dkimPubKeys []types.DkimPubKey

	for allDkimPubKeys.Next() {
		dkimPubKey, err := allDkimPubKeys.Value()
		if err != nil {
			panic(err)
		}
		dkimPubKeys = append(dkimPubKeys, types.DkimPubKey{
			Domain:   dkimPubKey.Domain,
			PubKey:   dkimPubKey.PubKey,
			Selector: dkimPubKey.Selector,
			Version:  types.Version(dkimPubKey.Version),
			KeyType:  types.KeyType(dkimPubKey.KeyType),
		})
	}
	// this line is used by starport scaffolding # genesis/module/export

	return &types.GenesisState{
		Params:      params,
		DkimPubkeys: dkimPubKeys,
	}
}
