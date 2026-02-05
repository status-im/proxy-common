package l2

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/status-im/proxy-common/cache"
	"github.com/status-im/proxy-common/cache/mock"
	"github.com/status-im/proxy-common/models"
)

func TestNewKeyDBCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient)

	assert.NotNil(t, c)
	keydbCache, ok := c.(*KeyDBCache)
	assert.True(t, ok)
	assert.Equal(t, mockClient, keydbCache.client)
	assert.Equal(t, cfg, keydbCache.cfg)
}

func TestKeyDBCache_Get_Success_Fresh(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 100,
		StaleAt:   now + 100, // Fresh
		ExpiresAt: now + 200,
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := c.Get("test-key")

	// Assert
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.True(t, result.IsFresh())
	assert.Equal(t, []byte("test-data"), result.Data)
}

func TestKeyDBCache_Get_Success_Stale(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 200,
		StaleAt:   now - 50, // Stale but not expired
		ExpiresAt: now + 100,
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := c.Get("test-key")

	// Assert
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.False(t, result.IsFresh())
	assert.Equal(t, []byte("test-data"), result.Data)
}

func TestKeyDBCache_Get_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", redis.Nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := c.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Get_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", errors.New("connection error"))
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := c.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Get_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	result, found := c.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Get_CorruptedEntry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations - return invalid JSON
	stringCmd := redis.NewStringResult("invalid-json", nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	result, found := c.Get("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_GetStale_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 200,
		StaleAt:   now - 50, // Stale but not expired
		ExpiresAt: now + 100,
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := c.GetStale("test-key")

	// Assert
	assert.True(t, found)
	assert.NotNil(t, result)
	assert.Equal(t, []byte("test-data"), result.Data)
}

func TestKeyDBCache_GetStale_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations
	stringCmd := redis.NewStringResult("", redis.Nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	// Execute
	result, found := c.GetStale("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_GetStale_Expired(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Prepare test data
	now := time.Now().Unix()
	entry := models.CacheEntry{
		Data:      []byte("test-data"),
		CreatedAt: now - 300,
		StaleAt:   now - 200,
		ExpiresAt: now - 100, // Expired
	}
	entryJSON, _ := json.Marshal(entry)

	// Mock expectations
	stringCmd := redis.NewStringResult(string(entryJSON), nil)
	mockClient.EXPECT().Get(gomock.Any(), "test-key").Return(stringCmd)

	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	result, found := c.GetStale("test-key")

	// Assert
	assert.False(t, found)
	assert.Nil(t, result)
}

func TestKeyDBCache_Set_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	ttl := models.TTL{
		Fresh: 60 * time.Second,
		Stale: 30 * time.Second,
	}

	// Mock expectations
	statusCmd := redis.NewStatusResult("OK", nil)
	mockClient.EXPECT().Set(gomock.Any(), "test-key", gomock.Any(), 90*time.Second).Return(statusCmd)

	// Execute
	c.Set("test-key", []byte("test-data"), ttl)

	// No assertions needed as Set doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Set_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	ttl := models.TTL{
		Fresh: 60 * time.Second,
		Stale: 30 * time.Second,
	}

	// Mock expectations
	statusCmd := redis.NewStatusResult("", errors.New("set error"))
	mockClient.EXPECT().Set(gomock.Any(), "test-key", gomock.Any(), 90*time.Second).Return(statusCmd)

	// Execute
	c.Set("test-key", []byte("test-data"), ttl)

	// No assertions needed as Set doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Delete_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations
	intCmd := redis.NewIntResult(1, nil)
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	c.Delete("test-key")

	// No assertions needed as Delete doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Delete_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations
	intCmd := redis.NewIntResult(0, errors.New("delete error"))
	mockClient.EXPECT().Del(gomock.Any(), "test-key").Return(intCmd)

	// Execute
	c.Delete("test-key")

	// No assertions needed as Delete doesn't return anything
	// The test passes if no panic occurs and mock expectations are met
}

func TestKeyDBCache_Close(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	// Mock expectations
	mockClient.EXPECT().Close().Return(nil)

	// Execute
	err := c.Close()

	// Assert
	assert.NoError(t, err)
}

func TestKeyDBCache_Close_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mock.NewMockKeyDbClient(ctrl)
	cfg := &cache.KeyDBConfig{}

	c := NewKeyDBCache(cfg, mockClient).(*KeyDBCache)

	expectedErr := errors.New("close error")

	// Mock expectations
	mockClient.EXPECT().Close().Return(expectedErr)

	// Execute
	err := c.Close()

	// Assert
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
}
