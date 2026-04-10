package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}
	if len(cfg.SuppressedViolations) != 0 {
		t.Error("default config should have no suppressions")
	}
	if cfg.DisableLayerCheck {
		t.Error("layer check should be enabled by default")
	}
	if cfg.DisableCycleCheck {
		t.Error("cycle check should be enabled by default")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	cfg, err := LoadConfig("/nonexistent/config.json")
	if err != nil {
		t.Fatalf("expected default config for missing file, got error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected default config, got nil")
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	content := `{
  "suppressed_violations": ["call-manager->billing-manager", "flow-manager->queue-manager"],
  "max_cycles": 5,
  "min_health_score": 50
}`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.SuppressedViolations) != 2 {
		t.Errorf("suppressed = %d, want 2", len(cfg.SuppressedViolations))
	}
	if cfg.MaxCycles != 5 {
		t.Errorf("max_cycles = %d, want 5", cfg.MaxCycles)
	}
	if cfg.MinHealthScore != 50 {
		t.Errorf("min_health_score = %d, want 50", cfg.MinHealthScore)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := LoadConfig(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestIsViolationSuppressed(t *testing.T) {
	cfg := &Config{
		SuppressedViolations: []string{
			"call-manager->billing-manager",
			"flow-manager->queue-manager",
		},
	}

	if !cfg.IsViolationSuppressed("call-manager", "billing-manager") {
		t.Error("should be suppressed")
	}
	if cfg.IsViolationSuppressed("call-manager", "flow-manager") {
		t.Error("should not be suppressed")
	}
}

func TestFilterViolations(t *testing.T) {
	cfg := &Config{
		SuppressedViolations: []string{
			"a->b",
		},
	}

	violations := []LayerViolation{
		{From: "a", To: "b", FromLayer: LayerCore, ToLayer: LayerBusiness},
		{From: "c", To: "d", FromLayer: LayerProxy, ToLayer: LayerBusiness},
	}

	filtered := cfg.FilterViolations(violations)
	if len(filtered) != 1 {
		t.Errorf("filtered = %d, want 1", len(filtered))
	}
	if filtered[0].From != "c" {
		t.Errorf("remaining violation from = %q, want 'c'", filtered[0].From)
	}
}

func TestFilterViolations_AllSuppressed(t *testing.T) {
	cfg := &Config{
		SuppressedViolations: []string{"a->b", "c->d"},
	}

	violations := []LayerViolation{
		{From: "a", To: "b"},
		{From: "c", To: "d"},
	}

	filtered := cfg.FilterViolations(violations)
	if len(filtered) != 0 {
		t.Errorf("filtered = %d, want 0", len(filtered))
	}
}

func TestFilterViolations_NoneSuppressed(t *testing.T) {
	cfg := DefaultConfig()

	violations := []LayerViolation{
		{From: "a", To: "b"},
	}

	filtered := cfg.FilterViolations(violations)
	if len(filtered) != 1 {
		t.Errorf("filtered = %d, want 1", len(filtered))
	}
}
