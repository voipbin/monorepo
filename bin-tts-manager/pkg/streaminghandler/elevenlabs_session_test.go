package streaminghandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

// newLocalWSPair returns a connected client/server WebSocket pair on a local
// httptest server. Both ends are closed by t.Cleanup.
func newLocalWSPair(t *testing.T) (client *websocket.Conn, server *websocket.Conn) {
	t.Helper()

	upgrader := websocket.Upgrader{}
	serverCh := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverCh <- conn
	}))

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("could not dial the local websocket server: %v", err)
	}
	s := <-serverCh

	t.Cleanup(func() {
		_ = c.Close()
		_ = s.Close()
		srv.Close()
	})

	return c, s
}

// readServerJSON reads one text message from the server side and decodes it.
func readServerJSON(t *testing.T, server *websocket.Conn) ElevenlabsMessage {
	t.Helper()

	if err := server.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("could not set the read deadline: %v", err)
	}
	_, data, err := server.ReadMessage()
	if err != nil {
		t.Fatalf("could not read from the server side: %v", err)
	}

	var res ElevenlabsMessage
	if errUnmarshal := json.Unmarshal(data, &res); errUnmarshal != nil {
		t.Fatalf("could not decode the message %q: %v", string(data), errUnmarshal)
	}
	return res
}

func newTestElevenlabsConfig(connWebsock *websocket.Conn) *ElevenlabsConfig {
	ctx, cancel := context.WithCancel(context.Background())
	return &ElevenlabsConfig{
		Streaming: &streaming.Streaming{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("5b6a3c1e-0000-4000-8000-000000000001"),
			},
		},
		Ctx:         ctx,
		Cancel:      cancel,
		ConnWebsock: connWebsock,
		Message: &message.Message{
			Identity: commonidentity.Identity{
				ID: uuid.FromStringOrNil("5b6a3c1e-0000-4000-8000-000000000002"),
			},
		},
	}
}

func Test_elevenlabsHandler_invalidVendorConfig(t *testing.T) {
	h := &elevenlabsHandler{}

	var typedNil *ElevenlabsConfig
	tests := []struct {
		name string
		cfg  any
	}{
		{"wrong type", &GCPConfig{}},
		{"untyped nil", nil},
		{"typed nil", typedNil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := h.Run(tt.cfg); err == nil {
				t.Errorf("Run: expected an error for %s", tt.name)
			}
			if err := h.SayAdd(tt.cfg, "text"); err == nil {
				t.Errorf("SayAdd: expected an error for %s", tt.name)
			}
			if err := h.SayFlush(tt.cfg); err == nil {
				t.Errorf("SayFlush: expected an error for %s", tt.name)
			}
			if err := h.SayStop(tt.cfg); err == nil {
				t.Errorf("SayStop: expected an error for %s", tt.name)
			}
			if err := h.SayFinish(tt.cfg); err == nil {
				t.Errorf("SayFinish: expected an error for %s", tt.name)
			}
		})
	}
}

func Test_elevenlabsHandler_SayAdd(t *testing.T) {
	client, server := newLocalWSPair(t)
	cf := newTestElevenlabsConfig(client)
	h := &elevenlabsHandler{}

	if err := h.SayAdd(cf, "Hello "); err != nil {
		t.Fatalf("SayAdd: unexpected error: %v", err)
	}
	if got := readServerJSON(t, server); got.Text != "Hello " || !got.Flush {
		t.Errorf("Wrong frame. expected text %q with flush, got: %+v", "Hello ", got)
	}

	if err := h.SayAdd(cf, "world"); err != nil {
		t.Fatalf("SayAdd: unexpected error on the second call: %v", err)
	}
	if got := readServerJSON(t, server); got.Text != "world" {
		t.Errorf("Wrong second frame. expected text %q, got: %+v", "world", got)
	}

	if cf.Message.TotalMessage != "Hello world" {
		t.Errorf("Wrong TotalMessage. expected: %q, got: %q", "Hello world", cf.Message.TotalMessage)
	}
	if cf.Message.TotalCount != 2 {
		t.Errorf("Wrong TotalCount. expected: 2, got: %d", cf.Message.TotalCount)
	}
}

func Test_elevenlabsHandler_SayAdd_errors(t *testing.T) {
	h := &elevenlabsHandler{}

	t.Run("nil websocket connection", func(t *testing.T) {
		cf := newTestElevenlabsConfig(nil)
		if err := h.SayAdd(cf, "text"); err == nil {
			t.Fatal("expected an error")
		}
		if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
			t.Errorf("message must not change on error. got: %+v", cf.Message)
		}
	})

	t.Run("write failure on a closed connection", func(t *testing.T) {
		client, _ := newLocalWSPair(t)
		cf := newTestElevenlabsConfig(client)
		_ = client.Close()

		if err := h.SayAdd(cf, "text"); err == nil {
			t.Fatal("expected an error")
		}
		if cf.Message.TotalCount != 0 || cf.Message.TotalMessage != "" {
			t.Errorf("message must not change on a failed write. got: %+v", cf.Message)
		}
	})
}

func Test_elevenlabsHandler_SayFlush(t *testing.T) {
	h := &elevenlabsHandler{}

	t.Run("sends an empty flush frame", func(t *testing.T) {
		client, server := newLocalWSPair(t)
		cf := newTestElevenlabsConfig(client)

		if err := h.SayFlush(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := readServerJSON(t, server); got.Text != "" || !got.Flush {
			t.Errorf("Wrong frame. expected an empty text with flush, got: %+v", got)
		}
		if cf.Message.TotalCount != 0 {
			t.Errorf("flush must not count as a message. got: %d", cf.Message.TotalCount)
		}
	})

	t.Run("nil websocket connection", func(t *testing.T) {
		cf := newTestElevenlabsConfig(nil)
		if err := h.SayFlush(cf); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func Test_elevenlabsHandler_SayStop(t *testing.T) {
	cf := newTestElevenlabsConfig(nil)
	h := &elevenlabsHandler{}

	if err := h.SayStop(cf); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled")
	}
}

func Test_elevenlabsHandler_SayFinish(t *testing.T) {
	h := &elevenlabsHandler{}

	t.Run("everything is already played, so the session is torn down", func(t *testing.T) {
		client, _ := newLocalWSPair(t)
		cf := newTestElevenlabsConfig(client)

		if err := h.SayFinish(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cf.Message.Finish {
			t.Error("Finish flag must be set")
		}
		if cf.ConnWebsock != nil {
			t.Error("the websocket connection must be released")
		}
		select {
		case <-cf.Ctx.Done():
		default:
			t.Error("the session context must be cancelled")
		}
	})

	t.Run("audio is still pending, so the session stays alive", func(t *testing.T) {
		client, _ := newLocalWSPair(t)
		cf := newTestElevenlabsConfig(client)
		cf.Message.TotalCount = 2
		cf.Message.PlayedCount = 1

		if err := h.SayFinish(cf); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cf.Message.Finish {
			t.Error("Finish flag must be set")
		}
		if cf.ConnWebsock == nil {
			t.Error("the websocket connection must be kept while audio is pending")
		}
		select {
		case <-cf.Ctx.Done():
			t.Error("the session context must stay alive while audio is pending")
		default:
		}
	})
}

func Test_elevenlabsHandler_terminate_nilConnection(t *testing.T) {
	cf := newTestElevenlabsConfig(nil)
	h := &elevenlabsHandler{}

	h.terminate(cf)

	select {
	case <-cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled")
	}
}

func Test_elevenlabsHandler_runKeepAlive_exitsOnCancel(t *testing.T) {
	cf := newTestElevenlabsConfig(nil)
	cf.Cancel()
	h := &elevenlabsHandler{}

	done := make(chan struct{})
	go func() {
		h.runKeepAlive(cf)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("runKeepAlive did not exit after the context was cancelled")
	}
}

// elevenlabsProcessEnv bundles what runProcess needs: the ElevenLabs side
// (server end is driven by the test) and the Asterisk side (server end
// collects the audio frames the handler writes).
type elevenlabsProcessEnv struct {
	cf *ElevenlabsConfig
	h  *elevenlabsHandler

	elevenServer *websocket.Conn
	astServer    *websocket.Conn

	astClient *websocket.Conn
	finished  chan struct{}
}

func newElevenlabsProcessEnv(t *testing.T, mc *gomock.Controller) *elevenlabsProcessEnv {
	t.Helper()

	elevenClient, elevenServer := newLocalWSPair(t)
	astClient, astServer := newLocalWSPair(t)

	cf := newTestElevenlabsConfig(elevenClient)
	cf.ConnAst = astClient

	finished := make(chan struct{})
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypePlayStarted, gomock.Any())
	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypePlayFinished, gomock.Any()).Do(
		func(ctx context.Context, eventType string, data any) { close(finished) },
	)

	env := &elevenlabsProcessEnv{
		cf:           cf,
		h:            &elevenlabsHandler{notifyHandler: mockNotify},
		elevenServer: elevenServer,
		astServer:    astServer,
		astClient:    astClient,
		finished:     finished,
	}
	t.Cleanup(cf.Cancel)

	return env
}

func (e *elevenlabsProcessEnv) runAndWait(t *testing.T) {
	t.Helper()

	go e.h.runProcess(e.cf)

	select {
	case <-e.finished:
	case <-time.After(5 * time.Second):
		t.Fatal("runProcess did not finish in time")
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("could not marshal: %v", err)
	}
	return data
}

func Test_elevenlabsHandler_runProcess_playsAudioUntilFinished(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newElevenlabsProcessEnv(t, mc)
	env.cf.Message.Finish = true
	env.cf.Message.TotalCount = 2
	env.cf.Message.TotalMessage = "hi!"

	audio := []byte{1, 2, 3, 4, 5, 6, 7, 8}

	go func() {
		first := ElevenlabsResponse{
			Audio:     base64.StdEncoding.EncodeToString(audio),
			Alignment: ElevenlabsAlignment{Chars: []string{"h", "i"}},
		}
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, first))

		// control messages must not count as played
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, []byte(`{"status":"ok","error":"minor warning"}`))

		second := ElevenlabsResponse{Alignment: ElevenlabsAlignment{Chars: []string{"!"}}}
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, second))
	}()

	env.runAndWait(t)

	if env.cf.Message.PlayedMessage != "hi!" {
		t.Errorf("Wrong PlayedMessage. expected: %q, got: %q", "hi!", env.cf.Message.PlayedMessage)
	}
	if env.cf.Message.PlayedCount != 2 {
		t.Errorf("Wrong PlayedCount. expected: 2, got: %d", env.cf.Message.PlayedCount)
	}

	if err := env.astServer.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("could not set the read deadline: %v", err)
	}
	msgType, frame, err := env.astServer.ReadMessage()
	if err != nil {
		t.Fatalf("the asterisk side did not receive the audio: %v", err)
	}
	if msgType != websocket.BinaryMessage || string(frame) != string(audio) {
		t.Errorf("Wrong audio frame. expected binary %v, got type %d: %v", audio, msgType, frame)
	}
}

func Test_elevenlabsHandler_runProcess_doesNotFinishWhileMessagesAreOutstanding(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newElevenlabsProcessEnv(t, mc)
	env.cf.Message.Finish = true
	env.cf.Message.TotalCount = 2

	go func() {
		one := ElevenlabsResponse{Alignment: ElevenlabsAlignment{Chars: []string{"a"}}}
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, one))

		// the close frame is read only after the message above was handled
		_ = env.elevenServer.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}()

	env.runAndWait(t)

	// the loop ended because of the close frame, so only one message was played
	if env.cf.Message.PlayedCount != 1 || env.cf.Message.PlayedMessage != "a" {
		t.Errorf("unexpected played state: count %d message %q", env.cf.Message.PlayedCount, env.cf.Message.PlayedMessage)
	}
}

func Test_elevenlabsHandler_runProcess_skipsMalformedJSON(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newElevenlabsProcessEnv(t, mc)
	env.cf.Message.Finish = true
	env.cf.Message.TotalCount = 1

	go func() {
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, []byte("this is not json"))
		ok := ElevenlabsResponse{Alignment: ElevenlabsAlignment{Chars: []string{"o", "k"}}}
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, ok))
	}()

	env.runAndWait(t)

	if env.cf.Message.PlayedMessage != "ok" || env.cf.Message.PlayedCount != 1 {
		t.Errorf("a malformed message must be skipped. got message %q count %d",
			env.cf.Message.PlayedMessage, env.cf.Message.PlayedCount)
	}
}

func Test_elevenlabsHandler_runProcess_stopsOnInvalidAudio(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newElevenlabsProcessEnv(t, mc)

	go func() {
		bad := ElevenlabsResponse{Audio: "***not base64***", Alignment: ElevenlabsAlignment{Chars: []string{"x"}}}
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, bad))
	}()

	env.runAndWait(t)

	if env.cf.Message.PlayedCount != 0 || env.cf.Message.PlayedMessage != "" {
		t.Errorf("nothing may be counted after an undecodable audio. got: %+v", env.cf.Message)
	}
}

func Test_elevenlabsHandler_runProcess_stopsWhenAsteriskWriteFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newElevenlabsProcessEnv(t, mc)
	_ = env.astClient.Close()

	go func() {
		resp := ElevenlabsResponse{
			Audio:     base64.StdEncoding.EncodeToString([]byte{9, 9, 9}),
			Alignment: ElevenlabsAlignment{Chars: []string{"x"}},
		}
		_ = env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, resp))
	}()

	env.runAndWait(t)

	if env.cf.Message.PlayedCount != 0 {
		t.Errorf("a failed audio write must end the loop before counting. got: %d", env.cf.Message.PlayedCount)
	}
}

func Test_elevenlabsHandler_runProcess_endsOnServerDisconnect(t *testing.T) {
	tests := []struct {
		name  string
		close func(server *websocket.Conn)
	}{
		{
			name: "normal closure",
			close: func(server *websocket.Conn) {
				_ = server.WriteMessage(websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
			},
		},
		{
			name: "abrupt disconnect",
			close: func(server *websocket.Conn) {
				_ = server.Close()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			env := newElevenlabsProcessEnv(t, mc)
			tt.close(env.elevenServer)

			env.runAndWait(t)

			select {
			case <-env.cf.Ctx.Done():
			default:
				t.Error("runProcess must cancel the session context when it ends")
			}
		})
	}
}

func Test_elevenlabsHandler_Run_playsAudioAndTearsDownOnAsteriskDisconnect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	env := newElevenlabsProcessEnv(t, mc)
	env.cf.ConnAstDone = make(chan struct{})

	h := env.h
	runDone := make(chan error, 1)
	go func() { runDone <- h.Run(env.cf) }()

	audio := []byte{4, 4, 4, 4}
	resp := ElevenlabsResponse{Audio: base64.StdEncoding.EncodeToString(audio)}
	if err := env.elevenServer.WriteMessage(websocket.TextMessage, mustMarshal(t, resp)); err != nil {
		t.Fatalf("could not send the audio: %v", err)
	}

	// wait until the audio reached Asterisk, which proves runProcess is past
	// its use of the ElevenLabs connection, then simulate the Asterisk drop.
	if err := env.astServer.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("could not set the read deadline: %v", err)
	}
	if _, frame, err := env.astServer.ReadMessage(); err != nil || string(frame) != string(audio) {
		t.Fatalf("Wrong audio frame. err: %v, got: %v", err, frame)
	}
	close(env.cf.ConnAstDone)

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run returned an error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after the Asterisk side was lost")
	}

	select {
	case <-env.finished:
	case <-time.After(5 * time.Second):
		t.Fatal("PlayFinished was not published after the teardown")
	}

	select {
	case <-env.cf.Ctx.Done():
	default:
		t.Error("the session context must be cancelled after Run returns")
	}
}
