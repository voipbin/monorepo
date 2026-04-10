package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"monorepo/monitoring/service-analyzer/pkg/analyzer"
	"monorepo/monitoring/service-analyzer/pkg/reporter"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	jsonOutput := hasFlag("--json")

	// determine monorepo root: use --root flag or auto-detect
	root := findMonorepoRoot()
	if root == "" {
		fmt.Fprintf(os.Stderr, "error: could not find monorepo root (no bin-common-handler found)\n")
		fmt.Fprintf(os.Stderr, "hint: run from inside the monorepo or pass --root /path/to/monorepo\n")
		os.Exit(1)
	}

	scanner := analyzer.NewScanner(root)

	switch command {
	case "graph":
		runGraph(scanner, jsonOutput)
	case "metrics":
		runMetrics(scanner)
	case "impact":
		svcName := findPositionalArg(2)
		if svcName == "" {
			fmt.Fprintf(os.Stderr, "usage: service-analyzer impact <service-name>\n")
			fmt.Fprintf(os.Stderr, "example: service-analyzer impact call-manager\n")
			os.Exit(1)
		}
		runImpact(scanner, svcName, jsonOutput)
	case "callers":
		svcName := findPositionalArg(2)
		if svcName == "" {
			fmt.Fprintf(os.Stderr, "usage: service-analyzer callers <service-name>\n")
			fmt.Fprintf(os.Stderr, "example: service-analyzer callers call-manager\n")
			os.Exit(1)
		}
		runCallers(scanner, svcName)
	case "deps":
		svcName := findPositionalArg(2)
		if svcName == "" {
			fmt.Fprintf(os.Stderr, "usage: service-analyzer deps <service-name>\n")
			fmt.Fprintf(os.Stderr, "example: service-analyzer deps flow-manager\n")
			os.Exit(1)
		}
		runDeps(scanner, svcName)
	case "hotspots":
		runHotspots(scanner)
	case "circular":
		runCircular(scanner)
	case "list":
		runList(scanner)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`service-analyzer - VoIPbin monorepo service dependency analyzer

Usage:
  service-analyzer <command> [args] [--json] [--root /path]

Commands:
  graph             Generate Mermaid dependency graph
  metrics           Show fan-in/fan-out metrics for each service
  impact <svc>      Analyze cascade impact if <svc> goes down
  callers <svc>     Show which services call <svc> (reverse lookup)
  deps <svc>        Show which services <svc> depends on
  hotspots          Identify high-coupling architectural risk points
  circular          Detect circular RPC dependency chains
  list              List all discovered services
  help              Show this help message

Flags:
  --json            Output in JSON format (graph, impact)
  --root /path      Specify monorepo root directory

Examples:
  service-analyzer graph                  # Mermaid graph to stdout
  service-analyzer graph --json           # JSON graph to stdout
  service-analyzer metrics                # dependency metrics table
  service-analyzer impact call-manager    # what breaks if call-manager is down?
  service-analyzer callers call-manager   # who calls call-manager?
  service-analyzer deps flow-manager      # what does flow-manager depend on?
  service-analyzer list                   # list all services`)
}

func findMonorepoRoot() string {
	// check if --root is passed
	for i, arg := range os.Args {
		if arg == "--root" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "--root=") {
			return strings.TrimPrefix(arg, "--root=")
		}
	}

	// walk up from cwd to find monorepo root
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		marker := filepath.Join(dir, "bin-common-handler")
		if info, err := os.Stat(marker); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func runGraph(scanner *analyzer.Scanner, jsonOutput bool) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	if jsonOutput {
		data, jsonErr := reporter.GenerateJSON(g)
		if jsonErr != nil {
			fmt.Fprintf(os.Stderr, "error generating JSON: %v\n", jsonErr)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	mermaid := reporter.GenerateMermaid(g)

	fmt.Println("# Service Dependency Graph (Auto-Generated)")
	fmt.Println()
	fmt.Println("```mermaid")
	fmt.Print(mermaid)
	fmt.Println("```")
	fmt.Println()
	fmt.Printf("Total services: %d\n", len(g.Services))
	fmt.Printf("Total dependencies: %d (RPC: %d, Event: %d)\n",
		len(g.Dependencies), countByType(g.Dependencies, analyzer.DepRPC), countByType(g.Dependencies, analyzer.DepEvent))
}

func runMetrics(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	metrics := analyzer.ComputeMetrics(g)
	fmt.Print(reporter.FormatMetrics(metrics))
}

func runImpact(scanner *analyzer.Scanner, serviceName string, jsonOutput bool) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	validateServiceExists(g, serviceName)

	result := reporter.AnalyzeImpact(g, serviceName)

	if jsonOutput {
		data, jsonErr := reporter.GenerateImpactJSON(result)
		if jsonErr != nil {
			fmt.Fprintf(os.Stderr, "error generating JSON: %v\n", jsonErr)
			os.Exit(1)
		}
		fmt.Println(string(data))
		return
	}

	fmt.Print(reporter.FormatImpact(result))
}

func runCallers(scanner *analyzer.Scanner, serviceName string) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	validateServiceExists(g, serviceName)

	fmt.Printf("Services that call %s:\n\n", serviceName)

	rpcCount := 0
	eventCount := 0
	for _, dep := range g.Dependencies {
		if dep.To == serviceName && dep.Type == analyzer.DepRPC {
			fmt.Printf("  [RPC]   %-25s  methods: %s\n", dep.From, strings.Join(dep.Methods, ", "))
			rpcCount++
		}
	}
	for _, dep := range g.Dependencies {
		if dep.To == serviceName && dep.Type == analyzer.DepEvent {
			fmt.Printf("  [EVENT] %-25s  events: %s\n", dep.From, strings.Join(dep.Methods, ", "))
			eventCount++
		}
	}

	if rpcCount == 0 && eventCount == 0 {
		fmt.Println("  (none)")
	}
	fmt.Printf("\nTotal: %d RPC callers, %d event subscribers\n", rpcCount, eventCount)
}

func runDeps(scanner *analyzer.Scanner, serviceName string) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	validateServiceExists(g, serviceName)

	fmt.Printf("Dependencies of %s:\n\n", serviceName)

	rpcCount := 0
	eventCount := 0
	for _, dep := range g.Dependencies {
		if dep.From == serviceName && dep.Type == analyzer.DepRPC {
			fmt.Printf("  [RPC]   %-25s  methods: %s\n", dep.To, strings.Join(dep.Methods, ", "))
			rpcCount++
		}
	}
	for _, dep := range g.Dependencies {
		if dep.From == serviceName && dep.Type == analyzer.DepEvent {
			fmt.Printf("  [EVENT] %-25s  events: %s\n", dep.To, strings.Join(dep.Methods, ", "))
			eventCount++
		}
	}

	if rpcCount == 0 && eventCount == 0 {
		fmt.Println("  (none)")
	}
	fmt.Printf("\nTotal: %d RPC targets, %d event publishers\n", rpcCount, eventCount)
}

func runHotspots(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	hotspots := analyzer.DetectHotspots(g)

	fmt.Println("Architectural Hotspots (High Coupling Risk)")
	fmt.Println(strings.Repeat("=", 75))
	fmt.Println()
	fmt.Printf("%-25s %8s %8s %8s %10s\n", "Service", "Fan-In", "Fan-Out", "Total", "Risk")
	fmt.Println(strings.Repeat("-", 75))

	for _, h := range hotspots {
		riskLabel := h.RiskLevel
		switch h.RiskLevel {
		case "critical":
			riskLabel = "CRITICAL"
		case "high":
			riskLabel = "HIGH"
		case "medium":
			riskLabel = "MEDIUM"
		case "low":
			riskLabel = "low"
		}
		fmt.Printf("%-25s %8d %8d %8d %10s\n", h.Name, h.FanIn, h.FanOut, h.Coupling, riskLabel)
	}

	fmt.Println()
	fmt.Println("Risk levels: CRITICAL (>=20), HIGH (>=10), MEDIUM (>=5), low (<5)")
	fmt.Println("High fan-in + high fan-out = high blast radius for changes")
}

func runCircular(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	cycles := analyzer.DetectCircularDeps(g)

	if len(cycles) == 0 {
		fmt.Println("No circular RPC dependencies detected.")
		return
	}

	fmt.Printf("Circular RPC Dependencies Detected: %d\n", len(cycles))
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	for i, c := range cycles {
		chain := strings.Join(c.Services, " -> ")
		chain += " -> " + c.Services[0] // complete the loop
		fmt.Printf("  Cycle %d: %s\n", i+1, chain)
	}

	fmt.Println()
	fmt.Println("Circular dependencies increase coupling and can cause")
	fmt.Println("cascading failures. Consider breaking cycles by:")
	fmt.Println("  - Converting synchronous RPC to async events")
	fmt.Println("  - Extracting shared logic into a new service")
	fmt.Println("  - Using the Saga pattern for distributed workflows")
}

func validateServiceExists(g *analyzer.Graph, serviceName string) {
	for _, svc := range g.Services {
		if svc.Name == serviceName {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "error: service %q not found\n", serviceName)
	fmt.Fprintf(os.Stderr, "hint: run 'service-analyzer list' to see available services\n")
	os.Exit(1)
}

func runList(scanner *analyzer.Scanner) {
	services, err := scanner.DiscoverServices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Discovered %d services:\n\n", len(services))
	for _, svc := range services {
		fmt.Printf("  %-25s  %s\n", svc.Name, svc.Directory)
	}
}

func countByType(deps []analyzer.Dependency, t analyzer.DependencyType) int {
	count := 0
	for _, d := range deps {
		if d.Type == t {
			count++
		}
	}
	return count
}

func hasFlag(flag string) bool {
	for _, arg := range os.Args {
		if arg == flag {
			return true
		}
	}
	return false
}

// findPositionalArg finds the nth non-flag argument (skipping --key value pairs).
func findPositionalArg(pos int) string {
	idx := 0
	skip := false
	for _, arg := range os.Args {
		if skip {
			skip = false
			continue
		}
		if arg == "--root" {
			skip = true
			continue
		}
		if strings.HasPrefix(arg, "--") {
			continue
		}
		if idx == pos {
			return arg
		}
		idx++
	}
	return ""
}
