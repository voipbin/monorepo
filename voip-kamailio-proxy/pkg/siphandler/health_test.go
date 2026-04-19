package siphandler

import (
	"context"
	"testing"
	"time"
)

func Test_SendOptionsCheck_timeout(t *testing.T) {
	// Use TEST-NET (192.0.2.1) which never responds — RFC 5737
	result, err := SendOptionsCheck(context.Background(), "192.0.2.1", 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if result.Healthy {
		t.Errorf("Expected unhealthy for unreachable host, got healthy")
	}
	if result.ResponseCode != "timeout" {
		t.Errorf("Expected 'timeout' response code, got: %s", result.ResponseCode)
	}
}

func Test_SendOptionsCheck_invalidHost(t *testing.T) {
	result, err := SendOptionsCheck(context.Background(), "not-a-valid-host-xyz.invalid", 100*time.Millisecond)
	if err != nil {
		t.Errorf("Expected nil error, got: %v", err)
	}
	if result.Healthy {
		t.Errorf("Expected unhealthy for invalid host, got healthy")
	}
	if result.ResponseCode != "timeout" {
		t.Errorf("Expected 'timeout' response code, got: %s", result.ResponseCode)
	}
}
