package streaminghandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

// newClosingServer returns a ws:// URL of a server that accepts the websocket
// upgrade and closes the connection without sending any message.
func newClosingServer(t *testing.T) string {
	t.Helper()

	upgrader := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		_ = conn.Close()
	}))
	t.Cleanup(srv.Close)

	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func Test_websocketConnect_closedBeforeMediaStart(t *testing.T) {
	conn, err := websocketConnect(context.Background(), newClosingServer(t))
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected an error when the peer closes before MEDIA_START")
	}
	if conn != nil {
		t.Errorf("expected no connection with an error, got: %v", conn)
	}
	if !strings.Contains(err.Error(), "could not read MEDIA_START message") {
		t.Errorf("Wrong error: %v", err)
	}
}

func Test_websocketConnect_unreachable(t *testing.T) {
	// A server that is shut down right away leaves a refused local port.
	srv := httptest.NewServer(http.NotFoundHandler())
	uri := "ws" + strings.TrimPrefix(srv.URL, "http")
	srv.Close()

	conn, err := websocketConnect(context.Background(), uri)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected a dial error")
	}
	if conn != nil || !strings.Contains(err.Error(), "could not dial WebSocket") {
		t.Errorf("Wrong result. conn: %v, err: %v", conn, err)
	}
}

func Test_Start_createFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	// no external media request is expected: the failure happens before it.
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	duplicateID := uuid.FromStringOrNil("e2b13e22-e9df-11ef-81b9-dfb396f7f633")
	h := &streamingHandler{
		utilHandler:    mockUtil,
		requestHandler: mockReq,
		notifyHandler:  mockNotify,
		mapStreaming: map[uuid.UUID]*streaming.Streaming{
			duplicateID: {},
		},
	}

	mockUtil.EXPECT().UUIDCreate().Return(duplicateID)

	res, err := h.Start(context.Background(), uuid.Nil, uuid.Nil, streaming.ReferenceTypeCall, uuid.Nil, "en-US", streaming.GenderFemale, streaming.DirectionIncoming)
	if res != nil || err == nil || !strings.Contains(err.Error(), "streaming already exists") {
		t.Errorf("Wrong result. res: %v, err: %v", res, err)
	}
	if len(h.mapStreaming) != 1 {
		t.Errorf("the existing streaming must be left untouched, got %d entries", len(h.mapStreaming))
	}
}
