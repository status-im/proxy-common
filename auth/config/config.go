package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/status-im/proxy-common/auth/puzzle"
)

type Config struct {
	Algorithm          string              `json:"algorithm"`
	JWTSecret          string              `json:"jwt_secret"`
	PuzzleDifficulty   int                 `json:"puzzle_difficulty"`
	RequestsPerToken   int                 `json:"requests_per_token"`
	TokenExpiryMinutes int                 `json:"token_expiry_minutes"`
	Argon2Params       puzzle.Argon2Config `json:"argon2_params"`
}

type Option func(*Config)

func New(opts ...Option) *Config {
	cfg := &Config{
		Algorithm:          "argon2id",
		PuzzleDifficulty:   1,
		RequestsPerToken:   100,
		TokenExpiryMinutes: 10,
		Argon2Params: puzzle.Argon2Config{
			MemoryKB: 16384,
			Time:     4,
			Threads:  4,
			KeyLen:   32,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func WithJWTSecret(secret string) Option {
	return func(c *Config) {
		c.JWTSecret = secret
	}
}

func WithAlgorithm(algorithm string) Option {
	return func(c *Config) {
		c.Algorithm = algorithm
	}
}

func WithDifficulty(difficulty int) Option {
	return func(c *Config) {
		c.PuzzleDifficulty = difficulty
	}
}

func WithRequestsPerToken(requests int) Option {
	return func(c *Config) {
		c.RequestsPerToken = requests
	}
}

func WithTokenExpiry(minutes int) Option {
	return func(c *Config) {
		c.TokenExpiryMinutes = minutes
	}
}

func WithArgon2Params(params puzzle.Argon2Config) Option {
	return func(c *Config) {
		c.Argon2Params = params
	}
}

func WithArgon2Memory(memoryKB int) Option {
	return func(c *Config) {
		c.Argon2Params.MemoryKB = memoryKB
	}
}

func WithArgon2Time(time int) Option {
	return func(c *Config) {
		c.Argon2Params.Time = time
	}
}

func WithArgon2Threads(threads int) Option {
	return func(c *Config) {
		c.Argon2Params.Threads = threads
	}
}

func WithArgon2KeyLen(keyLen int) Option {
	return func(c *Config) {
		c.Argon2Params.KeyLen = keyLen
	}
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Load loads configuration from the CONFIG_FILE environment variable
// or from the default path "auth_config.json"
func Load() (*Config, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		configFile = "auth_config.json"
	}
	return LoadFromFile(configFile)
}

// LoadFromEnv loads configuration from environment variables
// This provides a way to configure without a config file
func LoadFromEnv() (*Config, error) {
	cfg := New()

	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWTSecret = secret
	} else {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	if algorithm := os.Getenv("ALGORITHM"); algorithm != "" {
		cfg.Algorithm = algorithm
	}

	if diffStr := os.Getenv("PUZZLE_DIFFICULTY"); diffStr != "" {
		if diff, err := strconv.Atoi(diffStr); err == nil {
			cfg.PuzzleDifficulty = diff
		}
	}

	if reqStr := os.Getenv("REQUESTS_PER_TOKEN"); reqStr != "" {
		if req, err := strconv.Atoi(reqStr); err == nil {
			cfg.RequestsPerToken = req
		}
	}

	if expStr := os.Getenv("TOKEN_EXPIRY_MINUTES"); expStr != "" {
		if exp, err := strconv.Atoi(expStr); err == nil {
			cfg.TokenExpiryMinutes = exp
		}
	}

	if memStr := os.Getenv("ARGON2_MEMORY_KB"); memStr != "" {
		if mem, err := strconv.Atoi(memStr); err == nil {
			cfg.Argon2Params.MemoryKB = mem
		}
	}

	if timeStr := os.Getenv("ARGON2_TIME"); timeStr != "" {
		if time, err := strconv.Atoi(timeStr); err == nil {
			cfg.Argon2Params.Time = time
		}
	}

	if threadsStr := os.Getenv("ARGON2_THREADS"); threadsStr != "" {
		if threads, err := strconv.Atoi(threadsStr); err == nil {
			cfg.Argon2Params.Threads = threads
		}
	}

	if keyLenStr := os.Getenv("ARGON2_KEY_LEN"); keyLenStr != "" {
		if keyLen, err := strconv.Atoi(keyLenStr); err == nil {
			cfg.Argon2Params.KeyLen = keyLen
		}
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT secret is required")
	}

	if c.PuzzleDifficulty < 0 {
		return fmt.Errorf("puzzle difficulty must be non-negative")
	}

	if c.RequestsPerToken <= 0 {
		return fmt.Errorf("requests per token must be positive")
	}

	if c.TokenExpiryMinutes <= 0 {
		return fmt.Errorf("token expiry must be positive")
	}

	if c.Argon2Params.MemoryKB <= 0 {
		return fmt.Errorf("Argon2 memory must be positive")
	}

	if c.Argon2Params.Time <= 0 {
		return fmt.Errorf("Argon2 time must be positive")
	}

	if c.Argon2Params.Threads <= 0 {
		return fmt.Errorf("Argon2 threads must be positive")
	}

	if c.Argon2Params.KeyLen <= 0 {
		return fmt.Errorf("Argon2 key length must be positive")
	}

	return nil
}
