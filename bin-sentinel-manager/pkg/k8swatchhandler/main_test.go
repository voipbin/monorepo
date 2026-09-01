package k8swatchhandler

import (
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"
	"k8s.io/client-go/kubernetes/fake"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/monitoringbackend"
)

// monitoringbackendUnresolvedCounter reaches the SHARED counter both backends populate.
func monitoringbackendUnresolvedCounter(t *testing.T, containerName string) prometheus.Counter {
	t.Helper()

	return monitoringbackend.PromContainerUnresolvedAsteriskIDCounter.WithLabelValues(containerName)
}

func Test_NewK8sWatchHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := NewK8sWatchHandler(
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		utilhandler.NewMockUtilHandler(mc),
		fake.NewClientset(),
	)

	if h == nil {
		t.Fatalf("Wrong match. expect: non-nil handler, got: nil")
	}

	res, ok := h.(*k8sWatchHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *k8sWatchHandler, got: %T", h)
	}
	if res.clientset == nil {
		t.Errorf("Wrong match. expect: an injected clientset, got: nil")
	}
	if res.cacheSyncTimeout != cacheSyncTimeout {
		t.Errorf("Wrong match. expect: %v, got: %v", cacheSyncTimeout, res.cacheSyncTimeout)
	}
	if res.watchHealthInterval != watchHealthInterval {
		t.Errorf("Wrong match. expect: %v, got: %v", watchHealthInterval, res.watchHealthInterval)
	}
	if res.maxWatchFailures != maxConsecutiveWatchFailures {
		t.Errorf("Wrong match. expect: %d, got: %d", maxConsecutiveWatchFailures, res.maxWatchFailures)
	}
}

// Test_NewK8sWatchHandler_tuningIsSane guards the two constants whose zero values would silently
// disable the protections they exist for.
func Test_NewK8sWatchHandler_tuningIsSane(t *testing.T) {
	if cacheSyncTimeout <= 0 {
		t.Errorf("Wrong match. expect: a positive cache sync deadline, got: %v", cacheSyncTimeout)
	}
	if maxConsecutiveWatchFailures <= 0 {
		t.Errorf("Wrong match. expect: a positive watch failure budget, got: %d", maxConsecutiveWatchFailures)
	}
	if watchHealthInterval <= 0 {
		t.Errorf("Wrong match. expect: a positive watch health interval, got: %v", watchHealthInterval)
	}
	// the health check must be able to fire several times inside the budget's lifetime, or the
	// resource-version recovery signal would never get a chance to reset the counter.
	if watchHealthInterval >= cacheSyncTimeout {
		t.Errorf("Wrong match. expect: the health interval below the sync deadline, got: %v vs %v", watchHealthInterval, cacheSyncTimeout)
	}
}

// Test_mapService pins that the pod `app` label is mapped through an EXPLICIT lookup, never passed
// through. Event.Service is a typed constant bin-call-manager filters on exactly; a raw label
// passthrough means a typo produces an event whose filter silently never matches (design §8.3).
func Test_mapService(t *testing.T) {
	tests := []struct {
		name string

		labelApp string

		expectService string
		expectOK      bool
	}{
		{
			name: "asterisk call",

			labelApp: "asterisk-call",

			expectService: smcontainer.ServiceAsteriskCall,
			expectOK:      true,
		},
		{
			name: "asterisk conference",

			labelApp: "asterisk-conference",

			expectService: smcontainer.ServiceAsteriskConference,
			expectOK:      true,
		},
		{
			name: "asterisk registrar",

			labelApp: "asterisk-registrar",

			expectService: smcontainer.ServiceAsteriskRegistrar,
			expectOK:      true,
		},
		{
			name: "typo is rejected",

			labelApp: "asterisk-cal",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "unrelated workload is rejected",

			labelApp: "kamailio",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "empty label is rejected",

			labelApp: "",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "case mismatch is rejected",

			labelApp: "Asterisk-Call",

			expectService: "",
			expectOK:      false,
		},
		{
			name: "leading whitespace is rejected",

			labelApp: " asterisk-call",

			expectService: "",
			expectOK:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resService, resOK := mapService(tt.labelApp)

			if resOK != tt.expectOK {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectOK, resOK)
			}
			if resService != tt.expectService {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectService, resService)
			}
		})
	}
}

// Test_watchTargets_matchTheServiceMap pins the two lists against each other.
//
// They are declared separately (one is a label-selector string, the other a mapping), so nothing
// structural stops a service being added to one and forgotten in the other. A selector with no
// mapping would watch pods and then reject every one of them at the publish boundary; a mapping
// with no selector would be dead code for pods nobody watches.
func Test_watchTargets_matchTheServiceMap(t *testing.T) {
	if len(watchTargets) != len(serviceByPodLabelApp) {
		t.Fatalf("Wrong match. expect: %d watch targets, got: %d", len(serviceByPodLabelApp), len(watchTargets))
	}

	seen := map[string]bool{}
	for _, target := range watchTargets {
		if target.Namespace != watchedNamespace {
			t.Errorf("Wrong match. expect: namespace %s, got: %s", watchedNamespace, target.Namespace)
		}

		labelApp, ok := strings.CutPrefix(target.LabelSelector, podLabelApp+"=")
		if !ok {
			t.Fatalf("Wrong match. expect: an %s= selector, got: %s", podLabelApp, target.LabelSelector)
		}

		if _, mapped := serviceByPodLabelApp[labelApp]; !mapped {
			t.Errorf("Wrong match. expect: selector %q to have a service mapping, got: none", target.LabelSelector)
		}
		seen[labelApp] = true
	}

	for labelApp := range serviceByPodLabelApp {
		if !seen[labelApp] {
			t.Errorf("Wrong match. expect: app label %q to have a watch target, got: none", labelApp)
		}
	}
}

// Test_watchTargets_golden pins the exact selectors, which must stay identical to the ones the
// pre-VOIP-1418 cmd/sentinel-manager hardcoded — a self-hoster's existing pods carry these labels.
func Test_watchTargets_golden(t *testing.T) {
	expect := []string{
		"app=asterisk-call",
		"app=asterisk-conference",
		"app=asterisk-registrar",
	}

	if len(watchTargets) != len(expect) {
		t.Fatalf("Wrong match. expect: %d targets, got: %d", len(expect), len(watchTargets))
	}
	for i, selector := range expect {
		if watchTargets[i].LabelSelector != selector {
			t.Errorf("Wrong match at %d. expect: %s, got: %s", i, selector, watchTargets[i].LabelSelector)
		}
	}

	if watchedNamespace != "voip" {
		t.Errorf("Wrong match. expect: voip, got: %s", watchedNamespace)
	}
}

func Test_podMetadataKeys(t *testing.T) {
	if podLabelApp != "app" {
		t.Errorf("Wrong match. expect: app, got: %s", podLabelApp)
	}
	// this exact annotation key is what voip-asterisk-proxy self-patches; changing it here would
	// silently strand every recovery.
	if podAnnotationAsteriskID != "asterisk-id" {
		t.Errorf("Wrong match. expect: asterisk-id, got: %s", podAnnotationAsteriskID)
	}
}

func Test_prometheusMetrics(t *testing.T) {
	if promWatchHealthCounter == nil {
		t.Errorf("Wrong match. expect: initialized promWatchHealthCounter, got: nil")
	}
	if promDiedDetectionCounter == nil {
		t.Errorf("Wrong match. expect: initialized promDiedDetectionCounter, got: nil")
	}
	// the shared counters must be reachable from this package, or the two backends would silently
	// populate different series.
	if monitoringbackend.PromContainerStateChangeCounter == nil {
		t.Errorf("Wrong match. expect: initialized shared state-change counter, got: nil")
	}
	if monitoringbackend.PromContainerUnresolvedAsteriskIDCounter == nil {
		t.Errorf("Wrong match. expect: initialized shared unresolved-id counter, got: nil")
	}
}

func Test_watchOutcomeAndSourceLabels(t *testing.T) {
	outcomes := map[string]string{
		watchOutcomeResynced:       "resynced",
		watchOutcomeTransientError: "transient-error",
		watchOutcomeFatal:          "fatal",
	}
	for got, expect := range outcomes {
		if got != expect {
			t.Errorf("Wrong match. expect: %s, got: %s", expect, got)
		}
	}
	if len(outcomes) != 3 {
		t.Errorf("Wrong match. expect: 3 distinct outcome labels, got: %d", len(outcomes))
	}

	sources := map[string]string{
		diedSourceLive:          "live",
		diedSourceTombstone:     "tombstone",
		diedSourceUIDMismatch:   "uid-mismatch",
		diedSourceUnrecoverable: "unrecoverable",
	}
	for got, expect := range sources {
		if got != expect {
			t.Errorf("Wrong match. expect: %s, got: %s", expect, got)
		}
	}
	if len(sources) != 4 {
		t.Errorf("Wrong match. expect: 4 distinct source labels, got: %d", len(sources))
	}
}

// Test_handlerSatisfiesTheSharedContract is a compile-time-ish guard duplicated as a runtime
// assertion: cmd/sentinel-manager selects backends purely through this interface.
func Test_handlerSatisfiesTheSharedContract(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	var backend monitoringbackend.MonitoringBackend = NewK8sWatchHandler(
		requesthandler.NewMockRequestHandler(mc),
		notifyhandler.NewMockNotifyHandler(mc),
		utilhandler.NewMockUtilHandler(mc),
		fake.NewClientset(),
	)

	if backend == nil {
		t.Errorf("Wrong match. expect: non-nil backend, got: nil")
	}
}
