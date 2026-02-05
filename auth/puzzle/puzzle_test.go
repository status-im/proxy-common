package puzzle

import (
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if p.Challenge == "" {
		t.Error("expected non-empty challenge")
	}
	if p.Salt == "" {
		t.Error("expected non-empty salt")
	}
	if p.Difficulty != difficulty {
		t.Errorf("expected difficulty %d, got %d", difficulty, p.Difficulty)
	}
	if p.HMAC == "" {
		t.Error("expected non-empty HMAC")
	}

	// Check expiration time
	expectedExp := time.Now().Add(time.Duration(ttlMinutes) * time.Minute)
	diff := p.ExpiresAt.Sub(expectedExp).Abs()
	if diff > 2*time.Second {
		t.Errorf("expiration time differs by %v", diff)
	}

	// Check that challenge and salt are hex strings (32 chars for 16 bytes)
	if len(p.Challenge) != 32 {
		t.Errorf("expected challenge length 32, got %d", len(p.Challenge))
	}
	if len(p.Salt) != 32 {
		t.Errorf("expected salt length 32, got %d", len(p.Salt))
	}
}

func TestGenerateUniqueness(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"

	// Generate multiple puzzles and check they're unique
	puzzles := make([]*Puzzle, 10)
	for i := 0; i < 10; i++ {
		p, err := Generate(difficulty, ttlMinutes, jwtSecret)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}
		puzzles[i] = p
	}

	// Check all challenges are unique
	for i := 0; i < len(puzzles); i++ {
		for j := i + 1; j < len(puzzles); j++ {
			if puzzles[i].Challenge == puzzles[j].Challenge {
				t.Error("expected unique challenges")
			}
			if puzzles[i].Salt == puzzles[j].Salt {
				t.Error("expected unique salts")
			}
		}
	}
}

func TestSolve(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	solution, err := Solve(p, argon2Config)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	if solution.ArgonHash == "" {
		t.Error("expected non-empty argon hash")
	}

	// Check that solution hash meets difficulty
	if !checkDifficulty(solution.ArgonHash, difficulty) {
		t.Error("solution does not meet difficulty requirement")
	}
}

func TestSolveExpiredPuzzle(t *testing.T) {
	difficulty := 1
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	// Create expired puzzle
	p, err := Generate(difficulty, 1, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Set expiration to past
	p.ExpiresAt = time.Now().Add(-1 * time.Minute)

	_, err = Solve(p, argon2Config)
	if err == nil {
		t.Error("expected error for expired puzzle")
	}
}

func TestValidateHMACProtectedSolution(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	solution, err := Solve(p, argon2Config)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Test valid solution
	valid := ValidateHMACProtectedSolution(p, solution, argon2Config, jwtSecret)
	if !valid {
		t.Error("expected valid solution")
	}
}

func TestValidateHMACProtectedSolutionInvalidHMAC(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	solution, err := Solve(p, argon2Config)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Tamper with HMAC
	p.HMAC = "invalid-hmac"

	valid := ValidateHMACProtectedSolution(p, solution, argon2Config, jwtSecret)
	if valid {
		t.Error("expected invalid solution due to tampered HMAC")
	}
}

func TestValidateHMACProtectedSolutionExpired(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	solution, err := Solve(p, argon2Config)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Set expiration to past
	p.ExpiresAt = time.Now().Add(-1 * time.Minute)

	// Regenerate HMAC with expired time
	puzzleData := computeHMAC(
		p.Challenge+p.Salt+string(rune(p.Difficulty))+p.ExpiresAt.Format(time.RFC3339),
		jwtSecret,
	)
	p.HMAC = puzzleData

	valid := ValidateHMACProtectedSolution(p, solution, argon2Config, jwtSecret)
	if valid {
		t.Error("expected invalid solution due to expiration")
	}
}

func TestValidateHMACProtectedSolutionInvalidHash(t *testing.T) {
	difficulty := 1
	ttlMinutes := 10
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Create invalid solution
	solution := &Solution{
		ArgonHash: "invalid-hash",
		Nonce:     12345,
	}

	valid := ValidateHMACProtectedSolution(p, solution, argon2Config, jwtSecret)
	if valid {
		t.Error("expected invalid solution due to wrong hash")
	}
}

func TestValidateHMACProtectedSolutionWrongDifficulty(t *testing.T) {
	difficulty := 3 // Higher difficulty
	ttlMinutes := 10
	jwtSecret := "test-secret"
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	p, err := Generate(difficulty, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Create solution for lower difficulty
	p2, err := Generate(1, ttlMinutes, jwtSecret)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	solution, err := Solve(p2, argon2Config)
	if err != nil {
		t.Fatalf("Solve failed: %v", err)
	}

	// Try to use low-difficulty solution for high-difficulty puzzle
	p.Challenge = p2.Challenge
	p.Salt = p2.Salt

	valid := ValidateHMACProtectedSolution(p, solution, argon2Config, jwtSecret)
	if valid {
		t.Error("expected invalid solution due to difficulty mismatch")
	}
}

func TestCheckDifficulty(t *testing.T) {
	tests := []struct {
		name       string
		hash       string
		difficulty int
		want       bool
	}{
		{
			name:       "difficulty 0",
			hash:       "abc123",
			difficulty: 0,
			want:       true,
		},
		{
			name:       "difficulty 1 with leading zero",
			hash:       "0abc123",
			difficulty: 1,
			want:       true,
		},
		{
			name:       "difficulty 1 without leading zero",
			hash:       "abc123",
			difficulty: 1,
			want:       false,
		},
		{
			name:       "difficulty 2 with two leading zeros",
			hash:       "00abc123",
			difficulty: 2,
			want:       true,
		},
		{
			name:       "difficulty 2 with one leading zero",
			hash:       "0abc123",
			difficulty: 2,
			want:       false,
		},
		{
			name:       "difficulty 3 with three leading zeros",
			hash:       "000abc123",
			difficulty: 3,
			want:       true,
		},
		{
			name:       "hash too short",
			hash:       "00",
			difficulty: 3,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkDifficulty(tt.hash, tt.difficulty)
			if got != tt.want {
				t.Errorf("checkDifficulty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeArgon2HashWithConfig(t *testing.T) {
	challenge := "test-challenge"
	salt := "test-salt"
	nonce := uint64(123)
	difficulty := 1
	argon2Config := Argon2Config{
		MemoryKB: 16384,
		Time:     4,
		Threads:  4,
		KeyLen:   32,
	}

	hash := computeArgon2HashWithConfig(challenge, salt, nonce, difficulty, argon2Config)
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Hash should be deterministic
	hash2 := computeArgon2HashWithConfig(challenge, salt, nonce, difficulty, argon2Config)
	if hash != hash2 {
		t.Error("expected deterministic hash")
	}

	// Different nonce should produce different hash
	hash3 := computeArgon2HashWithConfig(challenge, salt, 456, difficulty, argon2Config)
	if hash == hash3 {
		t.Error("expected different hash for different nonce")
	}
}

func TestComputeHMAC(t *testing.T) {
	data := "test-data"
	secret := "test-secret"

	hmac1 := computeHMAC(data, secret)
	if hmac1 == "" {
		t.Error("expected non-empty HMAC")
	}

	// HMAC should be deterministic
	hmac2 := computeHMAC(data, secret)
	if hmac1 != hmac2 {
		t.Error("expected deterministic HMAC")
	}

	// Different secret should produce different HMAC
	hmac3 := computeHMAC(data, "different-secret")
	if hmac1 == hmac3 {
		t.Error("expected different HMAC for different secret")
	}

	// Different data should produce different HMAC
	hmac4 := computeHMAC("different-data", secret)
	if hmac1 == hmac4 {
		t.Error("expected different HMAC for different data")
	}
}
