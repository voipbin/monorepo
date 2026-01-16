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
	commonservice "monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ServiceStart_serviceStartReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		aiID          uuid.UUID
		activeflowID  uuid.UUID
		referenceType aicall.ReferenceType
		referenceID   uuid.UUID
		gender        aicall.Gender
		language      string

		responseAI                *ai.AI
		responseConfbridge        *cmconfbridge.Confbridge
		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcall        uuid.UUID
		responseMessages          []*message.Message
		responseUUIDAction        uuid.UUID

		expectAIcall       *aicall.AIcall
		expectMessageTexts []string
		expectLLMMessages  []map[string]any
		expectLLMType      pmpipecatcall.LLMType
		expectSTType       pmpipecatcall.STTType
		expectTTSType      pmpipecatcall.TTSType
		expectTTSVoiceID   string
		expectRes          *commonservice.Service
	}{
		{
			name:          "normal",
			aiID:          uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
			activeflowID:  uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
			referenceType: aicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
			gender:        aicall.GenderFemale,
			language:      "en-US",

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineModel: ai.EngineModel("openai.gpt-4"),
				InitPrompt:  "hello, this is init prompt message.",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "ee2d23be-b884-11f0-89b5-2f91294e7b2a",
				STTType:     ai.STTTypeDeepgram,
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				},
			},
			responseUUIDPipecatcallID: uuid.FromStringOrNil("025e1aa6-b87f-11f0-9a90-63680416f9cb"),
			responseUUIDAIcall:        uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
			responseMessages: []*message.Message{
				{
					Role:    "system",
					Content: "hello, this is init prompt message.",
				},
				{
					Role:    "system",
					Content: defaultCommonAIcallSystemPrompt,
				},
			},
			responseUUIDAction: uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:          uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
				ActiveflowID:  uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
				AIEngineModel: ai.EngineModel("openai.gpt-4"),
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "ee2d23be-b884-11f0-89b5-2f91294e7b2a",
				AISTTType:     ai.STTTypeDeepgram,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID: uuid.FromStringOrNil("025e1aa6-b87f-11f0-9a90-63680416f9cb"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
			},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				"hello, this is init prompt message.",
			},
			expectLLMType: pmpipecatcall.LLMType("openai.gpt-4"),
			expectLLMMessages: []map[string]any{
				{
					"role":    "system",
					"content": defaultCommonAIcallSystemPrompt,
				},
				{
					"role":    "system",
					"content": "hello, this is init prompt message.",
				},
			},
			expectSTType:     pmpipecatcall.STTTypeDeepgram,
			expectTTSType:    pmpipecatcall.TTSTypeElevenLabs,
			expectTTSVoiceID: "ee2d23be-b884-11f0-89b5-2f91294e7b2a",
			expectRes: &commonservice.Service{
				ID:   uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				Type: commonservice.TypeAIcall,
				PushActions: []fmaction.Action{
					{
						ID:   uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),
						Type: fmaction.TypeConfbridgeJoin,
						Option: map[string]any{
							"confbridge_id": "ec6d153d-dd5a-4eef-bc27-8fcebe100704",
						},
					},
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				messageHandler: mockMessage,
				aiHandler:      mockAI,
			}
			ctx := context.Background()

			// Start
			mockAI.EXPECT().Get(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().CallV1ConfbridgeCreate(
				ctx,
				cmcustomer.IDAIManager,
				tt.activeflowID,
				cmconfbridge.ReferenceTypeAI,
				tt.responseAI.ID,
				cmconfbridge.TypeConference,
			).Return(tt.responseConfbridge, nil)

			// startAIcall
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcall)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcall).Return(tt.expectAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.expectAIcall)

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.expectAIcall.ActiveflowID, gomock.Any()).Return(nil)

			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.expectAIcall.ActiveflowID, tt.responseAI.InitPrompt).Return(tt.responseAI.InitPrompt, nil)

			for i := range 2 {
				mockMessage.EXPECT().Create(
					ctx,
					tt.expectAIcall.CustomerID,
					tt.expectAIcall.ID,
					message.DirectionOutgoing,
					message.RoleSystem,
					tt.expectMessageTexts[i],
					nil,
					"",
				).Return(&message.Message{}, nil)
			}

			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)

			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.expectAIcall.PipecatcallID,
				tt.expectAIcall.CustomerID,
				tt.expectAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.expectAIcall.ID,
				tt.expectLLMType,
				tt.expectLLMMessages,
				tt.expectSTType,
				tt.expectAIcall.Language,
				tt.expectTTSType,
				tt.expectAIcall.Language,
				tt.expectTTSVoiceID,
			).Return(&pmpipecatcall.Pipecatcall{}, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAction)

			res, err := h.ServiceStart(ctx, tt.aiID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Expected result %#v, got %#v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceStart_serviceStartReferenceTypeConversation(t *testing.T) {
	tests := []struct {
		name string

		aiID          uuid.UUID
		activeflowID  uuid.UUID
		referenceType aicall.ReferenceType
		referenceID   uuid.UUID
		gender        aicall.Gender
		language      string

		responseAI           *ai.AI
		responseFlowVariable *fmvariable.Variable
		responseAIcall       *aicall.AIcall

		responseUUIDPipecatcallID uuid.UUID
		responseMessages          []*message.Message
		responsePipecatcall       *pmpipecatcall.Pipecatcall

		expectMessageText string
		expectLLMMessages []map[string]any
		expectLLMType     pmpipecatcall.LLMType
		expectSTType      pmpipecatcall.STTType
		expectTTSType     pmpipecatcall.TTSType
		expectTTSVoiceID  string
		expectRes         *commonservice.Service
	}{
		{
			name:          "normal",
			aiID:          uuid.FromStringOrNil("c3cd8518-b885-11f0-bae4-fb5033fa2df2"),
			activeflowID:  uuid.FromStringOrNil("c3ff93fa-b885-11f0-82cb-3f47ec04d13d"),
			referenceType: aicall.ReferenceTypeConversation,
			referenceID:   uuid.FromStringOrNil("c436f642-b885-11f0-8b5f-7b234b8f9158"),
			gender:        aicall.GenderFemale,
			language:      "en-US",

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3cd8518-b885-11f0-bae4-fb5033fa2df2"),
					CustomerID: uuid.FromStringOrNil("c468f26e-b885-11f0-b106-fb180bad9fd1"),
				},
				EngineModel: ai.EngineModel("openai.gpt-4"),
				InitPrompt:  "hello, this is init prompt message.",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "ee2d23be-b884-11f0-89b5-2f91294e7b2a",
				STTType:     ai.STTTypeDeepgram,
			},
			responseFlowVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"voipbin.conversation_message.text": "hello world",
				},
			},
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("15e0eebc-b886-11f0-9165-53c1245e306f"),
					CustomerID: uuid.FromStringOrNil("c468f26e-b885-11f0-b106-fb180bad9fd1"),
				},
				AIID:          uuid.FromStringOrNil("c3cd8518-b885-11f0-bae4-fb5033fa2df2"),
				ActiveflowID:  uuid.FromStringOrNil("c3ff93fa-b885-11f0-82cb-3f47ec04d13d"),
				AIEngineModel: ai.EngineModel("openai.gpt-4"),
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "ee2d23be-b884-11f0-89b5-2f91294e7b2a",
				AISTTType:     ai.STTTypeDeepgram,
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c436f642-b885-11f0-8b5f-7b234b8f9158"),
				PipecatcallID: uuid.FromStringOrNil("c4c99736-b885-11f0-b96c-436111319838"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
			},

			responseUUIDPipecatcallID: uuid.FromStringOrNil("c4c99736-b885-11f0-b96c-436111319838"),
			responseMessages: []*message.Message{
				{
					Role:    "user",
					Content: "hello world",
				},
				{
					Role:    "system",
					Content: "hello, this is init prompt message.",
				},
				{
					Role:    "system",
					Content: defaultCommonAIcallSystemPrompt,
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("540051a8-d530-11f0-94c9-bb8688a942c4"),
				},
				HostID: "host-12345",
			},

			expectMessageText: "hello world",
			expectLLMType:     pmpipecatcall.LLMType("openai.gpt-4"),
			expectLLMMessages: []map[string]any{
				{
					"role":    "system",
					"content": defaultCommonAIcallSystemPrompt,
				},
				{
					"role":    "system",
					"content": "hello, this is init prompt message.",
				},
				{
					"role":    "user",
					"content": "hello world",
				},
			},
			expectSTType:     pmpipecatcall.STTTypeDeepgram,
			expectTTSType:    pmpipecatcall.TTSTypeElevenLabs,
			expectTTSVoiceID: "ee2d23be-b884-11f0-89b5-2f91294e7b2a",
			expectRes: &commonservice.Service{
				ID:   uuid.FromStringOrNil("15e0eebc-b886-11f0-9165-53c1245e306f"),
				Type: commonservice.TypeAIcall,
				PushActions: []fmaction.Action{
					{
						ID:     uuid.FromStringOrNil("15e0eebc-b886-11f0-9165-53c1245e306f"),
						Type:   fmaction.TypeBlock,
						Option: fmaction.ConvertOption(fmaction.OptionBlock{}),
					},
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				messageHandler: mockMessage,
				aiHandler:      mockAI,
			}
			ctx := context.Background()

			mockAI.EXPECT().Get(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseFlowVariable, nil)
			mockDB.EXPECT().AIcallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseAIcall, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockDB.EXPECT().AIcallUpdate(ctx, tt.responseAIcall.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseAIcall.ID).Return(tt.responseAIcall, nil)

			mockMessage.EXPECT().Create(ctx, tt.responseAIcall.CustomerID, tt.responseAIcall.ID, message.DirectionOutgoing, message.RoleUser, tt.expectMessageText, nil, "").Return(&message.Message{}, nil)

			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)

			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.responseAIcall.PipecatcallID,
				tt.responseAIcall.CustomerID,
				tt.responseAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.responseAIcall.ID,
				tt.expectLLMType,
				tt.expectLLMMessages,
				tt.expectSTType,
				tt.responseAIcall.Language,
				tt.expectTTSType,
				tt.responseAIcall.Language,
				tt.expectTTSVoiceID,
			).Return(tt.responsePipecatcall, nil)
			mockReq.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, tt.responsePipecatcall.HostID, tt.responsePipecatcall.ID, defaultAITaskTimeout).Return(nil)

			res, err := h.ServiceStart(ctx, tt.aiID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Expected result %#v, got %#v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceStartTypeTask(t *testing.T) {
	tests := []struct {
		name string

		aiID         uuid.UUID
		activeflowID uuid.UUID

		responseAI                *ai.AI
		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcallID      uuid.UUID

		responseMessages    []*message.Message
		responsePipecatcall *pmpipecatcall.Pipecatcall

		expectAIcall       *aicall.AIcall
		expectMessageTexts []string
		expectLLMMessages  []map[string]any
		expectLLMType      pmpipecatcall.LLMType
		expectRes          *commonservice.Service
	}{
		{
			name:         "normal",
			aiID:         uuid.FromStringOrNil("48021ad4-d70c-11f0-9a63-c38f93e192a7"),
			activeflowID: uuid.FromStringOrNil("4838bc74-d70c-11f0-b4ff-af530084525d"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("48021ad4-d70c-11f0-9a63-c38f93e192a7"),
					CustomerID: uuid.FromStringOrNil("c468f26e-b885-11f0-b106-fb180bad9fd1"),
				},
				EngineModel: ai.EngineModel("openai.gpt-4"),
				InitPrompt:  "hello, this is init prompt message.",
			},
			responseUUIDPipecatcallID: uuid.FromStringOrNil("c4c99736-b885-11f0-b96c-436111319838"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("486ab602-d70c-11f0-9665-1b75a7a17c15"),
			responseMessages: []*message.Message{
				{
					Role:    "system",
					Content: "hello, this is init prompt message.",
				},
				{
					Role:    "system",
					Content: defaultCommonAItaskSystemPrompt,
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("540051a8-d530-11f0-94c9-bb8688a942c4"),
				},
				HostID: "host-12345",
			},

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("486ab602-d70c-11f0-9665-1b75a7a17c15"),
					CustomerID: uuid.FromStringOrNil("c468f26e-b885-11f0-b106-fb180bad9fd1"),
				},
				AIID:          uuid.FromStringOrNil("48021ad4-d70c-11f0-9a63-c38f93e192a7"),
				AIEngineModel: ai.EngineModel("openai.gpt-4"),
				ActiveflowID:  uuid.FromStringOrNil("4838bc74-d70c-11f0-b4ff-af530084525d"),
				ReferenceType: aicall.ReferenceTypeTask,
				PipecatcallID: uuid.FromStringOrNil("c4c99736-b885-11f0-b96c-436111319838"),
				Status:        aicall.StatusInitiating,
			},
			expectMessageTexts: []string{
				defaultCommonAItaskSystemPrompt,
				"hello, this is init prompt message.",
			},
			expectLLMType: pmpipecatcall.LLMType("openai.gpt-4"),
			expectLLMMessages: []map[string]any{
				{
					"role":    "system",
					"content": defaultCommonAItaskSystemPrompt,
				},
				{
					"role":    "system",
					"content": "hello, this is init prompt message.",
				},
			},
			expectRes: &commonservice.Service{
				ID:   uuid.FromStringOrNil("486ab602-d70c-11f0-9665-1b75a7a17c15"),
				Type: commonservice.TypeAIcall,
				PushActions: []fmaction.Action{
					{
						ID:     uuid.FromStringOrNil("486ab602-d70c-11f0-9665-1b75a7a17c15"),
						Type:   fmaction.TypeBlock,
						Option: map[string]any{},
					},
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				messageHandler: mockMessage,
				aiHandler:      mockAI,
			}
			ctx := context.Background()

			mockAI.EXPECT().Get(ctx, tt.aiID).Return(tt.responseAI, nil)

			// start aicall
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.expectAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.expectAIcall)

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, gomock.Any()).Return(nil)

			// startInitMessages
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.responseAI.InitPrompt).Return(tt.responseAI.InitPrompt, nil)
			for i := range 2 {
				mockMessage.EXPECT().Create(
					ctx,
					tt.responseAI.CustomerID,
					tt.expectAIcall.ID,
					message.DirectionOutgoing,
					message.RoleSystem,
					tt.expectMessageTexts[i],
					nil,
					"",
				).Return(&message.Message{}, nil)
			}

			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)

			// start pipecatcall
			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.expectAIcall.PipecatcallID,
				tt.expectAIcall.CustomerID,
				tt.expectAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.expectAIcall.ID,
				tt.expectLLMType,
				tt.expectLLMMessages,
				pmpipecatcall.STTTypeNone,
				"",
				pmpipecatcall.TTSTypeNone,
				"",
				"",
			).Return(tt.responsePipecatcall, nil)

			mockReq.EXPECT().AIV1AIcallTerminateWithDelay(ctx, tt.expectAIcall.ID, defaultAITaskTimeout).Return(nil)

			res, err := h.ServiceStartTypeTask(ctx, tt.aiID, tt.activeflowID)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Expected result %v, got %v", tt.expectRes, res)
			}
		})
	}
}
