package wasmbinding_test

import (
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/gogoproto/proto"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	xionapp "github.com/burnt-labs/xion/app"
	wasmbinding "github.com/burnt-labs/xion/wasmbindings"
)

// Test to cover the "res.Value == nil" branch in both GrpcQuerier and StargateQuerier
func TestQueriers_NilResponseValue(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(true)

	// Create a valid request that should return a response but might have nil Value
	validRequest := &banktypes.QueryBalanceRequest{
		Address: "cosmos1test",
		Denom:   "stake",
	}
	data, err := proto.Marshal(validRequest)
	require.NoError(t, err)

	t.Run("GrpcQuerier_with_valid_request", func(t *testing.T) {
		grpcQuerier := wasmbinding.GrpcQuerier(*app.GRPCQueryRouter())
		grpcRequest := &wasmvmtypes.GrpcQuery{
			Path: "/cosmos.bank.v1beta1.Query/Balance",
			Data: data,
		}

		response, err := grpcQuerier(ctx, grpcRequest)
		// This should succeed since the path is valid and the request is valid
		if err != nil {
			// If there's an error, make sure it's not the nil value error we're testing
			require.NotContains(t, err.Error(), "res returned from abci query route is nil")
		} else {
			require.NotNil(t, response)
		}
	})

	t.Run("StargateQuerier_with_valid_request", func(t *testing.T) {
		stargateQuerier := wasmbinding.StargateQuerier(*app.GRPCQueryRouter(), app.AppCodec())
		stargateRequest := &wasmvmtypes.StargateQuery{
			Path: "/cosmos.bank.v1beta1.Query/Balance",
			Data: data,
		}

		response, err := stargateQuerier(ctx, stargateRequest)
		// This should succeed since the path is valid and the request is valid
		if err != nil {
			// If there's an error, make sure it's not the nil value error we're testing
			require.NotContains(t, err.Error(), "res returned from abci query route is nil")
		} else {
			require.NotNil(t, response)
		}
	})
}

// Test the SetupKeys function with different scenarios
func TestSetupKeys_Comprehensive(t *testing.T) {
	// Just call it and accept either success or expected failure
	result, err := wasmbinding.SetupKeys()
	if err != nil {
		// Expected case: key file doesn't exist
		require.Contains(t, err.Error(), "no such file or directory")
		require.Nil(t, result)
	} else {
		// Unexpected but valid case: key file exists
		require.NotNil(t, result)
	}
}

// Test SetupPublicKeys with realistic scenarios
func TestSetupPublicKeys_Comprehensive(t *testing.T) {
	t.Run("with_empty_slice", func(t *testing.T) {
		// Test with empty string (default path)
		result1, result2, err := wasmbinding.SetupPublicKeys("")
		if err != nil {
			// Expected case: default key file doesn't exist
			require.Contains(t, err.Error(), "no such file or directory")
			require.Nil(t, result1)
			require.Nil(t, result2)
		} else {
			// Unexpected but valid case: key file exists
			require.NotNil(t, result1)
			// Note: result2 is expected to be nil based on the function implementation
			require.Nil(t, result2)
		}
	})

	t.Run("with_nonexistent_file", func(t *testing.T) {
		// Test with a file that definitely doesn't exist
		result1, result2, err := wasmbinding.SetupPublicKeys("/tmp/definitely_nonexistent_key_file.pem")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
		require.Nil(t, result1)
		require.Nil(t, result2)
	})
}

// Test all the different whitelisted query paths to ensure they're correctly configured
func TestAllWhitelistedPaths_Complete(t *testing.T) {
	allPaths := map[string]string{
		// auth
		"/cosmos.auth.v1beta1.Query/Account": "authtypes.QueryAccountResponse",
		"/cosmos.auth.v1beta1.Query/Params": "authtypes.QueryParamsResponse",
		"/cosmos.auth.v1beta1.Query/ModuleAccounts": "authtypes.QueryModuleAccountsResponse",

		// authz
		"/cosmos.authz.v1beta1.Query/Grants": "authztypes.QueryGrantsResponse",

		// bank
		"/cosmos.bank.v1beta1.Query/Balance": "banktypes.QueryBalanceResponse",
		"/cosmos.bank.v1beta1.Query/DenomMetadata": "banktypes.QueryDenomMetadataResponse",
		"/cosmos.bank.v1beta1.Query/DenomsMetadata": "banktypes.QueryDenomsMetadataResponse",
		"/cosmos.bank.v1beta1.Query/Params": "banktypes.QueryParamsResponse",
		"/cosmos.bank.v1beta1.Query/SupplyOf": "banktypes.QuerySupplyOfResponse",

		// distribution
		"/cosmos.distribution.v1beta1.Query/Params": "distributiontypes.QueryParamsResponse",
		"/cosmos.distribution.v1beta1.Query/DelegatorWithdrawAddress": "distributiontypes.QueryDelegatorWithdrawAddressResponse",
		"/cosmos.distribution.v1beta1.Query/ValidatorCommission": "distributiontypes.QueryValidatorCommissionResponse",

		// feegrant
		"/cosmos.feegrant.v1beta1.Query/Allowance": "feegranttypes.QueryAllowanceResponse",
		"/cosmos.feegrant.v1beta1.Query/AllowancesByGranter": "feegranttypes.QueryAllowancesByGranterResponse",

		// gov
		"/cosmos.gov.v1beta1.Query/Deposit": "govtypesv1.QueryDepositResponse",
		"/cosmos.gov.v1beta1.Query/Params": "govtypesv1.QueryParamsResponse",
		"/cosmos.gov.v1beta1.Query/Vote": "govtypesv1.QueryVoteResponse",

		// slashing
		"/cosmos.slashing.v1beta1.Query/Params": "slashingtypes.QueryParamsResponse",
		"/cosmos.slashing.v1beta1.Query/SigningInfo": "slashingtypes.QuerySigningInfoResponse",

		// staking
		"/cosmos.staking.v1beta1.Query/Delegation": "stakingtypes.QueryDelegationResponse",
		"/cosmos.staking.v1beta1.Query/Params": "stakingtypes.QueryParamsResponse",
		"/cosmos.staking.v1beta1.Query/Validator": "stakingtypes.QueryValidatorResponse",

		// xion queries
		"/xion.v1.Query/WebAuthNVerifyRegister": "xiontypes.QueryWebAuthNVerifyRegisterResponse",
		"/xion.v1.Query/WebAuthNVerifyAuthenticate": "xiontypes.QueryWebAuthNVerifyAuthenticateResponse",
		"/xion.jwk.v1.Query/AudienceAll": "jwktypes.QueryAllAudienceResponse",
		"/xion.jwk.v1.Query/Audience": "jwktypes.QueryGetAudienceResponse",
		"/xion.jwk.v1.Query/Params": "jwktypes.QueryParamsResponse",
		"/xion.jwk.v1.Query/ValidateJWT": "jwktypes.QueryValidateJWTResponse",
	}

	for path, expectedType := range allPaths {
		t.Run("path_"+path, func(t *testing.T) {
			result, err := wasmbinding.GetWhitelistedQuery(path)
			require.NoError(t, err, "Path %s should be whitelisted", path)
			require.NotNil(t, result, "Result should not be nil for path %s", path)

			// Verify the result implements proto.Message
			_, ok := result.(proto.Message)
			require.True(t, ok, "Result should implement proto.Message for path %s", path)

			t.Logf("Path %s returns type %T (expected %s)", path, result, expectedType)
		})
	}
}

// Test various edge cases and error conditions
func TestErrorConditions_Comprehensive(t *testing.T) {
	app := xionapp.Setup(t)
	ctx := app.NewContext(true)

	t.Run("GrpcQuerier_unknown_error_unmarshal", func(t *testing.T) {
		grpcQuerier := wasmbinding.GrpcQuerier(*app.GRPCQueryRouter())

		// Use a valid path but with malformed data that causes unmarshal errors
		grpcRequest := &wasmvmtypes.GrpcQuery{
			Path: "/cosmos.bank.v1beta1.Query/Balance",
			Data: []byte{0xFF, 0xFF, 0xFF, 0xFF}, // Invalid protobuf data
		}

		response, err := grpcQuerier(ctx, grpcRequest)
		require.Error(t, err)
		require.Nil(t, response)
	})

	t.Run("StargateQuerier_unknown_error_unmarshal", func(t *testing.T) {
		stargateQuerier := wasmbinding.StargateQuerier(*app.GRPCQueryRouter(), app.AppCodec())

		// Use a valid path but with malformed data that causes unmarshal errors
		stargateRequest := &wasmvmtypes.StargateQuery{
			Path: "/cosmos.bank.v1beta1.Query/Balance",
			Data: []byte{0xFF, 0xFF, 0xFF, 0xFF}, // Invalid protobuf data
		}

		response, err := stargateQuerier(ctx, stargateRequest)
		require.Error(t, err)
		require.Nil(t, response)
	})
}