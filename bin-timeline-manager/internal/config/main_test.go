package config

import (
	"testing"
)

func TestGet(t *testing.T) {
	cfg := Get()
	if cfg == nil {
		t.Error("Get() returned nil")
	}
}
