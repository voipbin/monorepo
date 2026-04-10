package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config holds validation settings for the service analyzer.
type Config struct {
	// SuppressedViolations lists layer violations to ignore (format: "from->to")
	SuppressedViolations []string `json:"suppressed_violations,omitempty"`

	// MaxCycles is the maximum allowed direct circular dependencies (default: 0)
	MaxCycles int `json:"max_cycles,omitempty"`

	// MinHealthScore is the minimum acceptable health score (default: 0, meaning any score passes)
	MinHealthScore int `json:"min_health_score,omitempty"`

	// DisableLayerCheck skips layer violation checking
	DisableLayerCheck bool `json:"disable_layer_check,omitempty"`

	// DisableCycleCheck skips circular dependency checking
	DisableCycleCheck bool `json:"disable_cycle_check,omitempty"`
}

// DefaultConfig returns a config with all checks enabled and no suppressions.
func DefaultConfig() *Config {
	return &Config{}
}

// LoadConfig reads a config from a JSON file. Returns default config if file doesn't exist.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// IsViolationSuppressed checks if a specific layer violation is in the suppression list.
func (c *Config) IsViolationSuppressed(from, to string) bool {
	key := from + "->" + to
	for _, s := range c.SuppressedViolations {
		if s == key {
			return true
		}
	}
	return false
}

// FilterViolations removes suppressed violations from the list.
func (c *Config) FilterViolations(violations []LayerViolation) []LayerViolation {
	var filtered []LayerViolation
	for _, v := range violations {
		if !c.IsViolationSuppressed(v.From, v.To) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}
