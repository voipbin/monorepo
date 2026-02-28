package circuitbreakerhandler

//go:generate mockgen -package circuitbreakerhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// CircuitBreakerHandler manages circuit breakers for multiple targets.
type CircuitBreakerHandler interface {
	Allow(target string) error
	RecordSuccess(target string)
	RecordFailure(target string)
}

type circuitBreakerHandler struct {
	mu       sync.Mutex
	breakers map[string]*breaker

	promStateTransitions *prometheus.CounterVec
	promState            *prometheus.GaugeVec
	promRejected         *prometheus.CounterVec
}

// NewCircuitBreakerHandler creates a new CircuitBreakerHandler with Prometheus metrics.
func NewCircuitBreakerHandler(namespace string) CircuitBreakerHandler {
	return newCircuitBreakerHandlerWithRegisterer(namespace, prometheus.DefaultRegisterer)
}

// newCircuitBreakerHandlerWithRegisterer is for testing — allows injecting a custom Prometheus registry.
func newCircuitBreakerHandlerWithRegisterer(namespace string, registerer prometheus.Registerer) CircuitBreakerHandler {
	h := &circuitBreakerHandler{
		breakers: make(map[string]*breaker),
	}

	h.promStateTransitions = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_state_transitions_total",
			Help:      "Total number of circuit breaker state transitions.",
		},
		[]string{"target", "from", "to"},
	)

	h.promState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_state",
			Help:      "Current circuit breaker state per target (0=closed, 1=half-open, 2=open).",
		},
		[]string{"target"},
	)

	h.promRejected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "circuitbreaker_rejected_total",
			Help:      "Total number of requests rejected by the circuit breaker.",
		},
		[]string{"target"},
	)

	registerer.MustRegister(
		h.promStateTransitions,
		h.promState,
		h.promRejected,
	)

	return h
}

func (h *circuitBreakerHandler) getOrCreateBreaker(target string) *breaker {
	h.mu.Lock()
	defer h.mu.Unlock()

	b, ok := h.breakers[target]
	if !ok {
		b = newBreaker()
		h.breakers[target] = b
	}
	return b
}

func (h *circuitBreakerHandler) Allow(target string) error {
	b := h.getOrCreateBreaker(target)

	prevState := b.getState()
	err := b.allow()
	newState := b.getState()

	if prevState != newState {
		h.promStateTransitions.WithLabelValues(target, prevState.String(), newState.String()).Inc()
		h.promState.WithLabelValues(target).Set(float64(newState))
		log.Warnf("Circuit breaker state changed for target %s: %s -> %s", target, prevState.String(), newState.String())
	}

	if err != nil {
		h.promRejected.WithLabelValues(target).Inc()
		return fmt.Errorf("%w for target: %s", ErrCircuitOpen, target)
	}

	return nil
}

func (h *circuitBreakerHandler) RecordSuccess(target string) {
	b := h.getOrCreateBreaker(target)

	prevState := b.getState()
	b.recordSuccess()
	newState := b.getState()

	if prevState != newState {
		h.promStateTransitions.WithLabelValues(target, prevState.String(), newState.String()).Inc()
		h.promState.WithLabelValues(target).Set(float64(newState))
		log.Warnf("Circuit breaker state changed for target %s: %s -> %s", target, prevState.String(), newState.String())
	}
}

func (h *circuitBreakerHandler) RecordFailure(target string) {
	b := h.getOrCreateBreaker(target)

	prevState := b.getState()
	b.recordFailure()
	newState := b.getState()

	if prevState != newState {
		h.promStateTransitions.WithLabelValues(target, prevState.String(), newState.String()).Inc()
		h.promState.WithLabelValues(target).Set(float64(newState))
		log.Warnf("Circuit breaker state changed for target %s: %s -> %s", target, prevState.String(), newState.String())
	}
}
