package discovery

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverModules(t *testing.T) {
	t.Parallel()

	root := t.TempDir()

	// Create module directories with go.mod
	for _, name := range []string{"bin-call-manager", "bin-api-manager", "bin-common-handler"} {
		dir := filepath.Join(root, name)
		os.MkdirAll(dir, 0755)
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test/"+name+"\n\ngo 1.25.3\n"), 0644)
	}

	// Create nested module (e.g., monitoring/mono-ci)
	nestedDir := filepath.Join(root, "monitoring", "mono-ci")
	os.MkdirAll(nestedDir, 0755)
	os.WriteFile(filepath.Join(nestedDir, "go.mod"), []byte("module test/monitoring/mono-ci\n"), 0644)

	// Create a non-module directory (no go.mod)
	os.MkdirAll(filepath.Join(root, "docs"), 0755)

	// Create hidden directory (should be skipped)
	hiddenDir := filepath.Join(root, ".circleci")
	os.MkdirAll(hiddenDir, 0755)
	os.WriteFile(filepath.Join(hiddenDir, "go.mod"), []byte("module hidden\n"), 0644)

	modules, err := DiscoverModules(root)
	if err != nil {
		t.Fatalf("DiscoverModules failed: %v", err)
	}

	if len(modules) != 4 {
		t.Fatalf("expected 4 modules, got %d: %v", len(modules), modules)
	}

	// Verify sorted order
	expectedNames := []string{"bin-api-manager", "bin-call-manager", "bin-common-handler", "monitoring/mono-ci"}
	for i, m := range modules {
		if m.Name != expectedNames[i] {
			t.Errorf("module[%d] name = %q, want %q", i, m.Name, expectedNames[i])
		}
	}
}

func TestDiscoverModules_EmptyDir(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	modules, err := DiscoverModules(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(modules))
	}
}

func TestDiscoverModules_InvalidDir(t *testing.T) {
	t.Parallel()

	_, err := DiscoverModules("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestFilterByPrefix(t *testing.T) {
	t.Parallel()

	modules := []Module{
		{Name: "bin-api-manager"},
		{Name: "bin-call-manager"},
		{Name: "voip-asterisk-proxy"},
		{Name: "monitoring/mono-ci"},
	}

	filtered := FilterByPrefix(modules, []string{"bin-"})
	if len(filtered) != 2 {
		t.Errorf("expected 2 modules with prefix 'bin-', got %d", len(filtered))
	}

	filtered = FilterByPrefix(modules, []string{"voip-", "monitoring/"})
	if len(filtered) != 2 {
		t.Errorf("expected 2 modules with voip-/monitoring/ prefix, got %d", len(filtered))
	}

	// Empty prefix returns all
	filtered = FilterByPrefix(modules, nil)
	if len(filtered) != 4 {
		t.Errorf("empty prefix should return all, got %d", len(filtered))
	}
}

func TestFilterByNames(t *testing.T) {
	t.Parallel()

	modules := []Module{
		{Name: "bin-api-manager"},
		{Name: "bin-call-manager"},
		{Name: "voip-asterisk-proxy"},
	}

	filtered := FilterByNames(modules, []string{"bin-call-manager"})
	if len(filtered) != 1 || filtered[0].Name != "bin-call-manager" {
		t.Errorf("expected bin-call-manager, got %v", filtered)
	}

	// Empty names returns all
	filtered = FilterByNames(modules, nil)
	if len(filtered) != 3 {
		t.Errorf("empty names should return all, got %d", len(filtered))
	}
}

func TestShouldSkipDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		skip bool
	}{
		{".git", true},
		{".circleci", true},
		{"vendor", true},
		{"node_modules", true},
		{"testdata", true},
		{"bin-api-manager", false},
		{"monitoring", false},
	}

	for _, tt := range tests {
		if got := shouldSkipDir(tt.name); got != tt.skip {
			t.Errorf("shouldSkipDir(%q) = %v, want %v", tt.name, got, tt.skip)
		}
	}
}
