package pipecatcallhandler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
)

const logPolicySentinel = "SENTINEL-CONVERSATION-TEXT-9f3c"

// The line that reports a created pipecatcall (Info level) must not carry the llm messages, which hold
// the prompt and the user's sentences, and must keep the id.
func Test_logPolicy_Start_createdPipecatcall(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &pipecatcallHandler{
		db:                    mockDB,
		notifyHandler:         mockNotify,
		requestHandler:        mockReq,
		hostID:                "test-host",
		mapPipecatcallSession: make(map[uuid.UUID]*pipecatcall.Session),
		muPipecatcallSession:  sync.Mutex{},
	}

	id := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())
	created := &pipecatcall.Pipecatcall{
		Identity:      commonidentity.Identity{ID: id},
		ReferenceType: pipecatcall.ReferenceTypeCall,
		ReferenceID:   referenceID,
		LLMMessages:   []map[string]any{{"role": "user", "content": logPolicySentinel}},
	}
	mockDB.EXPECT().PipecatcallCreate(gomock.Any(), gomock.Any()).Return(nil)
	mockDB.EXPECT().PipecatcallGet(gomock.Any(), id).Return(created, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypeCreated, created)
	mockReq.EXPECT().CallV1CallGet(gomock.Any(), referenceID).Return(nil, errors.New("call not found"))

	_, err := h.Start(context.Background(), id, uuid.Must(uuid.NewV4()), uuid.Nil, pipecatcall.ReferenceTypeCall, referenceID,
		pipecatcall.LLMType("openai.gpt-5"), created.LLMMessages, pipecatcall.STTTypeDeepgram, "en-US", pipecatcall.TTSTypeElevenLabs, "en-US", "voice")
	if err == nil {
		t.Fatalf("expected the call lookup failure")
	}

	logs.AssertAbsent(t, logPolicySentinel)
	logs.AssertField(t, "pipecatcall_id")
	logs.AssertField(t, "llm_message_count")
}

// The lines of the received frames must not carry the user's transcription, the bot's text or the raw
// frame, and must keep the pipecatcall id and the text length.
func Test_logPolicy_RunnerWebsocketHandleOutput_frames(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	frame := func(f *pipecatframe.Frame) []byte {
		b, err := proto.Marshal(f)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return b
	}
	textFrame := frame(&pipecatframe.Frame{Frame: &pipecatframe.Frame_Text{Text: &pipecatframe.TextFrame{Id: 1, Name: "t", Text: logPolicySentinel}}})
	transFrame := frame(&pipecatframe.Frame{Frame: &pipecatframe.Frame_Transcription{Transcription: &pipecatframe.TranscriptionFrame{Id: 2, Name: "tr", Text: logPolicySentinel, UserId: "u"}}})

	// a real websocket pair, because the handler closes the connection
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	mockWS := NewMockWebsocketHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	id := uuid.Must(uuid.NewV4())
	se := &pipecatcall.Session{Identity: commonidentity.Identity{ID: id}, Ctx: context.Background()}
	h := &pipecatcallHandler{
		db:                    mockDB,
		notifyHandler:         mockNotify,
		websocketHandler:      mockWS,
		mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{id: se},
		muPipecatcallSession:  sync.Mutex{},
	}

	mockWS.EXPECT().Upgrade(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, nil)
	mockDB.EXPECT().PipecatcallGet(gomock.Any(), id).Return(&pipecatcall.Pipecatcall{
		Identity:    commonidentity.Identity{ID: id},
		LLMMessages: []map[string]any{{"role": "user", "content": logPolicySentinel}},
	}, nil)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypeInitialized, gomock.Any())
	gomock.InOrder(
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, textFrame, nil),
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, transFrame, nil),
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.TextMessage, []byte(logPolicySentinel), nil),
		mockWS.EXPECT().ReadMessage(conn).Return(0, nil, errors.New("done")),
	)

	gin.SetMode(gin.TestMode)
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = httptest.NewRequest(http.MethodGet, "/v1/pipecatcalls/"+id.String()+"/output", nil)
	gc.Request.Header.Set("Authorization", "Bearer "+logPolicySentinel)

	_ = h.RunnerWebsocketHandleOutput(id, gc)

	logs.AssertAbsent(t, logPolicySentinel)
	logs.AssertField(t, "pipecatcall_id")
	logs.AssertField(t, "text_len")
}

// The input websocket receives no application data. If the peer sends some anyway, the lines must not
// carry the raw message, and the request headers (an authorization value) must not be logged.
func Test_logPolicy_RunnerWebsocketHandleInput(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	mockWS := NewMockWebsocketHandler(mc)
	mockPF := NewMockPipecatframeHandler(mc)

	id := uuid.Must(uuid.NewV4())
	se := &pipecatcall.Session{Identity: commonidentity.Identity{ID: id}, Ctx: context.Background()}
	h := &pipecatcallHandler{
		websocketHandler:      mockWS,
		pipecatframeHandler:   mockPF,
		mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{id: se},
		muPipecatcallSession:  sync.Mutex{},
	}

	received := make(chan struct{})
	mockWS.EXPECT().Upgrade(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, nil)
	gomock.InOrder(
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.TextMessage, []byte(logPolicySentinel), nil),
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, []byte(logPolicySentinel), nil),
		mockWS.EXPECT().ReadMessage(conn).Return(99, []byte(logPolicySentinel), nil),
		mockWS.EXPECT().ReadMessage(conn).DoAndReturn(func(_ *websocket.Conn) (int, []byte, error) {
			close(received)
			return 0, nil, errors.New("done")
		}),
	)
	// the sender returns after the receiver has read everything
	mockPF.EXPECT().RunSender(se, conn).Do(func(_ *pipecatcall.Session, _ *websocket.Conn) { <-received })

	gin.SetMode(gin.TestMode)
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = httptest.NewRequest(http.MethodGet, "/v1/pipecatcalls/"+id.String()+"/input", nil)
	gc.Request.Header.Set("Authorization", "Bearer "+logPolicySentinel)

	if err := h.RunnerWebsocketHandleInput(id, gc); err != nil {
		t.Fatalf("RunnerWebsocketHandleInput: %v", err)
	}

	logs.AssertAbsent(t, logPolicySentinel)
	logs.AssertField(t, "message_len")
}

// The error of a failed stop request carries the status and the body length, not the body.
func Test_logPolicy_pythonRunnerStop_body(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(logPolicySentinel))
	}))
	defer srv.Close()

	origClient := httpClient
	httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		r.URL.Scheme = "http"
		r.URL.Host = strings.TrimPrefix(srv.URL, "http://")
		return http.DefaultTransport.RoundTrip(r)
	})}
	defer func() { httpClient = origClient }()

	err := (&pythonRunner{}).Stop(context.Background(), uuid.Must(uuid.NewV4()))
	if err == nil {
		t.Fatalf("expected the stop to fail")
	}
	if strings.Contains(err.Error(), logPolicySentinel) || !strings.Contains(err.Error(), "body_len:") {
		t.Errorf("the error must carry the body length and not the body: %v", err)
	}
}
