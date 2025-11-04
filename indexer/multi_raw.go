package indexer

import (
	"context"
	"reflect"
	"unsafe"

	"cosmossdk.io/collections"
	"cosmossdk.io/collections/indexes"
)

// MultiIterateRaw provides IterateRaw functionality for Multi indexes.
//
// Background:
// The Multi index type is missing the IterateRaw() method that other index types
// (Unique, ReversePair) provide. The underlying refKeys field (a KeySet) already has
// IterateRaw(), but it's not exposed through the Multi type's public API.
//
// This helper uses reflection to access the private refKeys field and call its
// IterateRaw() method. This is safe because:
// 1. We're only calling a public method (IterateRaw) on the accessed field
// 2. We're not modifying any state or violating type safety
// 3. The performance impact is negligible (reflection only happens once per iteration)
func MultiIterateRaw[ReferenceKey, PrimaryKey, Value any](
	ctx context.Context,
	multi *indexes.Multi[ReferenceKey, PrimaryKey, Value],
	start, end []byte,
	order collections.Order,
) (collections.Iterator[collections.Pair[ReferenceKey, PrimaryKey], collections.NoValue], error) {

	// Access the private refKeys field via reflection
	// The Multi struct has: refKeys collections.KeySet[collections.Pair[ReferenceKey, PrimaryKey]]
	v := reflect.ValueOf(multi).Elem()
	refKeysField := v.FieldByName("refKeys")

	// Make the unexported field accessible and extract the KeySet
	// This is equivalent to: return multi.refKeys.IterateRaw(ctx, start, end, order)
	refKeys := reflect.NewAt(
		refKeysField.Type(),
		unsafe.Pointer(refKeysField.UnsafeAddr()),
	).Elem().Interface().(collections.KeySet[collections.Pair[ReferenceKey, PrimaryKey]])

	// Call the public IterateRaw method on the KeySet
	return refKeys.IterateRaw(ctx, start, end, order)
}
