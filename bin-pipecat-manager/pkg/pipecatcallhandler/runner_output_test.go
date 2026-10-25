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
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/models/pipecatframe"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"
)

// newRunnerTestConn returns the client side of a real websocket pair. The handlers under test close
// the connection, so a real one is needed even though the reads are mocked.
func newRunnerTestConn(t *testing.T) *websocket.Conn {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		for {
			if _, _, errRead := c.ReadMessage(); errRead != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("could not dial the test server: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return conn
}

// runnerTestFrame marshals a protobuf frame for the mocked websocket reads.
func runnerTestFrame(t *testing.T, f *pipecatframe.Frame) []byte {
	t.Helper()

	b, err := proto.Marshal(f)
	if err != nil {
		t.Fatalf("could not marshal the frame: %v", err)
	}
	return b
}

func runnerTestGinContext(id uuid.UUID, direction string) *gin.Context {
	gin.SetMode(gin.TestMode)
	gc, _ := gin.CreateTestContext(httptest.NewRecorder())
	gc.Request = httptest.NewRequest(http.MethodGet, "/v1/pipecatcalls/"+id.String()+"/"+direction, nil)
	return gc
}

type outputTestEnv struct {
	h          *pipecatcallHandler
	mockWS     *MockWebsocketHandler
	mockDB     *dbhandler.MockDBHandler
	mockNotify *notifyhandler.MockNotifyHandler
	id         uuid.UUID
}

func newOutputTestEnv(mc *gomock.Controller) *outputTestEnv {
	id := uuid.FromStringOrNil("a1b2c3d4-3001-4000-8000-000000000001")
	env := &outputTestEnv{
		mockWS:     NewMockWebsocketHandler(mc),
		mockDB:     dbhandler.NewMockDBHandler(mc),
		mockNotify: notifyhandler.NewMockNotifyHandler(mc),
		id:         id,
	}

	se := &pipecatcall.Session{Identity: commonidentity.Identity{ID: id}, Ctx: context.Background()}
	env.h = &pipecatcallHandler{
		db:                    env.mockDB,
		notifyHandler:         env.mockNotify,
		websocketHandler:      env.mockWS,
		mapPipecatcallSession: map[uuid.UUID]*pipecatcall.Session{id: se},
		muPipecatcallSession:  sync.Mutex{},
	}
	return env
}

// expectConnected registers the calls made before the read loop starts.
func (e *outputTestEnv) expectConnected(conn *websocket.Conn) {
	e.mockWS.EXPECT().Upgrade(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, nil)
	e.mockDB.EXPECT().PipecatcallGet(gomock.Any(), e.id).Return(&pipecatcall.Pipecatcall{
		Identity: commonidentity.Identity{ID: e.id},
	}, nil)
	e.mockNotify.EXPECT().PublishEvent(gomock.Any(), pipecatcall.EventTypeInitialized, gomock.Any())
}

func Test_RunnerWebsocketHandleOutput_setupErrors(t *testing.T) {
	t.Run("upgrade fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no db or notify expectation: nothing may happen after a failed upgrade.
		env := newOutputTestEnv(mc)
		env.mockWS.EXPECT().Upgrade(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("upgrade refused"))

		err := env.h.RunnerWebsocketHandleOutput(env.id, runnerTestGinContext(env.id, "output"))
		if err == nil || !strings.Contains(err.Error(), "could not upgrade to WebSocket") || !strings.Contains(err.Error(), "upgrade refused") {
			t.Errorf("Wrong error: %v", err)
		}
	})

	t.Run("pipecatcall lookup fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no notify expectation: the initialized event must not be published.
		env := newOutputTestEnv(mc)
		conn := newRunnerTestConn(t)
		env.mockWS.EXPECT().Upgrade(gomock.Any(), gomock.Any(), gomock.Any()).Return(conn, nil)
		env.mockDB.EXPECT().PipecatcallGet(gomock.Any(), env.id).Return(nil, errors.New("db down"))

		err := env.h.RunnerWebsocketHandleOutput(env.id, runnerTestGinContext(env.id, "output"))
		if err == nil || !strings.Contains(err.Error(), "Could not get pipecatcall info") || !strings.Contains(err.Error(), "db down") {
			t.Errorf("Wrong error: %v", err)
		}
	})
}

func Test_RunnerWebsocketHandleOutput_readErrors(t *testing.T) {
	tests := []struct {
		name string

		readErr    error
		expectLogs string
	}{
		{name: "graceful close", readErr: &websocket.CloseError{Code: websocket.CloseNormalClosure}, expectLogs: "Client disconnected gracefully."},
		{name: "unexpected close code", readErr: &websocket.CloseError{Code: websocket.CloseInternalServerErr}, expectLogs: "Client disconnected unexpectedly"},
		{name: "plain read error", readErr: errors.New("connection reset"), expectLogs: "Error reading message from client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			logs := logtest.New(t)
			env := newOutputTestEnv(mc)
			conn := newRunnerTestConn(t)
			env.expectConnected(conn)

			// a single read: the loop must end on the first error.
			env.mockWS.EXPECT().ReadMessage(conn).Return(0, nil, tt.readErr)

			if err := env.h.RunnerWebsocketHandleOutput(env.id, runnerTestGinContext(env.id, "output")); err != nil {
				t.Errorf("expected nil, got: %v", err)
			}
			if !strings.Contains(logs.Dump(), tt.expectLogs) {
				t.Errorf("Wrong log. expect contains: %q", tt.expectLogs)
			}
		})
	}
}

func Test_RunnerWebsocketHandleOutput_messageKinds(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)
	env := newOutputTestEnv(mc)
	conn := newRunnerTestConn(t)
	env.expectConnected(conn)

	badMessage := runnerTestFrame(t, &pipecatframe.Frame{Frame: &pipecatframe.Frame_Message{Message: &pipecatframe.MessageFrame{Data: "{not json"}}})
	emptyFrame := runnerTestFrame(t, &pipecatframe.Frame{})

	// the loop must keep reading after each of these and end only on the close message,
	// so any missing or extra read fails the test.
	gomock.InOrder(
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, []byte{0xff, 0xff, 0xff}, nil),
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, emptyFrame, nil),
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, badMessage, nil),
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.PingMessage, nil, nil),
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.PongMessage, nil, nil),
		env.mockWS.EXPECT().ReadMessage(conn).Return(99, nil, nil),
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.CloseMessage, nil, nil),
	)

	if err := env.h.RunnerWebsocketHandleOutput(env.id, runnerTestGinContext(env.id, "output")); err != nil {
		t.Fatalf("expected nil, got: %v", err)
	}

	dump := logs.Dump()
	for _, expect := range []string{
		"Could not unmarshal Protobuf message",
		"Could not recognize the Protobuf Frame type",
		"Could not process MessageFrame",
		"Received Ping message from client",
		"Received Pong message from client",
		"Received unknown message type 99",
		"Received Close message from client.",
	} {
		if !strings.Contains(dump, expect) {
			t.Errorf("Wrong log. expect contains: %q", expect)
		}
	}
}

func Test_RunnerWebsocketHandleOutput_audioErrorLimit(t *testing.T) {
	// two channels are rejected by runnerWebsocketHandleAudio before any collaborator is used.
	badAudio := runnerTestFrame(t, &pipecatframe.Frame{Frame: &pipecatframe.Frame_Audio{
		Audio: &pipecatframe.AudioRawFrame{SampleRate: defaultMediaSampleRate, NumChannels: 2, Audio: []byte{1, 2}},
	}})
	// an empty mono frame at the media rate is accepted without any collaborator.
	goodAudio := runnerTestFrame(t, &pipecatframe.Frame{Frame: &pipecatframe.Frame_Audio{
		Audio: &pipecatframe.AudioRawFrame{SampleRate: defaultMediaSampleRate, NumChannels: 1},
	}})

	t.Run("stops after 501 consecutive errors", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		logs := logtest.New(t)
		env := newOutputTestEnv(mc)
		conn := newRunnerTestConn(t)
		env.expectConnected(conn)

		// exactly 501 reads: the 501st error ends the handler, so no further read may happen.
		env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, badAudio, nil).Times(501)

		if err := env.h.RunnerWebsocketHandleOutput(env.id, runnerTestGinContext(env.id, "output")); err != nil {
			t.Fatalf("expected nil, got: %v", err)
		}
		if !strings.Contains(logs.Dump(), "Too many consecutive audio write errors (501), stopping output handler.") {
			t.Errorf("the stop log is missing")
		}
	})

	t.Run("a success resets the error counter", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		env := newOutputTestEnv(mc)
		conn := newRunnerTestConn(t)
		env.expectConnected(conn)

		// 500 errors do not reach the limit, the success clears the counter, and another
		// 500 errors still do not reach it, so the close message is read at the end.
		gomock.InOrder(
			env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, badAudio, nil).Times(500),
			env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, goodAudio, nil),
			env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, badAudio, nil).Times(500),
			env.mockWS.EXPECT().ReadMessage(conn).Return(websocket.CloseMessage, nil, nil),
		)

		if err := env.h.RunnerWebsocketHandleOutput(env.id, runnerTestGinContext(env.id, "output")); err != nil {
			t.Fatalf("expected nil, got: %v", err)
		}
	})
}

func Test_RunnerWebsocketHandleInput_upgradeFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newOutputTestEnv(mc)
	env.mockWS.EXPECT().Upgrade(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errors.New("upgrade refused"))

	err := env.h.RunnerWebsocketHandleInput(env.id, runnerTestGinContext(env.id, "input"))
	if err == nil || !strings.Contains(err.Error(), "could not upgrade to WebSocket") || !strings.Contains(err.Error(), "upgrade refused") {
		t.Errorf("Wrong error: %v", err)
	}
}

func Test_runnerWebsocketHandleInputReceiver(t *testing.T) {
	se := &pipecatcall.Session{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-3002-4000-8000-000000000001")}, Ctx: context.Background()}

	t.Run("close message ends the receiver", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		logs := logtest.New(t)
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{websocketHandler: mockWS}
		var ws *websocket.Conn

		// the control messages are read and ignored, the close message is the last read.
		gomock.InOrder(
			mockWS.EXPECT().ReadMessage(ws).Return(websocket.PingMessage, nil, nil),
			mockWS.EXPECT().ReadMessage(ws).Return(websocket.PongMessage, nil, nil),
			mockWS.EXPECT().ReadMessage(ws).Return(websocket.CloseMessage, nil, nil),
		)

		h.runnerWebsocketHandleInputReceiver(se, ws)

		dump := logs.Dump()
		for _, expect := range []string{"Received Ping message from client", "Received Pong message from client", "Received Close message from client."} {
			if !strings.Contains(dump, expect) {
				t.Errorf("Wrong log. expect contains: %q", expect)
			}
		}
	})

	t.Run("read error ends the receiver", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		logs := logtest.New(t)
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{websocketHandler: mockWS}
		var ws *websocket.Conn

		mockWS.EXPECT().ReadMessage(ws).Return(0, nil, errors.New("connection reset"))

		h.runnerWebsocketHandleInputReceiver(se, ws)

		if !strings.Contains(logs.Dump(), "Could not read message from websocket") {
			t.Errorf("the read error log is missing")
		}
	})
}
