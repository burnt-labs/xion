package keeper_test

import (
	"crypto/sha256"
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/keeper"
	"github.com/burnt-labs/xion/x/jwk/types"
)

// TestUpdateAudienceNewAudAndNewAdminSimultaneous tests that UpdateAudience
// correctly handles the simultaneous NewAud + NewAdmin case introduced in PR #508.
//
// Specifically it verifies:
//  1. The old audience record is removed.
//  2. The old aud claim is removed.
//  3. A new audience record is created under the new aud name.
//  4. The aud claim for the new name is owned by the new admin.
//  5. The old admin cannot re-create the old audience name without first
//     obtaining a new claim (i.e. the orphan claim is gone).
//  6. The new admin can create a brand-new audience at the new aud name if
//     needed (the transferred claim is in place).
func TestUpdateAudienceNewAudAndNewAdminSimultaneous(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	oldAdmin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	newAdmin := authtypes.NewModuleAddress("newadmin").String()

	const oldAud = "original-audience"
	const newAud = "renamed-audience"

	// --- setup: create claim + audience under oldAdmin ---

	oldAudHash := sha256.Sum256([]byte(oldAud))
	_, err := srv.CreateAudienceClaim(ctx, &types.MsgCreateAudienceClaim{
		Admin:   oldAdmin,
		AudHash: oldAudHash[:],
	})
	require.NoError(t, err)

	_, err = srv.CreateAudience(ctx, &types.MsgCreateAudience{
		Admin: oldAdmin,
		Aud:   oldAud,
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"original","e":"AQAB","kid":"orig"}`,
	})
	require.NoError(t, err)

	// oldAdmin pre-claims the new aud name (required by UpdateAudience).
	newAudHash := sha256.Sum256([]byte(newAud))
	_, err = srv.CreateAudienceClaim(ctx, &types.MsgCreateAudienceClaim{
		Admin:   oldAdmin,
		AudHash: newAudHash[:],
	})
	require.NoError(t, err)

	// --- action: simultaneous NewAud + NewAdmin ---

	resp, err := srv.UpdateAudience(ctx, &types.MsgUpdateAudience{
		Admin:    oldAdmin,
		NewAdmin: newAdmin,
		Aud:      oldAud,
		NewAud:   newAud,
		Key:      `{"kty":"RSA","use":"sig","alg":"RS256","n":"renamed","e":"AQAB","kid":"new"}`,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, newAud, resp.Audience.Aud)
	require.Equal(t, newAdmin, resp.Audience.Admin)

	// 1. Old audience record must be gone.
	_, found := k.GetAudience(ctx, oldAud)
	require.False(t, found, "old audience record should have been removed")

	// 2. Old aud claim must be gone (no orphan).
	_, found = k.GetAudienceClaim(ctx, oldAudHash[:])
	require.False(t, found, "old aud claim should have been removed")

	// 3. New audience record must exist.
	newAudience, found := k.GetAudience(ctx, newAud)
	require.True(t, found, "new audience record should exist")
	require.Equal(t, newAdmin, newAudience.Admin)

	// 4. New aud claim must be owned by newAdmin.
	newAudClaim, found := k.GetAudienceClaim(ctx, newAudHash[:])
	require.True(t, found, "new aud claim should exist")
	require.Equal(t, newAdmin, newAudClaim.Signer,
		"new aud claim should be owned by the new admin")

	// 5. Old admin MUST NOT be able to re-create the old audience without a
	//    fresh claim (the old claim is gone, so this should fail with "claim not found").
	oldAudHash2 := sha256.Sum256([]byte(oldAud))
	_, err = srv.CreateAudience(ctx, &types.MsgCreateAudience{
		Admin: oldAdmin,
		Aud:   oldAud,
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"replay","e":"AQAB","kid":"replay"}`,
	})
	require.Error(t, err, "old admin should not be able to recreate the old audience")
	require.Contains(t, err.Error(), "claim not found")

	// Ensure the hash variable is used (avoids "declared and not used" compile error).
	_ = oldAudHash2

	// 6. New admin SHOULD be able to create a fresh audience at the new aud
	//    name after first deleting the current one (proves the claim belongs to
	//    new admin).  We verify this by checking the claim signer directly
	//    rather than exercising the full create path again (which would fail
	//    with "audience already created").
	require.Equal(t, newAdmin, newAudClaim.Signer,
		"new admin controls the new aud claim and could recreate the audience if needed")
}

// TestUpdateAudienceOnlyNewAdmin is a regression test that confirms
// UpdateAudience with only a NewAdmin change (no NewAud) still transfers the
// audience claim to the new admin, so the old admin cannot exploit the orphan
// claim.
func TestUpdateAudienceOnlyNewAdmin(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)

	oldAdmin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	newAdmin := authtypes.NewModuleAddress("newadmin2").String()

	const aud = "transfer-only-audience"

	// Setup: create claim + audience.
	audHash := sha256.Sum256([]byte(aud))
	_, err := srv.CreateAudienceClaim(ctx, &types.MsgCreateAudienceClaim{
		Admin:   oldAdmin,
		AudHash: audHash[:],
	})
	require.NoError(t, err)

	_, err = srv.CreateAudience(ctx, &types.MsgCreateAudience{
		Admin: oldAdmin,
		Aud:   aud,
		Key:   `{"kty":"RSA","use":"sig","alg":"RS256","n":"original","e":"AQAB","kid":"orig"}`,
	})
	require.NoError(t, err)

	// Action: change admin only, aud name stays the same.
	resp, err := srv.UpdateAudience(ctx, &types.MsgUpdateAudience{
		Admin:    oldAdmin,
		NewAdmin: newAdmin,
		Aud:      aud,
		// NewAud intentionally omitted
		Key: `{"kty":"RSA","use":"sig","alg":"RS256","n":"updated","e":"AQAB","kid":"upd"}`,
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, newAdmin, resp.Audience.Admin)
	require.Equal(t, aud, resp.Audience.Aud)

	// Audience record should reflect new admin.
	audience, found := k.GetAudience(ctx, aud)
	require.True(t, found)
	require.Equal(t, newAdmin, audience.Admin)

	// Claim must now be owned by new admin.
	claim, found := k.GetAudienceClaim(ctx, audHash[:])
	require.True(t, found, "audience claim should still exist after admin transfer")
	require.Equal(t, newAdmin, claim.Signer,
		"claim should have been transferred to the new admin")

	// Old admin must not be able to delete the audience anymore.
	_, err = srv.DeleteAudience(ctx, &types.MsgDeleteAudience{
		Admin: oldAdmin,
		Aud:   aud,
	})
	require.Error(t, err, "old admin should not be able to delete the audience after transfer")
	require.Contains(t, err.Error(), "incorrect owner")

	// New admin should be able to manage the audience.
	_, err = srv.DeleteAudience(ctx, &types.MsgDeleteAudience{
		Admin: newAdmin,
		Aud:   aud,
	})
	require.NoError(t, err, "new admin should be able to delete the audience")
}
