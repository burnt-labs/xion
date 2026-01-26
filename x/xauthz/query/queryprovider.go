package query

import (
	"context"
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// GasCostPerQuery is the base gas cost for each query operation.
	// This helps prevent DoS attacks through excessive queries.
	GasCostPerQuery = uint64(1000)
)

// WasmQuerier is the interface for querying wasm contract info.
type WasmQuerier interface {
	GetContractInfo(ctx context.Context, contractAddress sdk.AccAddress) *wasmtypes.ContractInfo
}

type Provider struct {
	wasmQuerier WasmQuerier
}

// NewProvider creates a new Provider with the given wasm keeper.
func NewProvider(wasmkeeper *wasmkeeper.Keeper) *Provider {
	if wasmkeeper == nil {
		panic("must provide wasmkeeper")
	}
	return &Provider{
		wasmQuerier: wasmkeeper,
	}
}

// NewProviderWithWasmQuerier creates a new Provider with a custom wasm querier.
// This is useful for testing with mock implementations.
func NewProviderWithWasmQuerier(wasmQuerier WasmQuerier) *Provider {
	if wasmQuerier == nil {
		panic("must provide wasmQuerier")
	}
	return &Provider{
		wasmQuerier: wasmQuerier,
	}
}

// QueryContractInfo provides access to an instance's contract information.
func (p *Provider) QueryContractInfo(ctx context.Context, contract string) (*wasmtypes.ContractInfo, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.GasMeter().ConsumeGas(GasCostPerQuery, "stateful authz query")
	addr, err := sdk.AccAddressFromBech32(contract)
	if err != nil {
		return nil, err
	}

	contractInfo := p.wasmQuerier.GetContractInfo(ctx, addr)
	if contractInfo == nil {
		return nil, fmt.Errorf("empty contract information")
	}
	return contractInfo, nil
}
