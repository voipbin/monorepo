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

func Test_runAsteriskReceivedMediaHandle(t *testing.T) {
	tests := []struct {
		name string

		readMessages []struct {
			msgType int
			data    []byte
			err     error
		}

		expectAudioFrames int
	}{
		{
			name: "receives binary audio frames",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 2,
		},
		{
			name: "skips non-binary messages",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.TextMessage, data: []byte("text"), err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 1,
		},
		{
			name: "skips empty binary messages",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: []byte{}, err: nil},
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: fmt.Errorf("connection closed")},
			},
			expectAudioFrames: 1,
		},
		{
			name:              "nil ConnAst returns immediately",
			readMessages:      nil,
			expectAudioFrames: 0,
		},
		{
			name: "websocket close normal closure",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: websocket.BinaryMessage, data: make([]byte, 640), err: nil},
				{msgType: 0, data: nil, err: &websocket.CloseError{Code: websocket.CloseNormalClosure, Text: "normal"}},
			},
			expectAudioFrames: 1,
		},
		{
			name: "websocket close going away",
			readMessages: []struct {
				msgType int
				data    []byte
				err     error
			}{
				{msgType: 0, data: nil, err: &websocket.CloseError{Code: websocket.CloseGoingAway, Text: "going away"}},
			},
			expectAudioFrames: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockWS := NewMockWebsocketHandler(mc)
			mockPF := NewMockPipecatframeHandler(mc)

			var conn *websocket.Conn
			if tt.readMessages != nil {
				conn = &websocket.Conn{}
				for _, msg := range tt.readMessages {
					mockWS.EXPECT().ReadMessage(conn).Return(msg.msgType, msg.data, msg.err)
				}
			}

			if tt.expectAudioFrames > 0 {
				mockPF.EXPECT().SendAudio(gomock.Any(), gomock.Any(), gomock.Any()).Times(tt.expectAudioFrames).Return(nil)
			}

			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID: uuid.Must(uuid.NewV4()),
				},
				Ctx:     context.Background(),
				ConnAst: conn,
			}

			h := &pipecatcallHandler{
				websocketHandler:    mockWS,
				pipecatframeHandler: mockPF,
			}

			h.runAsteriskReceivedMediaHandle(se)
		})
	}
}

func Test_runAsteriskReceivedMediaHandle_contextCancelled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockWS := NewMockWebsocketHandler(mc)
	mockPF := NewMockPipecatframeHandler(mc)
	// No ReadMessage expectations — context is cancelled before any read

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID: uuid.Must(uuid.NewV4()),
		},
		Ctx:     ctx,
		ConnAst: &websocket.Conn{},
	}

	h := &pipecatcallHandler{
		websocketHandler:    mockWS,
		pipecatframeHandler: mockPF,
	}

	h.runAsteriskReceivedMediaHandle(se)
	// Should return without panic or hanging
}
