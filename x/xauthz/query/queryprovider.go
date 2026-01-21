package query

import (
	"context"
	"fmt"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
