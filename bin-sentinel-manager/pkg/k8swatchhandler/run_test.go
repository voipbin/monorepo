package k8swatchhandler

import (
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
)

// newTestHandler builds a handler wired to a fake clientset and a mock notify handler.
func newTestHandler(t *testing.T, mc *gomock.Controller, objects ...runtime.Object) (*k8sWatchHandler, *notifyhandler.MockNotifyHandler) {
	t.Helper()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &k8sWatchHandler{
		util:          utilhandler.NewMockUtilHandler(mc),
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		notifyHandler: mockNotify,

		clientset: fake.NewClientset(objects...),

		cacheSyncTimeout:    5 * time.Second,
		watchHealthInterval: 10 * time.Millisecond,
		maxWatchFailures:    3,
	}

	return h, mockNotify
}

// newPod builds a watched pod. An empty asteriskID leaves the annotation absent entirely, which is
// the real shape of a pod that died before voip-asterisk-proxy patched it.
func newPod(name string, uid string, app string, asteriskID string) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: watchedNamespace,
			UID:       types.UID(uid),
			Labels:    map[string]string{podLabelApp: app},
		},
	}

	if asteriskID != "" {
		p.Annotations = map[string]string{podAnnotationAsteriskID: asteriskID}
	}

	return p
}

func testBudget() *watchFailureBudget {
	return newWatchFailureBudget(3, nil)
}

// Test_handleUpdate_uidMismatch is THE highest-scrutiny test in this package.
//
// client-go's reflector only synthesizes a Deleted callback for keys ABSENT from a relist's object
// set. A pod deleted and replaced under the same name while the watch was interrupted is still
// present in the relist, so it arrives as a Replaced delta through UpdateFunc and NO delete
// callback ever fires for the dead generation. Without the UID comparison, that death — the exact
// event this service exists to detect — is dropped silently: no panic, no error, no counter.
//
// Mutation-checked in both directions: a same-UID update must NOT invent a death, and a
// different-UID update MUST publish the death of the old generation before the new one's start.
func Test_handleUpdate_uidMismatch(t *testing.T) {
	tests := []struct {
		name string

		oldPod *corev1.Pod
		newPod *corev1.Pod

		expectDied    *smcontainer.Event
		expectStarted *smcontainer.Event
	}{
		{
			name: "same uid publishes only started",

			oldPod: newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
			newPod: newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),

			expectDied: nil,
			expectStarted: &smcontainer.Event{
				ContainerName: "asterisk-call-abc",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},
		},
		{
			name: "same uid with a changed annotation still publishes only started",

			oldPod: newPod("asterisk-call-abc", "uid-1", "asterisk-call", ""),
			newPod: newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),

			expectDied: nil,
			expectStarted: &smcontainer.Event{
				ContainerName: "asterisk-call-abc",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},
		},
		{
			name: "different uid publishes the old generation's death first",

			oldPod: newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
			newPod: newPod("asterisk-call-abc", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f"),

			expectDied: &smcontainer.Event{
				ContainerName: "asterisk-call-abc",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "3e:50:6b:43:bb:32",
			},
			expectStarted: &smcontainer.Event{
				ContainerName: "asterisk-call-abc",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "72:ce:24:e6:51:2f",
			},
		},
		{
			name: "different uid where the dead generation never got an annotation",

			oldPod: newPod("asterisk-call-abc", "uid-1", "asterisk-call", ""),
			newPod: newPod("asterisk-call-abc", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f"),

			expectDied: &smcontainer.Event{
				ContainerName: "asterisk-call-abc",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "",
			},
			expectStarted: &smcontainer.Event{
				ContainerName: "asterisk-call-abc",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "72:ce:24:e6:51:2f",
			},
		},
		{
			name: "different uid on a conference pod",

			oldPod: newPod("asterisk-conference-xyz", "uid-1", "asterisk-conference", "aa:bb:cc:dd:ee:ff"),
			newPod: newPod("asterisk-conference-xyz", "uid-2", "asterisk-conference", "ff:ee:dd:cc:bb:aa"),

			expectDied: &smcontainer.Event{
				ContainerName: "asterisk-conference-xyz",
				Service:       smcontainer.ServiceAsteriskConference,
				AsteriskID:    "aa:bb:cc:dd:ee:ff",
			},
			expectStarted: &smcontainer.Event{
				ContainerName: "asterisk-conference-xyz",
				Service:       smcontainer.ServiceAsteriskConference,
				AsteriskID:    "ff:ee:dd:cc:bb:aa",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockNotify := newTestHandler(t, mc)

			ctx := context.Background()

			// ordering is asserted, not just occurrence: a consumer must see the death of the old
			// generation before the birth of its replacement.
			if tt.expectDied != nil {
				gomock.InOrder(
					mockNotify.EXPECT().PublishEvent(ctx, smcontainer.EventTypeContainerDied, tt.expectDied),
					mockNotify.EXPECT().PublishEvent(ctx, smcontainer.EventTypeContainerStarted, tt.expectStarted),
				)
			} else {
				mockNotify.EXPECT().PublishEvent(ctx, smcontainer.EventTypeContainerStarted, tt.expectStarted)
			}

			h.handleUpdate(ctx, testBudget(), tt.oldPod, tt.newPod)
		})
	}
}

// Test_handleUpdate_uidMismatchIsCounted pins that the inferred death is observable. A spike here
// is direct evidence of watch instability, exactly like a tombstone.
func Test_handleUpdate_uidMismatchIsCounted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockNotify := newTestHandler(t, mc)
	ctx := context.Background()

	mockNotify.EXPECT().PublishEvent(gomock.Any(), smcontainer.EventTypeContainerDied, gomock.Any())
	mockNotify.EXPECT().PublishEvent(gomock.Any(), smcontainer.EventTypeContainerStarted, gomock.Any())

	before := testutil.ToFloat64(promDiedDetectionCounter.WithLabelValues(diedSourceUIDMismatch))

	h.handleUpdate(ctx, testBudget(),
		newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
		newPod("asterisk-call-abc", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f"),
	)

	if delta := testutil.ToFloat64(promDiedDetectionCounter.WithLabelValues(diedSourceUIDMismatch)) - before; delta != 1 {
		t.Errorf("Wrong match. expect: the uid-mismatch counter to advance by 1, got: %v", delta)
	}
}

// Test_handleUpdate_malformedObjects pins that a non-pod payload degrades rather than panicking.
func Test_handleUpdate_malformedObjects(t *testing.T) {
	tests := []struct {
		name string

		oldObj any
		newObj any

		expectStarted bool
	}{
		{
			name: "new object is not a pod",

			oldObj: newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
			newObj: "not-a-pod",

			expectStarted: false,
		},
		{
			name: "new object is a typed nil pod",

			oldObj: newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
			newObj: (*corev1.Pod)(nil),

			expectStarted: false,
		},
		{
			name: "old object is not a pod, new one still publishes started",

			oldObj: "not-a-pod",
			newObj: newPod("asterisk-call-abc", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f"),

			expectStarted: true,
		},
		{
			name: "old object is a typed nil pod, new one still publishes started",

			oldObj: (*corev1.Pod)(nil),
			newObj: newPod("asterisk-call-abc", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f"),

			expectStarted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockNotify := newTestHandler(t, mc)

			if tt.expectStarted {
				mockNotify.EXPECT().PublishEvent(gomock.Any(), smcontainer.EventTypeContainerStarted, gomock.Any())
			}

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("handleUpdate panicked: %v", r)
				}
			}()

			h.handleUpdate(context.Background(), testBudget(), tt.oldObj, tt.newObj)
		})
	}
}

// Test_handleDelete pins the tombstone unwrap.
//
// client-go delivers cache.DeletedFinalStateUnknown instead of a pod whenever a deletion was
// missed during a watch interruption and only discovered on the next relist. A bare type assertion
// panics on that shape; the reflexive "assert with ok, return on mismatch" fix is worse, because
// it silently drops the death. The failure budget makes this path reachable BY DESIGN — its whole
// purpose is surviving the interruptions after which a tombstone appears.
func Test_handleDelete(t *testing.T) {
	pod := newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32")

	expect := &smcontainer.Event{
		ContainerName: "asterisk-call-abc",
		Service:       smcontainer.ServiceAsteriskCall,
		AsteriskID:    "3e:50:6b:43:bb:32",
	}

	tests := []struct {
		name string

		obj any

		expectEvent  *smcontainer.Event
		expectSource string
	}{
		{
			name: "live pod object",

			obj: pod,

			expectEvent:  expect,
			expectSource: diedSourceLive,
		},
		{
			name: "tombstone wrapping the same pod yields an identical event",

			obj: cache.DeletedFinalStateUnknown{Key: watchedNamespace + "/asterisk-call-abc", Obj: pod},

			expectEvent:  expect,
			expectSource: diedSourceTombstone,
		},
		{
			name: "tombstone wrapping a pod with no annotation",

			obj: cache.DeletedFinalStateUnknown{
				Key: watchedNamespace + "/asterisk-call-def",
				Obj: newPod("asterisk-call-def", "uid-2", "asterisk-call", ""),
			},

			expectEvent: &smcontainer.Event{
				ContainerName: "asterisk-call-def",
				Service:       smcontainer.ServiceAsteriskCall,
				AsteriskID:    "",
			},
			expectSource: diedSourceTombstone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockNotify := newTestHandler(t, mc)
			ctx := context.Background()

			mockNotify.EXPECT().PublishEvent(ctx, smcontainer.EventTypeContainerDied, tt.expectEvent)

			before := testutil.ToFloat64(promDiedDetectionCounter.WithLabelValues(tt.expectSource))

			h.handleDelete(ctx, testBudget(), tt.obj)

			if delta := testutil.ToFloat64(promDiedDetectionCounter.WithLabelValues(tt.expectSource)) - before; delta != 1 {
				t.Errorf("Wrong match. expect: the %s counter to advance by 1, got: %v", tt.expectSource, delta)
			}
		})
	}
}

// Test_handleDelete_unrecoverableIsCountedNotSilent pins that a payload we genuinely cannot resolve
// to a pod still leaves a trace. Nothing can be published, so this counter is the only evidence a
// death was observed and lost — it must never be a bare silent return.
func Test_handleDelete_unrecoverableIsCountedNotSilent(t *testing.T) {
	tests := []struct {
		name string

		obj any
	}{
		{
			name: "unrecognized shape",

			obj: "not-a-pod-at-all",
		},
		{
			name: "tombstone wrapping a non-pod",

			obj: cache.DeletedFinalStateUnknown{Key: "voip/whatever", Obj: "not-a-pod"},
		},
		{
			name: "tombstone wrapping a typed nil pod",

			obj: cache.DeletedFinalStateUnknown{Key: "voip/whatever", Obj: (*corev1.Pod)(nil)},
		},
		{
			name: "nil object",

			obj: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no PublishEvent expectation: the strict controller is the assertion that nothing is
			// published from an unresolvable payload.
			h, _ := newTestHandler(t, mc)

			before := testutil.ToFloat64(promDiedDetectionCounter.WithLabelValues(diedSourceUnrecoverable))

			defer func() {
				if r := recover(); r != nil {
					t.Errorf("handleDelete panicked: %v", r)
				}
			}()

			h.handleDelete(context.Background(), testBudget(), tt.obj)

			if delta := testutil.ToFloat64(promDiedDetectionCounter.WithLabelValues(diedSourceUnrecoverable)) - before; delta != 1 {
				t.Errorf("Wrong match. expect: the unrecoverable counter to advance by 1, got: %v", delta)
			}
		})
	}
}

// Test_publish_unwatchedServiceIsRejected pins that an unexpected `app` label is rejected at the
// publish boundary rather than passed through unmapped. A bare passthrough would produce an event
// bin-call-manager's filter silently never matches (design §8.3).
func Test_publish_unwatchedServiceIsRejected(t *testing.T) {
	tests := []struct {
		name string

		app string
	}{
		{name: "typo", app: "asterisk-cal"},
		{name: "unrelated workload", app: "kamailio"},
		{name: "empty label", app: ""},
		{name: "case mismatch", app: "Asterisk-Call"},
		{name: "whitespace", app: " asterisk-call"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no PublishEvent expectation at all: the strict controller is the assertion.
			h, _ := newTestHandler(t, mc)
			ctx := context.Background()

			p := newPod("some-pod", "uid-1", tt.app, "3e:50:6b:43:bb:32")

			h.publishStarted(ctx, p)
			h.publishDied(ctx, p, diedSourceLive)
		})
	}
}

// Test_publishDied_unresolvedAsteriskIDIsCounted pins that the K8s backend feeds the SAME shared
// counter the Docker backend does, so one Grafana panel covers both deployments.
//
// Scope is deaths only: a started event legitimately carries an empty id during the annotation
// patch window, and counting those would swamp the signal.
func Test_publishDied_unresolvedAsteriskIDIsCounted(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockNotify := newTestHandler(t, mc)
	ctx := context.Background()

	mockNotify.EXPECT().PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	unresolved := newPod("asterisk-call-abc", "uid-1", "asterisk-call", "")
	resolved := newPod("asterisk-call-def", "uid-2", "asterisk-call", "3e:50:6b:43:bb:32")

	beforeUnresolved := testutil.ToFloat64(monitoringbackendUnresolvedCounter(t, "asterisk-call-abc"))
	beforeResolved := testutil.ToFloat64(monitoringbackendUnresolvedCounter(t, "asterisk-call-def"))
	beforeStarted := testutil.ToFloat64(monitoringbackendUnresolvedCounter(t, "asterisk-call-ghi"))

	h.publishDied(ctx, unresolved, diedSourceLive)
	h.publishDied(ctx, resolved, diedSourceLive)
	// a started with an empty id must NOT count -- it is expected during the patch window.
	h.publishStarted(ctx, newPod("asterisk-call-ghi", "uid-3", "asterisk-call", ""))

	if delta := testutil.ToFloat64(monitoringbackendUnresolvedCounter(t, "asterisk-call-abc")) - beforeUnresolved; delta != 1 {
		t.Errorf("Wrong match. expect: an unresolved death to count 1, got: %v", delta)
	}
	if delta := testutil.ToFloat64(monitoringbackendUnresolvedCounter(t, "asterisk-call-def")) - beforeResolved; delta != 0 {
		t.Errorf("Wrong match. expect: a resolved death to count 0, got: %v", delta)
	}
	if delta := testutil.ToFloat64(monitoringbackendUnresolvedCounter(t, "asterisk-call-ghi")) - beforeStarted; delta != 0 {
		t.Errorf("Wrong match. expect: an unresolved START to count 0, got: %v", delta)
	}
}

// Test_waitForCacheSync_timeoutIsFatal pins the bounded startup deadline.
//
// A bare cache.WaitForCacheSync returns false only when its stop channel closes. Under the
// canonical failure this rewrite exists to catch — missing pod-reader RBAC — the initial List is
// denied and client-go retries forever, so the bare call blocks and never returns false. The
// informer below is deliberately never Run, which reproduces exactly that never-syncs shape.
func Test_waitForCacheSync_timeoutIsFatal(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newTestHandler(t, mc)
	h.cacheSyncTimeout = 50 * time.Millisecond

	informer := h.newTestInformer()

	err := h.waitForCacheSync(context.Background(), informer)
	if err == nil {
		t.Fatalf("Wrong match. expect: error on a sync that never completes, got: nil")
	}
}

// Test_waitForCacheSync_parentCancellationIsGraceful pins that a shutdown DURING startup is not
// reported as a failure — otherwise a SIGTERM arriving in the first seconds would exit non-zero.
func Test_waitForCacheSync_parentCancellationIsGraceful(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newTestHandler(t, mc)
	h.cacheSyncTimeout = 10 * time.Second

	informer := h.newTestInformer()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	if err := h.waitForCacheSync(ctx, informer); err != nil {
		t.Errorf("Wrong match. expect: nil on parent cancellation, got: %v", err)
	}
}

// newTestInformer builds an informer against the fake clientset without running it.
func (h *k8sWatchHandler) newTestInformer() cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		// See the matching comment in run.go -- WithContext ListWatch fields, matching what
		// client-go's Reflector actually calls.
		&cache.ListWatch{
			ListWithContextFunc: func(listCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				return h.clientset.CoreV1().Pods(watchedNamespace).List(listCtx, options)
			},
			WatchFuncWithContext: func(watchCtx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				return h.clientset.CoreV1().Pods(watchedNamespace).Watch(watchCtx, options)
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)
}

// Test_Run_gracefulShutdownReturnsNil pins the errgroup fan-in's shutdown semantics.
//
// errgroup.WithContext cancels its derived context as soon as any goroutine fails, so every
// sibling observes a cancelled context at that moment too. If cancellation synthesized an error
// here, a normal SIGTERM would exit non-zero and one real failure would be reported as several.
func Test_Run_gracefulShutdownReturnsNil(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newTestHandler(t, mc,
		newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
		newPod("asterisk-conference-xyz", "uid-2", "asterisk-conference", "aa:bb:cc:dd:ee:ff"),
	)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- h.Run(ctx) }()

	// let the informers sync, then shut down.
	time.Sleep(200 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wrong match. expect: nil on graceful shutdown, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("Wrong match. expect: Run to return on context cancel, got: still running")
	}
}

// Test_Run_addFuncStaysNoOp pins the deliberate no-op.
//
// Informers replay every already-existing pod as a synthetic Add on the initial list. Publishing
// started for those would misrepresent long-lived pods as freshly started on every sentinel
// restart. The strict mock — zero PublishEvent expectations — is the assertion.
func Test_Run_addFuncStaysNoOp(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newTestHandler(t, mc,
		newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32"),
		newPod("asterisk-call-def", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f"),
		newPod("asterisk-registrar-ghi", "uid-3", "asterisk-registrar", "ff:ee:dd:cc:bb:aa"),
	)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() { done <- h.Run(ctx) }()

	time.Sleep(300 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wrong match. expect: nil, got: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("Wrong match. expect: Run to return, got: still running")
	}
}
