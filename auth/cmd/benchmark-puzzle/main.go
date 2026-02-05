package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/status-im/proxy-common/auth/puzzle"
)

// BenchmarkParams holds the parameters for a single benchmark configuration
type BenchmarkParams struct {
	Difficulty   int
	Argon2Config puzzle.Argon2Config
}

// BenchmarkResult holds the results of a benchmark run
type BenchmarkResult struct {
	Params       BenchmarkParams
	Durations    []time.Duration
	Median       time.Duration
	Percentile90 time.Duration
}

func main() {
	fmt.Println("Puzzle Solver Benchmark (10 runs per config)")
	fmt.Println("============================================")
	fmt.Println()

	// Define test configurations
	configs := []BenchmarkParams{
		// difficulty 1, 2, 3 with Memory=16KB
		{Difficulty: 1, Argon2Config: puzzle.Argon2Config{MemoryKB: 16, Time: 4, Threads: 4, KeyLen: 32}},
		{Difficulty: 2, Argon2Config: puzzle.Argon2Config{MemoryKB: 16, Time: 4, Threads: 4, KeyLen: 32}},
		{Difficulty: 3, Argon2Config: puzzle.Argon2Config{MemoryKB: 16, Time: 4, Threads: 4, KeyLen: 32}},
		// difficulty 1, 2, 3 with Memory=64KB
		{Difficulty: 1, Argon2Config: puzzle.Argon2Config{MemoryKB: 64, Time: 4, Threads: 4, KeyLen: 32}},
		{Difficulty: 2, Argon2Config: puzzle.Argon2Config{MemoryKB: 64, Time: 4, Threads: 4, KeyLen: 32}},
		{Difficulty: 3, Argon2Config: puzzle.Argon2Config{MemoryKB: 64, Time: 4, Threads: 4, KeyLen: 32}},
	}

	// Run benchmarks
	results := make([]BenchmarkResult, 0, len(configs))
	for i, config := range configs {
		fmt.Printf("Running benchmark %d/%d (Difficulty=%d, Memory=%dKB)...\n",
			i+1, len(configs), config.Difficulty, config.Argon2Config.MemoryKB)
		result := runBenchmark(config, 10)
		results = append(results, result)
	}

	fmt.Println()
	fmt.Println("Results:")
	fmt.Println()

	// Print table header
	fmt.Printf("| %-10s | %-10s | %-4s | %-7s | %-6s | %-12s | %-12s |\n",
		"Difficulty", "Memory(KB)", "Time", "Threads", "KeyLen", "Median", "P90")
	fmt.Printf("|------------|------------|------|---------|--------|--------------|--------------||\n")

	// Print results
	for _, result := range results {
		fmt.Printf("| %-10d | %-10d | %-4d | %-7d | %-6d | %-12s | %-12s |\n",
			result.Params.Difficulty,
			result.Params.Argon2Config.MemoryKB,
			result.Params.Argon2Config.Time,
			result.Params.Argon2Config.Threads,
			result.Params.Argon2Config.KeyLen,
			formatDuration(result.Median),
			formatDuration(result.Percentile90))
	}

	fmt.Println()
	fmt.Println("Benchmark completed!")
}

// runBenchmark runs the puzzle solver multiple times and collects timing data
func runBenchmark(params BenchmarkParams, iterations int) BenchmarkResult {
	durations := make([]time.Duration, 0, iterations)
	jwtSecret := "test-secret-key-for-benchmarking"

	for i := 0; i < iterations; i++ {
		// Generate a new puzzle
		p, err := puzzle.Generate(params.Difficulty, 5, jwtSecret)
		if err != nil {
			fmt.Printf("Error generating puzzle: %v\n", err)
			continue
		}

		// Measure solve time
		startTime := time.Now()
		_, err = puzzle.Solve(p, params.Argon2Config)
		duration := time.Since(startTime)

		if err != nil {
			fmt.Printf("Error solving puzzle: %v\n", err)
			continue
		}

		durations = append(durations, duration)
	}

	// Calculate statistics
	median := calculateMedian(durations)
	p90 := calculatePercentile(durations, 90)

	return BenchmarkResult{
		Params:       params,
		Durations:    durations,
		Median:       median,
		Percentile90: p90,
	}
}

// calculateMedian calculates the median of a slice of durations
func calculateMedian(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// calculatePercentile calculates the specified percentile of a slice of durations
func calculatePercentile(durations []time.Duration, percentile float64) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(durations))
	copy(sorted, durations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	n := len(sorted)
	index := int(float64(n-1) * percentile / 100.0)
	if index >= n {
		index = n - 1
	}

	return sorted[index]
}

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return fmt.Sprintf("%dns", d.Nanoseconds())
	} else if d < time.Millisecond {
		return fmt.Sprintf("%.1fµs", float64(d.Nanoseconds())/1000.0)
	} else if d < time.Second {
		return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
