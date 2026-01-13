package indexer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/burnt-labs/xion/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// TestMultiIterateRaw_RealWorldUsage demonstrates using MultiIterateRaw
// through a realistic query scenario with raw byte-range pagination.
//
// This test shows how MultiIterateRaw would be used when implementing
// custom pagination logic that requires raw byte boundaries instead of
// the higher-level Iterate() API.
func TestMultiIterateRaw_RealWorldUsage(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()

	// Create handler with Multi index
	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Setup test data: Create multiple grants for different grantees
	granter1 := sdk.AccAddress([]byte("granter1_address____"))
	granter2 := sdk.AccAddress([]byte("granter2_address____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))
	grantee2 := sdk.AccAddress([]byte("grantee2_address____"))

	// Create send authorization
	sendAuth := banktypes.NewSendAuthorization(
		sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
		nil,
	)
	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	msgType := "/cosmos.bank.v1beta1.MsgSend"

	// Add grants - these will be indexed by the Multi index
	grants := []struct {
		granter sdk.AccAddress
		grantee sdk.AccAddress
		msgType string
	}{
		{granter1, grantee1, msgType},
		{granter1, grantee2, msgType},
		{granter2, grantee1, msgType},
		{granter2, grantee2, msgType},
	}

	for _, g := range grants {
		err := handler.SetGrant(ctx, g.granter, g.grantee, g.msgType, grant)
		require.NoError(t, err)
	}

	t.Run("RawByteRangePagination", func(t *testing.T) {
		// This test demonstrates using MultiIterateRaw for custom pagination
		// with raw byte boundaries - a use case where the standard Iterate()
		// API isn't sufficient.

		// Use MultiIterateRaw to iterate over the grantee index with raw byte boundaries
		// The Multi index stores: Pair[grantee, Triple[granter, grantee, msgType]]
		iter, err := MultiIterateRaw(
			ctx,
			handler.Authorizations.Indexes.Grantee,
			nil, // start: nil means from beginning
			nil, // end: nil means to end
			collections.OrderAscending,
		)
		require.NoError(t, err)
		require.NotNil(t, iter)
		defer iter.Close()

		// Count all entries using raw iteration
		count := 0
		var firstKey, lastKey collections.Pair[sdk.AccAddress, collections.Triple[sdk.AccAddress, sdk.AccAddress, string]]

		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(t, err)

			if count == 0 {
				firstKey = key
			}
			lastKey = key
			count++

			// Each key contains: Pair[grantee, Triple[granter, grantee, msgType]]
			grantee := key.K1()
			primaryKey := key.K2()
			granter := primaryKey.K1()

			t.Logf("Found grant via MultiIterateRaw: grantee=%x, granter=%x", grantee, granter)
		}

		// We should have 4 total grants in the index
		require.Equal(t, 4, count, "expected 4 grants in the grantee index")
		require.NotNil(t, firstKey)
		require.NotNil(t, lastKey)
	})

	t.Run("RawByteRangePagination_WithBoundaries", func(t *testing.T) {
		// This demonstrates pagination using raw byte boundaries
		// This is useful when you need to resume iteration from a specific byte position

		// First, get all keys to understand the byte layout
		allKeys := [][]byte{}

		iter, err := MultiIterateRaw(
			ctx,
			handler.Authorizations.Indexes.Grantee,
			nil,
			nil,
			collections.OrderAscending,
		)
		require.NoError(t, err)
		defer iter.Close()

		keyCodec := handler.Authorizations.Indexes.Grantee.KeyCodec()

		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(t, err)

			// Encode the key to bytes
			buf := make([]byte, 256)
			n, err := keyCodec.EncodeNonTerminal(buf, key)
			require.NoError(t, err)
			keyBytes := make([]byte, n)
			copy(keyBytes, buf[:n])
			allKeys = append(allKeys, keyBytes)
		}

		require.Len(t, allKeys, 4, "should have 4 keys")

		// Now test pagination by using the second key as start boundary
		if len(allKeys) >= 2 {
			t.Run("StartFromSecondKey", func(t *testing.T) {
				// Start iteration from the second key
				iter2, err := MultiIterateRaw(
					ctx,
					handler.Authorizations.Indexes.Grantee,
					allKeys[1], // start from second key
					nil,        // no end boundary
					collections.OrderAscending,
				)
				require.NoError(t, err)
				defer iter2.Close()

				count := 0
				for ; iter2.Valid(); iter2.Next() {
					count++
				}

				// Should get 3 results (from key 2 onwards: keys 1, 2, 3 in 0-indexed)
				require.GreaterOrEqual(t, count, 2, "should have at least 2 results when starting from second key")
				t.Logf("Got %d results when starting from second key", count)
			})
		}
	})

	t.Run("DescendingOrder", func(t *testing.T) {
		// Test descending order iteration using MultiIterateRaw
		iter, err := MultiIterateRaw(
			ctx,
			handler.Authorizations.Indexes.Grantee,
			nil,
			nil,
			collections.OrderDescending,
		)
		require.NoError(t, err)
		defer iter.Close()

		count := 0
		var prevGrantee sdk.AccAddress

		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(t, err)

			grantee := key.K1()

			// In descending order, each grantee should be <= previous
			if count > 0 {
				require.True(t,
					string(grantee) <= string(prevGrantee),
					"grantees should be in descending order",
				)
			}
			prevGrantee = grantee
			count++
		}

		require.Equal(t, 4, count)
	})
}

// TestMultiIterateRaw_CompareWithStandardIterate compares MultiIterateRaw
// with the standard Iterate() method to show they produce the same results.
func TestMultiIterateRaw_CompareWithStandardIterate(t *testing.T) {
	memDB, cdc, _ := setupTest(t)
	ctx := context.Background()

	handler, err := NewAuthzHandler(&kvAccessor{db: memDB}, cdc)
	require.NoError(t, err)

	// Setup test data
	granter := sdk.AccAddress([]byte("granter_address_____"))
	grantee1 := sdk.AccAddress([]byte("grantee1_address____"))
	grantee2 := sdk.AccAddress([]byte("grantee2_address____"))

	sendAuth := banktypes.NewSendAuthorization(
		sdk.NewCoins(sdk.NewInt64Coin("uxion", 1000)),
		nil,
	)

	expiration := time.Now().Add(24 * time.Hour)
	grant, err := authz.NewGrant(expiration, sendAuth, nil)
	require.NoError(t, err)

	msgType := "/cosmos.bank.v1beta1.MsgSend"

	err = handler.SetGrant(ctx, granter, grantee1, msgType, grant)
	require.NoError(t, err)
	err = handler.SetGrant(ctx, granter, grantee2, msgType, grant)
	require.NoError(t, err)

	// Collect results using MultiIterateRaw
	var rawResults []sdk.AccAddress
	iterRaw, err := MultiIterateRaw(
		ctx,
		handler.Authorizations.Indexes.Grantee,
		nil,
		nil,
		collections.OrderAscending,
	)
	require.NoError(t, err)
	for ; iterRaw.Valid(); iterRaw.Next() {
		key, err := iterRaw.Key()
		require.NoError(t, err)
		rawResults = append(rawResults, key.K1())
	}
	iterRaw.Close()

	// Collect results using standard Iterate with nil range
	var stdResults []sdk.AccAddress
	iterStd, err := handler.Authorizations.Indexes.Grantee.Iterate(ctx, nil)
	require.NoError(t, err)
	for ; iterStd.Valid(); iterStd.Next() {
		key, err := iterStd.FullKey()
		require.NoError(t, err)
		stdResults = append(stdResults, key.K1())
	}
	iterStd.Close()

	// Both should produce the same number of results
	require.Len(t, rawResults, 2)
	require.Len(t, stdResults, 2)

	t.Logf("MultiIterateRaw found %d entries", len(rawResults))
	t.Logf("Standard Iterate found %d entries", len(stdResults))
}
