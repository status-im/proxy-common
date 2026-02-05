package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/status-im/proxy-common/auth/config"
	"github.com/status-im/proxy-common/auth/puzzle"
)

func getTestConfig() *config.Config {
	return config.New(
		config.WithJWTSecret("test-secret"),
		config.WithDifficulty(1),
		config.WithRequestsPerToken(100),
		config.WithTokenExpiry(10),
	)
}

func TestPuzzleHandler(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/puzzle", nil)
	w := httptest.NewRecorder()

	h.PuzzleHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := response["challenge"]; !ok {
		t.Error("expected challenge field in response")
	}
	if _, ok := response["salt"]; !ok {
		t.Error("expected salt field in response")
	}
	if _, ok := response["difficulty"]; !ok {
		t.Error("expected difficulty field in response")
	}
	if _, ok := response["expires_at"]; !ok {
		t.Error("expected expires_at field in response")
	}
	if _, ok := response["hmac"]; !ok {
		t.Error("expected hmac field in response")
	}
	if _, ok := response["algorithm"]; !ok {
		t.Error("expected algorithm field in response")
	}

	// Check difficulty matches config
	if difficulty, ok := response["difficulty"].(float64); ok {
		if int(difficulty) != cfg.PuzzleDifficulty {
			t.Errorf("expected difficulty %d, got %d", cfg.PuzzleDifficulty, int(difficulty))
		}
	}
}

func TestSolveHandler(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	// First get a puzzle
	puzzleReq := httptest.NewRequest(http.MethodGet, "/auth/puzzle", nil)
	puzzleW := httptest.NewRecorder()
	h.PuzzleHandler(puzzleW, puzzleReq)

	var puzzleResp map[string]interface{}
	if err := json.NewDecoder(puzzleW.Body).Decode(&puzzleResp); err != nil {
		t.Fatalf("failed to decode puzzle response: %v", err)
	}

	// Solve the puzzle
	p := &puzzle.Puzzle{
		Challenge:  puzzleResp["challenge"].(string),
		Salt:       puzzleResp["salt"].(string),
		Difficulty: int(puzzleResp["difficulty"].(float64)),
		HMAC:       puzzleResp["hmac"].(string),
	}

	expiresAtStr := puzzleResp["expires_at"].(string)
	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err != nil {
		t.Fatalf("failed to parse expires_at: %v", err)
	}
	p.ExpiresAt = expiresAt

	solution, err := puzzle.Solve(p, cfg.Argon2Params)
	if err != nil {
		t.Fatalf("failed to solve puzzle: %v", err)
	}

	// Submit solution
	solveReq := SolveRequest{
		Challenge: p.Challenge,
		Salt:      p.Salt,
		Nonce:     solution.Nonce,
		ArgonHash: solution.ArgonHash,
		HMAC:      p.HMAC,
		ExpiresAt: expiresAtStr,
	}

	body, err := json.Marshal(solveReq)
	if err != nil {
		t.Fatalf("failed to marshal solve request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/solve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.SolveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check token is present
	if _, ok := response["token"]; !ok {
		t.Error("expected token field in response")
	}
	if _, ok := response["expires_at"]; !ok {
		t.Error("expected expires_at field in response")
	}
	if _, ok := response["request_limit"]; !ok {
		t.Error("expected request_limit field in response")
	}
}

func TestSolveHandlerInvalidRequest(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	tests := []struct {
		name    string
		payload string
		status  int
	}{
		{
			name:    "empty body",
			payload: "{}",
			status:  http.StatusBadRequest,
		},
		{
			name:    "missing challenge",
			payload: `{"salt":"test","argon_hash":"test","hmac":"test","expires_at":"2026-01-01T00:00:00Z"}`,
			status:  http.StatusBadRequest,
		},
		{
			name:    "invalid json",
			payload: `{invalid}`,
			status:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/solve", bytes.NewReader([]byte(tt.payload)))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.SolveHandler(w, req)

			if w.Code != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, w.Code)
			}
		})
	}
}

func TestSolveHandlerInvalidSolution(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	// Submit invalid solution
	solveReq := SolveRequest{
		Challenge: "invalid-challenge",
		Salt:      "invalid-salt",
		Nonce:     12345,
		ArgonHash: "invalid-hash",
		HMAC:      "invalid-hmac",
		ExpiresAt: time.Now().Add(10 * time.Minute).Format(time.RFC3339),
	}

	body, err := json.Marshal(solveReq)
	if err != nil {
		t.Fatalf("failed to marshal solve request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/solve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.SolveHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestVerifyHandler(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	// First get a valid token by solving a puzzle
	puzzleReq := httptest.NewRequest(http.MethodGet, "/auth/puzzle", nil)
	puzzleW := httptest.NewRecorder()
	h.PuzzleHandler(puzzleW, puzzleReq)

	var puzzleResp map[string]interface{}
	if err := json.NewDecoder(puzzleW.Body).Decode(&puzzleResp); err != nil {
		t.Fatalf("failed to decode puzzle response: %v", err)
	}

	p := &puzzle.Puzzle{
		Challenge:  puzzleResp["challenge"].(string),
		Salt:       puzzleResp["salt"].(string),
		Difficulty: int(puzzleResp["difficulty"].(float64)),
		HMAC:       puzzleResp["hmac"].(string),
	}
	expiresAtStr := puzzleResp["expires_at"].(string)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
	p.ExpiresAt = expiresAt

	solution, _ := puzzle.Solve(p, cfg.Argon2Params)

	solveReq := SolveRequest{
		Challenge: p.Challenge,
		Salt:      p.Salt,
		Nonce:     solution.Nonce,
		ArgonHash: solution.ArgonHash,
		HMAC:      p.HMAC,
		ExpiresAt: expiresAtStr,
	}

	body, _ := json.Marshal(solveReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/solve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SolveHandler(w, req)

	var solveResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&solveResp)
	token := solveResp["token"].(string)

	// Test with Bearer token
	t.Run("with bearer token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		h.VerifyHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test with query parameter
	t.Run("with query parameter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify?token="+token, nil)
		w := httptest.NewRecorder()

		h.VerifyHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	// Test missing token
	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		w := httptest.NewRecorder()

		h.VerifyHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})

	// Test invalid token
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		h.VerifyHandler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
	})
}

func TestVerifyHandlerRateLimiting(t *testing.T) {
	cfg := config.New(
		config.WithJWTSecret("test-secret"),
		config.WithDifficulty(1),
		config.WithRequestsPerToken(3), // Low limit for testing
		config.WithTokenExpiry(10),
	)
	h := New(cfg)

	// Get a token
	puzzleReq := httptest.NewRequest(http.MethodGet, "/auth/puzzle", nil)
	puzzleW := httptest.NewRecorder()
	h.PuzzleHandler(puzzleW, puzzleReq)

	var puzzleResp map[string]interface{}
	json.NewDecoder(puzzleW.Body).Decode(&puzzleResp)

	p := &puzzle.Puzzle{
		Challenge:  puzzleResp["challenge"].(string),
		Salt:       puzzleResp["salt"].(string),
		Difficulty: int(puzzleResp["difficulty"].(float64)),
		HMAC:       puzzleResp["hmac"].(string),
	}
	expiresAtStr := puzzleResp["expires_at"].(string)
	expiresAt, _ := time.Parse(time.RFC3339, expiresAtStr)
	p.ExpiresAt = expiresAt

	solution, _ := puzzle.Solve(p, cfg.Argon2Params)

	solveReq := SolveRequest{
		Challenge: p.Challenge,
		Salt:      p.Salt,
		Nonce:     solution.Nonce,
		ArgonHash: solution.ArgonHash,
		HMAC:      p.HMAC,
		ExpiresAt: expiresAtStr,
	}

	body, _ := json.Marshal(solveReq)
	req := httptest.NewRequest(http.MethodPost, "/auth/solve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.SolveHandler(w, req)

	var solveResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&solveResp)
	token := solveResp["token"].(string)

	// Use token up to limit
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()

		h.VerifyHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	// Next request should be rate limited
	req = httptest.NewRequest(http.MethodGet, "/auth/verify", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()

	h.VerifyHandler(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

func TestStatusHandler(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/auth/status", nil)
	w := httptest.NewRecorder()

	h.StatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := response["puzzle_difficulty"]; !ok {
		t.Error("expected puzzle_difficulty field in response")
	}
	if _, ok := response["token_expiry_min"]; !ok {
		t.Error("expected token_expiry_min field in response")
	}
	if _, ok := response["requests_per_token"]; !ok {
		t.Error("expected requests_per_token field in response")
	}
	if _, ok := response["jwt_secret_present"]; !ok {
		t.Error("expected jwt_secret_present field in response")
	}
	if _, ok := response["algorithm"]; !ok {
		t.Error("expected algorithm field in response")
	}
	if _, ok := response["endpoints"]; !ok {
		t.Error("expected endpoints field in response")
	}

	// Check jwt_secret_present is true
	if present, ok := response["jwt_secret_present"].(bool); ok {
		if !present {
			t.Error("expected jwt_secret_present to be true")
		}
	}
}

func TestTestSolveHandler(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	req := httptest.NewRequest(http.MethodGet, "/dev/test-solve", nil)
	w := httptest.NewRecorder()

	h.TestSolveHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Check required fields
	if _, ok := response["test_puzzle"]; !ok {
		t.Error("expected test_puzzle field in response")
	}
	if _, ok := response["example_request"]; !ok {
		t.Error("expected example_request field in response")
	}
}
