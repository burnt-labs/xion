# Index Utilities

This package provides utility functions for working with Cosmos SDK collections indexes.

## MultiIterateRaw

### Background

The `cosmossdk.io/collections@v1.3.1` package has a missing method on the `Multi` index type. While other index types (`Unique`, `ReversePair`) provide an `IterateRaw()` method for raw byte-range iteration, the `Multi` index does not expose this functionality, even though the underlying `KeySet` supports it.

### The Problem

```go
// This works for Unique and ReversePair indexes:
iter, err := uniqueIndex.IterateRaw(ctx, startBytes, endBytes, order)

// But this does NOT work for Multi indexes in v1.3.1:
iter, err := multiIndex.IterateRaw(ctx, startBytes, endBytes, order)
// ❌ Error: multiIndex.IterateRaw undefined
```

### The Solution

This package provides `MultiIterateRaw()` as a workaround:

```go
import "github.com/burnt-labs/xion/app/indexer"

// Use the helper function:
iter, err := indexer.MultiIterateRaw(ctx, multiIndex, startBytes, endBytes, order)
// ✅ Works!
```

### Usage Example

```go
package mymodule

import (
    "context"

    "cosmossdk.io/collections"
    "cosmossdk.io/collections/indexes"
    "github.com/burnt-labs/xion/app/indexer"
)

func (k Keeper) IterateUsersByCategory(ctx context.Context) error {
    // Assuming you have a Multi index: Category (string) -> UserID (uint64)
    multiIndex := k.usersByCategory

    // Iterate over all entries
    iter, err := indexer.MultiIterateRaw(
        ctx,
        multiIndex,
        nil, // start: nil means from beginning
        nil, // end: nil means to end
        collections.OrderAscending,
    )
    if err != nil {
        return err
    }
    defer iter.Close()

    // Process entries
    for ; iter.Valid(); iter.Next() {
        key, err := iter.Key()
        if err != nil {
            return err
        }

        category := key.K1()  // ReferenceKey
        userID := key.K2()    // PrimaryKey

        // Your business logic here
        k.processUser(ctx, category, userID)
    }

    return nil
}
```

### Implementation Details

The helper function uses reflection to access the private `refKeys` field of the `Multi` struct and calls its public `IterateRaw()` method. This is safe because:

1. We're only calling a public method on the accessed field
2. No state is modified
3. Type safety is preserved
4. Performance impact is negligible (reflection happens once per iteration setup)

### Testing

Comprehensive tests are available in `multi_raw_test.go`. Run them with:

```bash
go test -v ./app/indexer -run TestMultiIterateRaw
```

Tests verify:

- Basic iteration over Multi indexes
- Ascending and descending order
- Type compatibility with different key types (string, uint64)
- Proper iterator lifecycle (Valid, Next, Close)
