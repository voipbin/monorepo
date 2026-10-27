package listenhandler

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/pkg/activeflowhandler"
	"monorepo/bin-flow-manager/pkg/flowhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

// logWaiter closes its channel when the standard logger writes a message that contains the text.
// It lets a test wait for the end of a background goroutine without any sleep.
type logWaiter struct {
	text string
	once sync.Once
	done chan struct{}
}

func (w *logWaiter) Levels() []logrus.Level { return logrus.AllLevels }

func (w *logWaiter) Fire(entry *logrus.Entry) error {
	if strings.Contains(entry.Message, w.text) {
		w.once.Do(func() { close(w.done) })
	}
	return nil
}

// newLogWaiter installs a logWaiter on the standard logger, and removes it when the test ends.
// Only the messages written at the error level are guaranteed to reach the hook.
func newLogWaiter(t *testing.T, text string) *logWaiter {
	t.Helper()

	w := &logWaiter{text: text, done: make(chan struct{})}
	old := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(w)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(old) })

	return w
}

func (w *logWaiter) wait(t *testing.T) {
	t.Helper()
	select {
	case <-w.done:
	case <-time.After(5 * time.Second):
		t.Fatalf("The log message was not written. text: %s", w.text)
	}
}

func newCov90ListenHandler(mc *gomock.Controller) *listenHandler {
	return &listenHandler{
		sockHandler:       sockhandler.NewMockSockHandler(mc),
		flowHandler:       flowhandler.NewMockFlowHandler(mc),
		activeflowHandler: activeflowhandler.NewMockActiveflowHandler(mc),
		variableHandler:   variablehandler.NewMockVariableHandler(mc),
	}
}

// Test_handlers_unparsableURI_coverage90 verifies that a request whose URI can not be parsed is
// refused before any business handler is called. The handlers have no expectation registered, so
// a call into a business handler fails the test.
func Test_handlers_unparsableURI_coverage90(t *testing.T) {
	ctx := context.Background()
	// an escape sequence that is not hexadecimal can not be parsed as a URL.
	badURI := "/v1/flows/%zz"

	tests := []struct {
		name string
		call func(h *listenHandler, m *sock.Request) (*sock.Response, error)
	}{
		{"activeflows get", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1ActiveflowsGet(ctx, m) }},
		{"flows id get", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1FlowsIDGet(ctx, m) }},
		{"flows id put", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1FlowsIDPut(ctx, m) }},
		{"flows id delete", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1FlowsIDDelete(ctx, m) }},
		{"flows get", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1FlowsGet(ctx, m) }},
		{"flows id actions put", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1FlowsIDActionsPut(ctx, m) }},
		{"flows id actions id get", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.v1FlowsIDActionsIDGet(ctx, m)
		}},
		{"flows id direct hash regenerate", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.processV1FlowsIDDirectHashRegeneratePost(ctx, m)
		}},
		{"variables id get", func(h *listenHandler, m *sock.Request) (*sock.Response, error) { return h.v1VariablesIDGet(ctx, m) }},
		{"variables id variables post", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.v1VariablesIDVariablesPost(ctx, m)
		}},
		{"variables id variables key delete", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.v1VariablesIDVariablesKeyDelete(ctx, m)
		}},
		{"variables id substitute post", func(h *listenHandler, m *sock.Request) (*sock.Response, error) {
			return h.v1VariablesIDSubstitutePost(ctx, m)
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := newCov90ListenHandler(mc)

			res, err := tt.call(h, &sock.Request{URI: badURI, Data: []byte(`{}`)})
			if err == nil {
				t.Errorf("expected a parse error, got nil")
			}
			if res != nil {
				t.Errorf("no response is expected with the error. got: %v", res)
			}
		})
	}
}

func Test_processRequest_notFound_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := newCov90ListenHandler(mc)

	res, err := h.processRequest(&sock.Request{URI: "/v1/unknown_resource", Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 404 {
		t.Errorf("An unknown resource must be answered with 404. got: %v", res)
	}
}

func Test_directHashRegenerate_shortPath_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := newCov90ListenHandler(mc)

	// the path has no flow id, so the flow handler must not be called.
	res, err := h.processV1FlowsIDDirectHashRegeneratePost(context.Background(), &sock.Request{URI: "/v1/flows"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 400 {
		t.Errorf("A path without the flow id must be answered with 400. got: %v", res)
	}
}

func Test_v1VariablesIDVariablesKeyDelete_badKeyEscape_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := newCov90ListenHandler(mc)
	mockVariable := h.variableHandler.(*variablehandler.MockVariableHandler)
	id := uuid.FromStringOrNil("c9e1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")

	// the path is decoded once by the URL parser, and the decoded key is not a valid escape sequence,
	// so the delete is requested with the empty key.
	mockVariable.EXPECT().DeleteVariable(gomock.Any(), id, "").Return(nil)

	res, err := h.v1VariablesIDVariablesKeyDelete(context.Background(), &sock.Request{URI: "/v1/variables/" + id.String() + "/variables/%25zz"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.StatusCode != 200 {
		t.Errorf("The delete must be answered with 200. got: %v", res)
	}
}

func Test_activeflowExecution_background_coverage90(t *testing.T) {
	id := uuid.FromStringOrNil("c9e1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")
	caID := uuid.FromStringOrNil("c9e1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c03")

	tests := []struct {
		name string

		call      func(h *listenHandler) (*sock.Response, error)
		setup     func(m *activeflowhandler.MockActiveflowHandler)
		expectLog string
	}{
		{
			name: "execute failure is logged and does not reach the caller",
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsIDExecutePost(context.Background(), &sock.Request{URI: "/v1/activeflows/" + id.String() + "/execute"})
			},
			setup: func(m *activeflowhandler.MockActiveflowHandler) {
				m.EXPECT().Execute(gomock.Any(), id).Return(errSynthetic)
			},
			expectLog: "Could not execute the activeflow correctly",
		},
		{
			name: "execute panic is recovered",
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsIDExecutePost(context.Background(), &sock.Request{URI: "/v1/activeflows/" + id.String() + "/execute"})
			},
			setup: func(m *activeflowhandler.MockActiveflowHandler) {
				m.EXPECT().Execute(gomock.Any(), id).DoAndReturn(func(_ context.Context, _ uuid.UUID) error {
					panic("synthetic panic")
				})
			},
			expectLog: "Recovered from panic during activeflow execution",
		},
		{
			name: "continue failure is logged and does not reach the caller",
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsIDContinuePost(context.Background(), &sock.Request{
					URI:  "/v1/activeflows/" + id.String() + "/continue",
					Data: []byte(`{"current_action_id":"` + caID.String() + `"}`),
				})
			},
			setup: func(m *activeflowhandler.MockActiveflowHandler) {
				m.EXPECT().ExecuteContinue(gomock.Any(), id, caID).Return(errSynthetic)
			},
			expectLog: "Could not continue the activeflow correctly",
		},
		{
			name: "continue panic is recovered",
			call: func(h *listenHandler) (*sock.Response, error) {
				return h.v1ActiveflowsIDContinuePost(context.Background(), &sock.Request{
					URI:  "/v1/activeflows/" + id.String() + "/continue",
					Data: []byte(`{"current_action_id":"` + caID.String() + `"}`),
				})
			},
			setup: func(m *activeflowhandler.MockActiveflowHandler) {
				m.EXPECT().ExecuteContinue(gomock.Any(), id, caID).DoAndReturn(func(_ context.Context, _, _ uuid.UUID) error {
					panic("synthetic panic")
				})
			},
			expectLog: "Recovered from panic during activeflow continue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := newCov90ListenHandler(mc)
			tt.setup(h.activeflowHandler.(*activeflowhandler.MockActiveflowHandler))
			waiter := newLogWaiter(t, tt.expectLog)

			res, err := tt.call(h)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res == nil || res.StatusCode != 200 {
				t.Errorf("The request must be answered with 200 at once. got: %v", res)
			}

			waiter.wait(t)
		})
	}
}

func Test_Run_coverage90(t *testing.T) {
	t.Run("the queue can not be created", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := newCov90ListenHandler(mc)
		h.sockHandler.(*sockhandler.MockSockHandler).EXPECT().QueueCreate("synthetic-queue", "normal").Return(errSynthetic)

		err := h.Run("synthetic-queue", "synthetic-delay")
		if err == nil || !strings.Contains(err.Error(), "could not declare the queue for listenHandler") {
			t.Errorf("expected the queue failure, got: %v", err)
		}
	})

	t.Run("the consume failure is logged and does not reach the caller", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := newCov90ListenHandler(mc)
		mockSock := h.sockHandler.(*sockhandler.MockSockHandler)
		waiter := newLogWaiter(t, "Could not consume the request message correctly")

		mockSock.EXPECT().QueueCreate("synthetic-queue", "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "synthetic-queue", "flow-manager", false, false, false, 10, gomock.Any()).Return(errSynthetic)

		if err := h.Run("synthetic-queue", "synthetic-delay"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		waiter.wait(t)
	})
}
