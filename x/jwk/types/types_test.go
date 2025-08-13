package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestJWKTypes(t *testing.T) {
	// Test DefaultGenesis
	defaultGenesis := types.DefaultGenesis()
	require.NotNil(t, defaultGenesis)

	// Test Validate
	err := defaultGenesis.Validate()
	require.NoError(t, err)

	// Test AudienceKey
	key := types.AudienceKey("test-audience")
	require.NotEmpty(t, key)
	require.Contains(t, string(key), "test-audience")

	// Test AudienceClaimKey
	claimKey := types.AudienceClaimKey([]byte("test-claim"))
	require.NotEmpty(t, claimKey)
	require.Contains(t, string(claimKey), "test-claim")

	// Test KeyPrefix
	prefix := types.KeyPrefix("test")
	require.Equal(t, []byte("test"), prefix)
}

func TestJWKParams(t *testing.T) {
	// Test NewParams
	params := types.NewParams(1000, 500)
	require.NotNil(t, params)
	require.Equal(t, uint64(1000), params.DeploymentGas)
	require.Equal(t, uint64(500), params.TimeOffset)

	// Test DefaultParams
	defaultParams := types.DefaultParams()
	require.NotNil(t, defaultParams)
	require.Equal(t, uint64(500_000), defaultParams.DeploymentGas)
	require.Equal(t, uint64(1200), defaultParams.TimeOffset)

	// Test ParamSetPairs
	pairs := defaultParams.ParamSetPairs()
	require.NotNil(t, pairs)
	require.Len(t, pairs, 2)

	// Test Validate
	err := defaultParams.Validate()
	require.NoError(t, err)

	// Test with invalid params - zero deployment gas
	invalidParams := types.Params{
		DeploymentGas: 0, // invalid
		TimeOffset:    500,
	}
	err = invalidParams.Validate()
	require.Error(t, err)
}

func TestJWKCodec(t *testing.T) {
	// Test RegisterCodec
	amino := codec.NewLegacyAmino()
	require.NotPanics(t, func() {
		types.RegisterCodec(amino)
	})

	// Test RegisterInterfaces
	registry := codectypes.NewInterfaceRegistry()
	require.NotPanics(t, func() {
		types.RegisterInterfaces(registry)
	})
}

func TestJWKMessages(t *testing.T) {
	// Test NewMsgCreateAudience
	admin := "xion1test"
	aud := "test-audience"
	key := "test-key"
	msg := types.NewMsgCreateAudience(admin, aud, key)
	require.NotNil(t, msg)
	require.Equal(t, admin, msg.Admin)
	require.Equal(t, aud, msg.Aud)
	require.Equal(t, key, msg.Key)

	// Test Route
	require.Equal(t, types.RouterKey, msg.Route())

	// Test Type
	require.Equal(t, "create_audience", msg.Type())

	// Test GetSignBytes
	bytes := msg.GetSignBytes()
	require.NotNil(t, bytes)

	// Test ValidateBasic with valid message
	err := msg.ValidateBasic()
	require.NoError(t, err)

	// Test ValidateBasic with invalid message (empty admin)
	invalidMsg := types.NewMsgCreateAudience("", aud, key)
	err = invalidMsg.ValidateBasic()
	require.Error(t, err)
}
