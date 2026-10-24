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

func newSenderTestSession(bufferSize int) (*pipecatcall.Session, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	return &pipecatcall.Session{
		Identity:            commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-3001-4000-8000-000000000001")},
		Ctx:                 ctx,
		Cancel:              cancel,
		RunnerWebsocketChan: make(chan *pipecatcall.SessionFrame, bufferSize),
	}, cancel
}

func Test_pushFrame_backpressure(t *testing.T) {
	first := &pipecatcall.SessionFrame{MessageType: websocket.BinaryMessage, Data: []byte("first")}
	second := &pipecatcall.SessionFrame{MessageType: websocket.BinaryMessage, Data: []byte("second")}

	t.Run("full channel without a receiver drops the frame after the timeout", func(t *testing.T) {
		se, cancel := newSenderTestSession(1)
		defer cancel()
		se.RunnerWebsocketChan <- first

		h := &pipecatframeHandler{}
		h.pushFrame(se, second)

		if got := se.DroppedFrames.Load(); got != 1 {
			t.Errorf("wrong dropped count. expect: 1, got: %d", got)
		}
		if len(se.RunnerWebsocketChan) != 1 {
			t.Fatalf("the channel must still hold exactly the first frame, len: %d", len(se.RunnerWebsocketChan))
		}
		if got := <-se.RunnerWebsocketChan; got != first {
			t.Errorf("the dropped frame must not replace the queued one")
		}
	})

	t.Run("cancelled context on a full channel returns without counting a drop", func(t *testing.T) {
		se, cancel := newSenderTestSession(1)
		se.RunnerWebsocketChan <- first
		cancel()

		h := &pipecatframeHandler{}
		h.pushFrame(se, second)

		if got := se.DroppedFrames.Load(); got != 0 {
			t.Errorf("a cancelled push must not count as a drop, got: %d", got)
		}
		if len(se.RunnerWebsocketChan) != 1 {
			t.Errorf("nothing may be queued, len: %d", len(se.RunnerWebsocketChan))
		}
	})

	t.Run("room in the channel queues the frame", func(t *testing.T) {
		se, cancel := newSenderTestSession(1)
		defer cancel()

		h := &pipecatframeHandler{}
		h.pushFrame(se, first)

		if got := <-se.RunnerWebsocketChan; got != first {
			t.Errorf("wrong queued frame")
		}
		if got := se.DroppedFrames.Load(); got != 0 {
			t.Errorf("wrong dropped count, got: %d", got)
		}
	})
}

func Test_RunSender_exitConditions(t *testing.T) {
	t.Run("cancelled context stops the sender", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// no WriteMessage expectation: nothing may be written.
		h := &pipecatframeHandler{websocketHandler: NewMockWebsocketHandler(mc)}

		se, cancel := newSenderTestSession(1)
		cancel()

		h.RunSender(se, nil)
	})

	t.Run("closed channel stops the sender", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatframeHandler{websocketHandler: NewMockWebsocketHandler(mc)}

		se, cancel := newSenderTestSession(1)
		defer cancel()
		close(se.RunnerWebsocketChan)

		h.RunSender(se, nil)
	})

	t.Run("write failure stops the sender and leaves later frames queued", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatframeHandler{websocketHandler: mockWS}

		se, cancel := newSenderTestSession(2)
		defer cancel()
		se.RunnerWebsocketChan <- &pipecatcall.SessionFrame{MessageType: websocket.BinaryMessage, Data: []byte("one")}
		se.RunnerWebsocketChan <- &pipecatcall.SessionFrame{MessageType: websocket.BinaryMessage, Data: []byte("two")}

		mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.BinaryMessage, []byte("one")).Return(fmt.Errorf("write failed")).Times(1)

		h.RunSender(se, nil)

		if len(se.RunnerWebsocketChan) != 1 {
			t.Errorf("the second frame must not be consumed, len: %d", len(se.RunnerWebsocketChan))
		}
	})

	t.Run("nil frame is skipped and later frames are sent", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatframeHandler{websocketHandler: mockWS}

		se, cancel := newSenderTestSession(3)
		defer cancel()
		se.RunnerWebsocketChan <- nil
		se.RunnerWebsocketChan <- &pipecatcall.SessionFrame{MessageType: websocket.TextMessage, Data: []byte("after-nil")}
		close(se.RunnerWebsocketChan)

		mockWS.EXPECT().WriteMessage(gomock.Any(), websocket.TextMessage, []byte("after-nil")).Return(nil).Times(1)

		h.RunSender(se, nil)
	})
}
