package apikeys

import (
	"testing"
	"time"
)

// MockKeyProvider implements KeyProvider for testing
type MockKeyProvider struct {
	keys map[KeyType][]string
}

func NewMockKeyProvider() *MockKeyProvider {
	return &MockKeyProvider{
		keys: make(map[KeyType][]string),
	}
}

func (m *MockKeyProvider) SetKeys(keyType KeyType, keys []string) {
	m.keys[keyType] = keys
}

func (m *MockKeyProvider) GetKeys(keyType KeyType) []string {
	if keys, ok := m.keys[keyType]; ok {
		return keys
	}
	return []string{}
}

// Helper function for tests to check if a key is in the list of available keys
func containsKey(keys []APIKey, key string, keyType KeyType) bool {
	for _, k := range keys {
		if k.Key == key && k.Type == keyType {
			return true
		}
	}
	return false
}

func TestAPIKeyManager_GetAvailableKeys(t *testing.T) {
	// Create mock key provider
	const (
		ProKey  KeyType = 1
		DemoKey KeyType = 2
		NoKey   KeyType = 0
	)

	provider := NewMockKeyProvider()
	provider.SetKeys(ProKey, []string{"pro1", "pro2"})
	provider.SetKeys(DemoKey, []string{"demo1", "demo2", "demo3"})

	// Create API key manager
	manager := NewAPIKeyManager(provider, []KeyType{ProKey, DemoKey, NoKey}, 5*time.Minute)

	// Initially all keys should be available
	availableKeys := manager.GetAvailableKeys()

	// We expect 5 available keys (2 pro, 3 demo)
	if len(availableKeys) != 5 {
		t.Errorf("Expected 5 available keys, got %d", len(availableKeys))
	}

	// Verify pro and demo keys are present
	if !containsKey(availableKeys, "pro1", ProKey) {
		t.Errorf("Expected pro1 to be available")
	}
	if !containsKey(availableKeys, "pro2", ProKey) {
		t.Errorf("Expected pro2 to be available")
	}
	if !containsKey(availableKeys, "demo1", DemoKey) {
		t.Errorf("Expected demo1 to be available")
	}

	// Mark one key as failed
	manager.MarkKeyAsFailed("pro1")

	// Now we should have one less available pro key
	availableKeys = manager.GetAvailableKeys()
	if len(availableKeys) != 4 {
		t.Errorf("Expected 4 available keys after marking one as failed, got %d", len(availableKeys))
	}

	// pro1 should no longer be in the list
	if containsKey(availableKeys, "pro1", ProKey) {
		t.Errorf("Expected pro1 to not be available after marking as failed")
	}

	// Test the special case: single Pro key should always be available
	singleProProvider := NewMockKeyProvider()
	singleProProvider.SetKeys(ProKey, []string{"solo-pro"})
	singleProProvider.SetKeys(DemoKey, []string{"demo1", "demo2"})

	singleProManager := NewAPIKeyManager(singleProProvider, []KeyType{ProKey, DemoKey}, 5*time.Minute)

	// Mark the pro key as failed
	singleProManager.MarkKeyAsFailed("solo-pro")

	// The pro key should still be available
	singleProKeys := singleProManager.GetAvailableKeys()
	if !containsKey(singleProKeys, "solo-pro", ProKey) {
		t.Errorf("Expected solo pro key to be available even when in backoff")
	}
}

func TestAPIKeyManager_MarkKeyAsFailed(t *testing.T) {
	const ProKey KeyType = 1

	// Create mock key provider
	provider := NewMockKeyProvider()
	provider.SetKeys(ProKey, []string{"pro1", "pro2", "pro3", "pro4"})

	// Create API key manager with a shorter backoff for testing
	manager := NewAPIKeyManager(provider, []KeyType{ProKey}, 100*time.Millisecond)

	// Get initial available keys
	initialKeys := manager.GetAvailableKeys()
	initialProCount := 0
	for _, key := range initialKeys {
		if key.Type == ProKey {
			initialProCount++
		}
	}

	// Mark key as failed
	manager.MarkKeyAsFailed("pro1")

	// Get available keys after marking one as failed
	afterFailKeys := manager.GetAvailableKeys()
	afterFailProCount := 0
	for _, key := range afterFailKeys {
		if key.Type == ProKey {
			afterFailProCount++
		}
	}

	// We should have one less Pro key
	if afterFailProCount != (initialProCount - 1) {
		t.Errorf("Expected %d Pro keys after marking one as failed, got %d", initialProCount-1, afterFailProCount)
	}

	// Wait for backoff to expire
	time.Sleep(150 * time.Millisecond)

	// Get available keys after backoff expired
	afterBackoffKeys := manager.GetAvailableKeys()
	afterBackoffProCount := 0
	for _, key := range afterBackoffKeys {
		if key.Type == ProKey {
			afterBackoffProCount++
		}
	}

	// We should have same number of Pro keys as initially
	if afterBackoffProCount != initialProCount {
		t.Errorf("Expected %d Pro keys after backoff expired, got %d", initialProCount, afterBackoffProCount)
	}
}

func TestTryWithKeys(t *testing.T) {
	const ProKey KeyType = 1

	keys := []APIKey{
		{Key: "key1", Type: ProKey},
		{Key: "key2", Type: ProKey},
		{Key: "key3", Type: ProKey},
	}

	t.Run("Success on first try", func(t *testing.T) {
		executor := func(apiKey APIKey) (interface{}, bool, error) {
			return "success", true, nil
		}

		result, err := TryWithKeys(keys, "test", executor, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != "success" {
			t.Errorf("Expected 'success', got: %v", result)
		}
	})

	t.Run("Success on second try", func(t *testing.T) {
		attempts := 0
		executor := func(apiKey APIKey) (interface{}, bool, error) {
			attempts++
			if attempts == 1 {
				return nil, false, nil
			}
			return "success", true, nil
		}

		result, err := TryWithKeys(keys, "test", executor, nil)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result != "success" {
			t.Errorf("Expected 'success', got: %v", result)
		}
		if attempts != 2 {
			t.Errorf("Expected 2 attempts, got: %d", attempts)
		}
	})

	t.Run("All keys fail", func(t *testing.T) {
		executor := func(apiKey APIKey) (interface{}, bool, error) {
			return nil, false, nil
		}

		result, err := TryWithKeys(keys, "test", executor, nil)
		if err == nil {
			t.Error("Expected error when all keys fail")
		}
		if result != nil {
			t.Errorf("Expected nil result, got: %v", result)
		}
	})
}
