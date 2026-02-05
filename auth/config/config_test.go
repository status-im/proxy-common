package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/status-im/proxy-common/auth/puzzle"
)

func TestNew(t *testing.T) {
	cfg := New()

	if cfg.Algorithm != "argon2id" {
		t.Errorf("expected algorithm argon2id, got %s", cfg.Algorithm)
	}
	if cfg.PuzzleDifficulty != 1 {
		t.Errorf("expected puzzle difficulty 1, got %d", cfg.PuzzleDifficulty)
	}
	if cfg.RequestsPerToken != 100 {
		t.Errorf("expected requests per token 100, got %d", cfg.RequestsPerToken)
	}
	if cfg.TokenExpiryMinutes != 10 {
		t.Errorf("expected token expiry 10, got %d", cfg.TokenExpiryMinutes)
	}
	if cfg.Argon2Params.MemoryKB != 16384 {
		t.Errorf("expected argon2 memory 16384, got %d", cfg.Argon2Params.MemoryKB)
	}
}

func TestWithOptions(t *testing.T) {
	cfg := New(
		WithJWTSecret("test-secret"),
		WithAlgorithm("argon2id"),
		WithDifficulty(3),
		WithRequestsPerToken(50),
		WithTokenExpiry(20),
		WithArgon2Memory(8192),
		WithArgon2Time(2),
		WithArgon2Threads(2),
		WithArgon2KeyLen(16),
	)

	if cfg.JWTSecret != "test-secret" {
		t.Errorf("expected jwt secret test-secret, got %s", cfg.JWTSecret)
	}
	if cfg.PuzzleDifficulty != 3 {
		t.Errorf("expected puzzle difficulty 3, got %d", cfg.PuzzleDifficulty)
	}
	if cfg.RequestsPerToken != 50 {
		t.Errorf("expected requests per token 50, got %d", cfg.RequestsPerToken)
	}
	if cfg.TokenExpiryMinutes != 20 {
		t.Errorf("expected token expiry 20, got %d", cfg.TokenExpiryMinutes)
	}
	if cfg.Argon2Params.MemoryKB != 8192 {
		t.Errorf("expected argon2 memory 8192, got %d", cfg.Argon2Params.MemoryKB)
	}
	if cfg.Argon2Params.Time != 2 {
		t.Errorf("expected argon2 time 2, got %d", cfg.Argon2Params.Time)
	}
	if cfg.Argon2Params.Threads != 2 {
		t.Errorf("expected argon2 threads 2, got %d", cfg.Argon2Params.Threads)
	}
	if cfg.Argon2Params.KeyLen != 16 {
		t.Errorf("expected argon2 key len 16, got %d", cfg.Argon2Params.KeyLen)
	}
}

func TestWithArgon2Params(t *testing.T) {
	params := puzzle.Argon2Config{
		MemoryKB: 4096,
		Time:     1,
		Threads:  1,
		KeyLen:   64,
	}
	cfg := New(WithArgon2Params(params))

	if cfg.Argon2Params != params {
		t.Errorf("expected argon2 params to match")
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	configContent := `{
		"algorithm": "argon2id",
		"jwt_secret": "test-secret-123",
		"puzzle_difficulty": 2,
		"requests_per_token": 200,
		"token_expiry_minutes": 15,
		"argon2_params": {
			"memory_kb": 8192,
			"time": 3,
			"threads": 2,
			"key_len": 32
		}
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to create test config file: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.Algorithm != "argon2id" {
		t.Errorf("expected algorithm argon2id, got %s", cfg.Algorithm)
	}
	if cfg.JWTSecret != "test-secret-123" {
		t.Errorf("expected jwt secret test-secret-123, got %s", cfg.JWTSecret)
	}
	if cfg.PuzzleDifficulty != 2 {
		t.Errorf("expected puzzle difficulty 2, got %d", cfg.PuzzleDifficulty)
	}
	if cfg.RequestsPerToken != 200 {
		t.Errorf("expected requests per token 200, got %d", cfg.RequestsPerToken)
	}
	if cfg.TokenExpiryMinutes != 15 {
		t.Errorf("expected token expiry 15, got %d", cfg.TokenExpiryMinutes)
	}
}

func TestLoadFromFileNotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/config.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadFromFileInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.json")

	if err := os.WriteFile(configPath, []byte("invalid json {"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env vars
	origVars := map[string]string{
		"JWT_SECRET":           os.Getenv("JWT_SECRET"),
		"ALGORITHM":            os.Getenv("ALGORITHM"),
		"PUZZLE_DIFFICULTY":    os.Getenv("PUZZLE_DIFFICULTY"),
		"REQUESTS_PER_TOKEN":   os.Getenv("REQUESTS_PER_TOKEN"),
		"TOKEN_EXPIRY_MINUTES": os.Getenv("TOKEN_EXPIRY_MINUTES"),
		"ARGON2_MEMORY_KB":     os.Getenv("ARGON2_MEMORY_KB"),
		"ARGON2_TIME":          os.Getenv("ARGON2_TIME"),
		"ARGON2_THREADS":       os.Getenv("ARGON2_THREADS"),
		"ARGON2_KEY_LEN":       os.Getenv("ARGON2_KEY_LEN"),
	}

	// Restore env vars after test
	defer func() {
		for key, val := range origVars {
			if val == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, val)
			}
		}
	}()

	// Set test env vars
	os.Setenv("JWT_SECRET", "env-secret")
	os.Setenv("ALGORITHM", "argon2id")
	os.Setenv("PUZZLE_DIFFICULTY", "3")
	os.Setenv("REQUESTS_PER_TOKEN", "150")
	os.Setenv("TOKEN_EXPIRY_MINUTES", "30")
	os.Setenv("ARGON2_MEMORY_KB", "4096")
	os.Setenv("ARGON2_TIME", "2")
	os.Setenv("ARGON2_THREADS", "1")
	os.Setenv("ARGON2_KEY_LEN", "16")

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv failed: %v", err)
	}

	if cfg.JWTSecret != "env-secret" {
		t.Errorf("expected jwt secret env-secret, got %s", cfg.JWTSecret)
	}
	if cfg.Algorithm != "argon2id" {
		t.Errorf("expected algorithm argon2id, got %s", cfg.Algorithm)
	}
	if cfg.PuzzleDifficulty != 3 {
		t.Errorf("expected puzzle difficulty 3, got %d", cfg.PuzzleDifficulty)
	}
	if cfg.RequestsPerToken != 150 {
		t.Errorf("expected requests per token 150, got %d", cfg.RequestsPerToken)
	}
	if cfg.TokenExpiryMinutes != 30 {
		t.Errorf("expected token expiry 30, got %d", cfg.TokenExpiryMinutes)
	}
	if cfg.Argon2Params.MemoryKB != 4096 {
		t.Errorf("expected argon2 memory 4096, got %d", cfg.Argon2Params.MemoryKB)
	}
	if cfg.Argon2Params.Time != 2 {
		t.Errorf("expected argon2 time 2, got %d", cfg.Argon2Params.Time)
	}
	if cfg.Argon2Params.Threads != 1 {
		t.Errorf("expected argon2 threads 1, got %d", cfg.Argon2Params.Threads)
	}
	if cfg.Argon2Params.KeyLen != 16 {
		t.Errorf("expected argon2 key len 16, got %d", cfg.Argon2Params.KeyLen)
	}
}

func TestLoadFromEnvMissingSecret(t *testing.T) {
	origSecret := os.Getenv("JWT_SECRET")
	defer func() {
		if origSecret == "" {
			os.Unsetenv("JWT_SECRET")
		} else {
			os.Setenv("JWT_SECRET", origSecret)
		}
	}()

	os.Unsetenv("JWT_SECRET")

	_, err := LoadFromEnv()
	if err == nil {
		t.Error("expected error when JWT_SECRET is missing")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: false,
		},
		{
			name: "missing jwt secret",
			config: &Config{
				JWTSecret:          "",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "negative puzzle difficulty",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   -1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "zero requests per token",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   0,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "zero token expiry",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 0,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid argon2 memory",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 0,
					Time:     4,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid argon2 time",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     0,
					Threads:  4,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid argon2 threads",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  0,
					KeyLen:   32,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid argon2 key length",
			config: &Config{
				JWTSecret:          "secret",
				PuzzleDifficulty:   1,
				RequestsPerToken:   100,
				TokenExpiryMinutes: 10,
				Argon2Params: puzzle.Argon2Config{
					MemoryKB: 16384,
					Time:     4,
					Threads:  4,
					KeyLen:   0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
