package multi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/cache/mock"
	"github.com/status-im/proxy-common/models"
)

func TestNewMultiCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	assert.NotNil(t, multiCache)
	mc := multiCache.(*MultiCache)
	assert.Equal(t, 2, len(mc.caches))
	assert.Equal(t, cache1, mc.caches[0])
	assert.Equal(t, cache2, mc.caches[1])
}

func TestMultiCache_Get_FirstCacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	expectedEntry := &models.CacheEntry{
		Data:      []byte("test-value"),
		CreatedAt: time.Now().Unix(),
		StaleAt:   time.Now().Unix() + 60,
		ExpiresAt: time.Now().Unix() + 120,
	}
	cache1.EXPECT().Get("test-key").Return(expectedEntry, true).Times(1)
	// cache2.Get should not be called since cache1 has the value

	entry, found := multiCache.Get("test-key")

	assert.True(t, found)
	assert.Equal(t, expectedEntry, entry)
}

func TestMultiCache_Get_SecondCacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	expectedEntry := &models.CacheEntry{
		Data:      []byte("test-value"),
		CreatedAt: time.Now().Unix(),
		StaleAt:   time.Now().Unix() + 60,
		ExpiresAt: time.Now().Unix() + 120,
	}

	cache1.EXPECT().Get("test-key").Return(nil, false).Times(1)
	cache2.EXPECT().Get("test-key").Return(expectedEntry, true).Times(1)
	// Expect propagation to cache1
	cache1.EXPECT().Set("test-key", expectedEntry.Data, gomock.Any()).Times(1)

	entry, found := multiCache.Get("test-key")

	assert.True(t, found)
	assert.Equal(t, expectedEntry, entry)
}

func TestMultiCache_Get_AllCachesMiss(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	cache1.EXPECT().Get("test-key").Return(nil, false).Times(1)
	cache2.EXPECT().Get("test-key").Return(nil, false).Times(1)

	entry, found := multiCache.Get("test-key")

	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestMultiCache_Get_NoCaches(t *testing.T) {
	multiCache := NewMultiCache([]cache.Cache{}, true)

	entry, found := multiCache.Get("test-key")

	assert.False(t, found)
	assert.Nil(t, entry)
}

func TestMultiCache_GetStale_FirstCacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	expectedEntry := &models.CacheEntry{
		Data:      []byte("test-value"),
		CreatedAt: time.Now().Unix(),
		StaleAt:   time.Now().Unix() - 30, // stale but not expired
		ExpiresAt: time.Now().Unix() + 60,
	}
	cache1.EXPECT().GetStale("test-key").Return(expectedEntry, true).Times(1)

	entry, found := multiCache.GetStale("test-key")

	assert.True(t, found)
	assert.Equal(t, expectedEntry, entry)
}

func TestMultiCache_GetStale_SecondCacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	expectedEntry := &models.CacheEntry{
		Data:      []byte("test-value"),
		CreatedAt: time.Now().Unix(),
		StaleAt:   time.Now().Unix() - 30, // stale but not expired
		ExpiresAt: time.Now().Unix() + 60,
	}

	cache1.EXPECT().GetStale("test-key").Return(nil, false).Times(1)
	cache2.EXPECT().GetStale("test-key").Return(expectedEntry, true).Times(1)
	// Expect propagation to cache1
	cache1.EXPECT().Set("test-key", expectedEntry.Data, gomock.Any()).Times(1)

	entry, found := multiCache.GetStale("test-key")

	assert.True(t, found)
	assert.Equal(t, expectedEntry, entry)
}

func TestMultiCache_Set_AllCaches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	testVal := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	cache1.EXPECT().Set("test-key", testVal, testTTL).Times(1)
	cache2.EXPECT().Set("test-key", testVal, testTTL).Times(1)

	multiCache.Set("test-key", testVal, testTTL)
}

func TestMultiCache_Set_NoCaches(t *testing.T) {
	multiCache := NewMultiCache([]cache.Cache{}, true)

	testVal := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Should not panic
	multiCache.Set("test-key", testVal, testTTL)
}

func TestMultiCache_Delete_AllCaches(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)

	cache1.EXPECT().Delete("test-key").Times(1)
	cache2.EXPECT().Delete("test-key").Times(1)

	multiCache.Delete("test-key")
}

func TestMultiCache_Delete_NoCaches(t *testing.T) {
	multiCache := NewMultiCache([]cache.Cache{}, true)

	// Should not panic
	multiCache.Delete("test-key")
}

func TestMultiCache_GetCacheCount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []cache.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, true)
	mc := multiCache.(*MultiCache)

	assert.Equal(t, 2, mc.GetCacheCount())
}
