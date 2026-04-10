package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"monorepo/monitoring/mono-ci/pkg/discovery"
)

// Result holds the outcome of running tests or vet for a single module.
type Result struct {
	Module     discovery.Module
	Action     string // "test", "vet", "build"
	Passed     bool
	Output     string
	Coverage   float64 // percentage, -1 if not applicable
	Duration   time.Duration
	Error      string
	TestCount  int
	SkipCount  int
	FailCount  int
}

// Summary aggregates results across all modules.
type Summary struct {
	Results       []Result
	TotalDuration time.Duration
	PassedCount   int
	FailedCount   int
	SkippedCount  int
	AvgCoverage   float64
}

var coveragePattern = regexp.MustCompile(`coverage:\s+([\d.]+)%`)
var testCountPattern = regexp.MustCompile(`(?m)^ok\s+`)
var failCountPattern = regexp.MustCompile(`(?m)^FAIL\s+`)
var noTestPattern = regexp.MustCompile(`\[no test files\]`)

// RunTests runs `go test ./...` with coverage for each module, up to maxParallel concurrently.
func RunTests(modules []discovery.Module, maxParallel int, verbose bool) Summary {
	return runAction(modules, maxParallel, "test", func(m discovery.Module) Result {
		args := []string{"test", "-count=1", "-coverprofile=coverage.out", "-v"}
		args = append(args, "./...")

		return executeGoCommand(m, "test", args)
	})
}

// RunVet runs `go vet ./...` for each module.
func RunVet(modules []discovery.Module, maxParallel int) Summary {
	return runAction(modules, maxParallel, "vet", func(m discovery.Module) Result {
		return executeGoCommand(m, "vet", []string{"vet", "./..."})
	})
}

// RunBuild runs `go build ./...` for each module.
func RunBuild(modules []discovery.Module, maxParallel int) Summary {
	return runAction(modules, maxParallel, "build", func(m discovery.Module) Result {
		return executeGoCommand(m, "build", []string{"build", "./..."})
	})
}

func runAction(modules []discovery.Module, maxParallel int, action string, fn func(discovery.Module) Result) Summary {
	if maxParallel <= 0 {
		maxParallel = 4
	}

	start := time.Now()
	results := make([]Result, len(modules))
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	for i, m := range modules {
		wg.Add(1)
		go func(idx int, mod discovery.Module) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = fn(mod)
		}(i, m)
	}
	wg.Wait()

	return buildSummary(results, time.Since(start))
}

func executeGoCommand(m discovery.Module, action string, args []string) Result {
	start := time.Now()
	cmd := exec.Command("go", args...)
	cmd.Dir = m.Path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)
	output := stdout.String() + stderr.String()

	r := Result{
		Module:   m,
		Action:   action,
		Duration: duration,
		Output:   output,
	}

	if err != nil {
		r.Passed = false
		r.Error = err.Error()
	} else {
		r.Passed = true
	}

	if action == "test" {
		r.Coverage = extractCoverage(output)
		r.TestCount, r.FailCount, r.SkipCount = countTests(output)
	}

	return r
}

func extractCoverage(output string) float64 {
	matches := coveragePattern.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		if noTestPattern.MatchString(output) {
			return -1 // no test files
		}
		return -1
	}

	// Average coverage across all packages
	var total float64
	var count int
	for _, match := range matches {
		val, err := strconv.ParseFloat(match[1], 64)
		if err == nil {
			total += val
			count++
		}
	}
	if count == 0 {
		return -1
	}
	return total / float64(count)
}

func countTests(output string) (total, failed, skipped int) {
	total = len(testCountPattern.FindAllString(output, -1)) + len(failCountPattern.FindAllString(output, -1))
	failed = len(failCountPattern.FindAllString(output, -1))

	// Count SKIP lines
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "--- SKIP:") {
			skipped++
		}
	}

	return total, failed, skipped
}

func buildSummary(results []Result, totalDuration time.Duration) Summary {
	s := Summary{
		Results:       results,
		TotalDuration: totalDuration,
	}

	var coverageSum float64
	var coverageCount int

	for _, r := range results {
		if r.Passed {
			s.PassedCount++
		} else {
			s.FailedCount++
		}
		s.SkippedCount += r.SkipCount

		if r.Coverage >= 0 {
			coverageSum += r.Coverage
			coverageCount++
		}
	}

	if coverageCount > 0 {
		s.AvgCoverage = coverageSum / float64(coverageCount)
	}

	return s
}

// FormatSummary produces a human-readable summary of results.
func FormatSummary(s Summary) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("\n=== %s Summary ===\n", strings.ToUpper(s.Results[0].Action)))
	b.WriteString(fmt.Sprintf("Total modules: %d\n", len(s.Results)))
	b.WriteString(fmt.Sprintf("Passed: %d  Failed: %d\n", s.PassedCount, s.FailedCount))
	b.WriteString(fmt.Sprintf("Duration: %s\n\n", s.TotalDuration.Round(time.Millisecond)))

	// Show failed modules first
	for _, r := range s.Results {
		if r.Passed {
			continue
		}
		b.WriteString(fmt.Sprintf("FAIL  %-40s  %s\n", r.Module.Name, r.Duration.Round(time.Millisecond)))
		if r.Error != "" {
			b.WriteString(fmt.Sprintf("      error: %s\n", r.Error))
		}
	}

	// Then passed modules
	for _, r := range s.Results {
		if !r.Passed {
			continue
		}
		coverStr := ""
		if r.Coverage >= 0 {
			coverStr = fmt.Sprintf("  cover: %.1f%%", r.Coverage)
		}
		b.WriteString(fmt.Sprintf("ok    %-40s  %s%s\n", r.Module.Name, r.Duration.Round(time.Millisecond), coverStr))
	}

	if s.AvgCoverage > 0 {
		b.WriteString(fmt.Sprintf("\nAverage coverage: %.1f%%\n", s.AvgCoverage))
	}

	return b.String()
}
