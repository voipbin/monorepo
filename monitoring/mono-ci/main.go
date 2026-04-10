package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"monorepo/monitoring/mono-ci/pkg/changed"
	"monorepo/monitoring/mono-ci/pkg/discovery"
	"monorepo/monitoring/mono-ci/pkg/reporter"
	"monorepo/monitoring/mono-ci/pkg/runner"
)

const usage = `mono-ci - Monorepo CI orchestrator for Go microservices

Usage:
  mono-ci <command> [options]

Commands:
  list                   List all discovered Go modules
  test [--changed]       Run tests for all (or changed) modules
  vet [--changed]        Run go vet for all (or changed) modules
  build [--changed]      Run go build for all (or changed) modules
  validate [--changed]   Run vet + test + quality gates
  coverage               Show coverage report for all modules

Options:
  --root <path>          Monorepo root directory (auto-detected if omitted)
  --changed              Only run for modules with changes vs origin/main
  --base <ref>           Git base ref for change detection (default: origin/main)
  --parallel <n>         Max parallel module executions (default: 4)
  --prefix <prefix>      Filter modules by prefix (e.g., bin-, voip-)
  --config <path>        Config file for quality gates (default: .mono-ci.json)
  --json                 Output results as JSON
  --verbose              Show full test output
  --help                 Show this help

Examples:
  mono-ci list
  mono-ci test --changed --base origin/main
  mono-ci validate --prefix bin- --parallel 8
  mono-ci coverage --json
`

func main() {
	if len(os.Args) < 2 || os.Args[1] == "--help" || os.Args[1] == "-h" {
		fmt.Print(usage)
		os.Exit(0)
	}

	command := os.Args[1]

	root := findFlagValue("--root")
	if root == "" {
		var err error
		root, err = findMonorepoRoot()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: could not find monorepo root: %v\n", err)
			fmt.Fprintf(os.Stderr, "hint: use --root <path> to specify the monorepo root\n")
			os.Exit(1)
		}
	}

	baseRef := findFlagValue("--base")
	if baseRef == "" {
		baseRef = "origin/main"
	}

	parallelStr := findFlagValue("--parallel")
	parallel := 4
	if parallelStr != "" {
		if n, err := strconv.Atoi(parallelStr); err == nil && n > 0 {
			parallel = n
		}
	}

	prefix := findFlagValue("--prefix")
	configPath := findFlagValue("--config")
	if configPath == "" {
		configPath = filepath.Join(root, ".mono-ci.json")
	}

	changedOnly := hasFlag("--changed")
	jsonOutput := hasFlag("--json")
	verbose := hasFlag("--verbose")

	// Discover modules
	modules, err := discovery.DiscoverModules(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error discovering modules: %v\n", err)
		os.Exit(1)
	}

	// Filter by prefix
	if prefix != "" {
		modules = discovery.FilterByPrefix(modules, strings.Split(prefix, ","))
	}

	// Filter by changed
	if changedOnly {
		modules, err = changed.DetectChangedModules(root, baseRef, modules)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error detecting changes: %v\n", err)
			os.Exit(1)
		}
		if len(modules) == 0 {
			fmt.Println("No changed modules detected.")
			os.Exit(0)
		}
	}

	switch command {
	case "list":
		runList(modules, jsonOutput)
	case "test":
		runTest(modules, parallel, verbose, jsonOutput)
	case "vet":
		runVet(modules, parallel, jsonOutput)
	case "build":
		runBuild(modules, parallel, jsonOutput)
	case "validate":
		runValidate(modules, parallel, verbose, configPath, jsonOutput)
	case "coverage":
		runCoverage(modules, parallel, jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", command)
		fmt.Print(usage)
		os.Exit(1)
	}
}

func runList(modules []discovery.Module, jsonOutput bool) {
	if jsonOutput {
		fmt.Print("[")
		for i, m := range modules {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Printf(`{"name":"%s","path":"%s"}`, m.Name, m.Path)
		}
		fmt.Println("]")
		return
	}

	fmt.Printf("Discovered %d Go modules:\n\n", len(modules))
	for _, m := range modules {
		fmt.Printf("  %-40s %s\n", m.Name, m.Path)
	}
}

func runTest(modules []discovery.Module, parallel int, verbose, jsonOutput bool) {
	fmt.Printf("Running tests for %d modules (parallel=%d)...\n", len(modules), parallel)
	summary := runner.RunTests(modules, parallel, verbose)

	if jsonOutput {
		printJSON(summary)
	} else {
		fmt.Print(runner.FormatSummary(summary))
	}

	if summary.FailedCount > 0 {
		os.Exit(1)
	}
}

func runVet(modules []discovery.Module, parallel int, jsonOutput bool) {
	fmt.Printf("Running go vet for %d modules (parallel=%d)...\n", len(modules), parallel)
	summary := runner.RunVet(modules, parallel)

	if jsonOutput {
		printJSON(summary)
	} else {
		fmt.Print(runner.FormatSummary(summary))
	}

	if summary.FailedCount > 0 {
		os.Exit(1)
	}
}

func runBuild(modules []discovery.Module, parallel int, jsonOutput bool) {
	fmt.Printf("Running go build for %d modules (parallel=%d)...\n", len(modules), parallel)
	summary := runner.RunBuild(modules, parallel)

	if jsonOutput {
		printJSON(summary)
	} else {
		fmt.Print(runner.FormatSummary(summary))
	}

	if summary.FailedCount > 0 {
		os.Exit(1)
	}
}

func runValidate(modules []discovery.Module, parallel int, verbose bool, configPath string, jsonOutput bool) {
	cfg := reporter.LoadConfig(configPath)

	fmt.Printf("Running validation for %d modules (parallel=%d)...\n", len(modules), parallel)

	// Step 1: go vet
	fmt.Println("\n--- go vet ---")
	vetSummary := runner.RunVet(modules, parallel)
	if !jsonOutput {
		fmt.Print(runner.FormatSummary(vetSummary))
	}

	// Step 2: go test
	fmt.Println("\n--- go test ---")
	testSummary := runner.RunTests(modules, parallel, verbose)
	if !jsonOutput {
		fmt.Print(runner.FormatSummary(testSummary))
	}

	// Step 3: quality gates
	violations := reporter.ValidateResults(testSummary, &vetSummary, cfg)

	if jsonOutput {
		printJSON(testSummary)
	}

	fmt.Print(reporter.FormatViolations(violations))

	if len(violations) > 0 {
		os.Exit(1)
	}
}

func runCoverage(modules []discovery.Module, parallel int, jsonOutput bool) {
	fmt.Printf("Running coverage analysis for %d modules (parallel=%d)...\n", len(modules), parallel)
	summary := runner.RunTests(modules, parallel, false)

	if jsonOutput {
		printJSON(summary)
		return
	}

	fmt.Printf("\n=== Coverage Report ===\n\n")

	// Sort by coverage (lowest first for attention)
	for _, r := range summary.Results {
		if r.Coverage < 0 {
			fmt.Printf("  %-40s  [no tests]\n", r.Module.Name)
		} else {
			bar := coverageBar(r.Coverage)
			fmt.Printf("  %-40s  %5.1f%%  %s\n", r.Module.Name, r.Coverage, bar)
		}
	}

	if summary.AvgCoverage > 0 {
		fmt.Printf("\n  Average coverage: %.1f%%\n", summary.AvgCoverage)
	}
}

func coverageBar(pct float64) string {
	const width = 30
	filled := int(pct / 100.0 * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func printJSON(summary runner.Summary) {
	data, err := reporter.GenerateJSON(summary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error generating JSON: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

func findMonorepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Look for bin-common-handler as a marker of the monorepo root
		if _, err := os.Stat(filepath.Join(dir, "bin-common-handler")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find monorepo root (looked for bin-common-handler)")
		}
		dir = parent
	}
}

func findFlagValue(flag string) string {
	for i, arg := range os.Args {
		if arg == flag && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
	}
	return ""
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}
