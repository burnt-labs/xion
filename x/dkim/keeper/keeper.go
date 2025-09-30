package keeper

import (
	"context"

	"github.com/vocdoni/circom2gnark/parser"

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
			types.DkimPrefix, // NOTE: add an actual prefix
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
		pk := apiv1.DkimPubKey{
			Domain:       dkimPubKey.Domain,
			PubKey:       dkimPubKey.PubKey,
			Selector:     dkimPubKey.Selector,
			PoseidonHash: dkimPubKey.PoseidonHash,
		}
		key := collections.Join(pk.Domain, pk.Selector)
		//nolint:govet // copylocks: unavoidable when storing protobuf messages in collections.Map
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

	return &types.GenesisState{
		Params:      params,
		DkimPubkeys: dkimPubKeys,
	}
}

func (k *Keeper) Verify(ctx context.Context, proof *parser.CircomProof, vkey *parser.CircomVerificationKey, inputs *[]string) (bool, error) {
	gnarkProof, err := parser.ConvertCircomToGnark(vkey, proof, *inputs)
	if err != nil {
		return false, err
	}
	return parser.VerifyProof(gnarkProof)
}
