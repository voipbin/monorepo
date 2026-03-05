package pipecatcallhandler

import (
	"context"
	"sync"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"
	gomock "go.uber.org/mock/gomock"
)

func Test_receiveMessageFrameTypeMessage(t *testing.T) {

	tests := []struct {
		name string

		se *pipecatcall.Session
		m  []byte

		responseUUID  uuid.UUID
		expectEvent   string
		expectMessage message.Message
	}{
		{
			name: "bot-transcription",

			se: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54416ae0-af23-11f0-8991-07dd3ffd4def"),
					CustomerID: uuid.FromStringOrNil("546f7606-af23-11f0-a7ca-c32fd2659ee7"),
				},
				Ctx: context.Background(),
			},
			m: []byte(`{
				"label": "rtvi-ai",
				"type": "bot-transcription",
				"data": {"text": " How can I assist you today?"}
			}`),

			responseUUID: uuid.FromStringOrNil("c15f98f8-af1f-11f0-b009-535ac8cbc876"),
			expectEvent:  message.EventTypeBotTranscription,
			expectMessage: message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c15f98f8-af1f-11f0-b009-535ac8cbc876"),
					CustomerID: uuid.FromStringOrNil("546f7606-af23-11f0-a7ca-c32fd2659ee7"),
				},
				PipecatcallID: uuid.FromStringOrNil("54416ae0-af23-11f0-8991-07dd3ffd4def"),
				Text:          " How can I assist you today?",
			},
		},
		{
			name: "user-transcription",
			se: &pipecatcall.Session{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54986764-af23-11f0-9793-db91dfe17f29"),
					CustomerID: uuid.FromStringOrNil("54c1efee-af23-11f0-af7c-a7f393ea7de5"),
				},
				Ctx: context.Background(),
			},
			m: []byte(`{
				"label": "rtvi-ai",
				"type": "user-transcription",
				"data": {"text": "to by the way, who are you?", "user_id": "", "timestamp": "2025-10-22T02:38:39.119+00:00", "final": true}
			}`),

			responseUUID: uuid.FromStringOrNil("54eb0456-af23-11f0-986c-4bb2d9cd75de"),
			expectEvent:  message.EventTypeUserTranscription,

			expectMessage: message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54eb0456-af23-11f0-986c-4bb2d9cd75de"),
					CustomerID: uuid.FromStringOrNil("54c1efee-af23-11f0-af7c-a7f393ea7de5"),
				},
				PipecatcallID: uuid.FromStringOrNil("54986764-af23-11f0-9793-db91dfe17f29"),
				Text:          "to by the way, who are you?",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := pipecatcallHandler{
				notifyHandler: mockNotify,
				utilHandler:   mockUtil,
			}

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			// PublishEvent is now called in a goroutine; use WaitGroup to synchronize.
			var wg sync.WaitGroup
			wg.Add(1)
			mockNotify.EXPECT().PublishEvent(tt.se.Ctx, tt.expectEvent, tt.expectMessage).Do(
				func(any, any, any) { wg.Done() },
			)

			if err := h.receiveMessageFrameTypeMessage(tt.se, tt.m); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			wg.Wait()
		})
	}
}

func Test_runnerWebsocketHandleAudio(t *testing.T) {

	t.Run("16kHz mono audio passes through without resampling", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{
			websocketHandler: mockWS,
		}

		// ConnAst is non-nil so WriteMessage will be called to forward
		// the audio data directly to Asterisk.
		conn := &websocket.Conn{}
		connAstReady := make(chan struct{})
		close(connAstReady)
		se := &pipecatcall.Session{
			Ctx:          context.Background(),
			ConnAst:      conn,
			ConnAstReady: connAstReady,
		}

		data := []byte{0x01, 0x02, 0x03, 0x04}
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, data).Return(nil)

		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, data); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("non-16kHz audio is resampled before writing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAudio := NewMockAudiosocketHandler(mc)
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{
			audiosocketHandler: mockAudio,
			websocketHandler:   mockWS,
		}

		conn := &websocket.Conn{}
		connAstReady := make(chan struct{})
		close(connAstReady)
		se := &pipecatcall.Session{
			Ctx:          context.Background(),
			ConnAst:      conn,
			ConnAstReady: connAstReady,
		}

		inputData := []byte{0x01, 0x02, 0x03, 0x04}
		resampledData := []byte{0x10, 0x20}

		mockAudio.EXPECT().GetDataSamples(8000, inputData).Return(resampledData, nil)
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, resampledData).Return(nil)

		if err := h.runnerWebsocketHandleAudio(se, 8000, 1, inputData); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("24kHz audio is resampled (pipecat default rate safety net)", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAudio := NewMockAudiosocketHandler(mc)
		mockWS := NewMockWebsocketHandler(mc)
		h := &pipecatcallHandler{
			audiosocketHandler: mockAudio,
			websocketHandler:   mockWS,
		}

		conn := &websocket.Conn{}
		connAstReady := make(chan struct{})
		close(connAstReady)
		se := &pipecatcall.Session{
			Ctx:          context.Background(),
			ConnAst:      conn,
			ConnAstReady: connAstReady,
		}

		// 24kHz is Pipecat's default audio_out_sample_rate. If PipelineParams
		// doesn't set audio_out_sample_rate=16000, TTS outputs 24kHz and this
		// resampling path runs per chunk — creating boundary artifacts.
		inputData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
		resampledData := []byte{0x10, 0x20, 0x30, 0x40}

		mockAudio.EXPECT().GetDataSamples(24000, inputData).Return(resampledData, nil)
		mockWS.EXPECT().WriteMessage(conn, websocket.BinaryMessage, resampledData).Return(nil)

		if err := h.runnerWebsocketHandleAudio(se, 24000, 1, inputData); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("stereo audio is rejected", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatcallHandler{}
		se := &pipecatcall.Session{
			Ctx: context.Background(),
		}

		err := h.runnerWebsocketHandleAudio(se, 16000, 2, []byte{0x01, 0x02})
		if err == nil {
			t.Errorf("expected error for stereo audio, got nil")
		}
	})

	t.Run("nil ConnAst returns nil without writing", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatcallHandler{}
		connAstReady := make(chan struct{})
		close(connAstReady) // simulate ready but nil conn (e.g., non-call reference type)
		se := &pipecatcall.Session{
			Ctx:          context.Background(),
			ConnAst:      nil,
			ConnAstReady: connAstReady,
		}

		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, []byte{0x01, 0x02}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty data returns nil", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h := &pipecatcallHandler{}
		conn := &websocket.Conn{}
		se := &pipecatcall.Session{
			Ctx:     context.Background(),
			ConnAst: conn,
		}

		// Empty data returns nil without calling WriteMessage
		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, []byte{}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("audio writes to jitter buffer when present", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		// No mock WS needed — audio should go to the jitter buffer, not the websocket.
		h := &pipecatcallHandler{}

		jb := pipecatcall.NewAudioJitterBuffer()
		se := &pipecatcall.Session{
			Ctx:          context.Background(),
			JitterBuffer: jb,
		}

		data := []byte{0x01, 0x02, 0x03, 0x04}
		if err := h.runnerWebsocketHandleAudio(se, 16000, 1, data); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if jb.Len() != len(data) {
			t.Errorf("expected jitter buffer length %d, got %d", len(data), jb.Len())
		}
	})

	t.Run("resampled audio writes to jitter buffer when present", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockAudio := NewMockAudiosocketHandler(mc)
		h := &pipecatcallHandler{
			audiosocketHandler: mockAudio,
		}

		jb := pipecatcall.NewAudioJitterBuffer()
		se := &pipecatcall.Session{
			Ctx:          context.Background(),
			JitterBuffer: jb,
		}

		inputData := []byte{0x01, 0x02, 0x03, 0x04}
		resampledData := []byte{0x10, 0x20}

		mockAudio.EXPECT().GetDataSamples(8000, inputData).Return(resampledData, nil)

		if err := h.runnerWebsocketHandleAudio(se, 8000, 1, inputData); err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if jb.Len() != len(resampledData) {
			t.Errorf("expected jitter buffer length %d, got %d", len(resampledData), jb.Len())
		}
	})
}

