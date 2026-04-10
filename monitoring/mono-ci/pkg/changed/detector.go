package changed

import (
	"os/exec"
	"path/filepath"
	"strings"

	"monorepo/monitoring/mono-ci/pkg/discovery"
)

// DetectChangedModules returns the subset of modules that have files changed
// compared to the given git base ref (e.g., "origin/main", "HEAD~1").
// It runs `git diff --name-only` to find changed files.
func DetectChangedModules(repoRoot string, baseRef string, modules []discovery.Module) ([]discovery.Module, error) {
	changedFiles, err := getChangedFiles(repoRoot, baseRef)
	if err != nil {
		return nil, err
	}

	return matchModules(repoRoot, changedFiles, modules), nil
}

// DetectUncommittedModules returns modules that have uncommitted changes (staged + unstaged).
func DetectUncommittedModules(repoRoot string, modules []discovery.Module) ([]discovery.Module, error) {
	changedFiles, err := getUncommittedFiles(repoRoot)
	if err != nil {
		return nil, err
	}

	return matchModules(repoRoot, changedFiles, modules), nil
}

func getChangedFiles(repoRoot, baseRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseRef)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseLines(string(out)), nil
}

func getUncommittedFiles(repoRoot string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Also include staged changes
	cmd2 := exec.Command("git", "diff", "--name-only", "--cached")
	cmd2.Dir = repoRoot
	out2, err := cmd2.Output()
	if err != nil {
		return nil, err
	}

	files := parseLines(string(out))
	files = append(files, parseLines(string(out2))...)
	return deduplicate(files), nil
}

func matchModules(repoRoot string, changedFiles []string, modules []discovery.Module) []discovery.Module {
	var matched []discovery.Module
	seen := make(map[string]bool)

	for _, m := range modules {
		relPath, err := filepath.Rel(repoRoot, m.Path)
		if err != nil {
			continue
		}
		// Normalize to forward slashes for comparison
		relPath = filepath.ToSlash(relPath)

		for _, f := range changedFiles {
			f = filepath.ToSlash(f)
			if strings.HasPrefix(f, relPath+"/") || f == relPath {
				if !seen[m.Name] {
					matched = append(matched, m)
					seen[m.Name] = true
				}
				break
			}
		}
	}

	return matched
}

func parseLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func deduplicate(items []string) []string {
	seen := make(map[string]bool, len(items))
	var result []string
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
