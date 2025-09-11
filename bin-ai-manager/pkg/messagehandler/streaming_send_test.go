package messagehandler

// package handler

import (
	"context"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	tmStreaming "monorepo/bin-tts-manager/models/streaming"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_StreamingSend(t *testing.T) {
	tests := []struct {
		name string

		aicallID       uuid.UUID
		role           message.Role
		content        string
		returnResponse bool

		responseAIcall        *aicall.AIcall
		responseUUID1         uuid.UUID // For outgoing message
		responseUUID2         uuid.UUID // For TTSStreamingSayInit msgID
		responseMessages      []*message.Message
		responseChanMessage   <-chan string
		responseChanAction    <-chan *fmaction.Action
		responseStreaming     *tmStreaming.Streaming
		responseStreamingMsgs []string

		// Expected created messages
		expectOutgoingMessage *message.Message
		expectIncomingMessage *message.Message

		expectMessages []*message.Message

		expectReturnMessage *message.Message
		expectError         error
	}{
		{
			name: "normal openai streaming send",

			aicallID:       uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
			role:           message.RoleUser,
			content:        "hello world!",
			returnResponse: false,

			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				ReferenceType:     aicall.ReferenceTypeCall,
				Status:            aicall.StatusProgressing,
				AIEngineModel:     ai.EngineModelOpenaiGPT3Dot5Turbo,
				TTSStreamingID:    uuid.FromStringOrNil("e22f1d9c-87a6-11f0-94ca-b32bb1be78da"),
				TTSStreamingPodID: "tts-pod-id-456",
			},
			responseUUID1: uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"), // Outgoing message ID
			responseUUID2: uuid.FromStringOrNil("7786dba8-f2bc-11ef-b9de-4b764cfeef4d"), // TTS msgID
			responseMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"),
					},
				},
			},
			responseChanMessage: func() <-chan string {
				ch := make(chan string, 3)
				ch <- "Hi"
				ch <- " there"
				ch <- "!"
				close(ch)
				return ch
			}(),
			responseChanAction: func() <-chan *fmaction.Action {
				ch := make(chan *fmaction.Action, 3)
				close(ch)
				return ch
			}(),
			responseStreaming: &tmStreaming.Streaming{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("7786dba8-f2bc-11ef-b9de-4b764cfeef4d"),
				},
			},
			responseStreamingMsgs: []string{"Hi", " there", "!"},

			expectOutgoingMessage: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				AIcallID:  uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
				Direction: message.DirectionOutgoing,
				Role:      message.RoleUser,
				Content:   "hello world!",
			},
			expectIncomingMessage: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7786dba8-f2bc-11ef-b9de-4b764cfeef4d"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				AIcallID:  uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
				Direction: message.DirectionIncoming,
				Role:      message.RoleAssistant,
				Content:   "Hi there!",
			},

			expectMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"),
					},
				},
			},

			expectReturnMessage: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				AIcallID:  uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
				Direction: message.DirectionOutgoing,
				Role:      message.RoleUser,
				Content:   "hello world!",
			},
			expectError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockGPT := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,

				engineOpenaiHandler: mockGPT,
				reqHandler:          mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1AIcallGet(ctx, tt.aicallID).Return(tt.responseAIcall, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID1) // Outgoing message ID
			mockDB.EXPECT().MessageCreate(ctx, tt.expectOutgoingMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectOutgoingMessage.ID).Return(tt.expectOutgoingMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectOutgoingMessage.CustomerID, message.EventTypeMessageCreated, tt.expectOutgoingMessage)

			// streamingSendOpenai
			mockDB.EXPECT().MessageGets(ctx, tt.responseAIcall.ID, uint64(1000), "", gomock.Any()).Return(tt.responseMessages, nil)
			mockGPT.EXPECT().StreamingSend(ctx, tt.responseAIcall, tt.expectMessages).Return(tt.responseChanMessage, tt.responseChanAction, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID2) // TTS msgID
			mockReq.EXPECT().TTSV1StreamingSayInit(ctx, tt.responseAIcall.TTSStreamingPodID, tt.responseAIcall.TTSStreamingID, tt.responseUUID2).Return(tt.responseStreaming, nil)

			for _, msg := range tt.responseStreamingMsgs {
				mockReq.EXPECT().TTSV1StreamingSayAdd(ctx, tt.responseAIcall.TTSStreamingPodID, tt.responseAIcall.TTSStreamingID, tt.responseUUID2, msg).Return(nil)
			}

			mockDB.EXPECT().MessageCreate(ctx, tt.expectIncomingMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.expectIncomingMessage.ID).Return(tt.expectIncomingMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectIncomingMessage.CustomerID, message.EventTypeMessageCreated, tt.expectIncomingMessage)

			res, err := h.StreamingSend(ctx, tt.aicallID, tt.role, tt.content, tt.returnResponse)
			if err != nil {
				t.Errorf("Wrong match. expected ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectReturnMessage) {
				t.Errorf("Wrong return message match.\nexpect: %v\ngot: %v", tt.expectReturnMessage, res)
			}
		})
	}
}
