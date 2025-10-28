package pipecatcallhandler

import (
	"context"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_receiveMessageFrameTypeMessage(t *testing.T) {

	tests := []struct {
		name string

		pc *pipecatcall.Pipecatcall
		m  []byte

		responseUUID  uuid.UUID
		expectEvent   string
		expectMessage message.Message
	}{
		{
			name: "bot-transcription",

			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54416ae0-af23-11f0-8991-07dd3ffd4def"),
					CustomerID: uuid.FromStringOrNil("546f7606-af23-11f0-a7ca-c32fd2659ee7"),
				},
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
			pc: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("54986764-af23-11f0-9793-db91dfe17f29"),
					CustomerID: uuid.FromStringOrNil("54c1efee-af23-11f0-af7c-a7f393ea7de5"),
				},
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
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockNotify.EXPECT().PublishEvent(ctx, tt.expectEvent, tt.expectMessage)

			if err := h.receiveMessageFrameTypeMessage(ctx, tt.pc, tt.m); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_runnerWebsocketHandleAudio(t *testing.T) {
	tests := []struct {
		name string

		pc          *pipecatcall.Pipecatcall
		sampleRate  int
		numChannels int
		data        []byte

		responseDataSamples []byte
		expectWriteData     []byte
	}{
		{
			name: "normal mono audio",

			pc: &pipecatcall.Pipecatcall{
				AsteriskConn: NewDummyConn(),
			},
			sampleRate:  16000,
			numChannels: 1,
			data:        []byte{0x01, 0x02, 0x03, 0x04},

			responseDataSamples: []byte{0x10, 0x20, 0x30, 0x40},
			expectWriteData:     []byte{0x10, 0x20, 0x30, 0x40},
		},
		{
			name: "empty data",

			pc: &pipecatcall.Pipecatcall{
				AsteriskConn: NewDummyConn(),
			},
			sampleRate:  8000,
			numChannels: 1,
			data:        []byte{},

			responseDataSamples: []byte{},
			expectWriteData:     []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAudio := NewMockAudiosocketHandler(mc)
			h := &pipecatcallHandler{
				audiosocketHandler: mockAudio,
			}
			ctx := context.Background()

			mockAudio.EXPECT().GetDataSamples(tt.sampleRate, tt.data).Return(tt.responseDataSamples, nil)
			mockAudio.EXPECT().Write(ctx, tt.pc.AsteriskConn, tt.expectWriteData).Return(nil)

			if err := h.runnerWebsocketHandleAudio(ctx, tt.pc, tt.sampleRate, tt.numChannels, tt.data); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

		})
	}
}

func Test_runnerGetURL(t *testing.T) {
	tests := []struct {
		name       string
		runnerPort int
		expectURL  string
	}{
		{
			name:       "normal port",
			runnerPort: 8080,
			expectURL:  "ws://localhost:8080/ws",
		},
		{
			name:       "different port",
			runnerPort: 12345,
			expectURL:  "ws://localhost:12345/ws",
		},
		{
			name:       "zero port",
			runnerPort: 0,
			expectURL:  "ws://localhost:0/ws",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &pipecatcallHandler{}
			pc := &pipecatcall.Pipecatcall{
				RunnerPort: tt.runnerPort,
			}

			got := h.runnerGetURL(pc)
			if got != tt.expectURL {
				t.Errorf("unexpected URL: expect %q, got %q", tt.expectURL, got)
			}
		})
	}
}
