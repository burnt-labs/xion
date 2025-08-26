package app_test

import (
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/burnt-labs/xion/app"
)

// TestAuthzBypassVulnerabilityPrevention demonstrates that the AuthzLimiterDecorator
// successfully prevents the authz bypass attack described in the security report
func TestAuthzBypassVulnerabilityPrevention(t *testing.T) {
	// Create a malicious MsgExecuteContract that would cause network shutdown
	maliciousContract := &wasmtypes.MsgExecuteContract{
		Sender:   sdk.AccAddress([]byte("attacker")).String(),
		Contract: sdk.AccAddress([]byte("malicious_contract")).String(),
		Msg:      []byte(`{"infinite_loop": {"iterations": 999999999}}`),
		Funds:    nil,
	}

	// Wrap it in an authz MsgExec to bypass ante handlers (the vulnerability)
	anyMsg, err := types.NewAnyWithValue(maliciousContract)
	require.NoError(t, err)

	authzMsg := &authz.MsgExec{
		Grantee: "xion1attacker",
		Msgs:    []*types.Any{anyMsg},
	}

	// Create the AuthzLimiterDecorator with the same restrictions as in production
	decorator := app.NewAuthzLimiterDecorator([]string{
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract2",
		"/cosmwasm.wasm.v1.MsgMigrateContract",
		"/cosmwasm.wasm.v1.MsgUpdateAdmin",
		"/cosmwasm.wasm.v1.MsgClearAdmin",
	})

	// Test that the attack is blocked
	err = decorator.ValidateAuthzMessages(authzMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "/cosmwasm.wasm.v1.MsgExecuteContract")
	require.Contains(t, err.Error(), "not allowed in authz execution")

	// Verify that the error message indicates security restrictions
	require.Contains(t, err.Error(), "security restrictions")
}

// TestAuthzLimiterAllowsLegitimateMessages ensures we don't break normal authz functionality
func TestAuthzLimiterAllowsLegitimateMessages(t *testing.T) {
	// Create a legitimate authz message (MsgGrant is safe)
	legitimateMsg := &authz.MsgGrant{
		Granter: "xion1granter",
		Grantee: "xion1grantee",
		Grant: authz.Grant{
			Authorization: &authz.GenericAuthorization{
				Msg: "/cosmos.bank.v1beta1.MsgSend",
			},
			Expiration: time.Now().Add(24 * time.Hour),
		},
	}

	// Wrap it in authz MsgExec
	anyMsg, err := types.NewAnyWithValue(legitimateMsg)
	require.NoError(t, err)

	authzMsg := &authz.MsgExec{
		Grantee: "xion1grantee",
		Msgs:    []*types.Any{anyMsg},
	}

	// Create the decorator
	decorator := app.NewAuthzLimiterDecorator([]string{
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract",
	})

	// Test that legitimate messages are allowed
	err = decorator.ValidateAuthzMessages(authzMsg)
	require.NoError(t, err)
}

// TestMultipleRestrictedMessages ensures all dangerous message types are blocked
func TestMultipleRestrictedMessages(t *testing.T) {
	restrictedMsgTypes := []sdk.Msg{
		&wasmtypes.MsgExecuteContract{
			Sender:   "xion1attacker",
			Contract: "xion1contract",
			Msg:      []byte("{}"),
		},
		&wasmtypes.MsgInstantiateContract{
			Sender: "xion1attacker",
			Admin:  "xion1attacker",
			CodeID: 1,
			Label:  "malicious",
			Msg:    []byte("{}"),
		},
		&wasmtypes.MsgInstantiateContract2{
			Sender: "xion1attacker",
			Admin:  "xion1attacker",
			CodeID: 1,
			Label:  "malicious",
			Salt:   []byte("salt"),
			Msg:    []byte("{}"),
		},
		&wasmtypes.MsgMigrateContract{
			Sender:   "xion1attacker",
			Contract: "xion1contract",
			CodeID:   2,
			Msg:      []byte("{}"),
		},
		&wasmtypes.MsgUpdateAdmin{
			Sender:   "xion1attacker",
			NewAdmin: "xion1newadmin",
			Contract: "xion1contract",
		},
		&wasmtypes.MsgClearAdmin{
			Sender:   "xion1attacker",
			Contract: "xion1contract",
		},
	}

	decorator := app.NewAuthzLimiterDecorator([]string{
		"/cosmwasm.wasm.v1.MsgExecuteContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract",
		"/cosmwasm.wasm.v1.MsgInstantiateContract2",
		"/cosmwasm.wasm.v1.MsgMigrateContract",
		"/cosmwasm.wasm.v1.MsgUpdateAdmin",
		"/cosmwasm.wasm.v1.MsgClearAdmin",
	})

	for i, msg := range restrictedMsgTypes {
		t.Run(sdk.MsgTypeURL(msg), func(t *testing.T) {
			// Wrap in authz
			anyMsg, err := types.NewAnyWithValue(msg)
			require.NoError(t, err)

			authzMsg := &authz.MsgExec{
				Grantee: "xion1attacker",
				Msgs:    []*types.Any{anyMsg},
			}

			// Should be blocked
			err = decorator.ValidateAuthzMessages(authzMsg)
			require.Error(t, err, "Message type %d should be blocked", i)
			require.Contains(t, err.Error(), "not allowed in authz execution")
		})
	}
}
