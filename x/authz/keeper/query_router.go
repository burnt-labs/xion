package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/burnt-labs/xion/x/authz"
)

const (
	// GasCostPerQuery is the base gas cost for each query operation.
	// This helps prevent DoS attacks through excessive queries.
	GasCostPerQuery = uint64(1000)
)

// queryRouter provides controlled, read-only access to blockchain state.
// It implements the authz.QueryRouter interface.
type queryRouter struct {
	ctx        sdk.Context
	wasmKeeper authz.WasmKeeper
	maxGas     uint64
	gasUsed    uint64
}

// NewQueryRouter creates a new QueryRouter with the provided keepers.
// The maxGas parameter limits the total gas that can be consumed by queries.
func NewQueryRouter(
	ctx sdk.Context,
	wasmKeeper authz.WasmKeeper,
	maxGas uint64,
) authz.QueryRouter {
	return &queryRouter{
		ctx:        ctx,
		wasmKeeper: wasmKeeper,
		maxGas:     maxGas,
		gasUsed:    0,
	}
}

// consumeGas tracks gas usage and consumes from the context's gas meter.
func (qr *queryRouter) consumeGas(amount uint64, descriptor string) error {
	qr.gasUsed += amount
	if qr.gasUsed > qr.maxGas {
		return fmt.Errorf("query router out of gas: used %d, max %d", qr.gasUsed, qr.maxGas)
	}
	qr.ctx.GasMeter().ConsumeGas(amount, descriptor)
	return nil
}

// RemainingGas returns the remaining gas available for queries.
func (qr *queryRouter) RemainingGas() uint64 {
	if qr.gasUsed >= qr.maxGas {
		return 0
	}
	return qr.maxGas - qr.gasUsed
}

// Wasm queries

// QueryContractInfo returns contract metadata.
// Returns nil if contract doesn't exist.
func (qr *queryRouter) QueryContractInfo(ctx context.Context, contractAddr string) (*wasmtypes.ContractInfo, error) {
	if qr.wasmKeeper == nil {
		return nil, fmt.Errorf("wasm keeper not available")
	}

	if err := qr.consumeGas(GasCostPerQuery, "authz query contract info"); err != nil {
		return nil, err
	}

	accAddr, err := sdk.AccAddressFromBech32(contractAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	return qr.wasmKeeper.GetContractInfo(qr.ctx, accAddr), nil
}

// QueryContractState executes a smart query against a contract.
// The queryData should be JSON-encoded query message.
func (qr *queryRouter) QueryContractState(ctx context.Context, contractAddr string, queryData []byte) ([]byte, error) {
	if qr.wasmKeeper == nil {
		return nil, fmt.Errorf("wasm keeper not available")
	}

	// Smart queries can be expensive, charge extra gas
	if err := qr.consumeGas(GasCostPerQuery*5, "authz query contract state"); err != nil {
		return nil, err
	}

	accAddr, err := sdk.AccAddressFromBech32(contractAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	return qr.wasmKeeper.QuerySmart(qr.ctx, accAddr, queryData)
}
