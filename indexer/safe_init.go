package indexer

import (
	"context"
	"path/filepath"

	abci "github.com/cometbft/cometbft/abci/types"

	db "github.com/cosmos/cosmos-db"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
)

// IndexerService defines the interface for indexer services
type IndexerService interface {
	ListenFinalizeBlock(context.Context, abci.RequestFinalizeBlock, abci.ResponseFinalizeBlock) error
	ListenCommit(context.Context, abci.ResponseCommit, []*storetypes.StoreKVPair) error
	RegisterServices(module.Configurator) error
	Close() error
}

// NewSafeIndexer creates an indexer service with graceful error handling
// Returns a no-op service if initialization fails instead of panicking
func NewSafeIndexer(homeDir string, cdc codec.Codec, addrCodec address.Codec, logger log.Logger) IndexerService {
	logger = logger.With("module", "indexer")

	// Validate required parameters
	if cdc == nil {
		logger.Error("Failed to initialize indexer: codec is nil, running in degraded mode")
		return NewNoOpStreamService(logger)
	}

	// Try to create the database
	dataDir := filepath.Join(homeDir, "data")
	storeDB, err := db.NewPebbleDB("xion_indexer", dataDir, nil)
	if err != nil {
		logger.Error("Failed to initialize indexer database, running in degraded mode",
			"error", err,
			"path", filepath.Join(dataDir, "xion_indexer.db"))
		return NewNoOpStreamService(logger)
	}

	// Try to create handlers
	authzDB := db.NewPrefixDB(storeDB, AuthzStorePrefix)
	feegrantDb := db.NewPrefixDB(storeDB, FeeGrantStorePrefix)

	authzHandler, err := NewAuthzHandler(&kvAccessor{db: authzDB}, cdc)
	if err != nil {
		logger.Error("Failed to initialize authz handler, running in degraded mode", "error", err)
		storeDB.Close() // Clean up
		return NewNoOpStreamService(logger)
	}

	feeGrantHandler, err := NewFeeGrantHandler(&kvAccessor{db: feegrantDb}, cdc)
	if err != nil {
		logger.Error("Failed to initialize feegrant handler, running in degraded mode", "error", err)
		storeDB.Close() // Clean up
		return NewNoOpStreamService(logger)
	}

	logger.Info("Indexer initialized successfully")

	return &StreamService{
		db:              storeDB,
		log:             logger,
		authzHandler:    authzHandler,
		feeGrantHandler: feeGrantHandler,
		authzQuerier:    NewAuthzQuerier(authzHandler, cdc, addrCodec),
		feegrantQuerier: NewFeegrantQuerier(feeGrantHandler, cdc, addrCodec),
		addrCodec:       addrCodec,
		cdc:             cdc,
	}
}
