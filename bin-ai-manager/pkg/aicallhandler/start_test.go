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
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
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

		responseConfbridge        *cmconfbridge.Confbridge
		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcallID      uuid.UUID
		responseAIcall            *aicall.AIcall
		responseMessages          []*message.Message
		responsePipecatcall       *pmpipecatcall.Pipecatcall

		expectAIcall              *aicall.AIcall
		expectVariables           map[string]string
		expectPipecatcallMessages []map[string]any
		expectMessage             *message.Message
		expectSTTType             pmpipecatcall.STTType
		expectTTSType             pmpipecatcall.TTSType
		expectTTSVoiceID          string
		expectRes                 *aicall.AIcall
	}{
		{
			name: "normal",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType:  ai.EngineTypeNone,
				EngineModel: "openai.o1",
				InitPrompt:  "hello, this is init prompt message.",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				STTType:     ai.STTTypeDeepgram,
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
			responseUUIDPipecatcallID: uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:          uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel: "openai.o1",
				ActiveflowID:  uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				AIEngineType:  ai.EngineTypeNone,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID: uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
			},
			responseMessages: []*message.Message{
				{
					Role:    "assistant",
					Content: "test assistant message.",
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				},
			},

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:          uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel: "openai.o1",
				ActiveflowID:  uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				AIEngineType:  ai.EngineTypeNone,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID: uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
			},
			expectVariables: map[string]string{
				variableID:            "a6cd01d0-d785-467f-9069-684e46cc2644",
				variableAIID:          "a4107e6e-f06d-11ef-9b7a-03c848b3bb41",
				variableAIEngineModel: "openai.o1",
				variableConfbridgeID:  "ec6d153d-dd5a-4eef-bc27-8fcebe100704",
				variableGender:        string(aicall.GenderFemale),
				variableLanguage:      "en-US",
				variablePipecatcallID: "a4e5c7ae-b539-11f0-ac68-c38f244d145b",
			},
			expectMessage: &message.Message{
				Role:    message.RoleSystem,
				Content: "hello, this is init prompt message.",
			},
			expectPipecatcallMessages: []map[string]any{
				{"role": "system", "content": defaultCommonSystemPrompt},
				{"role": "system", "content": "hello, this is init prompt message."},
				{"role": "assistant", "content": "test assistant message."},
			},
			expectTTSType:    pmpipecatcall.TTSTypeElevenLabs,
			expectTTSVoiceID: "21m00Tcm4TlvDq8ikWAM",
			expectSTTType:    pmpipecatcall.STTTypeDeepgram,
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:          uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel: "openai.o1",
				ActiveflowID:  uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				AIEngineType:  ai.EngineTypeNone,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID: uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectVariables).Return(nil)
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.responseAIcall.ActiveflowID, tt.ai.InitPrompt).Return(tt.ai.InitPrompt, nil)
			mockMessage.EXPECT().Gets(ctx, tt.ai.ID, uint64(100), "", map[string]string{}).Return(tt.responseMessages, nil)
			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.responseAIcall.PipecatcallID,
				tt.responseAIcall.CustomerID,
				tt.responseAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.responseAIcall.ID,
				pmpipecatcall.LLMType(tt.ai.EngineModel),
				tt.expectPipecatcallMessages,
				tt.expectSTTType,
				tt.expectTTSType,
				tt.expectTTSVoiceID,
			).Return(tt.responsePipecatcall, nil)

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

		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcallID      uuid.UUID
		responseAIcall            *aicall.AIcall
		responseMessage           *message.Message

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

			responseUUIDPipecatcallID: uuid.FromStringOrNil("78a31220-b465-11f0-a3f2-b77bb59ccdcd"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
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
				AIID:          uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
				AIEngineType:  ai.EngineTypeNone,
				PipecatcallID: uuid.FromStringOrNil("78a31220-b465-11f0-a3f2-b77bb59ccdcd"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
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
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			mockMessage.EXPECT().Send(ctx, tt.responseAIcall.ID, message.RoleSystem, tt.ai.InitPrompt, true).Return(tt.responseMessage, nil)

			mockDB.EXPECT().AIcallUpdateStatus(ctx, tt.responseAIcall.ID, aicall.StatusProgressing).Return(nil)
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

		responseUUIDAIcallID      uuid.UUID
		responseUUIDPipecatcallID uuid.UUID
		responseAIcall            *aicall.AIcall
		responseVarible           *fmvariable.Variable

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

			responseUUIDAIcallID:      uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
			responseUUIDPipecatcallID: uuid.FromStringOrNil("19f84290-b465-11f0-81ef-07fea8c8aa82"),
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
				PipecatcallID: uuid.FromStringOrNil("19f84290-b465-11f0-81ef-07fea8c8aa82"),
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
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.responseAIcall, nil)
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
