package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/status-im/proxy-common/auth/config"
	"github.com/status-im/proxy-common/auth/jwt"
	"github.com/status-im/proxy-common/auth/metrics"
	"github.com/status-im/proxy-common/auth/puzzle"
)

type Handlers struct {
	config     *config.Config
	metrics    metrics.MetricsRecorder
	tokenUsage map[string]int
	tokenMutex sync.RWMutex
}

type Option func(*Handlers)

func WithMetrics(m metrics.MetricsRecorder) Option {
	return func(h *Handlers) {
		h.metrics = m
	}
}

func New(cfg *config.Config, opts ...Option) *Handlers {
	h := &Handlers{
		config:     cfg,
		metrics:    metrics.NewNoopMetrics(),
		tokenUsage: make(map[string]int),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

func (h *Handlers) PuzzleHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	p, err := puzzle.Generate(h.config.PuzzleDifficulty, h.config.TokenExpiryMinutes, h.config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to generate puzzle", 500)
		return
	}

	response := map[string]interface{}{
		"challenge":     p.Challenge,
		"salt":          p.Salt,
		"difficulty":    p.Difficulty,
		"expires_at":    p.ExpiresAt.Format(time.RFC3339),
		"hmac":          p.HMAC,
		"algorithm":     h.config.Algorithm,
		"argon2_params": h.config.Argon2Params,
		"solve_request_format": map[string]interface{}{
			"required_fields": []string{"challenge", "salt", "nonce", "argon_hash", "hmac", "expires_at"},
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode puzzle response", "error", err)
	}
}

type SolveRequest struct {
	Challenge string `json:"challenge"`
	Salt      string `json:"salt"`
	Nonce     uint64 `json:"nonce"`
	ArgonHash string `json:"argon_hash"`
	HMAC      string `json:"hmac"`
	ExpiresAt string `json:"expires_at"`
}

// SolveHandler handles HMAC protected solutions only
func (h *Handlers) SolveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.metrics.RecordPuzzleAttempt("invalid_request")
		http.Error(w, "bad request", 400)
		return
	}

	if req.Challenge == "" || req.Salt == "" || req.ArgonHash == "" || req.HMAC == "" {
		h.metrics.RecordPuzzleAttempt("missing_fields")
		http.Error(w, "missing required fields: challenge, salt, argon_hash, hmac", 400)
		return
	}

	exp, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		h.metrics.RecordPuzzleAttempt("invalid_expiry")
		http.Error(w, "bad expires_at format", 400)
		return
	}

	if time.Now().After(exp) {
		h.metrics.RecordPuzzleAttempt("expired")
		http.Error(w, "puzzle has expired", 400)
		return
	}

	puzzleObj := &puzzle.Puzzle{
		Challenge:  req.Challenge,
		Salt:       req.Salt,
		Difficulty: h.config.PuzzleDifficulty,
		ExpiresAt:  exp,
		HMAC:       req.HMAC,
	}

	solution := &puzzle.Solution{
		Nonce:     req.Nonce,
		ArgonHash: req.ArgonHash,
	}

	if !puzzle.ValidateHMACProtectedSolution(puzzleObj, solution, h.config.Argon2Params, h.config.JWTSecret) {
		h.metrics.RecordPuzzleAttempt("invalid_solution")
		http.Error(w, "invalid solution or HMAC verification failed", 400)
		return
	}

	h.metrics.RecordPuzzleAttempt("success")
	h.metrics.IncrementPuzzlesSolved()

	token, expiresAt, err := jwt.Generate(h.config.JWTSecret, req.Challenge, h.config.TokenExpiryMinutes, h.config.RequestsPerToken)
	if err != nil {
		http.Error(w, "failed to generate token", 500)
		return
	}

	h.metrics.IncrementTokensIssued()

	response := map[string]interface{}{
		"token":         token,
		"expires_at":    expiresAt.Format(time.RFC3339),
		"request_limit": h.config.RequestsPerToken,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode solve response", "error", err)
	}
}

// TestSolveHandler provides a test endpoint that generates a valid solution with HMAC
func (h *Handlers) TestSolveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	p, err := puzzle.Generate(h.config.PuzzleDifficulty, h.config.TokenExpiryMinutes, h.config.JWTSecret)
	if err != nil {
		http.Error(w, "failed to generate test puzzle", 500)
		return
	}

	solution, err := puzzle.Solve(p, h.config.Argon2Params)
	if err != nil {
		http.Error(w, "failed to solve test puzzle", 500)
		return
	}

	response := map[string]interface{}{
		"test_puzzle": map[string]interface{}{
			"challenge":  p.Challenge,
			"salt":       p.Salt,
			"difficulty": p.Difficulty,
			"expires_at": p.ExpiresAt.Format(time.RFC3339),
		},
		"example_request": map[string]interface{}{
			"challenge":  p.Challenge,
			"salt":       p.Salt,
			"nonce":      solution.Nonce,
			"argon_hash": solution.ArgonHash,
			"hmac":       p.HMAC,
			"expires_at": p.ExpiresAt.Format(time.RFC3339),
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode test-solve response", "error", err)
	}
}

// VerifyHandler handles JWT token verification for nginx auth_request
func (h *Handlers) VerifyHandler(w http.ResponseWriter, r *http.Request) {
	var tokenString string

	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString = parts[1]
		}
	}

	if tokenString == "" {
		tokenString = r.URL.Query().Get("token")
	}

	if tokenString == "" {
		h.metrics.RecordTokenVerification("missing_token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	claims, err := jwt.Verify(tokenString, h.config.JWTSecret)
	if err != nil {
		h.metrics.RecordTokenVerification("invalid_token")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokenID := claims.ID
	if tokenID != "" {
		h.tokenMutex.Lock()
		currentUsage := h.tokenUsage[tokenID]
		if currentUsage >= h.config.RequestsPerToken {
			h.tokenMutex.Unlock()
			h.metrics.RecordTokenVerification("rate_limited")
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.config.RequestsPerToken))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		h.tokenUsage[tokenID] = currentUsage + 1
		newUsage := h.tokenUsage[tokenID]
		h.tokenMutex.Unlock()

		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", h.config.RequestsPerToken))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", h.config.RequestsPerToken-newUsage))
	}

	h.metrics.RecordTokenVerification("success")

	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) StatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"puzzle_difficulty":  h.config.PuzzleDifficulty,
		"token_expiry_min":   h.config.TokenExpiryMinutes,
		"requests_per_token": h.config.RequestsPerToken,
		"jwt_secret_present": h.config.JWTSecret != "",
		"algorithm":          h.config.Algorithm,
		"argon2_params":      h.config.Argon2Params,
		"endpoints": map[string]interface{}{
			"puzzle": "/auth/puzzle",
			"solve":  "/auth/solve",
			"verify": "/auth/verify",
			"status": "/auth/status",
		},
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("failed to encode status response", "error", err)
	}
}
