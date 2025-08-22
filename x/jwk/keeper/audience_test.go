package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/burnt-labs/xion/x/jwk/types"
)

func TestAudienceOperations(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Test audience creation
	admin := authtypes.NewModuleAddress(govtypes.ModuleName).String()
	audience := types.Audience{
		Aud:   "test-audience",
		Admin: admin,
		Key:   "test-key",
	}

	// Test SetAudience
	k.SetAudience(ctx, audience)

	// Test GetAudience
	retrievedAudience, found := k.GetAudience(ctx, "test-audience")
	require.True(t, found)
	require.Equal(t, audience, retrievedAudience)

	// Test GetAudience with non-existent audience
	_, found = k.GetAudience(ctx, "non-existent")
	require.False(t, found)

	// Test GetAllAudience
	allAudiences := k.GetAllAudience(ctx)
	require.Len(t, allAudiences, 1)
	require.Equal(t, audience, allAudiences[0])

	// Add another audience
	audience2 := types.Audience{
		Aud:   "test-audience-2",
		Admin: admin,
		Key:   "test-key-2",
	}
	k.SetAudience(ctx, audience2)

	// Verify we now have 2 audiences
	allAudiences = k.GetAllAudience(ctx)
	require.Len(t, allAudiences, 2)

	// Test RemoveAudience
	k.RemoveAudience(ctx, "test-audience")

	// Verify audience was removed
	_, found = k.GetAudience(ctx, "test-audience")
	require.False(t, found)

	// Verify we now have 1 audience
	allAudiences = k.GetAllAudience(ctx)
	require.Len(t, allAudiences, 1)
	require.Equal(t, audience2, allAudiences[0])
}

func TestAudienceClaimOperations(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Test audience claim creation
	claim := []byte("test-claim-hash")
	signer := authtypes.NewModuleAddress(govtypes.ModuleName)

	// Test SetAudienceClaim
	k.SetAudienceClaim(ctx, claim, signer)

	// Test GetAudienceClaim
	retrievedClaim, found := k.GetAudienceClaim(ctx, claim)
	require.True(t, found)
	require.Equal(t, signer.String(), retrievedClaim.Signer)

	// Test GetAudienceClaim with non-existent claim
	_, found = k.GetAudienceClaim(ctx, []byte("non-existent"))
	require.False(t, found)

	// Test RemoveAudienceClaim
	k.RemoveAudienceClaim(ctx, claim)

	// Verify claim was removed
	_, found = k.GetAudienceClaim(ctx, claim)
	require.False(t, found)
}
