package aicallhandler

import (
	"context"
	"fmt"
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
	fmvariable "monorepo/bin-flow-manager/models/variable"
	tmstreaming "monorepo/bin-tts-manager/models/streaming"
	reflect "reflect"

	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmaction "monorepo/bin-flow-manager/models/action"
	"testing"
	"time"

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
		resume        bool

		responseAI           *ai.AI
		responseConfbridge   *cmconfbridge.Confbridge
		responseTTSStreaming *tmstreaming.Streaming
		responseUUIDAIcall   uuid.UUID
		responseAIcall       *aicall.AIcall
		responseMessage      *message.Message
		responseUUIDAction   uuid.UUID

		expectAIcall *aicall.AIcall
		expectRes    *commonservice.Service
	}{
		{
			name:          "normal - english female",
			aiID:          uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
			activeflowID:  uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
			referenceType: aicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
			gender:        aicall.GenderFemale,
			language:      "en-US",
			resume:        false,

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType: ai.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				},
			},
			responseTTSStreaming: &tmstreaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("457c3846-817e-11f0-9090-6309abe0b661"),
				},
				PodID: "45e53dd2-817e-11f0-a608-7f6bb70061e4",
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
			responseUUIDAction: uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:              uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
				ActiveflowID:      uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
				AIEngineType:      ai.EngineTypeNone,
				ReferenceType:     aicall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
				ConfbridgeID:      uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				Gender:            aicall.GenderFemale,
				Language:          "en-US",
				Status:            aicall.StatusInitiating,
				TTSStreamingID:    uuid.FromStringOrNil("457c3846-817e-11f0-9090-6309abe0b661"),
				TTSStreamingPodID: "45e53dd2-817e-11f0-a608-7f6bb70061e4",
			},
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
			mockAI.EXPECT().Get(ctx, tt.aiID).Return(tt.responseAI, nil)
			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, tt.activeflowID, cmconfbridge.ReferenceTypeAI, tt.responseAI.ID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)
			mockReq.EXPECT().TTSV1StreamingCreate(
				ctx,
				tt.responseAI.CustomerID,
				tt.activeflowID,
				tmstreaming.ReferenceTypeCall,
				tt.referenceID,
				tt.language,
				tmstreaming.Gender(tt.gender),
				tmstreaming.DirectionOutgoing,
			).Return(tt.responseTTSStreaming, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcall)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcall).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			mockMessage.EXPECT().StreamingSend(ctx, tt.responseAIcall.ID, message.RoleSystem, tt.responseAI.InitPrompt, true).Return(tt.responseMessage, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAction)

			res, err := h.ServiceStart(ctx, tt.aiID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language, tt.resume)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Expected result %#v, got %#v", tt.expectRes, res)
			}

			time.Sleep(100 * time.Millisecond)
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
		resume        bool

		responseAI         *ai.AI
		responseConfbridge *cmconfbridge.Confbridge
		responseUUIDAIcall uuid.UUID
		responseAIcall     *aicall.AIcall
		responseVariable   *fmvariable.Variable
		responseMessage    *message.Message
		responseUUIDAction uuid.UUID

		expectAIcall         *aicall.AIcall
		expectMessageContent string
		expectRes            *commonservice.Service
	}{
		{
			name:          "normal",
			aiID:          uuid.FromStringOrNil("979b54dc-30f1-11f0-b20f-cf68bd028351"),
			activeflowID:  uuid.FromStringOrNil("97c49694-30f1-11f0-9312-77d7d1f35c66"),
			referenceType: aicall.ReferenceTypeConversation,
			referenceID:   uuid.FromStringOrNil("97edda2c-30f1-11f0-8341-f38ceaa8013d"),
			gender:        aicall.GenderFemale,
			language:      "en-US",
			resume:        false,

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("979b54dc-30f1-11f0-b20f-cf68bd028351"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType: ai.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				},
			},
			responseUUIDAIcall: uuid.FromStringOrNil("983b70ca-30f1-11f0-b3a1-1bc84ea9dc87"),
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("983b70ca-30f1-11f0-b3a1-1bc84ea9dc87"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"voipbin.conversation_message.text": "test assistant message.",
				},
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},
			responseUUIDAction: uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("983b70ca-30f1-11f0-b3a1-1bc84ea9dc87"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AIID:          uuid.FromStringOrNil("979b54dc-30f1-11f0-b20f-cf68bd028351"),
				ActiveflowID:  uuid.FromStringOrNil("97c49694-30f1-11f0-9312-77d7d1f35c66"),
				AIEngineType:  ai.EngineTypeNone,
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   uuid.FromStringOrNil("97edda2c-30f1-11f0-8341-f38ceaa8013d"),
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
			},
			expectMessageContent: "test assistant message.",
			expectRes: &commonservice.Service{
				ID:          uuid.FromStringOrNil("983b70ca-30f1-11f0-b3a1-1bc84ea9dc87"),
				Type:        commonservice.TypeAIcall,
				PushActions: []fmaction.Action{},
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

			mockAI.EXPECT().Get(ctx, tt.aiID).Return(tt.responseAI, nil)

			mockDB.EXPECT().AIcallGetByReferenceID(ctx, tt.referenceID).Return(nil, fmt.Errorf(""))

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcall)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcall).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)

			mockMessage.EXPECT().Send(ctx, tt.responseAIcall.ID, message.RoleUser, tt.expectMessageContent, false).Return(tt.responseMessage, nil)

			res, err := h.ServiceStart(ctx, tt.aiID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language, tt.resume)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Expected result %#v, got %#v", tt.expectRes, res)
			}

			time.Sleep(100 * time.Millisecond)
		})
	}
}
