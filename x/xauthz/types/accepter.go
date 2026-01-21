package types

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

var (
	_ authz.Accepter = &AuthzAccepter{}
)

type AuthzAccepter struct {
	qp QueryProvider
}

func NewAccepter(qp QueryProvider) authz.Accepter {
	return &AuthzAccepter{
		qp,
	}
}

func (a *AuthzAccepter) Accept(ctx context.Context, msg sdk.Msg, auth authz.Authorization) (authz.AcceptResponse, error) {
	if stateful, ok := auth.(StatefulAccepter); ok {
		return stateful.AcceptWith(ctx, msg, a.qp)
	}
	// fallback action to run the original authz logic
	return auth.Accept(ctx, msg)
}

type QueryProvider interface {
	QueryContractInfo(ctx context.Context, contract string) (*wasmtypes.ContractInfo, error)
}

type StatefulAccepter interface {
	AcceptWith(ctx context.Context, msg sdk.Msg, qp QueryProvider) (authz.AcceptResponse, error)
}
