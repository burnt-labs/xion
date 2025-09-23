package wasmbinding_test

import (
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/gogoproto/proto"

	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
)

// Test to try to hit the type assertion failure in GetWhitelistedQuery
// Since we can't directly modify the sync.Map, we'll create a comprehensive test
// that ensures all current paths work correctly
func TestGetWhitelistedQuery_ComprehensiveTypeCheck(t *testing.T) {
	// Test with a known non-whitelisted path to ensure it fails as expected
	result, err := wasmbinding.GetWhitelistedQuery("/definitely.not.whitelisted/Query")
	require.Error(t, err)
	require.Nil(t, result)

	// Verify it's the right type of error
	unsupportedErr, ok := err.(wasmvmtypes.UnsupportedRequest)
	require.True(t, ok, "Should be UnsupportedRequest error")
	require.Contains(t, unsupportedErr.Error(), "path is not allowed from the contract")

	// Test multiple valid paths to ensure they all return proper proto.Message types
	validPaths := []string{
		"/cosmos.bank.v1beta1.Query/Balance",
		"/cosmos.auth.v1beta1.Query/Account",
		"/cosmos.authz.v1beta1.Query/Grants",
		"/xion.v1.Query/WebAuthNVerifyRegister",
		"/xion.jwk.v1.Query/Audience",
	}

	for _, path := range validPaths {
		result, err := wasmbinding.GetWhitelistedQuery(path)
		require.NoError(t, err, "Path should be whitelisted: %s", path)
		require.NotNil(t, result, "Result should not be nil for path: %s", path)

		// Verify it implements proto.Message
		protoMsg, ok := result.(proto.Message)
		require.True(t, ok, "Result should implement proto.Message for path: %s", path)
		require.NotNil(t, protoMsg)
	}
}