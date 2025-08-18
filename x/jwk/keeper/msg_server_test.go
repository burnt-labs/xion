package keeper_test

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestMsgServerCreate(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	require.NotNil(t, srv)

	// Test CreateAudience
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// First create an audience claim (required for CreateAudience)
	audHash := sha256.Sum256([]byte("test-audience"))
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:],
	}

	claimResp, err := srv.CreateAudienceClaim(wctx, claimMsg)
	require.NoError(t, err)
	require.NotNil(t, claimResp)

	// Now create the audience
	msg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "test-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
	}

	resp, err := srv.CreateAudience(wctx, msg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify audience was created
	audience, found := k.GetAudience(ctx, "test-audience")
	require.True(t, found)
	require.Equal(t, "test-audience", audience.Aud)
	require.Equal(t, admin, audience.Admin)

	// Test CreateAudienceClaim
	claimMsg2 := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: []byte("test-hash"),
	}

	claimResp2, err := srv.CreateAudienceClaim(wctx, claimMsg2)
	require.NoError(t, err)
	require.NotNil(t, claimResp2)

	// Verify claim was created
	claim, found := k.GetAudienceClaim(ctx, []byte("test-hash"))
	require.True(t, found)
	require.Equal(t, admin, claim.Signer)
}

func TestMsgServerUpdate(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// First create an audience claim (required for CreateAudience)
	audHash := sha256.Sum256([]byte("test-audience"))
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:],
	}

	_, err := srv.CreateAudienceClaim(wctx, claimMsg)
	require.NoError(t, err)

	// Now create an audience
	createMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "test-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
	}
	_, err = srv.CreateAudience(wctx, createMsg)
	require.NoError(t, err)

	// Test UpdateAudience - basic update
	updateMsg := &types.MsgUpdateAudience{
		Admin:    admin,
		NewAdmin: admin, // Keep the same admin
		Aud:      "test-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"updated","n":"updated","e":"AQAB"}`,
	}

	resp, err := srv.UpdateAudience(wctx, updateMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify audience was updated
	audience, found := k.GetAudience(ctx, "test-audience")
	require.True(t, found)
	require.Contains(t, audience.Key, "updated")

	// Test UpdateAudience with NewAdmin
	newAdmin := authtypes.NewModuleAddress("newadmin").String()
	updateWithNewAdminMsg := &types.MsgUpdateAudience{
		Admin:    admin,
		Aud:      "test-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"new-admin","n":"test","e":"AQAB"}`,
		NewAdmin: newAdmin,
	}

	resp2, err := srv.UpdateAudience(wctx, updateWithNewAdminMsg)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.Equal(t, newAdmin, resp2.Audience.Admin)

	// Update admin variable for subsequent tests
	admin = newAdmin

	// Test UpdateAudience - audience not found
	updateNonExistentMsg := &types.MsgUpdateAudience{
		Admin:    admin,
		NewAdmin: admin,
		Aud:      "non-existent-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"test","n":"test","e":"AQAB"}`,
	}

	_, err = srv.UpdateAudience(wctx, updateNonExistentMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "index not set")

	// Test UpdateAudience - unauthorized (wrong admin)
	wrongAdmin := authtypes.NewModuleAddress("wrongadmin").String()
	updateUnauthorizedMsg := &types.MsgUpdateAudience{
		Admin:    wrongAdmin,
		NewAdmin: wrongAdmin,
		Aud:      "test-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"test","n":"test","e":"AQAB"}`,
	}

	_, err = srv.UpdateAudience(wctx, updateUnauthorizedMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "incorrect owner")
}

func TestMsgServerUpdateWithNewAud(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Create initial audience claim and audience
	audHash := sha256.Sum256([]byte("test-audience"))
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:],
	}
	_, err := srv.CreateAudienceClaim(wctx, claimMsg)
	require.NoError(t, err)

	createMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "test-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
	}
	_, err = srv.CreateAudience(wctx, createMsg)
	require.NoError(t, err)

	// Create claim for new audience name
	newAudHash := sha256.Sum256([]byte("new-audience"))
	newClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: newAudHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, newClaimMsg)
	require.NoError(t, err)

	// Test UpdateAudience with NewAud
	updateMsg := &types.MsgUpdateAudience{
		Admin:    admin,
		NewAdmin: admin,
		Aud:      "test-audience",
		NewAud:   "new-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"updated","n":"updated","e":"AQAB"}`,
	}

	resp, err := srv.UpdateAudience(wctx, updateMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "new-audience", resp.Audience.Aud)

	// Verify old audience was removed and new one exists
	_, found := k.GetAudience(ctx, "test-audience")
	require.False(t, found)

	newAudience, found := k.GetAudience(ctx, "new-audience")
	require.True(t, found)
	require.Equal(t, "new-audience", newAudience.Aud)

	// Test UpdateAudience with NewAud that already exists
	existingAudHash := sha256.Sum256([]byte("existing-audience"))
	existingClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: existingAudHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, existingClaimMsg)
	require.NoError(t, err)

	existingCreateMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "existing-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
	}
	_, err = srv.CreateAudience(wctx, existingCreateMsg)
	require.NoError(t, err)

	// Use the actual admin of the "new-audience" audience for the update request
	updateToExistingMsg := &types.MsgUpdateAudience{
		Admin:    newAudience.Admin, // Use the admin from the actual audience
		NewAdmin: newAudience.Admin,
		Aud:      "new-audience",
		NewAud:   "existing-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"test","n":"test","e":"AQAB"}`,
	}

	_, err = srv.UpdateAudience(wctx, updateToExistingMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "audience already created")

	// Test UpdateAudience with NewAud but no claim
	updateWithoutClaimMsg := &types.MsgUpdateAudience{
		Admin:    newAudience.Admin, // Use the admin from the actual audience
		NewAdmin: newAudience.Admin,
		Aud:      "new-audience",
		NewAud:   "no-claim-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"test","n":"test","e":"AQAB"}`,
	}

	_, err = srv.UpdateAudience(wctx, updateWithoutClaimMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "claim not found")

	// Test UpdateAudience with NewAud but wrong signer in claim
	wrongSigner := authtypes.NewModuleAddress("wrongsigner").String()
	wrongSignerAudHash := sha256.Sum256([]byte("wrong-signer-audience"))
	wrongSignerClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   wrongSigner,
		AudHash: wrongSignerAudHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, wrongSignerClaimMsg)
	require.NoError(t, err)

	updateWithWrongSignerMsg := &types.MsgUpdateAudience{
		Admin:    newAudience.Admin, // Use the admin from the actual audience
		NewAdmin: newAudience.Admin,
		Aud:      "new-audience",
		NewAud:   "wrong-signer-audience",
		Key:      `{"kty":"RSA","use":"sig","kid":"test","n":"test","e":"AQAB"}`,
	}

	_, err = srv.UpdateAudience(wctx, updateWithWrongSignerMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected "+wrongSigner+", got")
}

func TestMsgServerDelete(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// First create an audience claim (required for CreateAudience)
	audHash := sha256.Sum256([]byte("test-audience"))
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:],
	}

	_, err := srv.CreateAudienceClaim(wctx, claimMsg)
	require.NoError(t, err)

	// Now create an audience
	createMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "test-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS","e":"AQAB","kid":"test-key"}`,
	}
	_, err = srv.CreateAudience(wctx, createMsg)
	require.NoError(t, err)

	// Test DeleteAudienceClaim
	deleteClaimMsg := &types.MsgDeleteAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:], // Use the same hash that was created
	}

	resp, err := srv.DeleteAudienceClaim(wctx, deleteClaimMsg)
	require.NoError(t, err)
	require.NotNil(t, resp)

	// Verify claim was deleted
	_, found := k.GetAudienceClaim(ctx, audHash[:])
	require.False(t, found)

	// Test DeleteAudience
	deleteMsg := &types.MsgDeleteAudience{
		Admin: admin,
		Aud:   "test-audience",
	}

	deleteResp, err := srv.DeleteAudience(wctx, deleteMsg)
	require.NoError(t, err)
	require.NotNil(t, deleteResp)

	// Verify audience was deleted
	_, found = k.GetAudience(ctx, "test-audience")
	require.False(t, found)
}

func TestMsgServerErrorCases(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// Test CreateAudienceClaim with invalid admin address
	audHash := sha256.Sum256([]byte("test-audience"))
	invalidClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   "invalid-address",
		AudHash: audHash[:],
	}
	_, err := srv.CreateAudienceClaim(wctx, invalidClaimMsg)
	require.Error(t, err)

	// Test CreateAudienceClaim duplicate claim (already exists)
	validClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, validClaimMsg)
	require.NoError(t, err)

	// Try to create the same claim again - should fail
	_, err = srv.CreateAudienceClaim(wctx, validClaimMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "audience already claimed")

	// Test DeleteAudienceClaim on non-existent claim
	deleteClaimMsg := &types.MsgDeleteAudienceClaim{
		Admin:   admin,
		AudHash: []byte("non-existent"),
	}
	_, err = srv.DeleteAudienceClaim(wctx, deleteClaimMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "index not set")

	// Test CreateAudience without claim
	createMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "no-claim-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, createMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "claim not found")

	// Test DeleteAudience on non-existent audience
	deleteMsg := &types.MsgDeleteAudience{
		Admin: admin,
		Aud:   "non-existent",
	}
	_, err = srv.DeleteAudience(wctx, deleteMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// Test UpdateAudience on non-existent audience
	updateMsg := &types.MsgUpdateAudience{
		Admin:    admin,
		NewAdmin: admin,
		Aud:      "non-existent",
		NewAud:   "new-aud",
		Key:      `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.UpdateAudience(wctx, updateMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestMsgServerComprehensiveErrorCoverage(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	wrongAdmin := authtypes.NewModuleAddress("wrongadmin").String()

	// Test CreateAudience without claim
	createMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "no-claim-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err := srv.CreateAudience(wctx, createMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "claim not found")

	// Create claim with wrong signer
	audHash := sha256.Sum256([]byte("wrong-signer-audience"))
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin:   wrongAdmin,
		AudHash: audHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, claimMsg)
	require.NoError(t, err)

	// Test CreateAudience with wrong signer
	createMsgWrongSigner := &types.MsgCreateAudience{
		Admin: admin, // Different from claim signer
		Aud:   "wrong-signer-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, createMsgWrongSigner)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected "+wrongAdmin+", got")

	// Test DeleteAudienceClaim on non-existent claim
	deleteClaimMsg := &types.MsgDeleteAudienceClaim{
		Admin:   admin,
		AudHash: []byte("non-existent-hash"),
	}
	_, err = srv.DeleteAudienceClaim(wctx, deleteClaimMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")

	// Create a valid claim and audience for testing unauthorized delete
	validAudHash := sha256.Sum256([]byte("valid-audience"))
	validClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: validAudHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, validClaimMsg)
	require.NoError(t, err)

	validCreateMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "valid-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, validCreateMsg)
	require.NoError(t, err)

	// Test DeleteAudienceClaim with wrong admin
	wrongDeleteClaimMsg := &types.MsgDeleteAudienceClaim{
		Admin:   wrongAdmin,
		AudHash: validAudHash[:],
	}
	_, err = srv.DeleteAudienceClaim(wctx, wrongDeleteClaimMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "incorrect owner")

	// Test DeleteAudience with wrong admin
	wrongDeleteMsg := &types.MsgDeleteAudience{
		Admin: wrongAdmin,
		Aud:   "valid-audience",
	}
	_, err = srv.DeleteAudience(wctx, wrongDeleteMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "incorrect owner")
}

// Additional tests to improve CreateAudience coverage to 100%
func TestMsgServerCreateAudienceComprehensiveErrorPaths(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	wctx := sdk.WrapSDKContext(ctx)

	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	wrongAdmin := "cosmos1wrong"

	// Test 1: CreateAudience when audience already exists
	audHash := sha256.Sum256([]byte("existing-audience"))
	claimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: audHash[:],
	}
	_, err := srv.CreateAudienceClaim(wctx, claimMsg)
	require.NoError(t, err)

	// Create initial audience
	createMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "existing-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, createMsg)
	require.NoError(t, err)

	// Try to create same audience again - should fail
	duplicateMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "existing-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test2","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, duplicateMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "audience already created")

	// Test 2: CreateAudience when claim doesn't exist
	noclaimMsg := &types.MsgCreateAudience{
		Admin: admin,
		Aud:   "no-claim-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, noclaimMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "claim not found for aud")

	// Test 3: CreateAudience when claim signer doesn't match admin
	wrongAdminHash := sha256.Sum256([]byte("wrong-admin-audience"))
	wrongAdminClaimMsg := &types.MsgCreateAudienceClaim{
		Admin:   admin,
		AudHash: wrongAdminHash[:],
	}
	_, err = srv.CreateAudienceClaim(wctx, wrongAdminClaimMsg)
	require.NoError(t, err)

	// Try to create audience with different admin
	wrongAdminAudienceMsg := &types.MsgCreateAudience{
		Admin: wrongAdmin,
		Aud:   "wrong-admin-audience",
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"test","e":"AQAB"}`,
	}
	_, err = srv.CreateAudience(wctx, wrongAdminAudienceMsg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "expected")
	require.Contains(t, err.Error(), "got")
}
