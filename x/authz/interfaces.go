package authz

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

// StatefulAuthorization extends Authorization with state query capability.
// Authorizations implementing this interface can query blockchain state
// during the Accept decision.
//
// When the keeper encounters a StatefulAuthorization, it calls AcceptWithQuery
// instead of Accept, providing a QueryRouter for controlled state access.
type StatefulAuthorization interface {
	Authorization // must implement base interface

	// AcceptWithQuery is called instead of Accept when the authorization
	// implements this interface. It provides access to a QueryRouter for
	// querying the chain using provided queries only.
	AcceptWithQuery(
		ctx context.Context,
		msg sdk.Msg,
		queryRouter QueryRouter,
	) (AcceptResponse, error)
}

// QueryRouter provides controlled, read-only access to blockchain state.
// All methods are deterministic and gas-metered to prevent DoS attacks.
//
// Implementations should:
// - Consume gas for each query operation
// - Only allow deterministic state reads
// - Return errors rather than panic on invalid inputs
type QueryRouter interface {
	// Wasm queries

	// QueryContractInfo returns contract metadata.
	// Returns nil if contract doesn't exist.
	QueryContractInfo(ctx context.Context, contractAddr string) (*wasmtypes.ContractInfo, error)

	// QueryContractState executes a smart query against a contract.
	// The queryData should be JSON-encoded query message.
	QueryContractState(ctx context.Context, contractAddr string, queryData []byte) ([]byte, error)

	// Gas management

	// RemainingGas returns the remaining gas available for queries.
	RemainingGas() uint64
}
