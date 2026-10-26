package listenhandler

import (
	stderrors "errors"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-agent-manager/pkg/agenthandler"
)

func newTestAgent(id uuid.UUID) *agent.Agent {
	return &agent.Agent{
		Identity: commonidentity.Identity{ID: id},
		Username: "direct@example.com",
	}
}

func Test_processRequest_unknownAndParseFailure(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectCode int
	}{
		{
			name: "unknown uri",

			request: &sock.Request{
				URI:    "/v1/unknown",
				Method: sock.RequestMethodGet,
			},

			expectCode: 404,
		},
		{
			name: "list request whose uri cannot be parsed",

			// the control character matches the route but makes the uri unparsable.
			request: &sock.Request{
				URI:    "/v1/agents?page_size=\x7f",
				Method: sock.RequestMethodGet,
			},

			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no agent handler call is expected for any of these requests.
			h := &listenHandler{
				sockHandler:  sockhandler.NewMockSockHandler(mc),
				agentHandler: agenthandler.NewMockAgentHandler(mc),
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectCode {
				t.Errorf("Wrong match. expect status: %d, got: %v", tt.expectCode, res)
			}
		})
	}
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAgent := agenthandler.NewMockAgentHandler(mc)

	res, ok := NewListenHandler(mockSock, mockAgent).(*listenHandler)
	if !ok {
		t.Fatalf("Wrong type. expect: *listenHandler")
	}

	if res.sockHandler != mockSock {
		t.Errorf("Wrong sockHandler wiring.")
	}
	if res.agentHandler != mockAgent {
		t.Errorf("Wrong agentHandler wiring.")
	}
	if res.utilHandler == nil {
		t.Errorf("Wrong utilHandler wiring. expect: not nil")
	}
}

func Test_Run_QueueCreateFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	errBoom := stderrors.New("boom")

	// ConsumeRPC must not be called when the queue could not be created.
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errBoom)

	h := NewListenHandler(mockSock, agenthandler.NewMockAgentHandler(mc))

	if err := h.Run("test-queue", "delay"); !stderrors.Is(err, errBoom) {
		t.Errorf("Wrong error. expect: wrapped boom, got: %v", err)
	}
}

func Test_Run_consume(t *testing.T) {
	tests := []struct {
		name string

		consumeErr error
	}{
		{
			name: "consume starts",
		},
		{
			name: "consume failure is only logged",

			consumeErr: stderrors.New("boom"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			consumed := make(chan struct{})

			mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
			mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", string(commonoutline.ServiceNameAgentManager), false, false, false, 10, gomock.Any()).
				DoAndReturn(func(_ any, _, _ string, _, _, _ bool, _ int, _ any) error {
					close(consumed)
					return tt.consumeErr
				}).Times(1)

			h := NewListenHandler(mockSock, agenthandler.NewMockAgentHandler(mc))

			if err := h.Run("test-queue", "delay"); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(5 * time.Second):
				t.Errorf("ConsumeRPC was not called.")
			}
		})
	}
}
