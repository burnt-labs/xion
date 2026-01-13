package indexer

import (
	"bytes"
	"context"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	abci "github.com/cometbft/cometbft/abci/types"

	db "github.com/cosmos/cosmos-db"

	"cosmossdk.io/collections/corecompat"
	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	"cosmossdk.io/x/feegrant"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/burnt-labs/xion/x/authz"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"

	indexerauthz "github.com/burnt-labs/xion/indexer/authz"
	indexerfeegrant "github.com/burnt-labs/xion/indexer/feegrant"
)

var (
	AuthzStorePrefix    = []byte("authz")
	FeeGrantStorePrefix = []byte("feegrant")
)

type StreamService struct {
	db              db.DB
	log             log.Logger
	authzHandler    *AuthzHandler
	feeGrantHandler *FeeGrantHandler
	authzQuerier    indexerauthz.QueryServer
	feegrantQuerier indexerfeegrant.QueryServer
	addrCodec       address.Codec
	cdc             codec.Codec
}

type kvAccessor struct {
	db db.DB
}

func (k *kvAccessor) OpenKVStore(ctx context.Context) corecompat.KVStore {
	return k.db
}

// New creates a new StreamService with a new PebbleDB instance
func New(homeDir string, cdc codec.Codec, addrCodec address.Codec, log log.Logger) *StreamService {
	dataDir := filepath.Join(homeDir, "data")
	storeDB, err := db.NewPebbleDB("xion_indexer", dataDir, nil)
	if err != nil {
		panic(err)
	}
	return NewWithDB(storeDB, cdc, addrCodec, log)
}

// NewWithDB creates a new StreamService with an existing db instance
func NewWithDB(store db.DB, cdc codec.Codec, addrCodec address.Codec, log log.Logger) *StreamService {
	authzDB := db.NewPrefixDB(store, AuthzStorePrefix)
	feegrantDb := db.NewPrefixDB(store, FeeGrantStorePrefix)

	authzHandler, err := NewAuthzHandler(&kvAccessor{db: authzDB}, cdc)
	if err != nil {
		panic(err)
	}
	feeGrantHandler, err := NewFeeGrantHandler(&kvAccessor{db: feegrantDb}, cdc)
	if err != nil {
		panic(err)
	}
	return &StreamService{
		db:              store,
		log:             log,
		authzHandler:    authzHandler,
		feeGrantHandler: feeGrantHandler,
		authzQuerier:    NewAuthzQuerier(authzHandler, cdc, addrCodec),
		feegrantQuerier: NewFeegrantQuerier(feeGrantHandler, cdc, addrCodec),
		addrCodec:       addrCodec,
		cdc:             cdc,
	}
}

func (ss *StreamService) AuthzHandler() *AuthzHandler {
	return ss.authzHandler
}

func (ss *StreamService) FeeGrantHandler() *FeeGrantHandler {
	return ss.feeGrantHandler
}

func (ss *StreamService) AuthzQuerier() indexerauthz.QueryServer {
	return ss.authzQuerier
}

func (ss *StreamService) FeeGrantQuerier() indexerfeegrant.QueryServer {
	return ss.feegrantQuerier
}

func (ss *StreamService) RegisterGRPCGatewayRoutes(clientCtx client.Context, mux *runtime.ServeMux) {
	ss.log.Info("registering authz querier grpc gateway routes")
	_ = indexerauthz.RegisterQueryHandlerClient(context.Background(), mux, indexerauthz.NewQueryClient(clientCtx))
	ss.log.Info("registering feegrant querier grpc gateway routes")
	_ = indexerfeegrant.RegisterQueryHandlerClient(context.Background(), mux, indexerfeegrant.NewQueryClient(clientCtx))
}

func (ss *StreamService) RegisterServices(cfg module.Configurator) error {
	ss.log.Info("registering authz querier services")
	indexerauthz.RegisterQueryServer(cfg.QueryServer(), ss.authzQuerier)
	ss.log.Info("registering feegrant querier services")
	indexerfeegrant.RegisterQueryServer(cfg.QueryServer(), ss.feegrantQuerier)
	return nil
}

func (ss *StreamService) Close() error {
	ss.log.Info("closing xion_indexer.db")
	return ss.db.Close()
}

// ListenFinalizeBlock will receive the request and response of a block
// currently not used because we are indexing mostly by change sets and not transaction events
func (ss *StreamService) ListenFinalizeBlock(ctx context.Context, req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	return nil
}

// ListenCommit will receive the request and response of a block
// and the change set of the block
// NOTE: in order to receive change sets, the app must be configured with StoreListeners.
func (ss *StreamService) ListenCommit(ctx context.Context, res abci.ResponseCommit, changeSet []*storetypes.StoreKVPair) error {
	for _, pair := range changeSet {
		switch pair.StoreKey {
		case authz.ModuleName:
			// if the key is a grant index it
			if bytes.HasPrefix(pair.Key, authzkeeper.GrantKey) {
				// Use safe handler with logging
				if err := SafeAuthzHandlerUpdate(ctx, ss.authzHandler, pair, ss.log); err != nil {
					// Even with safe handlers, log any unexpected errors
					ss.log.Error("Unexpected error in authz handler", "error", err)
					// Don't return error - continue processing
				}
			}
		case feegrant.ModuleName:
			if bytes.HasPrefix(pair.Key, feegrant.FeeAllowanceKeyPrefix) {
				// Use safe handler with logging
				if err := SafeFeeGrantHandlerUpdate(ctx, ss.feeGrantHandler, pair, ss.log); err != nil {
					// Even with safe handlers, log any unexpected errors
					ss.log.Error("Unexpected error in feegrant handler", "error", err)
					// Don't return error - continue processing
				}
			}
		}
	}
	return nil
}
