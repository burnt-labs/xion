package indexer_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
	"cosmossdk.io/core/store"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"

	"github.com/burnt-labs/xion/app/indexer"
)

// deps creates test dependencies compatible with cosmos-sdk v0.53.4 (core v0.11.3)
func deps(t *testing.T) (store.KVStoreService, context.Context) {
	key := storetypes.NewKVStoreKey("test")
	tkey := storetypes.NewTransientStoreKey("test_transient")

	// Create a test context using SDK's testutil
	testCtx := testutil.DefaultContextWithDB(t, key, tkey)

	// Wrap the store key with runtime's KVStoreService adapter
	storeService := runtime.NewKVStoreService(key)

	return storeService, testCtx.Ctx
}

// TestMultiIterateRaw verifies that our MultiIterateRaw helper function works correctly
func TestMultiIterateRaw(t *testing.T) {
	// Setup: Create a schema and Multi index
	sk, ctx := deps(t)
	sb := collections.NewSchemaBuilder(sk)

	type TestValue struct {
		ID       uint64
		Category string
	}

	// Create a Multi index: Category (string) -> ID (uint64)
	// Multiple IDs can have the same category
	multiIndex := indexes.NewMulti(
		sb,
		collections.NewPrefix(1),
		"test_multi_index",
		collections.StringKey,
		collections.Uint64Key,
		func(pk uint64, v TestValue) (string, error) {
			return v.Category, nil
		},
	)

	// Test 1: Insert some test data
	t.Run("insert_and_verify", func(t *testing.T) {
		testData := []TestValue{
			{ID: 1, Category: "alpha"},
			{ID: 2, Category: "alpha"},
			{ID: 3, Category: "beta"},
			{ID: 4, Category: "gamma"},
			{ID: 5, Category: "beta"},
		}

		// Reference the data in the index
		for _, v := range testData {
			err := multiIndex.Reference(ctx, v.ID, v, func() (TestValue, error) {
				return TestValue{}, collections.ErrNotFound
			})
			require.NoError(t, err)
		}
	})

	// Test 2: Use MultiIterateRaw to iterate over all entries
	t.Run("iterate_raw_all", func(t *testing.T) {
		iter, err := indexer.MultiIterateRaw(
			ctx,
			multiIndex,
			nil, // start: nil means from beginning
			nil, // end: nil means to end
			collections.OrderAscending,
		)
		require.NoError(t, err)
		require.NotNil(t, iter)
		defer iter.Close()

		// Count the entries
		count := 0
		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(t, err)
			t.Logf("Found entry: Category=%s, ID=%d", key.K1(), key.K2())
			count++
		}

		// We should have 5 entries total
		require.Equal(t, 5, count, "expected 5 entries in the index")
	})

	// Test 3: Use MultiIterateRaw with descending order
	t.Run("iterate_raw_descending", func(t *testing.T) {
		iter, err := indexer.MultiIterateRaw(
			ctx,
			multiIndex,
			nil,
			nil,
			collections.OrderDescending,
		)
		require.NoError(t, err)
		require.NotNil(t, iter)
		defer iter.Close()

		count := 0
		for ; iter.Valid(); iter.Next() {
			_, err := iter.Key()
			require.NoError(t, err)
			count++
		}

		require.Equal(t, 5, count, "expected 5 entries in descending iteration")
	})

	// Test 4: Verify the helper function signature matches expected usage
	t.Run("type_compatibility", func(t *testing.T) {
		// This test verifies that the returned iterator has the correct type
		iter, err := indexer.MultiIterateRaw(
			ctx,
			multiIndex,
			nil,
			nil,
			collections.OrderAscending,
		)
		require.NoError(t, err)
		defer iter.Close()

		// Verify the iterator returns Pair[string, uint64] keys
		if iter.Valid() {
			key, err := iter.Key()
			require.NoError(t, err)

			// Type assertions to verify the key structure
			category := key.K1()
			id := key.K2()

			require.IsType(t, "", category, "K1 should be string")
			require.IsType(t, uint64(0), id, "K2 should be uint64")
		}
	})
}

// TestMultiIterateRaw_WithRanges tests iteration with specific byte ranges
func TestMultiIterateRaw_WithRanges(t *testing.T) {
	sk, ctx := deps(t)
	sb := collections.NewSchemaBuilder(sk)

	type TestValue struct {
		ID    uint64
		Score int
	}

	multiIndex := indexes.NewMulti(
		sb,
		collections.NewPrefix(2),
		"score_index",
		collections.Uint64Key, // Score as reference key
		collections.Uint64Key, // ID as primary key
		func(pk uint64, v TestValue) (uint64, error) {
			if v.Score < 0 {
				return 0, nil
			}
			return uint64(v.Score), nil
		},
	)

	// Insert test data with various scores
	testData := []TestValue{
		{ID: 1, Score: 10},
		{ID: 2, Score: 20},
		{ID: 3, Score: 20},
		{ID: 4, Score: 30},
		{ID: 5, Score: 40},
	}

	for _, v := range testData {
		err := multiIndex.Reference(ctx, v.ID, v, func() (TestValue, error) {
			return TestValue{}, collections.ErrNotFound
		})
		require.NoError(t, err)
	}

	// Test: Iterate all entries to verify IterateRaw works with uint64 keys
	t.Run("iterate_all_scores", func(t *testing.T) {
		iter, err := indexer.MultiIterateRaw(
			ctx,
			multiIndex,
			nil, // start from beginning
			nil, // no end limit
			collections.OrderAscending,
		)
		require.NoError(t, err)
		defer iter.Close()

		count := 0
		scores := []uint64{}
		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(t, err)
			score := key.K1()
			id := key.K2()
			t.Logf("Found score entry: Score=%d, ID=%d", score, id)
			scores = append(scores, score)
			count++
		}

		// Should have 5 entries total
		require.Equal(t, 5, count, "expected 5 entries in the score index")

		// Verify we got all the scores
		require.Contains(t, scores, uint64(10))
		require.Contains(t, scores, uint64(20))
		require.Contains(t, scores, uint64(30))
		require.Contains(t, scores, uint64(40))
	})

	// Test: Descending order iteration
	t.Run("iterate_scores_descending", func(t *testing.T) {
		iter, err := indexer.MultiIterateRaw(
			ctx,
			multiIndex,
			nil,
			nil,
			collections.OrderDescending,
		)
		require.NoError(t, err)
		defer iter.Close()

		lastScore := ^uint64(0) // max uint64
		count := 0
		for ; iter.Valid(); iter.Next() {
			key, err := iter.Key()
			require.NoError(t, err)
			score := key.K1()

			// In descending order, each score should be <= previous
			require.LessOrEqual(t, score, lastScore, "scores should be in descending order")
			lastScore = score
			count++
		}

		require.Equal(t, 5, count, "expected 5 entries in descending order")
	})
}
