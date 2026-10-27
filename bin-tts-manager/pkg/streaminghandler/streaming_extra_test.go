package streaminghandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/mock/gomock"
)

func Test_websocketWrite_fragments(t *testing.T) {
	client, server := newLocalWSPair(t)

	// 7 bytes at 3 bytes per frame: two full frames and a short last frame
	data := []byte{1, 2, 3, 4, 5, 6, 7}
	if err := websocketWrite(context.Background(), client, data, 3); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	expect := [][]byte{{1, 2, 3}, {4, 5, 6}, {7}}
	for i, want := range expect {
		if err := server.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatalf("could not set the read deadline: %v", err)
		}
		msgType, got, err := server.ReadMessage()
		if err != nil {
			t.Fatalf("could not read frame %d: %v", i, err)
		}
		if msgType != websocket.BinaryMessage || !reflect.DeepEqual(got, want) {
			t.Errorf("Wrong frame %d. expect: %v, got: %v (type %d)", i, want, got, msgType)
		}
	}
}

func Test_streamingHandler_Delete_unknownID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no event is expected: the mock fails the test if one is published
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	existingID := uuid.FromStringOrNil("6c2d1e00-0000-4000-8000-0000000000c1")
	h := &streamingHandler{
		notifyHandler: mockNotify,
		mapStreaming:  map[uuid.UUID]*streaming.Streaming{existingID: {}},
	}

	h.Delete(context.Background(), uuid.FromStringOrNil("6c2d1e00-0000-4000-8000-0000000000c2"))

	if _, ok := h.mapStreaming[existingID]; !ok {
		t.Error("deleting an unknown streaming must not touch the others")
	}
}
