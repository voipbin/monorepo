package streaminghandler

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/message"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

// elevenlabsRequestSeen is what the local test server observed of the
// handshake request the handler sent.
type elevenlabsRequestSeen struct {
	host   string
	path   string
	query  map[string]string
	apiKey string
}

// useLocalElevenlabsServer points websocket.DefaultDialer at a local TLS test
// server for the duration of the test, so the handler's fixed wss host is
// reached without any external connection. When accept is false the server
// refuses the upgrade. The observed handshake requests are sent on the
// returned channel. The original dialer is restored by t.Cleanup.
func useLocalElevenlabsServer(t *testing.T, accept bool) <-chan elevenlabsRequestSeen {
	t.Helper()

	seen := make(chan elevenlabsRequestSeen, 4)
	release := make(chan struct{})
	upgrader := websocket.Upgrader{}

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- elevenlabsRequestSeen{
			host: r.Host,
			path: r.URL.Path,
			query: map[string]string{
				"model_id":      r.URL.Query().Get("model_id"),
				"output_format": r.URL.Query().Get("output_format"),
			},
			apiKey: r.Header.Get("xi-api-key"),
		}

		if !accept {
			http.Error(w, "refused", http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()

		<-release
	}))

	pool := x509.NewCertPool()
	pool.AddCert(srv.Certificate())
	addr := srv.Listener.Addr().String()

	original := websocket.DefaultDialer
	websocket.DefaultDialer = &websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
		// the test certificate is issued for this name
		TLSClientConfig:  &tls.Config{RootCAs: pool, ServerName: "example.com", MinVersion: tls.VersionTLS12},
		HandshakeTimeout: 5 * time.Second,
	}

	// cleanups run last in, first out: the dialer is restored, the handlers are
	// released, then the server is closed
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) })
	t.Cleanup(func() { websocket.DefaultDialer = original })

	return seen
}

func receiveElevenlabsRequest(t *testing.T, seen <-chan elevenlabsRequestSeen) elevenlabsRequestSeen {
	t.Helper()

	select {
	case res := <-seen:
		return res
	case <-time.After(5 * time.Second):
		t.Fatal("the local server did not receive a handshake request")
		return elevenlabsRequestSeen{}
	}
}

func Test_elevenlabsHandler_Init(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	seen := useLocalElevenlabsServer(t, true)

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &elevenlabsHandler{
		notifyHandler: mockNotify,
		apiKey:        "fake-api-key",
	}

	streamingID := uuid.FromStringOrNil("5b6a3c1e-0000-4000-8000-0000000000a1")
	messageID := uuid.FromStringOrNil("5b6a3c1e-0000-4000-8000-0000000000a2")
	customerID := uuid.FromStringOrNil("5b6a3c1e-0000-4000-8000-0000000000a3")

	st := &streaming.Streaming{
		VoiceID:     "fake-voice-0000000001",
		MessageID:   messageID,
		ConnAstDone: make(chan struct{}),
	}
	st.ID = streamingID
	st.CustomerID = customerID

	var published *message.Message
	mockNotify.EXPECT().PublishEvent(gomock.Any(), message.EventTypeInitiated, gomock.Any()).Do(
		func(ctx context.Context, eventType string, data any) {
			published, _ = data.(*message.Message)
		},
	)

	res, err := h.Init(context.Background(), st)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	cf, ok := res.(*ElevenlabsConfig)
	if !ok || cf == nil {
		t.Fatalf("Wrong match. expect: *ElevenlabsConfig, got: %T", res)
	}
	t.Cleanup(cf.Cancel)
	t.Cleanup(func() { _ = cf.ConnWebsock.Close() })

	req := receiveElevenlabsRequest(t, seen)
	if req.host != "api.elevenlabs.io" {
		t.Errorf("Wrong host. expect: api.elevenlabs.io, got: %s", req.host)
	}
	if req.path != "/v1/text-to-speech/fake-voice-0000000001/stream-input" {
		t.Errorf("Wrong path. got: %s", req.path)
	}
	if req.query["model_id"] != "eleven_multilingual_v2" || req.query["output_format"] != "pcm_16000" {
		t.Errorf("Wrong query. got: %v", req.query)
	}
	if req.apiKey != "fake-api-key" {
		t.Errorf("Wrong API key header. got: %q", req.apiKey)
	}

	if cf.Streaming != st || cf.ConnWebsock == nil || cf.Ctx == nil || cf.Cancel == nil {
		t.Errorf("Wrong config. got: %+v", cf)
	}
	if cf.ConnAstDone != st.ConnAstDone {
		t.Error("the Asterisk disconnect channel must be taken from the streaming")
	}
	if cf.Message == nil || cf.Message.ID != messageID || cf.Message.CustomerID != customerID || cf.Message.StreamingID != streamingID {
		t.Errorf("Wrong message. got: %+v", cf.Message)
	}
	if published != cf.Message {
		t.Error("the initiated event must carry the session message")
	}
	if cf.Ctx.Err() != nil {
		t.Error("the session context must stay alive after Init")
	}
}

func Test_elevenlabsHandler_Init_connectFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	seen := useLocalElevenlabsServer(t, false)

	// no event is expected: the mock fails the test if one is published
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &elevenlabsHandler{
		notifyHandler: mockNotify,
		apiKey:        "fake-api-key",
	}

	st := &streaming.Streaming{VoiceID: "fake-voice-0000000002"}

	res, err := h.Init(context.Background(), st)
	if res != nil || err == nil {
		t.Fatalf("Wrong match. expect: error, got: %v, %v", res, err)
	}
	for _, expect := range []string{"failed to initialize ElevenLabs WebSocket connection", "failed to connect to ElevenLabs WebSocket", "bad handshake"} {
		if !strings.Contains(err.Error(), expect) {
			t.Errorf("Wrong error. expect to contain: %q, got: %v", expect, err)
		}
	}

	req := receiveElevenlabsRequest(t, seen)
	if req.apiKey != "fake-api-key" {
		t.Errorf("the refused request must still carry the API key. got: %q", req.apiKey)
	}
}

func Test_elevenlabsHandler_connect_voiceSelection(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	seen := useLocalElevenlabsServer(t, true)
	h := &elevenlabsHandler{apiKey: "fake-api-key"}

	// no explicit voice and no activeflow: the language and gender mapping applies
	st := &streaming.Streaming{Language: "korean", Gender: streaming.GenderFemale}

	conn, err := h.connect(context.Background(), st)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	req := receiveElevenlabsRequest(t, seen)
	expectPath := "/v1/text-to-speech/" + elevenlabsVoiceIDMap["korean_female"] + "/stream-input"
	if req.path != expectPath {
		t.Errorf("Wrong path. expect: %s, got: %s", expectPath, req.path)
	}
}

// secondCheckCancelledContext reports the context as alive on the first Done
// call and as cancelled on every later one. readConnWebsock checks Done once
// at the top of its loop and again while handing a message over, so this makes
// the second check deterministic without timing.
type secondCheckCancelledContext struct {
	context.Context

	calls  atomic.Int32
	open   chan struct{}
	closed chan struct{}
}

func newSecondCheckCancelledContext() *secondCheckCancelledContext {
	closed := make(chan struct{})
	close(closed)

	return &secondCheckCancelledContext{
		Context: context.Background(),
		open:    make(chan struct{}),
		closed:  closed,
	}
}

func (c *secondCheckCancelledContext) Done() <-chan struct{} {
	if c.calls.Add(1) == 1 {
		return c.open
	}
	return c.closed
}

func Test_readConnWebsock_droppedWhenContextCancelled(t *testing.T) {
	client, server := newLocalWSPair(t)

	// nobody receives from msgCh, so the message can only be dropped
	msgCh := make(chan []byte)
	errCh := make(chan error, 1)
	h := &elevenlabsHandler{}

	if err := server.WriteMessage(websocket.TextMessage, []byte(`{"audio":""}`)); err != nil {
		t.Fatalf("could not write to the client: %v", err)
	}

	done := make(chan struct{})
	go func() {
		h.readConnWebsock(newSecondCheckCancelledContext(), uuid.Nil, client, msgCh, errCh)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("readConnWebsock did not return after the context was cancelled")
	}

	if v, ok := <-msgCh; ok {
		t.Errorf("Wrong match. expect: msgCh closed without a message, got: %q", v)
	}
	if err, ok := <-errCh; ok {
		t.Errorf("Wrong match. expect: no read error, got: %v", err)
	}
}
