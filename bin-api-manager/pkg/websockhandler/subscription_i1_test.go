package websockhandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/models/hook"
	"monorepo/bin-api-manager/pkg/pubsubhandler"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

const (
	i1Queue    = "pod-queue-i1"
	i1Exchange = "bin-manager.webhook-manager.event.topic"
	i1Topic    = "customer_id:5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb"
	i1Pattern  = "customer_id.5257dd3e-9b5d-11ea-8eda-6b53f19ec1eb.#"
)

func i1SuperAdmin() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{Permission: amagent.PermissionProjectSuperAdmin})
}

func i1NoPermission() *auth.AuthIdentity {
	return auth.NewAgentIdentity(&amagent.Agent{})
}

func Test_subscriptionHandleMessage_errors(t *testing.T) {
	tests := []struct {
		name string
		a    *auth.AuthIdentity
		msg  *hook.Hook

		setup func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler)

		expectErr  string
		expectHeld map[string]int
	}{
		{
			name:       "invalid topics are rejected before any subscribe",
			a:          i1NoPermission(),
			msg:        &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{"bad"}},
			setup:      func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {},
			expectErr:  "invalid topics",
			expectHeld: map[string]int{},
		},
		{
			name: "subscribe error is returned",
			a:    i1SuperAdmin(),
			msg:  &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{i1Topic}},
			setup: func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {
				sub.EXPECT().Subscribe(i1Topic).Return(errors.New("sub failed"))
			},
			expectErr:  "sub failed",
			expectHeld: map[string]int{},
		},
		{
			name: "unsubscribe error is returned",
			a:    i1SuperAdmin(),
			msg:  &hook.Hook{Type: hook.TypeUnsubscribe, Topics: []string{i1Topic}},
			setup: func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {
				sub.EXPECT().Unsubscribe(i1Topic).Return(errors.New("unsub failed"))
			},
			expectErr:  "unsub failed",
			expectHeld: map[string]int{},
		},
		{
			name: "subscribe with unconvertible topic skips the binding",
			a:    i1SuperAdmin(),
			msg:  &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{"nocolon"}},
			setup: func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {
				sub.EXPECT().Subscribe("nocolon").Return(nil)
			},
			expectHeld: map[string]int{},
		},
		{
			name: "subscribe continues when the AMQP binding fails",
			a:    i1SuperAdmin(),
			msg:  &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{i1Topic}},
			setup: func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {
				sub.EXPECT().Subscribe(i1Topic).Return(nil)
				sock.EXPECT().QueueBind(i1Queue, i1Pattern, i1Exchange, false, nil).Return(errors.New("bind failed"))
			},
			expectHeld: map[string]int{},
		},
		{
			name: "unsubscribe with unconvertible topic skips the release",
			a:    i1SuperAdmin(),
			msg:  &hook.Hook{Type: hook.TypeUnsubscribe, Topics: []string{"nocolon"}},
			setup: func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {
				sub.EXPECT().Unsubscribe("nocolon").Return(nil)
			},
			expectHeld: map[string]int{},
		},
		{
			name: "unsubscribe of an untracked pattern is a no-op",
			a:    i1SuperAdmin(),
			msg:  &hook.Hook{Type: hook.TypeUnsubscribe, Topics: []string{i1Topic}},
			setup: func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {
				sub.EXPECT().Unsubscribe(i1Topic).Return(nil)
			},
			expectHeld: map[string]int{},
		},
		{
			name:       "unknown message type does nothing",
			a:          i1SuperAdmin(),
			msg:        &hook.Hook{Type: hook.Type("other"), Topics: []string{i1Topic}},
			setup:      func(sub *pubsubhandler.MockSubHandler, sock *sockhandler.MockSockHandler) {},
			expectHeld: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSub := pubsubhandler.NewMockSubHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			tt.setup(mockSub, mockSock)

			h := &websockHandler{scopeRefCount: newScopeRefCount(mockSock, i1Queue, i1Exchange)}
			held := map[string]int{}
			var heldMu sync.Mutex

			err := h.subscriptionHandleMessage(context.Background(), tt.a, mockSub, tt.msg, held, &heldMu)
			if tt.expectErr == "" && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectErr != "" && (err == nil || err.Error() != tt.expectErr) {
				t.Errorf("wrong error. expect: %s, got: %v", tt.expectErr, err)
			}
			if !reflect.DeepEqual(held, tt.expectHeld) {
				t.Errorf("wrong held patterns. expect: %v, got: %v", tt.expectHeld, held)
			}
		})
	}
}

func Test_subscriptionHandleMessage_unsubscribeReleaseError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSub := pubsubhandler.NewMockSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &websockHandler{scopeRefCount: newScopeRefCount(mockSock, i1Queue, i1Exchange)}

	mockSub.EXPECT().Subscribe(i1Topic).Return(nil)
	mockSock.EXPECT().QueueBind(i1Queue, i1Pattern, i1Exchange, false, nil).Return(nil)
	mockSub.EXPECT().Unsubscribe(i1Topic).Return(nil)
	mockSock.EXPECT().QueueUnbind(i1Queue, i1Pattern, i1Exchange, nil).Return(errors.New("unbind failed"))

	held := map[string]int{}
	var heldMu sync.Mutex
	a := i1SuperAdmin()

	if err := h.subscriptionHandleMessage(context.Background(), a, mockSub, &hook.Hook{Type: hook.TypeSubscribe, Topics: []string{i1Topic}}, held, &heldMu); err != nil {
		t.Fatalf("subscribe: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(held, map[string]int{i1Pattern: 1}) {
		t.Fatalf("wrong held after subscribe: %v", held)
	}

	// a release failure is logged only; the pattern is still dropped from the held set.
	if err := h.subscriptionHandleMessage(context.Background(), a, mockSub, &hook.Hook{Type: hook.TypeUnsubscribe, Topics: []string{i1Topic}}, held, &heldMu); err != nil {
		t.Fatalf("unsubscribe: unexpected error: %v", err)
	}
	if !reflect.DeepEqual(held, map[string]int{}) {
		t.Errorf("wrong held after unsubscribe: %v", held)
	}
}

func Test_subscriptionRunPubsubSub(t *testing.T) {
	tests := []struct {
		name   string
		runErr error
	}{
		{name: "run error cancels the context", runErr: errors.New("run failed")},
		{name: "run finished without error cancels the context", runErr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSub := pubsubhandler.NewMockSubHandler(mc)
			var writeMu sync.Mutex
			ctx := context.Background()

			mockSub.EXPECT().RunWithMutex(ctx, nil, &writeMu).Return(tt.runErr)

			cancelled := make(chan struct{})
			h := &websockHandler{}
			h.subscriptionRunPubsubSub(ctx, func() { close(cancelled) }, nil, mockSub, &writeMu)

			select {
			case <-cancelled:
			default:
				t.Errorf("cancel was not called")
			}
		})
	}
}

func Test_subscriptionRunPinger_contextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cancelled := make(chan struct{})
	done := make(chan struct{})
	var writeMu sync.Mutex
	h := &websockHandler{}
	go func() {
		h.subscriptionRunPinger(ctx, func() { close(cancelled) }, nil, &writeMu)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("pinger did not return after the context was canceled")
	}
	select {
	case <-cancelled:
	default:
		t.Errorf("cancel was not called")
	}
}

func Test_subscriptionRun_notUpgrade(t *testing.T) {
	h := &websockHandler{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1.0/ws", nil)

	err := h.subscriptionRun(context.Background(), w, r, i1SuperAdmin())
	if err == nil {
		t.Fatalf("expected upgrade error, got nil")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("wrong status. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}

func Test_RunSubscription_notUpgrade(t *testing.T) {
	h := &websockHandler{}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/v1.0/ws", nil)

	err := h.RunSubscription(context.Background(), w, r, i1SuperAdmin())
	if err == nil {
		t.Fatalf("expected upgrade error, got nil")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("wrong status. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
}

// i1RunServer starts an httptest server whose handler runs subscriptionRun and reports its
// result on the returned channel.
func i1RunServer(t *testing.T, h *websockHandler, a *auth.AuthIdentity) (*websocket.Conn, <-chan error) {
	t.Helper()

	result := make(chan error, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result <- h.subscriptionRun(context.Background(), w, r, a)
	}))
	t.Cleanup(srv.Close)

	client, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("could not dial: %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	return client, result
}

func i1WaitErr(t *testing.T, ch <-chan error) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("subscriptionRun did not return in time")
		return nil
	}
}

func Test_subscriptionRun_clientDisconnectReleasesAll(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockBroker := pubsubhandler.NewMockBrokerHandler(mc)
	mockSub := pubsubhandler.NewMockSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		pubsubBroker:  mockBroker,
		scopeRefCount: newScopeRefCount(mockSock, i1Queue, i1Exchange),
	}

	mockBroker.EXPECT().NewSubHandler().Return(mockSub)
	mockSub.EXPECT().RunWithMutex(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ws *websocket.Conn, writeMu *sync.Mutex) error {
			<-ctx.Done()
			return ctx.Err()
		})
	mockSub.EXPECT().Terminate()

	subscribed := make(chan struct{})
	mockSub.EXPECT().Subscribe(i1Topic).Return(nil)
	mockSock.EXPECT().QueueBind(i1Queue, i1Pattern, i1Exchange, false, nil).DoAndReturn(
		func(queue, pattern, exchange string, noWait bool, args interface{}) error {
			close(subscribed)
			return nil
		})
	released := make(chan struct{})
	mockSock.EXPECT().QueueUnbind(i1Queue, i1Pattern, i1Exchange, nil).DoAndReturn(
		func(queue, pattern, exchange string, args interface{}) error {
			close(released)
			return nil
		})

	client, result := i1RunServer(t, h, i1SuperAdmin())

	// a malformed message is skipped and must not end the session.
	if err := client.WriteMessage(websocket.TextMessage, []byte("not-json")); err != nil {
		t.Fatalf("could not write: %v", err)
	}
	if err := client.WriteJSON(&hook.Hook{Type: hook.TypeSubscribe, Topics: []string{i1Topic}}); err != nil {
		t.Fatalf("could not write: %v", err)
	}

	select {
	case <-subscribed:
	case <-time.After(5 * time.Second):
		t.Fatalf("subscribe was not processed")
	}

	_ = client.Close()

	if err := i1WaitErr(t, result); err != nil {
		t.Errorf("expected nil from subscriptionRun, got: %v", err)
	}
	select {
	case <-released:
	case <-time.After(5 * time.Second):
		t.Errorf("held pattern was not released on disconnect")
	}
}

func Test_subscriptionRun_invalidTopicEndsSession(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockBroker := pubsubhandler.NewMockBrokerHandler(mc)
	mockSub := pubsubhandler.NewMockSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		pubsubBroker:  mockBroker,
		scopeRefCount: newScopeRefCount(mockSock, i1Queue, i1Exchange),
	}

	mockBroker.EXPECT().NewSubHandler().Return(mockSub)
	mockSub.EXPECT().RunWithMutex(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, ws *websocket.Conn, writeMu *sync.Mutex) error {
			<-ctx.Done()
			return ctx.Err()
		})
	mockSub.EXPECT().Terminate()
	// no Subscribe / QueueBind expected: the unauthorized topic must be rejected first.

	client, result := i1RunServer(t, h, i1NoPermission())
	if err := client.WriteJSON(&hook.Hook{Type: hook.TypeSubscribe, Topics: []string{i1Topic}}); err != nil {
		t.Fatalf("could not write: %v", err)
	}

	if err := i1WaitErr(t, result); err != nil {
		t.Errorf("expected nil from subscriptionRun, got: %v", err)
	}
}

func Test_subscriptionRun_pubsubErrorEndsSession(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockBroker := pubsubhandler.NewMockBrokerHandler(mc)
	mockSub := pubsubhandler.NewMockSubHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	h := &websockHandler{
		pubsubBroker:  mockBroker,
		scopeRefCount: newScopeRefCount(mockSock, i1Queue, i1Exchange),
	}

	mockBroker.EXPECT().NewSubHandler().Return(mockSub)
	mockSub.EXPECT().RunWithMutex(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("run failed"))
	mockSub.EXPECT().Terminate()

	_, result := i1RunServer(t, h, i1SuperAdmin())

	if err := i1WaitErr(t, result); err != nil {
		t.Errorf("expected nil from subscriptionRun, got: %v", err)
	}
}

func Test_NewWebsockHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockBroker := pubsubhandler.NewMockBrokerHandler(mc)
	mockSock := sockhandler.NewMockSockHandler(mc)

	res := NewWebsockHandler(nil, nil, mockSock, mockBroker, i1Queue)
	h, ok := res.(*websockHandler)
	if !ok {
		t.Fatalf("expected *websockHandler, got %T", res)
	}
	if h.pubsubBroker != mockBroker {
		t.Errorf("pubsubBroker was not set")
	}
	if h.scopeRefCount == nil || h.scopeRefCount.queue != i1Queue || h.scopeRefCount.exchange != i1Exchange || h.scopeRefCount.sock != mockSock {
		t.Errorf("scopeRefCount was not wired correctly: %+v", h.scopeRefCount)
	}

	// endpointInit side effect: every port in [minPort, maxPort) is free.
	lockAvailable.RLock()
	defer lockAvailable.RUnlock()
	for port := minPort; port < maxPort; port++ {
		id, ok := gPortsAvailable[port]
		if !ok || id.String() != "00000000-0000-0000-0000-000000000000" {
			t.Errorf("port %d not initialised as free: %v (present: %v)", port, id, ok)
		}
	}
}
