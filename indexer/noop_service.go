package indexer

import (
	"context"

	abci "github.com/cometbft/cometbft/abci/types"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/types/module"
)

// NoOpStreamService is a no-operation implementation of the StreamService
// Used when the indexer cannot be initialized but we want the node to continue
type NoOpStreamService struct {
	log log.Logger
}

// NewNoOpStreamService creates a new no-op stream service
func NewNoOpStreamService(logger log.Logger) *NoOpStreamService {
	return &NoOpStreamService{
		log: logger.With("module", "indexer", "mode", "noop"),
	}
}

// ListenFinalizeBlock implements ABCIListener - no-op
func (n *NoOpStreamService) ListenFinalizeBlock(ctx context.Context, req abci.RequestFinalizeBlock, res abci.ResponseFinalizeBlock) error {
	// No-op: just return nil to allow consensus to continue
	return nil
}

// ListenCommit implements ABCIListener - no-op
func (n *NoOpStreamService) ListenCommit(ctx context.Context, res abci.ResponseCommit, changeSet []*storetypes.StoreKVPair) error {
	// No-op: just return nil to allow consensus to continue
	return nil
}

// Close implements io.Closer - no-op
func (n *NoOpStreamService) Close() error {
	n.log.Info("Closing no-op indexer service")
	return nil
}

// RegisterServices implements the required interface - no-op
func (n *NoOpStreamService) RegisterServices(configurator module.Configurator) error {
	n.log.Info("No-op indexer: skipping service registration")
	return nil
}
