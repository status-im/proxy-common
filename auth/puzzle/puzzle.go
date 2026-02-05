package puzzle

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
)

type Argon2Config struct {
	MemoryKB int `json:"memory_kb"`
	Time     int `json:"time"`
	Threads  int `json:"threads"`
	KeyLen   int `json:"key_len"`
}

type Puzzle struct {
	Challenge  string    `json:"challenge"`
	Salt       string    `json:"salt"`
	Difficulty int       `json:"difficulty"`
	ExpiresAt  time.Time `json:"expires_at"`
	HMAC       string    `json:"hmac"`
}

type Solution struct {
	ArgonHash string `json:"argon_hash"`
	Nonce     uint64 `json:"nonce"`
}

func Generate(difficulty int, ttlMinutes int, jwtSecret string) (*Puzzle, error) {
	challengeBytes := make([]byte, 16)
	if _, err := rand.Read(challengeBytes); err != nil {
		return nil, fmt.Errorf("failed to generate challenge: %w", err)
	}

	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	challenge := hex.EncodeToString(challengeBytes)
	salt := hex.EncodeToString(saltBytes)
	expiresAt := time.Now().Add(time.Duration(ttlMinutes) * time.Minute)

	// Create HMAC for puzzle verification (challenge + salt + difficulty + expires_at)
	puzzleData := fmt.Sprintf("%s%s%d%s", challenge, salt, difficulty, expiresAt.Format(time.RFC3339))
	puzzleHMAC := computeHMAC(puzzleData, jwtSecret)

	return &Puzzle{
		Challenge:  challenge,
		Salt:       salt,
		Difficulty: difficulty,
		ExpiresAt:  expiresAt,
		HMAC:       puzzleHMAC,
	}, nil
}

// ValidateHMACProtectedSolution validates a solution with HMAC protection (only secure method)
func ValidateHMACProtectedSolution(puzzle *Puzzle, solution *Solution, argon2Config Argon2Config, jwtSecret string) bool {
	// Step 1: Check HMAC signature of puzzle conditions FIRST (most important security check)
	puzzleData := fmt.Sprintf("%s%s%d%s", puzzle.Challenge, puzzle.Salt, puzzle.Difficulty, puzzle.ExpiresAt.Format(time.RFC3339))
	expectedHMAC := computeHMAC(puzzleData, jwtSecret)
	if !hmac.Equal([]byte(expectedHMAC), []byte(puzzle.HMAC)) {
		return false
	}

	if time.Now().After(puzzle.ExpiresAt) {
		return false
	}

	computedArgonHash := computeArgon2HashWithConfig(puzzle.Challenge, puzzle.Salt, solution.Nonce, puzzle.Difficulty, argon2Config)

	if computedArgonHash != solution.ArgonHash {
		return false
	}

	return checkDifficulty(computedArgonHash, puzzle.Difficulty)
}

func Solve(puzzle *Puzzle, argon2Config Argon2Config) (*Solution, error) {
	if time.Now().After(puzzle.ExpiresAt) {
		return nil, fmt.Errorf("puzzle has expired")
	}

	for nonce := uint64(0); nonce < 1000000; nonce++ {
		argonHash := computeArgon2HashWithConfig(puzzle.Challenge, puzzle.Salt, nonce, puzzle.Difficulty, argon2Config)

		if checkDifficulty(argonHash, puzzle.Difficulty) {
			return &Solution{
				ArgonHash: argonHash,
				Nonce:     nonce,
			}, nil
		}
	}

	return nil, fmt.Errorf("failed to solve puzzle within attempt limit")
}

func computeHMAC(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func computeArgon2HashWithConfig(challenge, salt string, nonce uint64, difficulty int, argon2Config Argon2Config) string {
	input := fmt.Sprintf("%s%s%d", challenge, salt, nonce)

	saltBytes, err := hex.DecodeString(salt)
	if err != nil {
		saltBytes = []byte(salt)
	}

	memory := uint32(argon2Config.MemoryKB)
	time := uint32(argon2Config.Time)
	threads := uint8(argon2Config.Threads)
	keyLen := uint32(argon2Config.KeyLen)

	hash := argon2.IDKey([]byte(input), saltBytes, time, memory, threads, keyLen)

	return hex.EncodeToString(hash)
}

func checkDifficulty(hash string, difficulty int) bool {
	if len(hash) < difficulty {
		return false
	}

	for i := 0; i < difficulty; i++ {
		if hash[i] != '0' {
			return false
		}
	}

	return true
}
