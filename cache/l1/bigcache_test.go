package l1

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/models"
)

// Helper function to create test BigCache config
func createTestBigCacheConfig() *cache.BigCacheConfig {
	return &cache.BigCacheConfig{
		Enabled: true,
		Size:    10,
	}
}

func TestNewBigCache(t *testing.T) {
	cfg := createTestBigCacheConfig()

	c, err := NewBigCache(cfg)

	assert.NoError(t, err)
	assert.NotNil(t, c)

	bigCache, ok := c.(*BigCache)
	assert.True(t, ok)
	assert.NotNil(t, bigCache.cache)
}

func TestBigCache_Set_And_Get_Fresh(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	testData := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set the value
	c.Set("test-key", testData, testTTL)

	// Get the value immediately (should be fresh)
	result, found := c.Get("test-key")

	assert.True(t, found)
	assert.NotNil(t, result)
	assert.True(t, result.IsFresh())
	assert.Equal(t, testData, result.Data)
}

func TestBigCache_Get_NotFound(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Try to get non-existent key
	result, found := c.Get("non-existent-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_Set_And_Get_Stale(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Create a cache entry that's already stale
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create a stale entry by setting timestamps in the past
	bigCache := c.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 200,
		StaleAt:   now - 50,  // Already stale
		ExpiresAt: now + 100, // Not expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	_ = bigCache.cache.Set("test-key", entryJSON)

	// Get the value (should be stale but not expired)
	result, found := c.Get("test-key")

	assert.True(t, found)
	assert.NotNil(t, result)
	assert.False(t, result.IsFresh())
	assert.Equal(t, testData, result.Data)
}

func TestBigCache_Set_And_Get_Expired(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Create a cache entry that's already expired
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create an expired entry
	bigCache := c.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Already expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	_ = bigCache.cache.Set("test-key", entryJSON)

	// Get the value (should be expired and not found)
	result, found := c.Get("test-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_GetStale_Success(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Create a cache entry that's stale but not expired
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create a stale entry
	bigCache := c.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 200,
		StaleAt:   now - 50,  // Already stale
		ExpiresAt: now + 100, // Not expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	_ = bigCache.cache.Set("test-key", entryJSON)

	// Get stale value
	result, found := c.GetStale("test-key")

	assert.True(t, found)
	assert.NotNil(t, result)
	assert.Equal(t, testData, result.Data)
}

func TestBigCache_GetStale_NotFound(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Try to get stale value for non-existent key
	result, found := c.GetStale("non-existent-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_GetStale_Expired(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Create a cache entry that's completely expired
	now := time.Now().Unix()
	testData := []byte("test-value")

	// Manually create an expired entry
	bigCache := c.(*BigCache)
	entry := models.CacheEntry{
		Data:      testData,
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Already expired
	}

	// Manually marshal and set the entry
	entryJSON, _ := json.Marshal(entry)
	_ = bigCache.cache.Set("test-key", entryJSON)

	// Try to get stale value (should be expired)
	result, found := c.GetStale("test-key")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_Delete(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	testData := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set the value
	c.Set("test-key", testData, testTTL)

	// Verify it exists
	result, found := c.Get("test-key")
	assert.True(t, found)
	assert.NotNil(t, result)

	// Delete it
	c.Delete("test-key")

	// Verify it's gone
	result, found = c.Get("test-key")
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestBigCache_Delete_NonExistent(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	// Delete non-existent key (should not panic)
	c.Delete("non-existent-key")
}

func TestBigCache_Multiple_Keys(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Set multiple keys
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := []byte(fmt.Sprintf("value-%d", i))
		c.Set(key, value, testTTL)
	}

	// Verify all keys exist
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		expectedValue := []byte(fmt.Sprintf("value-%d", i))

		result, found := c.Get(key)
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, expectedValue, result.Data)
	}
}

func TestBigCache_Concurrent_Access(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}
	numGoroutines := 10
	numOperations := 100

	// Run concurrent operations
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("concurrent-key-%d-%d", id, j)
				value := []byte(fmt.Sprintf("value-%d-%d", id, j))

				// Set
				c.Set(key, value, testTTL)

				// Get
				result, found := c.Get(key)
				if found {
					assert.NotNil(t, result)
					assert.Equal(t, value, result.Data)
				}

				// Delete
				c.Delete(key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestBigCache_Edge_Cases(t *testing.T) {
	c, err := NewBigCache(createTestBigCacheConfig())
	assert.NoError(t, err)

	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	t.Run("empty key", func(t *testing.T) {
		c.Set("", []byte("value"), testTTL)
		result, found := c.Get("")
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, []byte("value"), result.Data)
	})

	t.Run("empty value", func(t *testing.T) {
		c.Set("empty-value-key", []byte(""), testTTL)
		result, found := c.Get("empty-value-key")
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Equal(t, []byte(""), result.Data)
	})

	t.Run("nil value", func(t *testing.T) {
		c.Set("nil-value-key", nil, testTTL)
		result, found := c.Get("nil-value-key")
		assert.True(t, found)
		assert.NotNil(t, result)
		assert.Nil(t, result.Data)
	})
}
