package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

func TestAuthzLimiterDecorator_ValidateAuthzMessages(t *testing.T) {
	// Create the decorator with restricted message types
	decorator := NewAuthzLimiterDecorator([]string{
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract",
	})

	// Test 1: Non-restricted message should pass
	allowedMsg := &authz.MsgGrant{}
	authzWithAllowed := createAuthzMsg(t, allowedMsg)

	err := decorator.ValidateAuthzMessages(authzWithAllowed)
	require.NoError(t, err)

	// Test 2: Restricted MsgExecuteContract should be blocked
	dangerousMsg := &wasmtypes.MsgExecuteContract{
		Sender:   "xion1attacker",
		Contract: "xion1malicious",
		Msg:      []byte(`{"infinite_loop": {"iterations": 999999999}}`),
		Funds:    nil,
	}

	authzWithDangerous := createAuthzMsg(t, dangerousMsg)
	err = decorator.ValidateAuthzMessages(authzWithDangerous)
	require.Error(t, err)
	require.Contains(t, err.Error(), "/cosmwasm.wasm.v1.MsgExecuteContract")
	require.Contains(t, err.Error(), "not allowed in authz execution")

	// Test 3: Restricted MsgInstantiateContract should be blocked
	dangerousMsg2 := &wasmtypes.MsgInstantiateContract{
		Sender: "xion1attacker",
		Admin:  "xion1attacker",
		CodeID: 1,
		Label:  "malicious",
		Msg:    []byte("{}"),
		Funds:  nil,
	}

	authzWithDangerous2 := createAuthzMsg(t, dangerousMsg2)
	err = decorator.ValidateAuthzMessages(authzWithDangerous2)
	require.Error(t, err)
	require.Contains(t, err.Error(), "/cosmwasm.wasm.v1.MsgInstantiateContract")
	require.Contains(t, err.Error(), "not allowed in authz execution")
}

func TestAuthzLimiterDecorator_EmptyRestrictedList(t *testing.T) {
	// Decorator with no restricted messages should allow everything
	decorator := NewAuthzLimiterDecorator([]string{})

	dangerousMsg := &wasmtypes.MsgExecuteContract{
		Sender:   "xion1sender",
		Contract: "xion1contract",
		Msg:      []byte("{}"),
		Funds:    nil,
	}

	authzMsg := createAuthzMsg(t, dangerousMsg)
	err := decorator.ValidateAuthzMessages(authzMsg)
	require.NoError(t, err)
}

func TestAuthzLimiterDecorator_NewCreation(t *testing.T) {
	restrictedTypes := []string{
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract",
	}

	decorator := NewAuthzLimiterDecorator(restrictedTypes)

	// Verify the decorator has the correct restricted messages
	require.True(t, decorator.restrictedMessages["/cosmwasm.wasm.v1.MsgExecuteContract"])
	require.True(t, decorator.restrictedMessages["/cosmwasm.wasm.v1.MsgInstantiateContract"])
	require.False(t, decorator.restrictedMessages["/cosmos.authz.v1beta1.MsgGrant"])
}

// Helper function
func createAuthzMsg(t *testing.T, nestedMsg sdk.Msg) *authz.MsgExec {
	anyMsg, err := types.NewAnyWithValue(nestedMsg)
	require.NoError(t, err)

	return &authz.MsgExec{
		Grantee: "xion1grantee",
		Msgs:    []*types.Any{anyMsg},
	}
}
