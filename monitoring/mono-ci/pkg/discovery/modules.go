package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Module represents a Go module discovered in the monorepo.
type Module struct {
	Name      string // directory name (e.g., "bin-call-manager")
	Path      string // absolute path to the module directory
	GoModPath string // absolute path to go.mod
}

// DiscoverModules walks the monorepo root and finds all directories containing a go.mod file.
// It skips hidden directories, vendor directories, and nested modules within modules.
func DiscoverModules(root string) ([]Module, error) {
	var modules []Module

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		if shouldSkipDir(name) {
			continue
		}

		dirPath := filepath.Join(root, name)

		// Check for go.mod directly in this directory
		goModPath := filepath.Join(dirPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			modules = append(modules, Module{
				Name:      name,
				Path:      dirPath,
				GoModPath: goModPath,
			})
			continue
		}

		// Check one level deeper (e.g., monitoring/mono-ci, monitoring/service-analyzer)
		subEntries, err := os.ReadDir(dirPath)
		if err != nil {
			continue
		}
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() {
				continue
			}
			subName := subEntry.Name()
			if shouldSkipDir(subName) {
				continue
			}
			subDirPath := filepath.Join(dirPath, subName)
			subGoModPath := filepath.Join(subDirPath, "go.mod")
			if _, err := os.Stat(subGoModPath); err == nil {
				modules = append(modules, Module{
					Name:      name + "/" + subName,
					Path:      subDirPath,
					GoModPath: subGoModPath,
				})
			}
		}
	}

	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Name < modules[j].Name
	})

	return modules, nil
}

// FilterByPrefix returns only modules whose names start with any of the given prefixes.
func FilterByPrefix(modules []Module, prefixes []string) []Module {
	if len(prefixes) == 0 {
		return modules
	}

	var filtered []Module
	for _, m := range modules {
		for _, p := range prefixes {
			if strings.HasPrefix(m.Name, p) {
				filtered = append(filtered, m)
				break
			}
		}
	}
	return filtered
}

// FilterByNames returns only modules whose names match any of the given names exactly.
func FilterByNames(modules []Module, names []string) []Module {
	if len(names) == 0 {
		return modules
	}

	nameSet := make(map[string]bool, len(names))
	for _, n := range names {
		nameSet[n] = true
	}

	var filtered []Module
	for _, m := range modules {
		if nameSet[m.Name] {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".") ||
		name == "vendor" ||
		name == "node_modules" ||
		name == "testdata"
}
