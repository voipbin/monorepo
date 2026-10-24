package streaminghandler

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"

	cmexternalmedia "monorepo/bin-call-manager/models/externalmedia"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-transcribe-manager/models/streaming"
	"monorepo/bin-transcribe-manager/models/transcribe"
	"monorepo/bin-transcribe-manager/models/transcript"
)

func newStartTestHandler(mc *gomock.Controller) (*streamingHandler, *utilhandler.MockUtilHandler, *requesthandler.MockRequestHandler, *notifyhandler.MockNotifyHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &streamingHandler{
		utilHandler:   mockUtil,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		mapStreaming:  make(map[uuid.UUID]*streaming.Streaming),
	}

	return h, mockUtil, mockReq, mockNotify
}

func Test_Start_createFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, _, _ := newStartTestHandler(mc)
	ctx := context.Background()

	// an entry with the generated id exists already, so Create refuses it. The request handler
	// has no expectation: the external media must not be started.
	id := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000001")
	h.mapStreaming[id] = &streaming.Streaming{Identity: commonidentity.Identity{ID: id}}
	mockUtil.EXPECT().UUIDCreate().Return(id)

	res, err := h.Start(ctx, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty)
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
	}
}

func Test_Start_externalMediaStartFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockReq, mockNotify := newStartTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000011")
	referenceID := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000012")
	startErr := fmt.Errorf("call manager unavailable")

	mockUtil.EXPECT().UUIDCreate().Return(id)
	mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, gomock.Any())
	mockReq.EXPECT().CallV1ExternalMediaStart(ctx, id, cmexternalmedia.ReferenceType(transcribe.ReferenceTypeCall), referenceID, "INCOMING", defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat, cmexternalmedia.Direction(transcript.DirectionOut), cmexternalmedia.DirectionNone).Return(nil, startErr)

	res, err := h.Start(ctx, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, referenceID, "en-US", transcript.DirectionOut, transcribe.ProviderEmpty)
	if err != startErr || res != nil {
		t.Errorf("Wrong match. expect: nil and the start error, got: %v and %v", res, err)
	}
}

func Test_Start_websocketFailureAndCleanupFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockReq, mockNotify := newStartTestHandler(mc)
	ctx := context.Background()

	id := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000021")
	em := &cmexternalmedia.ExternalMedia{ID: id, MediaURI: "ws://127.0.0.1:0/none"}

	mockUtil.EXPECT().UUIDCreate().Return(id)
	mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, gomock.Any())
	mockReq.EXPECT().CallV1ExternalMediaStart(ctx, id, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(em, nil)
	// the orphan cleanup fails; Start must still delete the record and report the websocket error
	mockReq.EXPECT().CallV1ExternalMediaStop(ctx, id).Return(nil, fmt.Errorf("stop failed"))
	mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStopped, gomock.Any())

	res, err := h.Start(ctx, uuid.Nil, uuid.Nil, transcribe.ReferenceTypeCall, uuid.Nil, "en-US", transcript.DirectionIn, transcribe.ProviderEmpty)
	if err == nil || res != nil {
		t.Fatalf("Wrong match. expect: nil and err, got: %v and %v", res, err)
	}
	if _, errGet := h.Get(ctx, id); errGet == nil {
		t.Errorf("Wrong match. expect the streaming record removed, got: still present")
	}
}

func Test_Start_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockUtil, mockReq, mockNotify := newStartTestHandler(mc)
	ctx := context.Background()

	url := newTestMediaServer(t, func(conn net.Conn) {
		serverWriteFrame(conn, websocket.TextMessage, []byte("MEDIA_START"))
		serverWaitClosed(conn)
	})

	id := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000031")
	customerID := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000032")
	transcribeID := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000033")
	em := &cmexternalmedia.ExternalMedia{ID: id, MediaURI: url}

	mockUtil.EXPECT().UUIDCreate().Return(id)
	mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStarted, gomock.Any())
	mockReq.EXPECT().CallV1ExternalMediaStart(ctx, id, cmexternalmedia.ReferenceType(transcribe.ReferenceTypeConfbridge), gomock.Any(), "INCOMING", defaultEncapsulation, defaultTransport, "", defaultConnectionType, defaultFormat, cmexternalmedia.Direction(transcript.DirectionIn), cmexternalmedia.DirectionNone).Return(em, nil)

	// no STT client is initialized in this handler, so the spawned runSTT returns at once
	res, err := h.Start(ctx, customerID, transcribeID, transcribe.ReferenceTypeConfbridge, uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000034"), "ko-KR", transcript.DirectionIn, transcribe.ProviderEmpty)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	defer func() { _ = res.ConnAst.Close() }()

	if res.ConnAst == nil {
		t.Fatalf("Wrong match. expect the websocket connection to be stored on the streaming")
	}
	if res.ID != id || res.CustomerID != customerID || res.TranscribeID != transcribeID || res.Language != "ko-KR" || res.Direction != transcript.DirectionIn {
		t.Errorf("Wrong match. got: %v", res)
	}

	stored, errGet := h.Get(ctx, id)
	if errGet != nil || stored != res {
		t.Errorf("Wrong match. expect the same streaming kept in the session map, got: %v (err: %v)", stored, errGet)
	}
}

func Test_runSTT_noInitializedProvider(t *testing.T) {
	tests := []struct {
		name string

		provider transcribe.Provider
	}{
		{"empty provider", transcribe.ProviderEmpty},
		{"requested provider is not initialized", transcribe.Provider("gcp")},
		{"requested aws provider is not initialized", transcribe.Provider("aws")},
		{"unknown provider falls back to the default order", transcribe.Provider("unknown")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &streamingHandler{}
			st := &streaming.Streaming{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000041")}}

			done := make(chan struct{})
			go func() {
				defer close(done)
				h.runSTT(st, tt.provider)
			}()

			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatalf("Wrong match. runSTT did not return without an initialized provider")
			}
		})
	}
}

func Test_Stop_missingStreaming(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// the request handler has no expectation: an unknown streaming must not reach the external media
	h, _, _, _ := newStartTestHandler(mc)

	res, err := h.Stop(context.Background(), uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000051"))
	if err == nil || res != nil {
		t.Errorf("Wrong match. expect: nil and err, got: %v and %v", res, err)
	}
}

func Test_Stop_closesAsteriskConnection(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockReq, mockNotify := newStartTestHandler(mc)
	ctx := context.Background()

	// the server reports whether the client closed the connection (true) or the wait timed out (false)
	closedByClient := make(chan bool, 1)
	url := newTestMediaServer(t, func(conn net.Conn) {
		serverWriteFrame(conn, websocket.TextMessage, []byte("MEDIA_START"))
		closedByClient <- serverWaitClosed(conn)
	})

	conn, err := websocketConnect(ctx, url)
	if err != nil {
		t.Fatalf("could not connect to the test server. err: %v", err)
	}

	id := uuid.FromStringOrNil("9e3f0c10-0000-11ee-8b11-000000000061")
	h.mapStreaming[id] = &streaming.Streaming{Identity: commonidentity.Identity{ID: id}, ConnAst: conn}

	mockReq.EXPECT().CallV1ExternalMediaStop(ctx, id).Return(&cmexternalmedia.ExternalMedia{ID: id}, nil)
	mockNotify.EXPECT().PublishEvent(ctx, streaming.EventTypeStreamingStopped, gomock.Any())

	res, errStop := h.Stop(ctx, id)
	if errStop != nil || res == nil || res.ID != id {
		t.Fatalf("Wrong match. expect: the stopped streaming, got: %v (err: %v)", res, errStop)
	}

	// keep the client connection reachable until here, so that only an explicit Close in Stop can
	// close it; otherwise the garbage collector could close the socket and hide a missing Close
	runtime.KeepAlive(conn)

	select {
	case closedByStop := <-closedByClient:
		if !closedByStop {
			t.Errorf("Wrong match. expect the server side to observe the connection closed by Stop, got: wait timed out")
		}
	case <-time.After(10 * time.Second):
		t.Errorf("Wrong match. expect the server side to report the connection state")
	}

	if _, errGet := h.Get(ctx, id); errGet == nil {
		t.Errorf("Wrong match. expect the streaming removed from the session map")
	}
}
