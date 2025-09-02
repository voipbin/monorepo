package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_startReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		ai           *ai.AI
		activeflowID uuid.UUID
		referenceID  uuid.UUID
		gender       aicall.Gender
		language     string

		responseConfbridge   *cmconfbridge.Confbridge
		responseTTSStreaming *tmstreaming.Streaming
		responseUUIDAIcall   uuid.UUID
		responseAIcall       *aicall.AIcall
		responseMessage      *message.Message

		expectAIcall         *aicall.AIcall
		expectAIcallMessages []message.Message
		expectMessage        *message.Message
		expectRes            *aicall.AIcall
	}{
		{
			name: "normal",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType: ai.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			activeflowID: uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
			referenceID:  uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
			gender:       aicall.GenderFemale,
			language:     "en-US",

			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				},
			},
			responseTTSStreaming: &tmstreaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f3e1ca38-817c-11f0-b195-eb59dc9cc7d9"),
				},
				PodID: "f42f870a-817c-11f0-baa2-6fadbb11d7a1",
			},
			responseUUIDAIcall: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},
			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:              uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				ActiveflowID:      uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				AIEngineType:      ai.EngineTypeNone,
				ReferenceType:     aicall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:      uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				Gender:            aicall.GenderFemale,
				Language:          "en-US",
				Status:            aicall.StatusInitiating,
				TTSStreamingID:    uuid.FromStringOrNil("f3e1ca38-817c-11f0-b195-eb59dc9cc7d9"),
				TTSStreamingPodID: "f42f870a-817c-11f0-baa2-6fadbb11d7a1",
			},
			expectMessage: &message.Message{
				Role:    message.RoleSystem,
				Content: "hello, this is init prompt message.",
			},
			expectAIcallMessages: []message.Message{
				{Role: "system", Content: "hello, this is init prompt message."},
				{Role: "assistant", Content: "test assistant message."},
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, tt.activeflowID, cmconfbridge.ReferenceTypeAI, tt.ai.ID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().TTSV1StreamingCreate(ctx, tt.ai.CustomerID, tt.activeflowID, tmstreaming.ReferenceTypeCall, tt.referenceID, tt.language, tmstreaming.Gender(tt.gender), tmstreaming.DirectionOutgoing).Return(tt.responseTTSStreaming, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcall)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcall).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			mockMessage.EXPECT().StreamingSend(ctx, tt.responseAIcall.ID, message.RoleSystem, tt.ai.InitPrompt, true).Return(tt.responseMessage, nil)

			res, err := h.startReferenceTypeCall(ctx, tt.ai, tt.activeflowID, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startReferenceTypeNone(t *testing.T) {
	tests := []struct {
		name string

		ai       *ai.AI
		gender   aicall.Gender
		language string

		responseUUIDAIcall uuid.UUID
		responseAIcall     *aicall.AIcall
		responseMessage    *message.Message

		expectAIcall *aicall.AIcall
		expectRes    *aicall.AIcall
	}{
		{
			name: "normal",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineType: ai.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			gender:   aicall.GenderFemale,
			language: "en-US",

			responseUUIDAIcall: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
				},
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},
			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				AIID:         uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
				AIEngineType: ai.EngineTypeNone,
				Gender:       aicall.GenderFemale,
				Language:     "en-US",
				Status:       aicall.StatusInitiating,
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcall)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcall).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			mockMessage.EXPECT().Send(ctx, tt.responseAIcall.ID, message.RoleSystem, tt.ai.InitPrompt, true).Return(tt.responseMessage, nil)

			mockDB.EXPECT().AIcallUpdateStatusProgressing(ctx, tt.responseAIcall.ID, uuid.Nil).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseAIcall.ID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusProgressing, tt.responseAIcall)

			res, err := h.startReferenceTypeNone(ctx, tt.ai, tt.gender, tt.language)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startReferenceTypeConversation(t *testing.T) {
	tests := []struct {
		name string

		ai           *ai.AI
		activeflowID uuid.UUID
		referenceID  uuid.UUID
		gender       aicall.Gender
		language     string

		responseUUIDAIcall uuid.UUID
		responseAIcall     *aicall.AIcall
		responseVarible    *fmvariable.Variable

		responseMessage *message.Message

		expectAIcall         *aicall.AIcall
		expectMessageContent string
		expectRes            *aicall.AIcall
	}{
		{
			name: "normal",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineType: ai.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			activeflowID: uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:  uuid.FromStringOrNil("d184c87c-30dd-11f0-8bbf-d773a2d31d73"),
			gender:       aicall.GenderFemale,
			language:     "en-US",

			responseUUIDAIcall: uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
				},
			},
			responseVarible: &fmvariable.Variable{
				Variables: map[string]string{
					"voipbin.conversation_message.text": "test user message.",
				},
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				AIID:          uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
				AIEngineType:  ai.EngineTypeNone,
				ActiveflowID:  uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   uuid.FromStringOrNil("d184c87c-30dd-11f0-8bbf-d773a2d31d73"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
			},
			expectMessageContent: "test user message.",
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGetByReferenceID(ctx, tt.referenceID).Return(nil, dbhandler.ErrNotFound)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcall)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcall).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			// get conversation message
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVarible, nil)

			mockMessage.EXPECT().Send(ctx, tt.responseAIcall.ID, message.RoleUser, tt.expectMessageContent, false).Return(tt.responseMessage, nil)

			res, err := h.startReferenceTypeConversation(ctx, tt.ai, tt.activeflowID, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
