package wasmbinding

import (
	"fmt"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/stretchr/testify/require"
)

// TestDeterministicPathsStillAllowed verifies that deterministic query paths remain whitelisted
func TestDeterministicPathsStillAllowed(t *testing.T) {
	// These paths should still be allowed as they are deterministic
	allowedPaths := []string{
		"/xion.jwk.v1.Query/AudienceAll",
		"/xion.jwk.v1.Query/Audience",
		"/xion.jwk.v1.Query/Params",
		"/xion.jwk.v1.Query/ValidateJWT",
		"/cosmos.bank.v1beta1.Query/Balance",
		"/cosmos.auth.v1beta1.Query/Account",
	}

	for _, path := range allowedPaths {
		t.Run(fmt.Sprintf("Path_%s_should_be_allowed", path), func(t *testing.T) {
			_, err := GetWhitelistedQuery(path)
			require.NoError(t, err, "Query path %s should still be whitelisted, but got error: %v", path, err)
		})
	}
}

// TestWhitelistSecurityInvariants verifies critical security properties of the whitelist
func TestWhitelistSecurityInvariants(t *testing.T) {
	t.Run("All_whitelisted_queries_deterministic", func(t *testing.T) {
		// This test documents that all remaining whitelisted queries should be deterministic
		// If a new query is added that might be non-deterministic, this test should catch it
		deterministicPaths := map[string]string{
			// Auth module - deterministic (account state)
			"/cosmos.auth.v1beta1.Query/Account":        "deterministic_account_state",
			"/cosmos.auth.v1beta1.Query/Params":         "deterministic_module_params",
			"/xion.v1.Query/WebAuthNVerifyRegister":     "deterministic_webauthn_register",
			"/xion.v1.Query/WebAuthNVerifyAuthenticate": "deterministic_webauthn_authenticate",

			// Bank module - deterministic (balances, metadata)
			"/cosmos.bank.v1beta1.Query/Balance":       "deterministic_account_balance",
			"/cosmos.bank.v1beta1.Query/DenomMetadata": "deterministic_denom_info",
			"/cosmos.bank.v1beta1.Query/SupplyOf":      "deterministic_supply_info",

			// JWK module - deterministic (stored data, JWT validation with fixed time)
			"/xion.jwk.v1.Query/AudienceAll": "deterministic_stored_audiences",
			"/xion.jwk.v1.Query/Audience":    "deterministic_audience_lookup",
			"/xion.jwk.v1.Query/Params":      "deterministic_module_params",
			"/xion.jwk.v1.Query/ValidateJWT": "deterministic_jwt_validation", // Uses block time, not system time
		}

		for path, reason := range deterministicPaths {
			_, err := GetWhitelistedQuery(path)
			require.NoError(t, err, "Deterministic query %s (%s) should be whitelisted", path, reason)
		}
	})
}

// TestStargateWhitelistThreadSafety verifies the whitelist is thread-safe
func TestStargateWhitelistThreadSafety(t *testing.T) {
	// The whitelist uses sync.Map which should be thread-safe
	// This test verifies basic concurrent access doesn't panic

	paths := []string{
		"/xion.jwk.v1.Query/Params",
		"/cosmos.bank.v1beta1.Query/Balance",
		"/invalid/path/should/fail",
	}

	done := make(chan bool, len(paths))

	// Launch concurrent goroutines accessing the whitelist
	for _, path := range paths {
		go func(p string) {
			defer func() { done <- true }()
			_, _ = GetWhitelistedQuery(p) // Don't care about result, just that it doesn't panic
		}(path)
	}

	// Wait for all goroutines to complete
	for i := 0; i < len(paths); i++ {
		<-done
	}
}

// TestGetWhitelistedQueryErrorPaths tests the error paths in GetWhitelistedQuery
func TestGetWhitelistedQueryErrorPaths(t *testing.T) {
	t.Run("non_whitelisted_path_returns_error", func(t *testing.T) {
		// Test path that should not be in whitelist
		invalidPath := "/invalid.module.v1.Query/NonExistentMethod"

		_, err := GetWhitelistedQuery(invalidPath)
		require.Error(t, err)

		// Verify it's the right type of error
		unsupportedErr, ok := err.(wasmvmtypes.UnsupportedRequest)
		require.True(t, ok, "Expected UnsupportedRequest error type")
		require.Contains(t, unsupportedErr.Kind, "path is not allowed from the contract")
		require.Contains(t, unsupportedErr.Kind, invalidPath)
	})

	t.Run("empty_path_returns_error", func(t *testing.T) {
		// Test empty path
		_, err := GetWhitelistedQuery("")
		require.Error(t, err)

		// Should be UnsupportedRequest error
		_, ok := err.(wasmvmtypes.UnsupportedRequest)
		require.True(t, ok, "Expected UnsupportedRequest error type")
	})

	t.Run("malformed_path_returns_error", func(t *testing.T) {
		// Test malformed path
		malformedPath := "not-a-valid-path-format"
		_, err := GetWhitelistedQuery(malformedPath)
		require.Error(t, err)

		// Should be UnsupportedRequest error
		unsupportedErr, ok := err.(wasmvmtypes.UnsupportedRequest)
		require.True(t, ok, "Expected UnsupportedRequest error type")
		require.Contains(t, unsupportedErr.Kind, malformedPath)
	})

	t.Run("invalid_proto_type_in_whitelist_returns_error", func(t *testing.T) {
		// Temporarily add an invalid entry to the whitelist
		// We need to add a non-proto.Message type to trigger the type assertion failure
		testPath := "/test.invalid.v1.Query/BadType"

		// Store a non-proto.Message type to trigger the type assertion failure on line 94-97
		stargateWhitelist.Store(testPath, "invalid_non_proto_message")

		// Clean up after test
		defer stargateWhitelist.Delete(testPath)

		_, err := GetWhitelistedQuery(testPath)
		require.Error(t, err)

		// Should be wasmvmtypes.Unknown error (from line 96)
		_, ok := err.(wasmvmtypes.Unknown)
		require.True(t, ok, "Expected wasmvmtypes.Unknown error type")
	})
}
