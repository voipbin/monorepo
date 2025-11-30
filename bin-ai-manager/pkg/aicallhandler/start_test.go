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
		expectLLMType             pmpipecatcall.LLMType
		expectLLMMessages         *message.Message
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
				AIEngineModel: ai.EngineModel("openai.o1"),
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				AISTTType:     ai.STTTypeDeepgram,
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
			expectLLMType: pmpipecatcall.LLMType("openai.o1"),
			expectLLMMessages: &message.Message{
				Role:    message.RoleSystem,
				Content: "hello, this is init prompt message.",
			},
			expectPipecatcallMessages: []map[string]any{
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
				AIEngineModel: ai.EngineModel("openai.o1"),
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				AISTTType:     ai.STTTypeDeepgram,
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

			// startAIcall
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, gomock.Any()).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
			mockMessage.EXPECT().Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&message.Message{}, nil)

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectVariables).Return(nil)
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.responseAIcall.ActiveflowID, tt.ai.InitPrompt).Return(tt.ai.InitPrompt, nil)

			// startPipecatcall
			mockMessage.EXPECT().Gets(ctx, tt.responseAIcall.ID, uint64(100), "", map[string]string{}).Return(tt.responseMessages, nil)
			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.responseAIcall.PipecatcallID,
				tt.responseAIcall.CustomerID,
				tt.responseAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.responseAIcall.ID,
				tt.expectLLMType,
				tt.expectPipecatcallMessages,
				tt.expectSTTType,
				tt.responseAIcall.Language,
				tt.expectTTSType,
				tt.responseAIcall.Language,
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

			// startAIcall
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, gomock.Any()).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
			mockMessage.EXPECT().Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&message.Message{}, nil)

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

		responseVarible           *fmvariable.Variable
		responseAIcall            *aicall.AIcall
		responseUUIDPipecatcallID uuid.UUID
		responseMessages          []*message.Message
		responsePipecatcall       *pmpipecatcall.Pipecatcall

		expectAIcall         *aicall.AIcall
		expectMessageContent string
		expectLLMType        pmpipecatcall.LLMType
		expectLLMMessages    []map[string]any
		expectSTTType        pmpipecatcall.STTType
		expectTTSType        pmpipecatcall.TTSType
		expectTTSVoiceID     string
		expectRes            *aicall.AIcall
	}{
		{
			name: "normal",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineType: ai.EngineType("openai.gpt-3.5-turbo"),
				InitPrompt: "hello, this is init prompt message.",
			},
			activeflowID: uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:  uuid.FromStringOrNil("d184c87c-30dd-11f0-8bbf-d773a2d31d73"),
			gender:       aicall.GenderFemale,
			language:     "en-US",

			responseVarible: &fmvariable.Variable{
				Variables: map[string]string{
					"voipbin.conversation_message.text": "test user message.",
				},
			},
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
				},
				AIEngineModel: "openai.gpt-3.5-turbo",
			},
			responseUUIDPipecatcallID: uuid.FromStringOrNil("017c1c12-b737-11f0-80ad-032b0dde6a93"),
			responseMessages: []*message.Message{
				{
					Role:    "system",
					Content: "test assistant message.",
				},
				{
					Role:    "system",
					Content: "default common system prompt.",
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
				},
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
			expectLLMType:        pmpipecatcall.LLMType("openai.gpt-3.5-turbo"),
			expectLLMMessages: []map[string]any{
				{
					"role":    "system",
					"content": "default common system prompt.",
				},
				{
					"role":    "system",
					"content": "test assistant message.",
				},
			},
			expectSTTType:    pmpipecatcall.STTTypeNone,
			expectTTSType:    pmpipecatcall.TTSTypeNone,
			expectTTSVoiceID: "",

			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
				},
				AIEngineModel: "openai.gpt-3.5-turbo",
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

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVarible, nil)

			// GetByReferenceID
			mockDB.EXPECT().AIcallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseAIcall, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockDB.EXPECT().AIcallUpdatePipecatcallID(ctx, tt.responseAIcall.ID, tt.responseUUIDPipecatcallID).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseAIcall.ID).Return(tt.responseAIcall, nil)

			// get conversation message
			mockMessage.EXPECT().Create(ctx, tt.responseAIcall.CustomerID, tt.responseAIcall.ID, message.DirectionOutgoing, message.RoleUser, tt.expectMessageContent, nil, "").Return(&message.Message{}, nil)

			// startPipecatcall
			mockMessage.EXPECT().Gets(ctx, tt.responseAIcall.ID, uint64(100), "", map[string]string{}).Return(tt.responseMessages, nil)
			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.responseAIcall.PipecatcallID,
				tt.responseAIcall.CustomerID,
				tt.responseAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.responseAIcall.ID,
				tt.expectLLMType,
				tt.expectLLMMessages,
				tt.expectSTTType,
				tt.responseAIcall.Language,
				tt.expectTTSType,
				tt.responseAIcall.Language,
				tt.expectTTSVoiceID,
			).Return(tt.responsePipecatcall, nil)

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

func Test_getPipecatcallMessages(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall

		responseMessages []*message.Message
		expectRes        []map[string]any
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6754cd08-b62a-11f0-b4a3-13e4f1d01c60"),
				},
			},

			responseMessages: []*message.Message{
				{
					Role:    message.RoleSystem,
					Content: "default common system prompt.",
				}, {
					Role:    message.RoleSystem,
					Content: "default initial system prompt.",
				},
			},
			expectRes: []map[string]any{
				{
					"role":    "system",
					"content": "default initial system prompt.",
				},
				{
					"role":    "system",
					"content": "default common system prompt.",
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

			mockMessage.EXPECT().Gets(ctx, tt.aicall.ID, uint64(100), "", map[string]string{}).Return(tt.responseMessages, nil)

			res, err := h.getPipecatcallMessages(ctx, tt.aicall)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getPipecatcallSTTType(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall

		expectRes pmpipecatcall.STTType
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				AISTTType: ai.STTTypeDeepgram,
			},

			expectRes: pmpipecatcall.STTTypeDeepgram,
		},
		{
			name: "stt type is none",

			aicall: &aicall.AIcall{
				AISTTType: ai.STTTypeNone,
			},

			expectRes: defaultPipecatcallSTTType,
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

			res := h.getPipecatcallSTTType(tt.aicall)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getPipecatcallTTSInfo(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall

		expectResTTSType    pmpipecatcall.TTSType
		expectResTTSVoiceID string
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
				AITTSType:    ai.TTSTypeElevenLabs,
				AITTSVoiceID: "c27e27ec-b657-11f0-805c-3b02525581bd",
			},

			expectResTTSType:    pmpipecatcall.TTSTypeElevenLabs,
			expectResTTSVoiceID: "c27e27ec-b657-11f0-805c-3b02525581bd",
		},
		{
			name: "stt type is none and voice id is empty",

			aicall: &aicall.AIcall{
				AITTSType:    ai.TTSTypeNone,
				AITTSVoiceID: "",
			},

			expectResTTSType:    defaultPipecatcallTTSType,
			expectResTTSVoiceID: defaultPipecatcallTTSVoiceID,
		},
		{
			name: "stt type is none",

			aicall: &aicall.AIcall{
				AITTSType:    ai.TTSTypeNone,
				AITTSVoiceID: "c2c1c7a4-b657-11f0-aed1-f74832c7c9dc",
			},

			expectResTTSType:    defaultPipecatcallTTSType,
			expectResTTSVoiceID: "c2c1c7a4-b657-11f0-aed1-f74832c7c9dc",
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

			resTTSType, resTTSVoiceID := h.getPipecatcallTTSInfo(tt.aicall)

			if !reflect.DeepEqual(resTTSType, tt.expectResTTSType) {
				t.Errorf("expected: %v, got: %v", tt.expectResTTSType, resTTSType)
			}
			if !reflect.DeepEqual(resTTSVoiceID, tt.expectResTTSVoiceID) {
				t.Errorf("expected: %v, got: %v", tt.expectResTTSVoiceID, resTTSVoiceID)
			}
		})
	}
}

// func Test_getTTSType(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		ttsType ai.TTSType

// 		expectRes ai.TTSType
// 	}{
// 		{
// 			name: "normal",

// 			ttsType: ai.TTSTypeCartesia,

// 			expectRes: ai.TTSTypeCartesia,
// 		},
// 		{
// 			name: "tts type is none",

// 			ttsType: ai.TTSTypeNone,

// 			expectRes: defaultTTSType,
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}

// 			res := h.getTTSType(tt.ttsType)

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// func Test_getPipecatcallTTSType(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		aicall  *aicall.AIcall
// 		ttsType ai.TTSType

// 		expectRes pmpipecatcall.TTSType
// 	}{
// 		{
// 			name: "normal",

// 			aicall: &aicall.AIcall{
// 				ReferenceType: aicall.ReferenceTypeCall,
// 			},
// 			ttsType: ai.TTSTypeElevenLabs,

// 			expectRes: pmpipecatcall.TTSTypeElevenLabs,
// 		},
// 		{
// 			name: "reference type is not call",

// 			aicall: &aicall.AIcall{
// 				ReferenceType: aicall.ReferenceTypeConversation,
// 			},
// 			ttsType: ai.TTSTypeElevenLabs,

// 			expectRes: pmpipecatcall.TTSTypeNone,
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}

// 			res := h.getPipecatcallTTSType(tt.aicall, tt.ttsType)

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

// func Test_getPipecatcallVoiceID(t *testing.T) {
// 	tests := []struct {
// 		name string

// 		ttsType ai.TTSType
// 		voiceID string

// 		expectRes string
// 	}{
// 		{
// 			name: "normal",

// 			ttsType: ai.TTSTypeElevenLabs,
// 			voiceID: "a18a75ec-b62d-11f0-9102-d3923076d044",

// 			expectRes: "a18a75ec-b62d-11f0-9102-d3923076d044",
// 		},
// 		{
// 			name: "voice id is empty",

// 			ttsType: ai.TTSTypeElevenLabs,
// 			voiceID: "",

// 			expectRes: mapDefaultTTSVoiceIDByTTSType[ai.TTSTypeElevenLabs],
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt
// 		t.Run(tt.name, func(t *testing.T) {
// 			t.Parallel()
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}

// 			res, err := h.getPipecatcallVoiceID(tt.ttsType, tt.voiceID)
// 			if err != nil {
// 				t.Errorf("unexpected error: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }

func Test_startPipecatcall(t *testing.T) {
	tests := []struct {
		name string

		aicall *aicall.AIcall

		responseMessages    []*message.Message
		responsePipecatcall *pmpipecatcall.Pipecatcall

		expectLLMType     pmpipecatcall.LLMType
		expectLLMMessages []map[string]any
		expectSTTType     pmpipecatcall.STTType
		expectTTSType     pmpipecatcall.TTSType
		expectTTSVoiceID  string
		expectRes         *pmpipecatcall.Pipecatcall
	}{
		{
			name: "reference type is call",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6754cd08-b62a-11f0-b4a3-13e4f1d01c60"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				PipecatcallID: uuid.FromStringOrNil("fa818566-b62b-11f0-b7d9-bf54e3ce1991"),
				AIEngineModel: "openai.gpt-3.5-turbo",
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "fa5c67b8-b62b-11f0-aab5-630972321af9",
				AISTTType:     ai.STTTypeDeepgram,
				Language:      "en-US",
			},

			responseMessages: []*message.Message{
				{
					Role:    message.RoleSystem,
					Content: "default common system prompt.",
				}, {
					Role:    message.RoleSystem,
					Content: "default initial system prompt.",
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a14f5408-b62d-11f0-938b-bfe88125c47f"),
				},
			},

			expectLLMType: pmpipecatcall.LLMType("openai.gpt-3.5-turbo"),
			expectLLMMessages: []map[string]any{
				{
					"role":    "system",
					"content": "default initial system prompt.",
				},
				{
					"role":    "system",
					"content": "default common system prompt.",
				},
			},
			expectSTTType:    pmpipecatcall.STTTypeDeepgram,
			expectTTSType:    pmpipecatcall.TTSTypeElevenLabs,
			expectTTSVoiceID: "fa5c67b8-b62b-11f0-aab5-630972321af9",
			expectRes: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a14f5408-b62d-11f0-938b-bfe88125c47f"),
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

			mockMessage.EXPECT().Gets(ctx, tt.aicall.ID, uint64(100), "", map[string]string{}).Return(tt.responseMessages, nil)
			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.aicall.PipecatcallID,
				tt.aicall.CustomerID,
				tt.aicall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.aicall.ID,
				tt.expectLLMType,
				tt.expectLLMMessages,
				tt.expectSTTType,
				tt.aicall.Language,
				tt.expectTTSType,
				tt.aicall.Language,
				tt.expectTTSVoiceID,
			).Return(tt.responsePipecatcall, nil)

			res, err := h.startPipecatcall(ctx, tt.aicall)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startAIcall(t *testing.T) {
	tests := []struct {
		name string

		ai            *ai.AI
		activeflowID  uuid.UUID
		referenceType aicall.ReferenceType
		referenceID   uuid.UUID
		confbridgeID  uuid.UUID
		gender        aicall.Gender
		language      string

		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcallID      uuid.UUID

		expectAIcall       *aicall.AIcall
		expectVariables    map[string]string
		expectMessageTexts []string

		expectRes *aicall.AIcall
	}{
		{
			name: "normal",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				InitPrompt: "You are a helpful assistant.",
			},
			activeflowID:  uuid.FromStringOrNil("b34140c8-b659-11f0-be3a-5fc8a6759b80"),
			referenceType: aicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b3662e38-b659-11f0-820a-833195d45f7e"),
			confbridgeID:  uuid.FromStringOrNil("b3864e5c-b659-11f0-ab17-6b281e446482"),
			gender:        aicall.GenderMale,
			language:      "en-US",

			responseUUIDPipecatcallID: uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AIID: uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				ActiveflowID:  uuid.FromStringOrNil("b34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b3662e38-b659-11f0-820a-833195d45f7e"),
				ConfbridgeID:  uuid.FromStringOrNil("b3864e5c-b659-11f0-ab17-6b281e446482"),
				PipecatcallID: uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,
				Gender:   aicall.GenderMale,
				Language: "en-US",
			},
			expectVariables: map[string]string{
				"voipbin.aicall.ai_engine_model": "",
				"voipbin.aicall.ai_id":           "b30ecf94-b659-11f0-b8ef-13f90dff9ee8",
				"voipbin.aicall.confbridge_id":   "b3864e5c-b659-11f0-ab17-6b281e446482",
				"voipbin.aicall.gender":          "male",
				"voipbin.aicall.id":              "b3af613e-b659-11f0-9a72-e3e004fae386",
				"voipbin.aicall.language":        "en-US",
				"voipbin.aicall.pipecatcall_id":  "b3af613e-b659-11f0-9a72-e3e004fae386",
			},
			expectMessageTexts: []string{
				defaultCommonSystemPrompt,
				"You are a helpful assistant.",
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AIID: uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				ActiveflowID:  uuid.FromStringOrNil("b34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b3662e38-b659-11f0-820a-833195d45f7e"),
				ConfbridgeID:  uuid.FromStringOrNil("b3864e5c-b659-11f0-ab17-6b281e446482"),
				PipecatcallID: uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,
				Gender:   aicall.GenderMale,
				Language: "en-US",
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.expectAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.expectAIcall)

			// setActiveflowVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.expectAIcall.ActiveflowID, tt.expectVariables).Return(nil)

			// startInitMessages
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.ai.InitPrompt).Return(tt.ai.InitPrompt, nil)
			for _, m := range tt.expectMessageTexts {
				mockMessage.EXPECT().Create(ctx, tt.expectAIcall.CustomerID, tt.expectAIcall.ID, message.DirectionOutgoing, message.RoleSystem, m, nil, "").Return(&message.Message{}, nil)
			}

			res, err := h.startAIcall(ctx, tt.ai, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getInitPrompt(t *testing.T) {
	tests := []struct {
		name string

		ai           *ai.AI
		activeflowID uuid.UUID

		responseSubstitute string
		expectRes          string
	}{
		{
			name: "normal",

			ai: &ai.AI{
				InitPrompt: "You are a helpful assistant.",
			},
			activeflowID: uuid.FromStringOrNil("f9e382ac-b659-11f0-97d2-7b9de23cd371"),

			responseSubstitute: "You are a super helpful assistant.",
			expectRes:          "You are a super helpful assistant.",
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

			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.ai.InitPrompt).Return(tt.responseSubstitute, nil)

			res := h.getInitPrompt(ctx, tt.ai, tt.activeflowID)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startInitMessages(t *testing.T) {
	tests := []struct {
		name string

		ai     *ai.AI
		aicall *aicall.AIcall

		responseSubstitutes []string

		expectMessageTexts []string
	}{
		{
			name: "has all",

			ai: &ai.AI{
				InitPrompt: "You are a helpful assistant.",
				EngineData: map[string]any{
					"initial_system_prompt": "Bruce Wayne is Batman.",
				},
			},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("79e1da30-c055-11f0-9ca3-6ff0383cb80c"),
				},
				ActiveflowID: uuid.FromStringOrNil("7a0651d0-c055-11f0-a70e-8fe16492a013"),
			},

			responseSubstitutes: []string{
				"You are a super helpful assistant.",
				"Bruce Wayne is Batman.",
			},
			expectMessageTexts: []string{
				defaultCommonSystemPrompt,
				"You are a super helpful assistant.",
				`{"initial_system_prompt":"Bruce Wayne is Batman."}`,
			},
		},
		{
			name: "has nothing",

			ai: &ai.AI{},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("60afea76-c054-11f0-a76a-272fe6ab293c"),
				},
				ActiveflowID: uuid.FromStringOrNil("60e6ed6e-c054-11f0-9d64-573dab8aa82d"),
			},

			expectMessageTexts: []string{
				defaultCommonSystemPrompt,
			},
		},
		{
			name: "has initial prompt",

			ai: &ai.AI{
				InitPrompt: "You are a helpful assistant.",
			},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fa0c92d2-b659-11f0-8e59-836f218753a9"),
				},
				ActiveflowID: uuid.FromStringOrNil("fa31d6a0-b659-11f0-8ec0-4f223b1fe9db"),
			},

			responseSubstitutes: []string{"You are a super helpful assistant."},
			expectMessageTexts: []string{
				defaultCommonSystemPrompt,
				"You are a super helpful assistant.",
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

			// getInitPrompt
			for _, m := range tt.responseSubstitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, gomock.Any(), gomock.Any()).Return(m, nil)
			}

			for _, m := range tt.expectMessageTexts {
				mockMessage.EXPECT().Create(ctx, tt.aicall.CustomerID, tt.aicall.ID, message.DirectionOutgoing, message.RoleSystem, m, nil, "").Return(&message.Message{}, nil)
			}

			if err := h.startInitMessages(ctx, tt.ai, tt.aicall); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
