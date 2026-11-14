package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/status-im/proxy-common/auth/config"
	"github.com/status-im/proxy-common/auth/puzzle"
)

// solvePuzzle asks h for a puzzle, solves it and returns the solve request.
func solvePuzzle(t *testing.T, h *Handlers, cfg *config.Config) SolveRequest {
	t.Helper()

	w := httptest.NewRecorder()
	h.PuzzleHandler(w, httptest.NewRequest(http.MethodGet, "/auth/puzzle", nil))

	var resp struct {
		Challenge  string `json:"challenge"`
		Salt       string `json:"salt"`
		Difficulty int    `json:"difficulty"`
		ExpiresAt  string `json:"expires_at"`
		HMAC       string `json:"hmac"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode puzzle response: %v", err)
	}
	expiresAt, err := time.Parse(time.RFC3339, resp.ExpiresAt)
	if err != nil {
		t.Fatalf("failed to parse expires_at: %v", err)
	}

	solution, err := puzzle.Solve(&puzzle.Puzzle{
		Challenge:  resp.Challenge,
		Salt:       resp.Salt,
		Difficulty: resp.Difficulty,
		ExpiresAt:  expiresAt,
		HMAC:       resp.HMAC,
	}, cfg.Argon2Params)
	if err != nil {
		t.Fatalf("failed to solve puzzle: %v", err)
	}

	return SolveRequest{
		Challenge: resp.Challenge,
		Salt:      resp.Salt,
		Nonce:     solution.Nonce,
		ArgonHash: solution.ArgonHash,
		HMAC:      resp.HMAC,
		ExpiresAt: resp.ExpiresAt,
	}
}

// submit posts a solve request to h and returns the response.
func submit(t *testing.T, h *Handlers, solveReq SolveRequest) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(solveReq)
	if err != nil {
		t.Fatalf("failed to marshal solve request: %v", err)
	}
	w := httptest.NewRecorder()
	h.SolveHandler(w, httptest.NewRequest(http.MethodPost, "/auth/solve", bytes.NewReader(body)))
	return w
}

func TestSolveHandlerIssuesOneTokenPerPuzzle(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)
	solveReq := solvePuzzle(t, h, cfg)

	if w := submit(t, h, solveReq); w.Code != http.StatusOK {
		t.Fatalf("first submission: expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	for attempt := 2; attempt <= 3; attempt++ {
		w := submit(t, h, solveReq)
		if w.Code != http.StatusConflict {
			t.Errorf("submission %d: expected status 409, got %d: %s", attempt, w.Code, w.Body.String())
		}
		if bytes.Contains(w.Body.Bytes(), []byte("token")) {
			t.Errorf("submission %d: got a token: %s", attempt, w.Body.String())
		}
	}
}

func TestSolveHandlerConcurrentSubmissionsGetOneToken(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)
	solveReq := solvePuzzle(t, h, cfg)

	const submissions = 20
	codes := make([]int, submissions)
	var wg sync.WaitGroup
	for i := range codes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes[i] = submit(t, h, solveReq).Code
		}()
	}
	wg.Wait()

	counts := map[int]int{}
	for _, code := range codes {
		counts[code]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != submissions-1 {
		t.Errorf("expected one 200 and %d 409, got %v", submissions-1, counts)
	}
}

func TestSolveHandlerEachPuzzleGetsItsToken(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)

	for i := 1; i <= 2; i++ {
		if w := submit(t, h, solvePuzzle(t, h, cfg)); w.Code != http.StatusOK {
			t.Errorf("puzzle %d: expected status 200, got %d: %s", i, w.Code, w.Body.String())
		}
	}
}

func TestSolveHandlerWrongSolutionDoesNotUseUpThePuzzle(t *testing.T) {
	cfg := getTestConfig()
	h := New(cfg)
	solveReq := solvePuzzle(t, h, cfg)

	wrong := solveReq
	wrong.ArgonHash = "ff" + solveReq.ArgonHash[2:]
	if w := submit(t, h, wrong); w.Code != http.StatusBadRequest {
		t.Fatalf("wrong solution: expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	if w := submit(t, h, solveReq); w.Code != http.StatusOK {
		t.Errorf("right solution after a wrong one: expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSolvedPuzzlesForgetsAPuzzleAfterItExpires(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := newSolvedPuzzles()
	s.now = func() time.Time { return now }
	expiresAt := now.Add(10 * time.Minute)

	if !s.claim("puzzle-1", expiresAt) {
		t.Fatal("first claim of a puzzle must succeed")
	}

	// Remembered for its whole life, and a little past it.
	for _, at := range []time.Duration{time.Second, 10 * time.Minute, 10*time.Minute + 30*time.Second} {
		now = expiresAt.Add(-10 * time.Minute).Add(at)
		if s.claim("puzzle-1", expiresAt) {
			t.Errorf("claimed again %v after the first claim", at)
		}
	}

	// Later claims of other puzzles sweep it out.
	now = expiresAt.Add(3 * time.Minute)
	s.claim("puzzle-2", now.Add(10*time.Minute))
	if _, remembered := s.expiresAt["puzzle-1"]; remembered {
		t.Error("an expired puzzle is still remembered")
	}
	if _, remembered := s.expiresAt["puzzle-2"]; !remembered {
		t.Error("a live puzzle was forgotten")
	}
}
