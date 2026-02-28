package circuitbreakerhandler

import (
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func newTestHandler() CircuitBreakerHandler {
	reg := prometheus.NewPedanticRegistry()
	return newCircuitBreakerHandlerWithRegisterer("test", reg)
}

func TestHandlerAllowNewTarget(t *testing.T) {
	h := newTestHandler()
	if err := h.Allow("target-a"); err != nil {
		t.Errorf("expected nil for new target, got %v", err)
	}
}

func TestHandlerIndependentBreakers(t *testing.T) {
	h := newTestHandler()

	for i := 0; i < defaultFailureThreshold; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	err := h.Allow("target-a")
	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen for target-a, got %v", err)
	}

	if err := h.Allow("target-b"); err != nil {
		t.Errorf("expected nil for target-b, got %v", err)
	}
}

func TestHandlerRecordSuccessResets(t *testing.T) {
	h := newTestHandler()

	for i := 0; i < defaultFailureThreshold-1; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	h.RecordSuccess("target-a")

	for i := 0; i < defaultFailureThreshold-1; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	if err := h.Allow("target-a"); err != nil {
		t.Errorf("expected nil after reset, got %v", err)
	}
}

func TestHandlerErrorWrapsTarget(t *testing.T) {
	h := newTestHandler()

	for i := 0; i < defaultFailureThreshold; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	err := h.Allow("target-a")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrCircuitOpen) {
		t.Errorf("expected error to wrap ErrCircuitOpen, got %v", err)
	}

	expected := "circuit breaker is open for target: target-a"
	if err.Error() != expected {
		t.Errorf("expected error message %q, got %q", expected, err.Error())
	}
}

func TestHandlerLazyCreation(t *testing.T) {
	h := newTestHandler()
	cbh := h.(*circuitBreakerHandler)

	if len(cbh.breakers) != 0 {
		t.Errorf("expected 0 breakers initially, got %d", len(cbh.breakers))
	}

	_ = h.Allow("target-a")
	_ = h.Allow("target-b")

	if len(cbh.breakers) != 2 {
		t.Errorf("expected 2 breakers after two targets, got %d", len(cbh.breakers))
	}

	_ = h.Allow("target-a")
	if len(cbh.breakers) != 2 {
		t.Errorf("expected still 2 breakers, got %d", len(cbh.breakers))
	}
}
