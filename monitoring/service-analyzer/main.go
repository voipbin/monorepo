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
		runGraph(scanner)
	case "metrics":
		runMetrics(scanner)
	case "impact":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "usage: service-analyzer impact <service-name>\n")
			fmt.Fprintf(os.Stderr, "example: service-analyzer impact call-manager\n")
			os.Exit(1)
		}
		runImpact(scanner, os.Args[2])
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
  service-analyzer <command> [args]

Commands:
  graph           Generate Mermaid dependency graph
  metrics         Show fan-in/fan-out metrics for each service
  impact <svc>    Analyze cascade impact if <svc> goes down
  list            List all discovered services
  help            Show this help message

Examples:
  service-analyzer graph              # print Mermaid graph to stdout
  service-analyzer graph > deps.md    # save to file
  service-analyzer metrics            # show dependency metrics table
  service-analyzer impact call-manager  # what breaks if call-manager is down?
  service-analyzer list               # list all services

Environment:
  The tool auto-detects the monorepo root by walking up from the current
  directory looking for bin-common-handler/. Alternatively, set the working
  directory to the monorepo root before running.`)
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

func runGraph(scanner *analyzer.Scanner) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
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

func runImpact(scanner *analyzer.Scanner, serviceName string) {
	g, err := scanner.BuildGraph()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building graph: %v\n", err)
		os.Exit(1)
	}

	// validate service exists
	found := false
	for _, svc := range g.Services {
		if svc.Name == serviceName {
			found = true
			break
		}
	}
	if !found {
		fmt.Fprintf(os.Stderr, "error: service %q not found\n", serviceName)
		fmt.Fprintf(os.Stderr, "hint: run 'service-analyzer list' to see available services\n")
		os.Exit(1)
	}

	result := reporter.AnalyzeImpact(g, serviceName)
	fmt.Print(reporter.FormatImpact(result))
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
