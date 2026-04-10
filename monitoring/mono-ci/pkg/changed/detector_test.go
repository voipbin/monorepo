package changed

import (
	"testing"

	"monorepo/monitoring/mono-ci/pkg/discovery"
)

func TestMatchModules(t *testing.T) {
	t.Parallel()

	modules := []discovery.Module{
		{Name: "bin-call-manager", Path: "/repo/bin-call-manager"},
		{Name: "bin-api-manager", Path: "/repo/bin-api-manager"},
		{Name: "monitoring/mono-ci", Path: "/repo/monitoring/mono-ci"},
	}

	tests := []struct {
		name     string
		files    []string
		expected int
	}{
		{
			name:     "single module changed",
			files:    []string{"bin-call-manager/main.go", "bin-call-manager/pkg/handler.go"},
			expected: 1,
		},
		{
			name:     "multiple modules changed",
			files:    []string{"bin-call-manager/main.go", "bin-api-manager/pkg/api.go"},
			expected: 2,
		},
		{
			name:     "nested module changed",
			files:    []string{"monitoring/mono-ci/main.go"},
			expected: 1,
		},
		{
			name:     "no module matched",
			files:    []string{"docs/README.md", ".circleci/config.yml"},
			expected: 0,
		},
		{
			name:     "empty file list",
			files:    nil,
			expected: 0,
		},
		{
			name:     "duplicate files in same module",
			files:    []string{"bin-call-manager/a.go", "bin-call-manager/b.go"},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matched := matchModules("/repo", tt.files, modules)
			if len(matched) != tt.expected {
				t.Errorf("expected %d matched modules, got %d: %v", tt.expected, len(matched), matched)
			}
		})
	}
}

func TestParseLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"normal", "a.go\nb.go\nc.go", 3},
		{"trailing newline", "a.go\nb.go\n", 2},
		{"empty", "", 0},
		{"whitespace only", "  \n  \n", 0},
		{"mixed whitespace", "  a.go  \n  b.go  ", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			lines := parseLines(tt.input)
			if len(lines) != tt.expected {
				t.Errorf("expected %d lines, got %d: %v", tt.expected, len(lines), lines)
			}
		})
	}
}

func TestDeduplicate(t *testing.T) {
	t.Parallel()

	result := deduplicate([]string{"a", "b", "a", "c", "b"})
	if len(result) != 3 {
		t.Errorf("expected 3 unique items, got %d", len(result))
	}

	result = deduplicate(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 for nil input, got %d", len(result))
	}
}
