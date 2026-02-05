package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNoopMetrics(t *testing.T) {
	m := NewNoopMetrics()

	// These should not panic
	m.RecordPuzzleAttempt("success")
	m.RecordPuzzleAttempt("failed")
	m.RecordTokenVerification("success")
	m.RecordTokenVerification("failed")
	m.IncrementTokensIssued()
	m.IncrementPuzzlesSolved()
}

func TestPrometheusMetrics(t *testing.T) {
	// Create a new registry for this test to avoid conflicts
	reg := prometheus.NewRegistry()

	// Create new counters for this test
	tokensIssued := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_auth_tokens_issued_total",
		Help: "Test tokens issued",
	})
	puzzlesSolved := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_auth_puzzles_solved_total",
		Help: "Test puzzles solved",
	})
	puzzleAttempts := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_auth_puzzle_attempts_total",
		Help: "Test puzzle attempts",
	}, []string{"status"})
	tokenVerifications := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_auth_token_verifications_total",
		Help: "Test token verifications",
	}, []string{"status"})

	reg.MustRegister(tokensIssued, puzzlesSolved, puzzleAttempts, tokenVerifications)

	m := &PrometheusMetrics{
		tokensIssued:       tokensIssued,
		puzzlesSolved:      puzzlesSolved,
		puzzleAttempts:     puzzleAttempts,
		tokenVerifications: tokenVerifications,
	}

	// Test IncrementTokensIssued
	m.IncrementTokensIssued()
	m.IncrementTokensIssued()
	count := testutil.ToFloat64(tokensIssued)
	if count != 2 {
		t.Errorf("expected tokens issued count 2, got %f", count)
	}

	// Test IncrementPuzzlesSolved
	m.IncrementPuzzlesSolved()
	m.IncrementPuzzlesSolved()
	m.IncrementPuzzlesSolved()
	count = testutil.ToFloat64(puzzlesSolved)
	if count != 3 {
		t.Errorf("expected puzzles solved count 3, got %f", count)
	}

	// Test RecordPuzzleAttempt
	m.RecordPuzzleAttempt("success")
	m.RecordPuzzleAttempt("success")
	m.RecordPuzzleAttempt("failed")

	successCount := testutil.ToFloat64(puzzleAttempts.WithLabelValues("success"))
	if successCount != 2 {
		t.Errorf("expected success puzzle attempts 2, got %f", successCount)
	}

	failedCount := testutil.ToFloat64(puzzleAttempts.WithLabelValues("failed"))
	if failedCount != 1 {
		t.Errorf("expected failed puzzle attempts 1, got %f", failedCount)
	}

	// Test RecordTokenVerification
	m.RecordTokenVerification("success")
	m.RecordTokenVerification("success")
	m.RecordTokenVerification("success")
	m.RecordTokenVerification("invalid_token")

	successVerCount := testutil.ToFloat64(tokenVerifications.WithLabelValues("success"))
	if successVerCount != 3 {
		t.Errorf("expected success token verifications 3, got %f", successVerCount)
	}

	invalidCount := testutil.ToFloat64(tokenVerifications.WithLabelValues("invalid_token"))
	if invalidCount != 1 {
		t.Errorf("expected invalid_token verifications 1, got %f", invalidCount)
	}
}

func TestLegacyFunctions(t *testing.T) {
	// Test legacy functions don't panic
	// Note: These operate on global metrics, so we can't easily test exact counts
	IncrementTokensIssued()
	IncrementPuzzlesSolved()
	RecordPuzzleAttempt("test")
	RecordTokenVerification("test")
}

func TestPrometheusMetricsInterface(t *testing.T) {
	// Verify that PrometheusMetrics implements MetricsRecorder
	var _ MetricsRecorder = &PrometheusMetrics{}
	var _ MetricsRecorder = &NoopMetrics{}
}

func TestNewPrometheusMetrics(t *testing.T) {
	m := NewPrometheusMetrics()
	if m == nil {
		t.Error("expected non-nil metrics recorder")
	}

	// Should not panic
	m.IncrementTokensIssued()
	m.IncrementPuzzlesSolved()
	m.RecordPuzzleAttempt("test")
	m.RecordTokenVerification("test")
}

func TestNoopMetricsInterface(t *testing.T) {
	var m MetricsRecorder = NewNoopMetrics()

	// Should not panic
	m.RecordPuzzleAttempt("success")
	m.RecordPuzzleAttempt("failed")
	m.RecordPuzzleAttempt("invalid_request")
	m.RecordPuzzleAttempt("missing_fields")
	m.RecordPuzzleAttempt("invalid_expiry")
	m.RecordPuzzleAttempt("expired")
	m.RecordPuzzleAttempt("invalid_solution")

	m.RecordTokenVerification("success")
	m.RecordTokenVerification("failed")
	m.RecordTokenVerification("expired")
	m.RecordTokenVerification("rate_limited")
	m.RecordTokenVerification("missing_token")
	m.RecordTokenVerification("invalid_token")

	m.IncrementTokensIssued()
	m.IncrementPuzzlesSolved()
}
