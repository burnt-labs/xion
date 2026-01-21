package types

import (
	"context"
	"fmt"
	"slices"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

var _ StatefulAccepter = &CodeExecutionAuthorization{}

// NewCodeExecutionAuthorization creates a new CodeExecutionAuthorization.
func NewCodeExecutionAuthorization(allowedCodeIDs []uint64) *CodeExecutionAuthorization {
	return &CodeExecutionAuthorization{
		AllowedCodeIds: allowedCodeIDs,
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a CodeExecutionAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&wasmtypes.MsgExecuteContract{})
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a CodeExecutionAuthorization) ValidateBasic() error {
	if len(a.AllowedCodeIds) == 0 {
		return fmt.Errorf("allowed_code_ids cannot be empty")
	}
	return nil
}

// Accept implements Authorization.Accept.
// This is called for non-stateful execution paths. Since we need to query
// contract info to check the code ID, we require the stateful path.
func (a CodeExecutionAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	return authz.AcceptResponse{Accept: false}, fmt.Errorf("CodeExecutionAuthorization requires stateful execution")
}

// AcceptWith implements StatefulAuthorization.AcceptWith.
// It checks if the contract being executed is instantiated from an allowed code ID.
func (a CodeExecutionAuthorization) AcceptWith(
	ctx context.Context,
	msg sdk.Msg,
	qr QueryProvider,
) (authz.AcceptResponse, error) {
	execMsg, ok := msg.(*wasmtypes.MsgExecuteContract)
	if !ok {
		return authz.AcceptResponse{Accept: false}, fmt.Errorf("expected MsgExecuteContract, got %T", msg)
	}

	// Query the contract info to get its code ID
	contractInfo, err := qr.QueryContractInfo(ctx, execMsg.Contract)
	if err != nil {
		return authz.AcceptResponse{Accept: false}, fmt.Errorf("failed to query contract info: %w", err)
	}

	// Check if the code ID is in the allowed list
	if slices.Contains(a.AllowedCodeIds, contractInfo.CodeID) {
		return authz.AcceptResponse{Accept: true}, nil
	}

	return authz.AcceptResponse{Accept: false}, fmt.Errorf(
		"contract code_id %d not in allowed list %v",
		contractInfo.CodeID,
		a.AllowedCodeIds,
	)
}
