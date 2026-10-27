package pipecatcallhandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	amai "monorepo/bin-ai-manager/models/ai"
	amaicall "monorepo/bin-ai-manager/models/aicall"
	aitool "monorepo/bin-ai-manager/models/tool"
	cmcall "monorepo/bin-call-manager/models/call"
	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/toolhandler"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

// startSuccessWait bounds every wait of the start success tests, so a regression shows as a
// failure and not as a hang.
const startSuccessWait = 5 * time.Second

// newLoopbackConn returns the client side of a websocket connection to a local test server. The
// server only drains the connection, so nothing leaves the host.
func newLoopbackConn(t *testing.T) *websocket.Conn {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = c.Close() }()
		for {
			if _, _, errRead := c.ReadMessage(); errRead != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	return dialTestServer(t, srv)
}

// expectAsteriskMediaLifecycle makes the websocket mock answer like an Asterisk media channel:
// the first read is the ready text message and the second read is a normal close, which ends the
// media handle and, through the lifecycle monitor, terminates the pipecatcall.
func expectAsteriskMediaLifecycle(mockWS *MockWebsocketHandler, mediaURI string, conn *websocket.Conn) {
	var reads atomic.Int32

	mockWS.EXPECT().DialContext(gomock.Any(), mediaURI, gomock.Any()).Return(conn, nil, nil)
	mockWS.EXPECT().ReadMessage(conn).DoAndReturn(func(_ *websocket.Conn) (int, []byte, error) {
		if reads.Add(1) == 1 {
			return websocket.TextMessage, []byte("MEDIA_START"), nil
		}
		return 0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}
	}).Times(2)
}

// expectPythonStartStop registers the python runner calls of a started and then terminated
// session. The returned channels report the session context of the start call and close on the
// stop call, the last mock call of the terminate goroutine.
func expectPythonStartStop(mockPython *MockPythonRunner, pcID uuid.UUID) (<-chan context.Context, <-chan struct{}) {
	ctxCh := make(chan context.Context, 1)
	stopCh := make(chan struct{})

	mockPython.EXPECT().Start(
		gomock.Any(), pcID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
	).DoAndReturn(func(
		ctx context.Context, _ uuid.UUID, _ string, _ string, _ []map[string]any, _ string, _ string, _ string, _ string, _ string,
		_ []aitool.Tool, _ *resolvedTeamData, _ *amai.VADConfig, _ bool, _ uuid.UUID, _ string, _ string,
	) error {
		ctxCh <- ctx
		return nil
	})
	mockPython.EXPECT().Stop(gomock.Any(), pcID).DoAndReturn(func(_ context.Context, _ uuid.UUID) error {
		close(stopCh)
		return nil
	})

	return ctxCh, stopCh
}

// assertStartedAndTerminated waits until the python runner was started and stopped and checks
// the outcome of the lifecycle: the session is gone and its Asterisk connection was closed.
func assertStartedAndTerminated(t *testing.T, h *pipecatcallHandler, pcID uuid.UUID, conn *websocket.Conn, ctxCh <-chan context.Context, stopCh <-chan struct{}) {
	t.Helper()

	select {
	case <-stopCh:
	case <-time.After(startSuccessWait):
		t.Fatalf("the pipecatcall was not terminated after the Asterisk connection closed")
	}

	select {
	case ctx := <-ctxCh:
		select {
		case <-ctx.Done():
		case <-time.After(startSuccessWait):
			t.Errorf("the session context was not cancelled")
		}
	case <-time.After(startSuccessWait):
		t.Fatalf("the python runner was not started")
	}

	if _, err := h.SessionGet(pcID); err == nil {
		t.Errorf("the session must be removed after the termination")
	}
	if err := conn.WriteMessage(websocket.TextMessage, []byte("x")); err == nil {
		t.Errorf("the Asterisk connection must be closed after the termination")
	}
}

func Test_startReferenceTypeCall_asteriskConnected(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockWS := NewMockWebsocketHandler(mc)
	mockPython := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		websocketHandler:      mockWS,
		pythonRunner:          mockPython,
		toolHandler:           mockTool,
		notifyHandler:         mockNotify,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
		muTerminated:          sync.Mutex{},
		terminatedPublished:   make(map[uuid.UUID]struct{}),
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-5001-4000-8000-000000000001")
	referenceID := uuid.FromStringOrNil("b2c3d4e5-5001-4000-8000-000000000002")
	callID := uuid.FromStringOrNil("d4e5f6a7-5001-4000-8000-000000000003")
	emID := uuid.FromStringOrNil("e5f6a7b8-5001-4000-8000-000000000004")
	mediaURI := "ws://media.example.invalid/ws/test"

	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
		LLMType:       pipecatcall.LLMType("openai.gpt-5"),
	}

	// the call is looked up for the start and again for the termination; it is not in progress, so
	// the termination does not stop the external media.
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(&cmcall.Call{
		Identity: commonidentity.Identity{ID: callID},
		Status:   cmcall.StatusHangup,
	}, nil).Times(2)
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(), pcID, cmexternalmedia.ReferenceTypeCall, callID, "INCOMING",
		defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat,
		cmexternalmedia.DirectionIn, cmexternalmedia.DirectionOut,
	).Return(&cmexternalmedia.ExternalMedia{ID: emID, MediaURI: mediaURI}, nil)

	conn := newLoopbackConn(t)
	expectAsteriskMediaLifecycle(mockWS, mediaURI, conn)
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	ctxCh, stopCh := expectPythonStartStop(mockPython, pcID)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypePipecatcallTerminated, pc).Times(1)

	if err := h.startReferenceTypeCall(context.Background(), pc); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertStartedAndTerminated(t, h, pcID, conn, ctxCh, stopCh)
}

func Test_startReferenceTypeAIcall_callBranchAsteriskConnected(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockWS := NewMockWebsocketHandler(mc)
	mockPython := NewMockPythonRunner(mc)
	mockTool := toolhandler.NewMockToolHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &pipecatcallHandler{
		requestHandler:        mockReq,
		websocketHandler:      mockWS,
		pythonRunner:          mockPython,
		toolHandler:           mockTool,
		notifyHandler:         mockNotify,
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
		muTerminated:          sync.Mutex{},
		terminatedPublished:   make(map[uuid.UUID]struct{}),
	}

	pcID := uuid.FromStringOrNil("a1b2c3d4-5002-4000-8000-000000000001")
	aicallID := uuid.FromStringOrNil("b2c3d4e5-5002-4000-8000-000000000002")
	aiID := uuid.FromStringOrNil("c3d4e5f6-5002-4000-8000-000000000003")
	callID := uuid.FromStringOrNil("d4e5f6a7-5002-4000-8000-000000000004")
	emID := uuid.FromStringOrNil("e5f6a7b8-5002-4000-8000-000000000005")
	mediaURI := "ws://media.example.invalid/ws/test"

	pc := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: pcID},
		ReferenceType: pipecatcall.ReferenceTypeAICall,
		ReferenceID:   aicallID,
		LLMType:       pipecatcall.LLMType("openai.gpt-5"),
	}

	// the start, the runner and the termination look the ai call up; every lookup succeeds, so the
	// termination reaches the python runner stop.
	mockReq.EXPECT().AIV1AIcallGet(gomock.Any(), aicallID).Return(&amaicall.AIcall{
		Identity:       commonidentity.Identity{ID: aicallID},
		ReferenceType:  amaicall.ReferenceTypeCall,
		ReferenceID:    callID,
		AssistanceType: amaicall.AssistanceTypeAI,
		AssistanceID:   aiID,
	}, nil).AnyTimes()
	mockReq.EXPECT().AIV1AIGet(gomock.Any(), aiID).Return(&amai.AI{
		Identity:    commonidentity.Identity{ID: aiID},
		Type:        amai.TypeNormal,
		EngineModel: amai.EngineModel("openai.gpt-5"),
		EngineKey:   "synthetic-key",
	}, nil).AnyTimes()
	mockReq.EXPECT().AIV1AIcallToolList(gomock.Any(), aicallID).Return(nil, nil).AnyTimes()
	mockTool.EXPECT().GetByNames(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
	mockReq.EXPECT().CallV1ExternalMediaStart(
		gomock.Any(), pcID, cmexternalmedia.ReferenceTypeCall, callID, "INCOMING",
		defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat,
		cmexternalmedia.DirectionIn, cmexternalmedia.DirectionOut,
	).Return(&cmexternalmedia.ExternalMedia{ID: emID, MediaURI: mediaURI}, nil)
	mockReq.EXPECT().CallV1ExternalMediaStop(gomock.Any(), pcID).Return(&cmexternalmedia.ExternalMedia{ID: emID}, nil)

	conn := newLoopbackConn(t)
	expectAsteriskMediaLifecycle(mockWS, mediaURI, conn)
	ctxCh, stopCh := expectPythonStartStop(mockPython, pcID)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypePipecatcallTerminated, pc).Times(1)

	if err := h.startReferenceTypeAIcall(context.Background(), pc); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	assertStartedAndTerminated(t, h, pcID, conn, ctxCh, stopCh)
}
