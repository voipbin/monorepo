package monitoringbackend

import (
	"context"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// stubBackend exists only to prove the interface is satisfiable from outside this package with
// nothing but a Run method — i.e. that the contract really is one method wide.
type stubBackend struct {
	err error
}

func (s *stubBackend) Run(ctx context.Context) error {
	<-ctx.Done()
	return s.err
}

func Test_MonitoringBackend_isOneMethodWide(t *testing.T) {
	var backend MonitoringBackend = &stubBackend{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := backend.Run(ctx); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

// Test_MetricsNamespace pins the shared prefix.
//
// This lives here rather than in either backend precisely so the two cannot drift onto different
// prefixes and split one logical counter into two differently-named series.
func Test_MetricsNamespace(t *testing.T) {
	if MetricsNamespace == "" {
		t.Fatalf("Wrong match. expect: a non-empty metrics namespace, got: empty")
	}

	if !strings.Contains(MetricsNamespace, "sentinel") {
		t.Errorf("Wrong match. expect: a sentinel-manager namespace, got: %s", MetricsNamespace)
	}
}

func Test_stateLabels(t *testing.T) {
	if StateStarted != "started" {
		t.Errorf("Wrong match. expect: started, got: %s", StateStarted)
	}
	if StateDied != "died" {
		t.Errorf("Wrong match. expect: died, got: %s", StateDied)
	}
	if StateStarted == StateDied {
		t.Errorf("Wrong match. expect: distinct state labels, got: both %s", StateStarted)
	}
}

// Test_sharedCountersAreRegisteredAndExported pins the two properties the K8s backend depends on:
// the counters must be EXPORTED (reachable from another package) and registered exactly once here
// rather than inside either backend.
func Test_sharedCountersAreRegisteredAndExported(t *testing.T) {
	if PromContainerStateChangeCounter == nil {
		t.Fatalf("Wrong match. expect: initialized PromContainerStateChangeCounter, got: nil")
	}
	if PromContainerUnresolvedAsteriskIDCounter == nil {
		t.Fatalf("Wrong match. expect: initialized PromContainerUnresolvedAsteriskIDCounter, got: nil")
	}

	// re-registering must fail: proof these are already in the default registry, so neither
	// backend needs to (or may) register them again.
	if err := prometheus.Register(PromContainerStateChangeCounter); err == nil {
		t.Errorf("Wrong match. expect: an AlreadyRegisteredError, got: nil")
	}
	if err := prometheus.Register(PromContainerUnresolvedAsteriskIDCounter); err == nil {
		t.Errorf("Wrong match. expect: an AlreadyRegisteredError, got: nil")
	}
}

// Test_sharedCountersAcceptBothBackendsLabelSets pins that the label cardinality works for either
// producer: a Docker container name and a Kubernetes pod name go into the same `container_name`
// slot, with no `backend` label anywhere (one process runs one backend for its whole life, so such
// a label could only ever hold one value).
func Test_sharedCountersAcceptBothBackendsLabelSets(t *testing.T) {
	tests := []struct {
		name string

		containerName string
		service       string
		state         string
	}{
		{
			name: "docker style container name",

			containerName: "voip-asterisk-call-docker-1",
			service:       "asterisk-call",
			state:         StateStarted,
		},
		{
			name: "kubernetes style pod name",

			containerName: "asterisk-call-7c9d5f8b6d-abcde",
			service:       "asterisk-call",
			state:         StateDied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := testutil.ToFloat64(PromContainerStateChangeCounter.WithLabelValues(tt.containerName, tt.service, tt.state))

			PromContainerStateChangeCounter.WithLabelValues(tt.containerName, tt.service, tt.state).Inc()

			after := testutil.ToFloat64(PromContainerStateChangeCounter.WithLabelValues(tt.containerName, tt.service, tt.state))
			if after-before != 1 {
				t.Errorf("Wrong match. expect: the counter to advance by 1, got: %v", after-before)
			}
		})
	}

	before := testutil.ToFloat64(PromContainerUnresolvedAsteriskIDCounter.WithLabelValues("asterisk-call-7c9d5f8b6d-abcde"))
	PromContainerUnresolvedAsteriskIDCounter.WithLabelValues("asterisk-call-7c9d5f8b6d-abcde").Inc()
	if delta := testutil.ToFloat64(PromContainerUnresolvedAsteriskIDCounter.WithLabelValues("asterisk-call-7c9d5f8b6d-abcde")) - before; delta != 1 {
		t.Errorf("Wrong match. expect: the counter to advance by 1, got: %v", delta)
	}
}
