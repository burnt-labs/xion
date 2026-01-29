package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/x/authz"

	"github.com/burnt-labs/xion/x/xauthz/types"
)

func TestRegisterLegacyAminoCodec(t *testing.T) {
	cdc := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		types.RegisterLegacyAminoCodec(cdc)
	})
}

func TestRegisterInterfaces(t *testing.T) {
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		types.RegisterInterfaces(registry)
	})

	// Verify CodeExecutionAuthorization is registered as an authz.Authorization
	auth := &types.CodeExecutionAuthorization{AllowedCodeIds: []uint64{1}}
	any, err := codectypes.NewAnyWithValue(auth)
	require.NoError(t, err)

	var resolved authz.Authorization
	err = registry.UnpackAny(any, &resolved)
	require.NoError(t, err)
	require.Equal(t, auth.MsgTypeURL(), resolved.MsgTypeURL())
}
