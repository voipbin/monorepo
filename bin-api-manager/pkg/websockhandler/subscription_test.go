package websockhandler

import (
	"context"
	"sync"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/hook"
	"monorepo/bin-api-manager/pkg/zmqsubhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
	"github.com/stretchr/testify/require"
)

// newSuperAdminIdentity returns an AuthIdentity that always passes validateTopics, so tests
// can focus on the scopeRefCount wiring rather than topic authorization rules.
func newSuperAdminIdentity() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{
		Permission: amagent.PermissionProjectSuperAdmin,
	})
}

func TestSubscriptionHandleMessage_AcquiresBindingOnSubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockZMQSub := zmqsubhandler.NewMockZMQSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		scopeRefCount: newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic"),
	}

	a := newSuperAdminIdentity()
	topic := "customer_id:5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb"

	mockZMQSub.EXPECT().Subscribe(topic).Return(nil)
	mockSock.EXPECT().QueueBind("pod-queue-1", "customer_id.5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb.#", "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)

	heldPatterns := make(map[string]int)
	var heldMu sync.Mutex

	m := &hook.Hook{
		Type:   hook.TypeSubscribe,
		Topics: []string{topic},
	}

	err := h.subscriptionHandleMessage(context.Background(), a, mockZMQSub, m, heldPatterns, &heldMu)
	require.NoError(t, err)
	require.Equal(t, 1, heldPatterns["customer_id.5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb.#"])
}

func TestSubscriptionHandleMessage_ReleasesBindingOnUnsubscribe(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockZMQSub := zmqsubhandler.NewMockZMQSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		scopeRefCount: newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic"),
	}

	a := newSuperAdminIdentity()
	topic := "customer_id:5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb"
	pattern := "customer_id.5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb.#"

	mockZMQSub.EXPECT().Subscribe(topic).Return(nil)
	mockSock.EXPECT().QueueBind("pod-queue-1", pattern, "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)
	mockZMQSub.EXPECT().Unsubscribe(topic).Return(nil)
	mockSock.EXPECT().QueueUnbind("pod-queue-1", pattern, "bin-manager.webhook-manager.event.topic", nil).Return(nil)

	heldPatterns := make(map[string]int)
	var heldMu sync.Mutex

	subMsg := &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{topic}}
	err := h.subscriptionHandleMessage(context.Background(), a, mockZMQSub, subMsg, heldPatterns, &heldMu)
	require.NoError(t, err)
	require.Equal(t, 1, heldPatterns[pattern])

	unsubMsg := &hook.Hook{Type: hook.TypeUnsubscribe, Topics: []string{topic}}
	err = h.subscriptionHandleMessage(context.Background(), a, mockZMQSub, unsubMsg, heldPatterns, &heldMu)
	require.NoError(t, err)
	require.Equal(t, 0, heldPatterns[pattern])
	_, stillPresent := heldPatterns[pattern]
	require.False(t, stillPresent)
}

// TestSubscriptionHandleMessage_DoubleSubscribeThenSingleUnsubscribeKeepsBinding regression-tests
// the refcount leak found in PR #1101 round-2 review: a connection that subscribes to the same
// topic twice (without an intervening unsubscribe -- a realistic client reconnect/retry pattern)
// must have its heldPatterns entry survive a single unsubscribe, since scopeRefCount's internal
// refcount is still 1 after that single Release. If heldPatterns were a boolean set instead of a
// counter, this single unsubscribe would erroneously drop the pattern from the held set, and the
// eventual abrupt-disconnect ReleaseAll would never issue the second Release needed to actually
// unbind the queue -- a permanent per-scope bind leak.
func TestSubscriptionHandleMessage_DoubleSubscribeThenSingleUnsubscribeKeepsBinding(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockZMQSub := zmqsubhandler.NewMockZMQSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		scopeRefCount: newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic"),
	}

	a := newSuperAdminIdentity()
	topic := "customer_id:5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb"
	pattern := "customer_id.5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb.#"

	// Only ONE QueueBind expected: scopeRefCount only binds on the 0->1 transition. The second
	// Acquire (from the second subscribe) increments the internal refcount to 2 without a
	// second AMQP call.
	mockZMQSub.EXPECT().Subscribe(topic).Return(nil).Times(2)
	mockSock.EXPECT().QueueBind("pod-queue-1", pattern, "bin-manager.webhook-manager.event.topic", false, nil).Return(nil).Times(1)

	heldPatterns := make(map[string]int)
	var heldMu sync.Mutex

	subMsg := &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{topic}}
	require.NoError(t, h.subscriptionHandleMessage(context.Background(), a, mockZMQSub, subMsg, heldPatterns, &heldMu))
	require.NoError(t, h.subscriptionHandleMessage(context.Background(), a, mockZMQSub, subMsg, heldPatterns, &heldMu))
	require.Equal(t, 2, heldPatterns[pattern])

	// A single unsubscribe must NOT drop the pattern from heldPatterns entirely -- the
	// connection still holds one outstanding Acquire. No QueueUnbind expected here (internal
	// refcount only drops from 2 to 1, still > 0).
	mockZMQSub.EXPECT().Unsubscribe(topic).Return(nil)
	unsubMsg := &hook.Hook{Type: hook.TypeUnsubscribe, Topics: []string{topic}}
	require.NoError(t, h.subscriptionHandleMessage(context.Background(), a, mockZMQSub, unsubMsg, heldPatterns, &heldMu))
	require.Equal(t, 1, heldPatterns[pattern])
	_, stillPresent := heldPatterns[pattern]
	require.True(t, stillPresent, "pattern must still be tracked after only one of two Acquires was released")

	// Now simulate the abrupt-disconnect cleanup path exactly as subscriptionRun does: expand
	// heldPatterns by count and ReleaseAll. This must issue exactly one more QueueUnbind (the
	// remaining outstanding Acquire), proving the leak is closed.
	mockSock.EXPECT().QueueUnbind("pod-queue-1", pattern, "bin-manager.webhook-manager.event.topic", nil).Return(nil).Times(1)
	heldMu.Lock()
	patterns := make([]string, 0, len(heldPatterns))
	for p, count := range heldPatterns {
		for i := 0; i < count; i++ {
			patterns = append(patterns, p)
		}
	}
	heldMu.Unlock()
	h.scopeRefCount.ReleaseAll(patterns)
}

func TestSubscriptionRun_ReleasesAllOnAbruptDisconnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		scopeRefCount: newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic"),
	}

	pattern1 := "customer_id.5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb.#"
	pattern2 := "agent_id.6367ee4f-ac6e-22fb-9feb-7c64f2af2fcc.#"

	// Simulate what subscriptionHandleMessage would have already done: two patterns acquired
	// via explicit subscribe messages, with no matching unsubscribe before disconnect.
	mockSock.EXPECT().QueueBind("pod-queue-1", pattern1, "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)
	mockSock.EXPECT().QueueBind("pod-queue-1", pattern2, "bin-manager.webhook-manager.event.topic", false, nil).Return(nil)
	require.NoError(t, h.scopeRefCount.Acquire(pattern1))
	require.NoError(t, h.scopeRefCount.Acquire(pattern2))

	heldPatterns := map[string]int{
		pattern1: 1,
		pattern2: 1,
	}
	var heldMu sync.Mutex

	// Assert both patterns get unbound (refcount 1->0) on the abrupt-disconnect cleanup path,
	// mirroring what subscriptionRun does after <-newCtx.Done() with no unsubscribe received.
	mockSock.EXPECT().QueueUnbind("pod-queue-1", pattern1, "bin-manager.webhook-manager.event.topic", nil).Return(nil)
	mockSock.EXPECT().QueueUnbind("pod-queue-1", pattern2, "bin-manager.webhook-manager.event.topic", nil).Return(nil)

	// exercise the cleanup logic exactly as subscriptionRun does after ctx cancellation
	heldMu.Lock()
	patterns := make([]string, 0, len(heldPatterns))
	for p, count := range heldPatterns {
		for i := 0; i < count; i++ {
			patterns = append(patterns, p)
		}
	}
	heldMu.Unlock()
	h.scopeRefCount.ReleaseAll(patterns)
}

// TestSubscriptionRun_ContextCancelTriggersReleaseAll drives subscriptionRun end-to-end (minus
// the real websocket upgrade, which we can't easily fake here) is out of scope; instead this
// test exercises subscriptionRunWebsock's context-cancel exit path together with the held-set
// bookkeeping, verifying that a canceled context stops the read loop promptly.
func TestSubscriptionRunWebsock_ExitsOnContextCancel(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockZMQSub := zmqsubhandler.NewMockZMQSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		scopeRefCount: newScopeRefCount(mockSock, "pod-queue-1", "bin-manager.webhook-manager.event.topic"),
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel so receiveTextFromWebsock's context check (via ctx.Done()) exits immediately

	a := newSuperAdminIdentity()
	heldPatterns := make(map[string]int)
	var heldMu sync.Mutex

	done := make(chan struct{})
	var innerCancelCalled bool
	var mu sync.Mutex
	go func() {
		h.subscriptionRunWebsock(ctx, func() {
			mu.Lock()
			innerCancelCalled = true
			mu.Unlock()
		}, a, nil, mockZMQSub, heldPatterns, &heldMu)
		close(done)
	}()

	select {
	case <-done:
		// good, the goroutine returned
	case <-time.After(5 * time.Second):
		t.Fatal("subscriptionRunWebsock did not return after context cancellation")
	}

	mu.Lock()
	defer mu.Unlock()
	require.True(t, innerCancelCalled)
}
