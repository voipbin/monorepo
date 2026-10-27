package k8swatchhandler

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
)

// feedTimeout bounds every wait on the fake apiserver. Success paths return as soon as the awaited
// event arrives, so a generous value costs nothing under -race.
const feedTimeout = 30 * time.Second

// annotationInitialEventsEnd marks the bookmark that ends a streaming initial list.
const annotationInitialEventsEnd = "k8s.io/initial-events-end"

// podFeed stands in for the apiserver's watch endpoint. Every watch the informers open gets its
// own controllable watcher, keyed by the label selector it asked for.
type podFeed struct {
	mu       sync.Mutex
	watchers map[string]*watch.FakeWatcher
	opened   chan string
}

// newPodFeed installs the feed on the handler's fake clientset.
func newPodFeed(h *k8sWatchHandler) *podFeed {
	f := &podFeed{
		watchers: map[string]*watch.FakeWatcher{},
		opened:   make(chan string, 64),
	}

	h.clientset.(*fake.Clientset).PrependWatchReactor("pods", func(action k8stesting.Action) (bool, watch.Interface, error) {
		selector := action.(k8stesting.WatchAction).GetWatchRestrictions().Labels.String()

		fw := watch.NewFake()
		f.mu.Lock()
		f.watchers[selector] = fw
		f.mu.Unlock()

		select {
		case f.opened <- selector:
		default:
		}

		return true, fw, nil
	})

	return f
}

// waitWatcher returns the latest watcher opened for the selector.
func (f *podFeed) waitWatcher(t *testing.T, selector string) *watch.FakeWatcher {
	t.Helper()

	deadline := time.Now().Add(feedTimeout)
	for time.Now().Before(deadline) {
		f.mu.Lock()
		fw := f.watchers[selector]
		f.mu.Unlock()
		if fw != nil {
			return fw
		}
		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("Wrong match. expect: a watch opened for %q, got: none within %v", selector, feedTimeout)
	return nil
}

// completeInitialList ends the streaming initial list of one informer, optionally replaying the
// given pods first, which is what lets the informer report that it has synced.
func completeInitialList(t *testing.T, fw *watch.FakeWatcher, existing ...*corev1.Pod) {
	t.Helper()

	for _, p := range existing {
		sendWithin(t, func() { fw.Add(p) })
	}

	end := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:            "initial-events-end",
		Namespace:       watchedNamespace,
		ResourceVersion: "100",
		Annotations:     map[string]string{annotationInitialEventsEnd: "true"},
	}}
	sendWithin(t, func() { fw.Action(watch.Bookmark, end) })
}

// sendWithin runs a blocking send on the fake watcher and fails instead of hanging when nothing
// receives it.
func sendWithin(t *testing.T, send func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		send()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(feedTimeout):
		t.Fatalf("Wrong match. expect: the informer to receive the event, got: no receiver within %v", feedTimeout)
	}
}

// publishedEvent is one PublishEvent call observed on the mock notify handler.
type publishedEvent struct {
	eventType string
	event     *smcontainer.Event
}

// Test_Run_watchEventsArePublished drives a whole Run through the watch seam: an update that
// replaces the pod's UID publishes the old generation's death and then the new start, and a
// delete publishes the death. The delivered events also exercise the informer's callbacks, which
// the direct handler tests do not reach.
func Test_Run_watchEventsArePublished(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockNotify := newTestHandler(t, mc)
	feed := newPodFeed(h)

	published := make(chan publishedEvent, 8)
	mockNotify.EXPECT().
		PublishEvent(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, eventType string, data any) {
			published <- publishedEvent{eventType: eventType, event: data.(*smcontainer.Event)}
		}).
		Times(3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- h.Run(ctx) }()

	oldPod := newPod("asterisk-call-abc", "uid-1", "asterisk-call", "3e:50:6b:43:bb:32")
	newerPod := newPod("asterisk-call-abc", "uid-2", "asterisk-call", "72:ce:24:e6:51:2f")

	for _, target := range watchTargets {
		fw := feed.waitWatcher(t, target.LabelSelector)
		if target.LabelSelector == "app=asterisk-call" {
			completeInitialList(t, fw, oldPod)
			continue
		}
		completeInitialList(t, fw)
	}

	callWatcher := feed.waitWatcher(t, "app=asterisk-call")
	sendWithin(t, func() { callWatcher.Modify(newerPod) })
	sendWithin(t, func() { callWatcher.Delete(newerPod) })

	expect := []publishedEvent{
		{eventType: smcontainer.EventTypeContainerDied, event: &smcontainer.Event{ContainerName: "asterisk-call-abc", Service: smcontainer.ServiceAsteriskCall, AsteriskID: "3e:50:6b:43:bb:32"}},
		{eventType: smcontainer.EventTypeContainerStarted, event: &smcontainer.Event{ContainerName: "asterisk-call-abc", Service: smcontainer.ServiceAsteriskCall, AsteriskID: "72:ce:24:e6:51:2f"}},
		{eventType: smcontainer.EventTypeContainerDied, event: &smcontainer.Event{ContainerName: "asterisk-call-abc", Service: smcontainer.ServiceAsteriskCall, AsteriskID: "72:ce:24:e6:51:2f"}},
	}

	for i, want := range expect {
		select {
		case got := <-published:
			if got.eventType != want.eventType || *got.event != *want.event {
				t.Errorf("Wrong match. event %d expect: %s %+v, got: %s %+v", i, want.eventType, *want.event, got.eventType, *got.event)
			}
		case <-time.After(feedTimeout):
			t.Fatalf("Wrong match. expect: event %d to be published, got: nothing within %v", i, feedTimeout)
		}
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Wrong match. expect: nil after the context is cancelled, got: %v", err)
		}
	case <-time.After(feedTimeout):
		t.Fatalf("Wrong match. expect: Run to return after cancel, got: still running")
	}
}

// Test_Run_watchFailuresExhaustTheBudget pins that a watch which keeps failing ends Run with an
// error once the budget is spent, instead of continuing without a working watch.
func Test_Run_watchFailuresExhaustTheBudget(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newTestHandler(t, mc)
	h.maxWatchFailures = 1

	h.clientset.(*fake.Clientset).PrependWatchReactor("pods", func(k8stesting.Action) (bool, watch.Interface, error) {
		return true, nil, errors.New("synthetic watch failure")
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- h.Run(ctx) }()

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("Wrong match. expect: error once the watch budget is spent, got: nil")
		}
		if !strings.Contains(err.Error(), "the pod watch stopped with an error") {
			t.Errorf("Wrong match. expect: the Run wrapper, got: %v", err)
		}
		if !strings.Contains(err.Error(), "refusing to keep running blind") {
			t.Errorf("Wrong match. expect: the budget exhaustion cause, got: %v", err)
		}
	case <-time.After(feedTimeout):
		t.Fatalf("Wrong match. expect: Run to return an error, got: still running")
	}
}

// Test_Run_initialSyncFailureIsReturned pins that an informer that cannot list ends Run with the
// sync error instead of hanging.
func Test_Run_initialSyncFailureIsReturned(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newTestHandler(t, mc)
	h.cacheSyncTimeout = 200 * time.Millisecond

	clientset := h.clientset.(*fake.Clientset)
	clientset.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("synthetic list denied")
	})
	clientset.PrependWatchReactor("pods", func(k8stesting.Action) (bool, watch.Interface, error) {
		return true, nil, errors.New("synthetic watch denied")
	})

	err := h.Run(context.Background())
	if err == nil {
		t.Fatalf("Wrong match. expect: error when the informer cannot sync, got: nil")
	}
	if !strings.Contains(err.Error(), "the pod watch stopped with an error") {
		t.Errorf("Wrong match. expect: the Run wrapper, got: %v", err)
	}
	if !strings.Contains(err.Error(), "did not complete its initial sync") {
		t.Errorf("Wrong match. expect: the sync failure cause, got: %v", err)
	}
}
