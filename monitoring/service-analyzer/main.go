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
	case "report":
		runReport(scanner)
	case "hotspots":
		runHotspots(scanner)
	case "circular":
		runCircular(scanner)
	case "paths":
		srcSvc := findPositionalArg(2)
		dstSvc := findPositionalArg(3)
		if srcSvc == "" || dstSvc == "" {
			fmt.Fprintf(os.Stderr, "usage: service-analyzer paths <from-service> <to-service>\n")
			fmt.Fprintf(os.Stderr, "example: service-analyzer paths call-manager campaign-manager\n")
			os.Exit(1)
		}
		runPaths(scanner, srcSvc, dstSvc)
	case "layers":
		runLayers(scanner)
	case "validate":
		runValidate(scanner)
	case "snapshot":
		runSnapshot(scanner)
	case "diff":
		runDiff()
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
  report            Full architectural health report with score
  hotspots          Identify high-coupling architectural risk points
  circular          Detect circular RPC dependency chains
  paths <a> <b>     Find all dependency paths between two services
  layers            Check for architectural layer violations (CI gate)
  validate          Combined CI gate: checks cycles, layers, hotspots (exit 1 on issues)
  snapshot          Save current dependency state to JSON file
  diff              Compare two snapshots to detect dependency changes
  list              List all discovered services
  help              Show this help message

Flags:
  --json            Output in JSON format (graph, impact)
  --root /path      Specify monorepo root directory
  --output /path    Output file path (snapshot)
  --config /path    Config file for validate (default: .service-analyzer.json)

Examples:
  service-analyzer graph                  # Mermaid graph to stdout
  service-analyzer graph --json           # JSON graph to stdout
  service-analyzer metrics                # dependency metrics table
  service-analyzer impact call-manager    # what breaks if call-manager is down?
  service-analyzer callers call-manager   # who calls call-manager?
  service-analyzer deps flow-manager      # what does flow-manager depend on?
  service-analyzer paths a b              # find all paths from service a to b
  service-analyzer snapshot               # save snapshot to deps-snapshot.json
  service-analyzer diff old.json new.json # compare two snapshots
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

func runReport(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(reporter.GenerateFullReport(g))
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

func runPaths(scanner *analyzer.Scanner, source, target string) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	validateServiceExists(g, source)
	validateServiceExists(g, target)

	shortest := analyzer.FindShortestPath(g, source, target)
	if shortest == nil {
		fmt.Printf("No dependency path from %s to %s\n", source, target)
		return
	}

	fmt.Printf("Dependency Paths: %s -> %s\n", source, target)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	fmt.Println("Shortest path:")
	printPath(shortest)
	fmt.Println()

	allPaths := analyzer.FindAllPaths(g, source, target, 8)
	if len(allPaths) > 1 {
		fmt.Printf("All paths (%d total):\n", len(allPaths))
		limit := 10
		if len(allPaths) < limit {
			limit = len(allPaths)
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  %d. ", i+1)
			printPathInline(&allPaths[i])
		}
		if len(allPaths) > 10 {
			fmt.Printf("  ... and %d more paths\n", len(allPaths)-10)
		}
	}
}

func printPath(p *analyzer.Path) {
	for i, svc := range p.Services {
		if i > 0 {
			depType := p.Types[i-1]
			if depType == analyzer.DepRPC {
				fmt.Printf("  --[RPC]--> %s\n", svc)
			} else {
				fmt.Printf("  --[EVENT]--> %s\n", svc)
			}
		} else {
			fmt.Printf("  %s\n", svc)
		}
	}
}

func printPathInline(p *analyzer.Path) {
	for i, svc := range p.Services {
		if i > 0 {
			if p.Types[i-1] == analyzer.DepRPC {
				fmt.Printf(" -[rpc]-> %s", svc)
			} else {
				fmt.Printf(" -[evt]-> %s", svc)
			}
		} else {
			fmt.Print(svc)
		}
	}
	fmt.Println()
}

func runLayers(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	layerMap := reporter.GetLayerMap()
	violations := analyzer.DetectLayerViolations(g, layerMap)

	if len(violations) == 0 {
		fmt.Println("No architectural layer violations detected.")
		fmt.Println("All service dependencies follow the expected layer hierarchy.")
		return
	}

	fmt.Printf("Architectural Layer Violations: %d\n", len(violations))
	fmt.Println(strings.Repeat("=", 65))
	fmt.Println()
	fmt.Printf("%-20s %-12s %-3s %-20s %-12s\n", "From", "Layer", "", "To", "Layer")
	fmt.Println(strings.Repeat("-", 65))

	for _, v := range violations {
		fmt.Printf("%-20s %-12s --> %-20s %-12s  [%s]\n",
			v.From, v.FromLayer, v.To, v.ToLayer, v.DepType)
	}

	fmt.Println()
	fmt.Println("Layer hierarchy (top depends on bottom):")
	fmt.Println("  Gateway > Business > Core")
	fmt.Println("  Business > Telephony > Core")
	fmt.Println("  * > Integration")
	fmt.Println("  Proxy -> Core only")
	fmt.Println()
	fmt.Println("Fix violations by refactoring to use allowed dependency paths.")

	os.Exit(1) // non-zero exit for CI gates
}

func runValidate(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	// load config from --config flag or default path
	configPath := findFlagValue("--config")
	if configPath == "" {
		configPath = ".service-analyzer.json"
	}
	cfg, err := analyzer.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Architectural Validation")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	issues := 0

	// 1. Check circular dependencies
	if !cfg.DisableCycleCheck {
		cycles := analyzer.DetectCircularDeps(g)
		directCycles := 0
		for _, c := range cycles {
			if len(c.Services) == 2 {
				directCycles++
			}
		}
		exceeds := directCycles > cfg.MaxCycles
		if exceeds {
			fmt.Printf("FAIL  Direct circular RPC deps: %d (max: %d)\n", directCycles, cfg.MaxCycles)
			for _, c := range cycles {
				if len(c.Services) == 2 {
					fmt.Printf("        %s <-> %s\n", c.Services[0], c.Services[1])
				}
			}
			issues += directCycles - cfg.MaxCycles
		} else if directCycles > 0 {
			fmt.Printf("PASS  Direct circular RPC deps: %d (within max: %d)\n", directCycles, cfg.MaxCycles)
		} else {
			fmt.Println("PASS  No direct circular RPC dependencies")
		}
	} else {
		fmt.Println("SKIP  Circular dependency check (disabled)")
	}

	// 2. Check layer violations
	if !cfg.DisableLayerCheck {
		layerMap := reporter.GetLayerMap()
		allViolations := analyzer.DetectLayerViolations(g, layerMap)
		violations := cfg.FilterViolations(allViolations)
		suppressed := len(allViolations) - len(violations)

		if len(violations) > 0 {
			fmt.Printf("FAIL  Layer violations: %d", len(violations))
			if suppressed > 0 {
				fmt.Printf(" (%d suppressed)", suppressed)
			}
			fmt.Println()
			for _, v := range violations {
				fmt.Printf("        %s (%s) -> %s (%s)\n", v.From, v.FromLayer, v.To, v.ToLayer)
			}
			issues += len(violations)
		} else {
			if suppressed > 0 {
				fmt.Printf("PASS  No new layer violations (%d suppressed)\n", suppressed)
			} else {
				fmt.Println("PASS  No architectural layer violations")
			}
		}
	} else {
		fmt.Println("SKIP  Layer violation check (disabled)")
	}

	// 3. Check critical hotspots
	hotspots := analyzer.DetectHotspots(g)
	critCount := 0
	for _, h := range hotspots {
		if h.RiskLevel == "critical" {
			critCount++
		}
	}
	if critCount > 0 {
		fmt.Printf("WARN  Critical coupling hotspots: %d\n", critCount)
		for _, h := range hotspots {
			if h.RiskLevel == "critical" {
				fmt.Printf("        %s (coupling: %d)\n", h.Name, h.Coupling)
			}
		}
	} else {
		fmt.Println("PASS  No critical coupling hotspots")
	}

	// 4. Health score
	highCount := 0
	for _, h := range hotspots {
		if h.RiskLevel == "high" {
			highCount++
		}
	}
	cycles := analyzer.DetectCircularDeps(g)
	directCycles := 0
	for _, c := range cycles {
		if len(c.Services) == 2 {
			directCycles++
		}
	}
	score := reporter.ComputeHealthScorePublic(len(g.Services), critCount, highCount, len(cycles), directCycles)
	fmt.Println()
	fmt.Printf("Health Score: %d/100\n", score)

	if cfg.MinHealthScore > 0 && score < cfg.MinHealthScore {
		fmt.Printf("FAIL  Health score %d below minimum %d\n", score, cfg.MinHealthScore)
		issues++
	}

	if issues > 0 {
		fmt.Printf("\nValidation FAILED with %d issue(s).\n", issues)
		fmt.Println("Hint: suppress known violations in .service-analyzer.json")
		os.Exit(1)
	}
	fmt.Println("\nValidation PASSED.")
}

func runSnapshot(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	snap := reporter.CreateSnapshot(g)

	outputPath := findFlagValue("--output")
	if outputPath == "" {
		outputPath = "deps-snapshot.json"
	}

	if err := reporter.SaveSnapshot(snap, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "error saving snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Snapshot saved to %s\n", outputPath)
	fmt.Printf("  Services: %d, Dependencies: %d (RPC: %d, Event: %d)\n",
		snap.Services, snap.Dependencies, snap.RPCCount, snap.EventCount)
	fmt.Printf("  Health Score: %d/100, Cycles: %d\n", snap.HealthScore, snap.Cycles)
}

func runDiff() {
	oldPath := findPositionalArg(2)
	newPath := findPositionalArg(3)

	if oldPath == "" || newPath == "" {
		fmt.Fprintf(os.Stderr, "usage: service-analyzer diff <old-snapshot.json> <new-snapshot.json>\n")
		fmt.Fprintf(os.Stderr, "example: service-analyzer diff deps-v1.json deps-v2.json\n")
		os.Exit(1)
	}

	old, err := reporter.LoadSnapshot(oldPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading old snapshot: %v\n", err)
		os.Exit(1)
	}

	new, err := reporter.LoadSnapshot(newPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading new snapshot: %v\n", err)
		os.Exit(1)
	}

	diff := reporter.DiffSnapshots(old, new)
	fmt.Print(reporter.FormatDiff(diff))
}

func findFlagValue(flag string) string {
	for i, arg := range os.Args {
		if arg == flag && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, flag+"=") {
			return strings.TrimPrefix(arg, flag+"=")
		}
	}
	return ""
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
