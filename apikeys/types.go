package apikeys

// KeyType defines the API key type
type KeyType int

// APIKey represents an API key with its type
type APIKey struct {
	Key   string
	Type  KeyType
	Label string // optional description
}

// KeyProvider defines an interface for providing API keys
type KeyProvider interface {
	// GetKeys returns a list of keys for a specific type
	GetKeys(keyType KeyType) []string
}
