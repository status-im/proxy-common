# batch

Generic chunk processing utilities for handling large datasets with configurable limits and delays.

## Installation

```go
import "github.com/status-im/proxy-common/batch"
```

## Key Functions

- `ChunkMapFetcher[T any]` - Process items in chunks, returns merged map
- `ChunkArrayFetcher[T any]` - Process items in chunks, returns merged array

Both functions use Go generics for type safety.

## Quick Start

### ChunkMapFetcher

Process items in chunks and merge results into a map:

```go
import (
    "context"
    "fmt"
    "time"
    
    "github.com/status-im/proxy-common/batch"
)

ctx := context.Background()
items := []string{"id1", "id2", "id3", ..., "id100"}

// Fetch function that processes a chunk
fetchFunc := func(ctx context.Context, chunk []string) (map[string]interface{}, error) {
    results := make(map[string]interface{})
    for _, id := range chunk {
        // Fetch data for this ID
        data := fetchDataFromAPI(id)
        results[id] = data
    }
    return results, nil
}

// Process in chunks of 10 items, max 1000 chars per chunk, 100ms delay between chunks
result, err := batch.ChunkMapFetcher(
    ctx,
    items,
    10,              // chunk size limit
    1000,            // string length limit (0 to disable)
    100*time.Millisecond, // delay between chunks
    fetchFunc,
)

if err != nil {
    fmt.Println("Error:", err)
} else {
    fmt.Printf("Processed %d items\n", len(result))
}
```

### ChunkArrayFetcher

Process items in chunks and merge results into an array:

```go
fetchFunc := func(ctx context.Context, chunk []string) ([]User, error) {
    var users []User
    for _, id := range chunk {
        user := fetchUser(id)
        users = append(users, user)
    }
    return users, nil
}

users, err := batch.ChunkArrayFetcher(
    ctx,
    userIDs,
    20,              // chunk size limit
    0,               // no string length limit
    200*time.Millisecond, // delay between chunks
    fetchFunc,
)
```

## Parameters

- `ctx` - Context for cancellation
- `items` - Slice of items to process
- `chunkElementsLimit` - Maximum items per chunk
- `chunkStringLengthLimit` - Maximum total string length per chunk (0 to disable)
- `delay` - Delay between chunks (0 for no delay)
- `fetchFunc` - Function to process each chunk

## Use Cases

- Batch API requests to avoid rate limits
- Process large datasets in manageable chunks
- Add delays between requests to be nice to APIs
- Respect string length limits in API requests
