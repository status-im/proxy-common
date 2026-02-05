package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsRecorder interface {
	RecordPuzzleAttempt(status string)
	RecordTokenVerification(status string)
	IncrementTokensIssued()
	IncrementPuzzlesSolved()
}

type NoopMetrics struct{}

func NewNoopMetrics() MetricsRecorder {
	return &NoopMetrics{}
}

func (n *NoopMetrics) RecordPuzzleAttempt(status string) {}

func (n *NoopMetrics) RecordTokenVerification(status string) {}

func (n *NoopMetrics) IncrementTokensIssued() {}

func (n *NoopMetrics) IncrementPuzzlesSolved() {}

type PrometheusMetrics struct {
	tokensIssued       prometheus.Counter
	puzzlesSolved      prometheus.Counter
	puzzleAttempts     *prometheus.CounterVec
	tokenVerifications *prometheus.CounterVec
}

func NewPrometheusMetrics() MetricsRecorder {
	return &PrometheusMetrics{
		tokensIssued:       TokensIssued,
		puzzlesSolved:      PuzzlesSolved,
		puzzleAttempts:     PuzzleAttempts,
		tokenVerifications: TokenVerifications,
	}
}

func (p *PrometheusMetrics) RecordPuzzleAttempt(status string) {
	p.puzzleAttempts.WithLabelValues(status).Inc()
}

func (p *PrometheusMetrics) RecordTokenVerification(status string) {
	p.tokenVerifications.WithLabelValues(status).Inc()
}

func (p *PrometheusMetrics) IncrementTokensIssued() {
	p.tokensIssued.Inc()
}

func (p *PrometheusMetrics) IncrementPuzzlesSolved() {
	p.puzzlesSolved.Inc()
}

// Legacy global metrics for backward compatibility
var (
	// TokensIssued tracks the total number of JWT tokens issued
	TokensIssued = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_tokens_issued_total",
		Help: "The total number of JWT tokens issued",
	})

	// PuzzlesSolved tracks the total number of puzzles solved successfully
	PuzzlesSolved = promauto.NewCounter(prometheus.CounterOpts{
		Name: "auth_puzzles_solved_total",
		Help: "The total number of puzzles solved successfully",
	})

	// PuzzleAttempts tracks puzzle solution attempts (including failed ones)
	PuzzleAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_puzzle_attempts_total",
		Help: "The total number of puzzle solution attempts",
	}, []string{"status"}) // status: "success", "failed", "invalid_hmac", "expired"

	// TokenVerifications tracks JWT token verification attempts
	TokenVerifications = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_token_verifications_total",
		Help: "The total number of token verification attempts",
	}, []string{"status"}) // status: "success", "failed", "expired", "rate_limited"
)

// Legacy functions for backward compatibility

func IncrementTokensIssued() {
	TokensIssued.Inc()
}

func IncrementPuzzlesSolved() {
	PuzzlesSolved.Inc()
}

func RecordPuzzleAttempt(status string) {
	PuzzleAttempts.WithLabelValues(status).Inc()
}

func RecordTokenVerification(status string) {
	TokenVerifications.WithLabelValues(status).Inc()
}
