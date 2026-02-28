package circuitbreakerhandler

import (
	"errors"
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func newTestHandlerWithRegistry() (CircuitBreakerHandler, *prometheus.Registry) {
	reg := prometheus.NewPedanticRegistry()
	return newCircuitBreakerHandlerWithRegisterer("test", reg), reg
}

func newTestHandler() CircuitBreakerHandler {
	h, _ := newTestHandlerWithRegistry()
	return h
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

func TestHandlerPrometheusRejectedMetric(t *testing.T) {
	h, reg := newTestHandlerWithRegistry()

	// Trip circuit to Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	// This should be rejected and increment the rejected counter
	_ = h.Allow("target-a")
	_ = h.Allow("target-a")

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	var rejectedCount float64
	for _, mf := range mfs {
		if mf.GetName() == "test_circuitbreaker_rejected_total" {
			for _, m := range mf.GetMetric() {
				rejectedCount += m.GetCounter().GetValue()
			}
		}
	}

	if rejectedCount != 2 {
		t.Errorf("expected 2 rejected requests, got %v", rejectedCount)
	}
}

func TestHandlerPrometheusStateTransitionMetric(t *testing.T) {
	h, reg := newTestHandlerWithRegistry()

	// Trip circuit: Closed -> Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	var transitionCount float64
	for _, mf := range mfs {
		if mf.GetName() == "test_circuitbreaker_state_transitions_total" {
			for _, m := range mf.GetMetric() {
				transitionCount += m.GetCounter().GetValue()
			}
		}
	}

	if transitionCount < 1 {
		t.Errorf("expected at least 1 state transition, got %v", transitionCount)
	}
}

func TestHandlerPrometheusStateGauge(t *testing.T) {
	h, reg := newTestHandlerWithRegistry()

	// Trip circuit: Closed -> Open
	for i := 0; i < defaultFailureThreshold; i++ {
		_ = h.Allow("target-a")
		h.RecordFailure("target-a")
	}

	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("failed to gather metrics: %v", err)
	}

	var stateGauge *dto.Metric
	for _, mf := range mfs {
		if mf.GetName() == "test_circuitbreaker_state" {
			for _, m := range mf.GetMetric() {
				stateGauge = m
			}
		}
	}

	if stateGauge == nil {
		t.Fatal("expected circuitbreaker_state metric, got none")
	}

	// StateOpen = 1 (iota: 0=Closed, 1=Open, 2=HalfOpen)
	if stateGauge.GetGauge().GetValue() != float64(StateOpen) {
		t.Errorf("expected state gauge to be %v (Open), got %v", float64(StateOpen), stateGauge.GetGauge().GetValue())
	}
}

func TestHandlerConcurrentAccess(t *testing.T) {
	h := newTestHandler()
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = h.Allow("target-a")
			h.RecordFailure("target-a")
			_ = h.Allow("target-b")
			h.RecordSuccess("target-b")
		}()
	}
	wg.Wait()
}
