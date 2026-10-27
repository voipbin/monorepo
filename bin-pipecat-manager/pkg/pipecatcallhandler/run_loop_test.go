package pipecatcallhandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func newMediaHandleSession(ctx context.Context, conn *websocket.Conn, done chan struct{}) *pipecatcall.Session {
	ready := make(chan struct{})
	close(ready)

	return &pipecatcall.Session{
		Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-6201-4000-8000-000000000001")},
		Ctx:          ctx,
		ConnAst:      conn,
		ConnAstDone:  done,
		ConnAstReady: ready,
	}
}

func Test_runAsteriskReceivedMediaHandle_sendFailureKeepsReading(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)
	mockPF := NewMockPipecatframeHandler(mc)
	h := &pipecatcallHandler{websocketHandler: mockWS, pipecatframeHandler: mockPF}

	conn := &websocket.Conn{}
	done := make(chan struct{})
	se := newMediaHandleSession(context.Background(), conn, done)

	frame0 := []byte{0x01, 0x02}
	frame1 := []byte{0x03, 0x04}
	gomock.InOrder(
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, frame0, nil),
		mockWS.EXPECT().ReadMessage(conn).Return(websocket.BinaryMessage, frame1, nil),
		mockWS.EXPECT().ReadMessage(conn).Return(0, nil, &websocket.CloseError{Code: websocket.CloseNormalClosure}),
	)
	// the first frame fails to send; the loop goes on and the packet id still advances.
	gomock.InOrder(
		mockPF.EXPECT().SendAudio(se, uint64(0), frame0).Return(fmt.Errorf("send failed")),
		mockPF.EXPECT().SendAudio(se, uint64(1), frame1).Return(nil),
	)

	h.runAsteriskReceivedMediaHandle(se)

	// the lifecycle monitor of the start path waits for this channel.
	select {
	case <-done:
	default:
		t.Errorf("the done channel must be closed when the media handle returns")
	}
}

func Test_runAsteriskReceivedMediaHandle_contextCancelledInLoop(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)
	h := &pipecatcallHandler{websocketHandler: mockWS}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &websocket.Conn{}
	done := make(chan struct{})
	se := newMediaHandleSession(ctx, conn, done)

	// the session ends while a read is in flight; the next loop turn must stop without a second read.
	mockWS.EXPECT().ReadMessage(conn).DoAndReturn(func(_ *websocket.Conn) (int, []byte, error) {
		cancel()
		return websocket.TextMessage, []byte("ignored"), nil
	}).Times(1)

	h.runAsteriskReceivedMediaHandle(se)

	select {
	case <-done:
	default:
		t.Errorf("the done channel must be closed when the media handle returns")
	}
}
