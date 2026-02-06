package apikeys

import (
	"log"
	"sync"
	"time"
)

// IAPIKeyManager defines the interface for API key management
type IAPIKeyManager interface {
	// GetAvailableKeys returns a list of available API keys
	GetAvailableKeys() []APIKey

	// MarkKeyAsFailed marks a key as failed, which will put it in backoff
	MarkKeyAsFailed(key string)
}

// APIKeyManager implements IAPIKeyManager with backoff support
type APIKeyManager struct {
	provider    KeyProvider
	keyTypes    []KeyType            // ordered by priority
	lastFailed  map[string]time.Time // Stores the time of the last failure for each key
	backoffTime time.Duration        // Backoff duration before retrying a failed key
	mu          sync.RWMutex
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(provider KeyProvider, keyTypes []KeyType, backoff time.Duration) *APIKeyManager {
	if backoff == 0 {
		backoff = 5 * time.Minute
	}
	return &APIKeyManager{
		provider:    provider,
		keyTypes:    keyTypes,
		lastFailed:  make(map[string]time.Time),
		backoffTime: backoff,
	}
}

// isKeyInBackoff checks if a key is currently in backoff period (private implementation)
func (m *APIKeyManager) isKeyInBackoff(key string) bool {
	if key == "" {
		return false
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if lastFailTime, exists := m.lastFailed[key]; exists {
		return time.Since(lastFailTime) < m.backoffTime
	}

	return false
}

// GetAvailableKeys returns a list of available API keys based on priority and backoff status
func (m *APIKeyManager) GetAvailableKeys() []APIKey {
	var availableKeys []APIKey

	for _, keyType := range m.keyTypes {
		keys := m.provider.GetKeys(keyType)

		// If there's exactly one key of this type, include it even if it's in backoff
		if len(keys) == 1 {
			availableKeys = append(availableKeys, APIKey{Key: keys[0], Type: keyType})
		} else if len(keys) > 1 {
			// For multiple keys, include only those not in backoff
			for _, key := range keys {
				if !m.isKeyInBackoff(key) {
					availableKeys = append(availableKeys, APIKey{Key: key, Type: keyType})
				}
			}
		}
	}

	return availableKeys
}

// MarkKeyAsFailed marks a key as non-working for some time
func (m *APIKeyManager) MarkKeyAsFailed(key string) {
	if key == "" {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.lastFailed[key] = time.Now()
	log.Printf("APIKeyManager: Marked key as failed for %v", m.backoffTime)
}
