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

type Provider struct {
	wasmkeeper *wasmkeeper.Keeper
}

func NewProvider(wasmkeeper *wasmkeeper.Keeper) *Provider {
	if wasmkeeper == nil {
		panic("must provide wasmkeeper")
	}
	return &Provider{
		wasmkeeper: wasmkeeper,
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

	contractInfo := p.wasmkeeper.GetContractInfo(ctx, addr)
	if contractInfo == nil {
		return nil, fmt.Errorf("empty contract information")
	}
	return contractInfo, nil
}
