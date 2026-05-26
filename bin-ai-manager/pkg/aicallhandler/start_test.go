package aicallhandler

import (
	"context"
	"errors"
	"fmt"
	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/team"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-ai-manager/pkg/participanthandler"
	"monorepo/bin-ai-manager/pkg/teamhandler"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	pmpipecatcall "monorepo/bin-pipecat-manager/models/pipecatcall"
	reflect "reflect"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	gomock "go.uber.org/mock/gomock"
)

func Test_startReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		ai             *ai.AI
		assistanceType aicall.AssistanceType
		assistanceID   uuid.UUID
		activeflowID   uuid.UUID
		referenceID    uuid.UUID


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
				EngineModel: "openai.gpt-5",
				InitPrompt:  "hello, this is init prompt message.",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
			activeflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
			referenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),


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
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				AITTSType:      ai.TTSTypeElevenLabs,
				AITTSVoiceID:   "21m00Tcm4TlvDq8ikWAM",
				AISTTType:      ai.STTTypeDeepgram,
				ActiveflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:   uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID:  uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),

				STTLanguage:    "en-US",
				Status:         aicall.StatusInitiating,
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
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel:  "openai.gpt-5",
				ActiveflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:   uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID:  uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),

				STTLanguage:    "en-US",
				Status:         aicall.StatusInitiating,
			},
			expectVariables: map[string]string{
				variableID:            "a6cd01d0-d785-467f-9069-684e46cc2644",
				variableAIID:          "a4107e6e-f06d-11ef-9b7a-03c848b3bb41",
				variableAIEngineModel: "openai.gpt-5",
				variableConfbridgeID:  "ec6d153d-dd5a-4eef-bc27-8fcebe100704",

				variableSTTLanguage:   "en-US",
				variablePipecatcallID: "a4e5c7ae-b539-11f0-ac68-c38f244d145b",
			},
			expectLLMType: pmpipecatcall.LLMType("openai.gpt-5"),
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
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				AITTSType:      ai.TTSTypeElevenLabs,
				AITTSVoiceID:   "21m00Tcm4TlvDq8ikWAM",
				AISTTType:      ai.STTTypeDeepgram,
				ActiveflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:   uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID:  uuid.FromStringOrNil("a4e5c7ae-b539-11f0-ac68-c38f244d145b"),

				STTLanguage:    "en-US",
				Status:         aicall.StatusInitiating,
			},
		},
		{
			name: "with participant handler",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineModel: "openai.gpt-5",
				InitPrompt:  "hello, this is init prompt message.",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
			activeflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
			referenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),

			responseConfbridge: &cmconfbridge.Confbridge{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				},
			},
			responseUUIDPipecatcallID: uuid.FromStringOrNil("b4e5c7ae-b539-11f0-ac68-c38f244d145b"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("b6cd01d0-d785-467f-9069-684e46cc2644"),
			responseAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				AITTSType:      ai.TTSTypeElevenLabs,
				AITTSVoiceID:   "21m00Tcm4TlvDq8ikWAM",
				AISTTType:      ai.STTTypeDeepgram,
				ActiveflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:   uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID:  uuid.FromStringOrNil("b4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				STTLanguage:    "en-US",
				Status:         aicall.StatusInitiating,
			},
			responseMessages: []*message.Message{
				{
					Role:    "assistant",
					Content: "test assistant message.",
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				},
			},

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel:  "openai.gpt-5",
				ActiveflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:   uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID:  uuid.FromStringOrNil("b4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				STTLanguage:    "en-US",
				Status:         aicall.StatusInitiating,
			},
			expectVariables: map[string]string{
				variableID:            "b6cd01d0-d785-467f-9069-684e46cc2644",
				variableAIID:          "a4107e6e-f06d-11ef-9b7a-03c848b3bb41",
				variableAIEngineModel: "openai.gpt-5",
				variableConfbridgeID:  "ec6d153d-dd5a-4eef-bc27-8fcebe100704",
				variableSTTLanguage:   "en-US",
				variablePipecatcallID: "b4e5c7ae-b539-11f0-ac68-c38f244d145b",
			},
			expectLLMType: pmpipecatcall.LLMType("openai.gpt-5"),
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
					ID:         uuid.FromStringOrNil("b6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				AIEngineModel:  ai.EngineModel("openai.gpt-5"),
				AITTSType:      ai.TTSTypeElevenLabs,
				AITTSVoiceID:   "21m00Tcm4TlvDq8ikWAM",
				AISTTType:      ai.STTTypeDeepgram,
				ActiveflowID:   uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:   uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				PipecatcallID:  uuid.FromStringOrNil("b4e5c7ae-b539-11f0-ac68-c38f244d145b"),
				STTLanguage:    "en-US",
				Status:         aicall.StatusInitiating,
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
			if tt.name == "with participant handler" {
				mockParticipant := participanthandler.NewMockParticipantHandler(mc)
				mockParticipant.EXPECT().Create(gomock.Any(), gomock.Any(), tt.ai.ID).Return(nil).Times(1)
				h.participantHandler = mockParticipant
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, cmcustomer.IDAIManager, tt.activeflowID, cmconfbridge.ReferenceTypeAI, tt.ai.ID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)

			// startAIcall
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, gomock.Any()).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
			mockMessage.EXPECT().Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&message.Message{}, nil)

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, tt.expectVariables).Return(nil)
			// startInitMessages + buildPromptSnapshots both substitute the AI init prompt
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.responseAIcall.ActiveflowID, tt.ai.InitPrompt).Return(tt.ai.InitPrompt, nil).Times(2)

			// startPipecatcall
			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)
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
				tt.responseAIcall.STTLanguage,
				tt.expectTTSType,
				"",
				tt.expectTTSVoiceID,
			).Return(tt.responsePipecatcall, nil)

			res, err := h.startReferenceTypeCall(ctx, tt.ai, tt.assistanceType, tt.assistanceID, tt.activeflowID, tt.referenceID, nil, uuid.Nil)
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

		ai             *ai.AI
		assistanceType aicall.AssistanceType
		assistanceID   uuid.UUID


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
				InitPrompt:  "hello, this is init prompt message.",
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),


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
			mockMessage.EXPECT().Create(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&message.Message{}, nil)

			mockDB.EXPECT().AIcallUpdate(ctx, tt.responseAIcall.ID, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseAIcall.ID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusProgressing, tt.responseAIcall)

			res, err := h.startReferenceTypeNone(ctx, tt.ai, tt.assistanceType, tt.assistanceID, uuid.Nil, nil, uuid.Nil)
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
	// ensure idle threshold is set deterministically for cases that depend on it
	config.SetAIcallConversationIdleTimeoutHoursForTest(24)

	freshTM := time.Now().Add(-1 * time.Hour)
	expiredTM := time.Now().Add(-25 * time.Hour)

	type mocks struct {
		util    *utilhandler.MockUtilHandler
		req     *requesthandler.MockRequestHandler
		notify  *notifyhandler.MockNotifyHandler
		db      *dbhandler.MockDBHandler
		ai      *aihandler.MockAIHandler
		team    *teamhandler.MockTeamHandler
		message *messagehandler.MockMessageHandler
	}

	tests := []struct {
		name string

		ai             *ai.AI
		assistanceType aicall.AssistanceType
		assistanceID   uuid.UUID
		activeflowID   uuid.UUID
		referenceID    uuid.UUID

		mockSetup func(ctx context.Context, m *mocks)

		expectRes *aicall.AIcall

		// expectErr — when true, the function MUST return a non-nil error
		// AND res MUST be nil. expectRes is ignored when expectErr is true,
		// but expectErrSubstring (when set) MUST be a substring of err.Error().
		expectErr          bool
		expectErrSubstring string

		// expectIdleExpiredInc — when true, the idle-expired counter
		// (promAIcallIdleExpiredTotal) MUST increment by exactly 1 across
		// the call. When false, it MUST NOT change.
		expectIdleExpiredInc bool
	}{
		{
			name: "reuse: alive previous pipecat — interrupt invoked",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				InitPrompt:  "hello, this is init prompt message.",
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("d184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				existingAIcallID := uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5")
				oldPCC := uuid.FromStringOrNil("aaaaaaaa-0001-11f0-aaaa-aaaaaaaaaaaa")
				newPCC := uuid.FromStringOrNil("017c1c12-b737-11f0-80ad-032b0dde6a93")
				existing := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         existingAIcallID,
						CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					AIEngineModel: "openai.gpt-5-nano",
					Status:        aicall.StatusProgressing,
					TMUpdate:      &freshTM,
					PipecatcallID: oldPCC,
				}
				pipecatcall := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{ID: oldPCC},
					HostID:   "host1",
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "test user message.",
					},
				}, nil)

				// GetByReferenceID returns existing reusable AIcall
				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(existing, nil)

				// interruptPreviousPipecatcall: get -> ping ok -> terminate
				m.req.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), oldPCC).Return(pipecatcall, nil)
				m.req.EXPECT().PipecatV1Ping(gomock.Any(), "host1").Return(nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminate(gomock.Any(), "host1", oldPCC).Return(nil, nil)

				// new pipecatcall ID + atomic UpdatePipecatcallIDAndActiveflowID
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.db.EXPECT().AIcallUpdate(ctx, existingAIcallID, map[aicall.Field]any{
					aicall.FieldPipecatcallID: newPCC,
					aicall.FieldActiveflowID:  uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
				}).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, existingAIcallID).Return(existing, nil)

				// conversation message create
				m.message.EXPECT().Create(ctx, uuid.Nil, existing.CustomerID, existing.ID, existing.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "test user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// startPipecatcall
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					existing.PipecatcallID,
					existing.CustomerID,
					existing.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					existing.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d1319db4-30dd-11f0-8747-a7f601e136a5"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				AIEngineModel: "openai.gpt-5-nano",
				Status:        aicall.StatusProgressing,
				TMUpdate:      &freshTM,
				PipecatcallID: uuid.FromStringOrNil("aaaaaaaa-0001-11f0-aaaa-aaaaaaaaaaaa"),
			},
		},
		{
			name: "reuse: dead previous pipecat — interrupt skipped after ping",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("d184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				existingAIcallID := uuid.FromStringOrNil("b1319db4-30dd-11f0-8747-a7f601e136a5")
				oldPCC := uuid.FromStringOrNil("bbbbbbbb-0001-11f0-bbbb-bbbbbbbbbbbb")
				newPCC := uuid.FromStringOrNil("117c1c12-b737-11f0-80ad-032b0dde6a93")
				existing := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         existingAIcallID,
						CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					AIEngineModel: "openai.gpt-5-nano",
					Status:        aicall.StatusProgressing,
					TMUpdate:      &freshTM,
					PipecatcallID: oldPCC,
				}
				pipecatcall := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{ID: oldPCC},
					HostID:   "host2",
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("4f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "another user message.",
					},
				}, nil)

				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(existing, nil)

				// interruptPreviousPipecatcall: get -> ping fails -> NO terminate
				m.req.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), oldPCC).Return(pipecatcall, nil)
				m.req.EXPECT().PipecatV1Ping(gomock.Any(), "host2").Return(context.DeadlineExceeded)
				// no PipecatV1PipecatcallTerminate expectation

				// new pipecatcall ID + atomic UpdatePipecatcallIDAndActiveflowID
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.db.EXPECT().AIcallUpdate(ctx, existingAIcallID, map[aicall.Field]any{
					aicall.FieldPipecatcallID: newPCC,
					aicall.FieldActiveflowID:  uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
				}).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, existingAIcallID).Return(existing, nil)

				m.message.EXPECT().Create(ctx, uuid.Nil, existing.CustomerID, existing.ID, existing.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "another user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					existing.PipecatcallID,
					existing.CustomerID,
					existing.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					existing.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b1319db4-30dd-11f0-8747-a7f601e136a5"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				AIEngineModel: "openai.gpt-5-nano",
				Status:        aicall.StatusProgressing,
				TMUpdate:      &freshTM,
				PipecatcallID: uuid.FromStringOrNil("bbbbbbbb-0001-11f0-bbbb-bbbbbbbbbbbb"),
			},
		},
		{
			name: "fresh: GetByReferenceID returns error — startAIcallByMessaging path",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("c0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("c184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				newPCC := uuid.FromStringOrNil("cccccccc-0001-11f0-cccc-cccccccccccc")
				newAIcallID := uuid.FromStringOrNil("cccccccc-0002-11f0-cccc-cccccccccccc")
				createdAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID:  uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
					AIEngineModel: ai.EngineModelOpenaiGPT5,
					Status:        aicall.StatusInitiating,
				}
				progressingAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID:  uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
					AIEngineModel: ai.EngineModelOpenaiGPT5,
					Status:        aicall.StatusProgressing,
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "fresh user message.",
					},
				}, nil)

				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(nil, errors.New("not found"))

				// startAIcallByMessaging internal calls: pipecatcall id + create
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.util.EXPECT().UUIDCreate().Return(newAIcallID)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(createdAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())

				// setActiveflowVariables
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)

				// startInitMessages — system prompt only (no init prompt set)
				m.message.EXPECT().Create(ctx, uuid.Nil, createdAIcall.CustomerID, createdAIcall.ID, createdAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// UpdateStatus -> Progressing
				m.db.EXPECT().AIcallUpdate(ctx, newAIcallID, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(progressingAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, progressingAIcall.CustomerID, aicall.EventTypeStatusProgressing, progressingAIcall)

				// conversation message create
				m.message.EXPECT().Create(ctx, uuid.Nil, progressingAIcall.CustomerID, progressingAIcall.ID, progressingAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "fresh user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// startPipecatcall
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					progressingAIcall.PipecatcallID,
					progressingAIcall.CustomerID,
					progressingAIcall.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					progressingAIcall.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cccccccc-0002-11f0-cccc-cccccccccccc"),
					CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				ActiveflowID:  uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
				AIEngineModel: ai.EngineModelOpenaiGPT5,
				Status:        aicall.StatusProgressing,
			},
		},
		{
			// Verifies that a transient DB failure on UpdateStatus(Progressing)
			// does NOT drop the user's message: the function logs a warning and
			// proceeds with messageHandler.Create + startPipecatcall using the
			// freshly created AIcall (still at StatusInitiating). The status
			// field is observability-only and AIcall behavior is correct
			// regardless.
			name: "fresh path: UpdateStatus fails — proceeds anyway",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("c0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("c184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				newPCC := uuid.FromStringOrNil("cccccccc-1001-11f0-cccc-cccccccccccc")
				newAIcallID := uuid.FromStringOrNil("cccccccc-1002-11f0-cccc-cccccccccccc")
				createdAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID:  uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
					AIEngineModel: ai.EngineModelOpenaiGPT5,
					Status:        aicall.StatusInitiating,
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("5f97dd3a-c663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "fresh user message — update fails.",
					},
				}, nil)

				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(nil, errors.New("not found"))

				// startAIcallByMessaging internal calls
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.util.EXPECT().UUIDCreate().Return(newAIcallID)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(createdAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())

				// setActiveflowVariables
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)

				// startInitMessages — system prompt only (no init prompt set)
				m.message.EXPECT().Create(ctx, uuid.Nil, createdAIcall.CustomerID, createdAIcall.ID, createdAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// UpdateStatus -> Progressing FAILS at AIcallUpdate (no Get / PublishWebhookEvent follow-up).
				// Per fix: log warn, do NOT fail the request. Subsequent calls should still happen
				// against the createdAIcall (still StatusInitiating since update failed).
				m.db.EXPECT().AIcallUpdate(ctx, newAIcallID, gomock.Any()).Return(errors.New("transient db error"))

				// conversation message create — proceeds despite UpdateStatus failure
				m.message.EXPECT().Create(ctx, uuid.Nil, createdAIcall.CustomerID, createdAIcall.ID, createdAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "fresh user message — update fails.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// startPipecatcall — proceeds despite UpdateStatus failure
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					createdAIcall.PipecatcallID,
					createdAIcall.CustomerID,
					createdAIcall.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					createdAIcall.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			// Returned AIcall is the pre-update copy (StatusInitiating) since UpdateStatus failed
			// and res was not reassigned.
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("cccccccc-1002-11f0-cccc-cccccccccccc"),
					CustomerID: uuid.FromStringOrNil("cdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				ActiveflowID:  uuid.FromStringOrNil("c15ae476-30dd-11f0-87af-67d3c47111a7"),
				AIEngineModel: ai.EngineModelOpenaiGPT5,
				Status:        aicall.StatusInitiating,
			},
		},
		{
			name: "fresh: existing AIcall is StatusTerminated — recreate",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("edbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("e0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("e15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("e184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				oldAIcallID := uuid.FromStringOrNil("dddddddd-0001-11f0-dddd-dddddddddddd")
				oldPCC := uuid.FromStringOrNil("dddddddd-0099-11f0-dddd-dddddddddddd")
				newPCC := uuid.FromStringOrNil("eeeeeeee-0001-11f0-eeee-eeeeeeeeeeee")
				newAIcallID := uuid.FromStringOrNil("eeeeeeee-0002-11f0-eeee-eeeeeeeeeeee")
				existingTerminated := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         oldAIcallID,
						CustomerID: uuid.FromStringOrNil("edbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					Status:        aicall.StatusTerminated,
					TMUpdate:      &freshTM,
					PipecatcallID: oldPCC,
				}
				createdAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("edbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID: uuid.FromStringOrNil("e15ae476-30dd-11f0-87af-67d3c47111a7"),
					Status:       aicall.StatusInitiating,
				}
				progressingAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("edbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID: uuid.FromStringOrNil("e15ae476-30dd-11f0-87af-67d3c47111a7"),
					Status:       aicall.StatusProgressing,
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "post-terminated user message.",
					},
				}, nil)

				// GetByReferenceID returns terminated AIcall
				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(existingTerminated, nil)

				// NO interruptPreviousPipecatcall (idle-expiry branch is short-circuited because status is already StatusTerminated)
				// NO UpdateStatus(StatusTerminated) on the old AIcall

				// startAIcallByMessaging
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.util.EXPECT().UUIDCreate().Return(newAIcallID)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(createdAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)
				m.message.EXPECT().Create(ctx, uuid.Nil, createdAIcall.CustomerID, createdAIcall.ID, createdAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// UpdateStatus -> Progressing on new AIcall
				m.db.EXPECT().AIcallUpdate(ctx, newAIcallID, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(progressingAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, progressingAIcall.CustomerID, aicall.EventTypeStatusProgressing, progressingAIcall)

				// conversation message create
				m.message.EXPECT().Create(ctx, uuid.Nil, progressingAIcall.CustomerID, progressingAIcall.ID, progressingAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "post-terminated user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// startPipecatcall
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					progressingAIcall.PipecatcallID,
					progressingAIcall.CustomerID,
					progressingAIcall.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					progressingAIcall.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("eeeeeeee-0002-11f0-eeee-eeeeeeeeeeee"),
					CustomerID: uuid.FromStringOrNil("edbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				ActiveflowID: uuid.FromStringOrNil("e15ae476-30dd-11f0-87af-67d3c47111a7"),
				Status:       aicall.StatusProgressing,
			},
		},
		{
			name: "fresh: idle-expired AIcall — terminate then recreate",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("fdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("f0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("f15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("f184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				oldAIcallID := uuid.FromStringOrNil("a1111111-0001-11f0-aaaa-aaaaaaaaaaaa")
				oldPCC := uuid.FromStringOrNil("a1111111-0099-11f0-aaaa-aaaaaaaaaaaa")
				newPCC := uuid.FromStringOrNil("a2222222-0001-11f0-aaaa-aaaaaaaaaaaa")
				newAIcallID := uuid.FromStringOrNil("a2222222-0002-11f0-aaaa-aaaaaaaaaaaa")
				existingIdle := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         oldAIcallID,
						CustomerID: uuid.FromStringOrNil("fdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					Status:        aicall.StatusProgressing,
					TMUpdate:      &expiredTM,
					PipecatcallID: oldPCC,
				}
				terminatedOld := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         oldAIcallID,
						CustomerID: uuid.FromStringOrNil("fdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					Status:        aicall.StatusTerminated,
					PipecatcallID: oldPCC,
				}
				createdAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("fdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID: uuid.FromStringOrNil("f15ae476-30dd-11f0-87af-67d3c47111a7"),
					Status:       aicall.StatusInitiating,
				}
				progressingAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("fdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					ActiveflowID: uuid.FromStringOrNil("f15ae476-30dd-11f0-87af-67d3c47111a7"),
					Status:       aicall.StatusProgressing,
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "after-idle user message.",
					},
				}, nil)

				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(existingIdle, nil)

				// idle-expiry branch: UpdateStatus(StatusTerminated) on the OLD AIcall
				m.db.EXPECT().AIcallUpdate(ctx, oldAIcallID, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, oldAIcallID).Return(terminatedOld, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, terminatedOld.CustomerID, aicall.EventTypeStatusTerminated, terminatedOld)

				// startAIcallByMessaging
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.util.EXPECT().UUIDCreate().Return(newAIcallID)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(createdAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)
				m.message.EXPECT().Create(ctx, uuid.Nil, createdAIcall.CustomerID, createdAIcall.ID, createdAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// UpdateStatus -> Progressing on the NEW AIcall
				m.db.EXPECT().AIcallUpdate(ctx, newAIcallID, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(progressingAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, progressingAIcall.CustomerID, aicall.EventTypeStatusProgressing, progressingAIcall)

				// conversation message create
				m.message.EXPECT().Create(ctx, uuid.Nil, progressingAIcall.CustomerID, progressingAIcall.ID, progressingAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "after-idle user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					progressingAIcall.PipecatcallID,
					progressingAIcall.CustomerID,
					progressingAIcall.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					progressingAIcall.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a2222222-0002-11f0-aaaa-aaaaaaaaaaaa"),
					CustomerID: uuid.FromStringOrNil("fdbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				ActiveflowID: uuid.FromStringOrNil("f15ae476-30dd-11f0-87af-67d3c47111a7"),
				Status:       aicall.StatusProgressing,
			},
			expectIdleExpiredInc: true,
		},
		{
			name: "team smoke: reuse + alive previous pipecat",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
			activeflowID:   uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
			referenceID:    uuid.FromStringOrNil("d184c87c-30dd-11f0-8bbf-d773a2d31d73"),

			mockSetup: func(ctx context.Context, m *mocks) {
				existingAIcallID := uuid.FromStringOrNil("a3333333-0001-11f0-9999-999999999999")
				oldPCC := uuid.FromStringOrNil("a3333333-0099-11f0-9999-999999999999")
				newPCC := uuid.FromStringOrNil("a4444444-0001-11f0-9999-999999999999")
				teamID := uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef")
				memberID := uuid.FromStringOrNil("a5555555-0001-11f0-9999-999999999999")
				memberAIID := uuid.FromStringOrNil("a6666666-0001-11f0-9999-999999999999")
				existing := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         existingAIcallID,
						CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
					},
					AssistanceType:  aicall.AssistanceTypeTeam,
					AssistanceID:    teamID,
					AIEngineModel:   "openai.gpt-5-nano", // stale snapshot; resolveTeamMemberForSend overrides it
					Status:          aicall.StatusProgressing,
					TMUpdate:        &freshTM,
					PipecatcallID:   oldPCC,
					CurrentMemberID: memberID,
				}
				pipecatcall := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{ID: oldPCC},
					HostID:   "host-team",
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8f97dd3a-b663-11f0-b2ae-0b46e18cb363"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "team user message.",
					},
				}, nil)

				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(existing, nil)

				// interruptPreviousPipecatcall: alive
				m.req.EXPECT().PipecatV1PipecatcallGet(gomock.Any(), oldPCC).Return(pipecatcall, nil)
				m.req.EXPECT().PipecatV1Ping(gomock.Any(), "host-team").Return(nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminate(gomock.Any(), "host-team", oldPCC).Return(nil, nil)

				// new pipecatcall ID + atomic UpdatePipecatcallIDAndActiveflowID
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.db.EXPECT().AIcallUpdate(ctx, existingAIcallID, map[aicall.Field]any{
					aicall.FieldPipecatcallID: newPCC,
					aicall.FieldActiveflowID:  uuid.FromStringOrNil("d15ae476-30dd-11f0-87af-67d3c47111a7"),
				}).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, existingAIcallID).Return(existing, nil)

				// resolveTeamMemberForSend: refresh AIEngineModel from current member's AI.
				// CurrentMemberID matches a team member, so no fallback / no UpdateCurrentMemberID call.
				m.team.EXPECT().Get(ctx, teamID).Return(&team.Team{
					Identity: commonidentity.Identity{ID: teamID},
					StartMemberID: uuid.FromStringOrNil("a7777777-0001-11f0-9999-999999999999"),
					Members: []team.Member{
						{ID: memberID, AIID: memberAIID},
					},
				}, nil)
				m.ai.EXPECT().Get(ctx, memberAIID).Return(&ai.AI{
					Identity: commonidentity.Identity{ID: memberAIID},
					EngineModel: "grok.grok-3", // resolved engine model overrides the stale snapshot
				}, nil)

				// resolveActiveAIIDFromAIcall: get team to find CurrentMemberID's AIID.
				m.team.EXPECT().Get(ctx, teamID).Return(&team.Team{
					Identity: commonidentity.Identity{ID: teamID},
					StartMemberID: uuid.FromStringOrNil("a7777777-0001-11f0-9999-999999999999"),
					Members: []team.Member{
						{ID: memberID, AIID: memberAIID},
					},
				}, nil)

				m.message.EXPECT().Create(ctx, uuid.Nil, existing.CustomerID, existing.ID, existing.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "team user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					existing.PipecatcallID,
					existing.CustomerID,
					existing.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					existing.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a3333333-0001-11f0-9999-999999999999"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("d0f2a050-30dd-11f0-b9f5-6fd58444fdef"),
				AIEngineModel:   "grok.grok-3", // overridden by resolveTeamMemberForSend
				Status:          aicall.StatusProgressing,
				TMUpdate:        &freshTM,
				PipecatcallID:   uuid.FromStringOrNil("a3333333-0099-11f0-9999-999999999999"),
				CurrentMemberID: uuid.FromStringOrNil("a5555555-0001-11f0-9999-999999999999"),
			},
		},
		{
			// FlowV1VariableGet returns an error — function aborts before any
			// downstream work (no GetByReferenceID, no startAIcallByMessaging,
			// no messageHandler.Create, no startPipecatcall). Confirms the
			// error is wrapped with the "could not get the activeflow variables"
			// context.
			name: "error: FlowV1VariableGet fails — aborts before any work",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0a111111-0000-11f0-aaaa-000000000001"),
					CustomerID: uuid.FromStringOrNil("0a111111-0000-11f0-aaaa-000000000002"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("0a111111-0000-11f0-aaaa-000000000001"),
			activeflowID:   uuid.FromStringOrNil("0a111111-0000-11f0-aaaa-000000000003"),
			referenceID:    uuid.FromStringOrNil("0a111111-0000-11f0-aaaa-000000000004"),

			mockSetup: func(ctx context.Context, m *mocks) {
				// Single FlowV1VariableGet expectation that fails. NO downstream
				// calls of any kind: no AIcallGetByReferenceID, no message Create,
				// no PipecatcallStart, no UpdateStatus, etc.
				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(nil, errors.New("flow rpc error"))
			},
			expectErr:          true,
			expectErrSubstring: "could not get the activeflow variables",
		},
		{
			// FlowV1VariableGet succeeds but the returned Variable does not
			// contain the "voipbin.conversation_message.text" key — function
			// returns an error. NO GetByReferenceID, NO messageHandler.Create,
			// NO startPipecatcall.
			name: "error: missing conversation_message.text variable — aborts",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0b222222-0000-11f0-bbbb-000000000001"),
					CustomerID: uuid.FromStringOrNil("0b222222-0000-11f0-bbbb-000000000002"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("0b222222-0000-11f0-bbbb-000000000001"),
			activeflowID:   uuid.FromStringOrNil("0b222222-0000-11f0-bbbb-000000000003"),
			referenceID:    uuid.FromStringOrNil("0b222222-0000-11f0-bbbb-000000000004"),

			mockSetup: func(ctx context.Context, m *mocks) {
				// FlowV1VariableGet returns Variable without the expected key.
				// NO downstream calls.
				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"some.other.variable": "value",
					},
				}, nil)
			},
			expectErr:          true,
			expectErrSubstring: "could not get the conversation message text from the activeflow variables",
		},
		{
			// Existing AIcall is in StatusTerminating — isAIcallReusable returns
			// false, so the fresh path runs. The idle-expiry branch is short-
			// circuited because Status == StatusTerminating, so NO
			// interruptPreviousPipecatcall, NO idle-expired counter increment,
			// NO UpdateStatus(StatusTerminated) on the old AIcall.
			// startAIcallByMessaging runs and UpdateStatus(StatusProgressing)
			// runs on the freshly created AIcall.
			name: "fresh: existing AIcall is StatusTerminating — recreate (no interrupt, no idle-expiry)",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000001"),
					CustomerID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000002"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000001"),
			activeflowID:   uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000003"),
			referenceID:    uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000004"),

			mockSetup: func(ctx context.Context, m *mocks) {
				oldAIcallID := uuid.FromStringOrNil("0c444444-0000-11f0-cccc-000000000001")
				oldPCC := uuid.FromStringOrNil("0c444444-0000-11f0-cccc-000000000099")
				newPCC := uuid.FromStringOrNil("0c555555-0000-11f0-cccc-000000000001")
				newAIcallID := uuid.FromStringOrNil("0c555555-0000-11f0-cccc-000000000002")

				existingTerminating := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         oldAIcallID,
						CustomerID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000002"),
					},
					Status:        aicall.StatusTerminating,
					TMUpdate:      &freshTM,
					PipecatcallID: oldPCC,
				}
				createdAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000002"),
					},
					ActiveflowID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000003"),
					Status:       aicall.StatusInitiating,
				}
				progressingAIcall := &aicall.AIcall{
					Identity: commonidentity.Identity{
						ID:         newAIcallID,
						CustomerID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000002"),
					},
					ActiveflowID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000003"),
					Status:       aicall.StatusProgressing,
				}
				responsePC := &pmpipecatcall.Pipecatcall{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("0c666666-0000-11f0-cccc-000000000001"),
					},
				}

				m.req.EXPECT().FlowV1VariableGet(ctx, gomock.Any()).Return(&fmvariable.Variable{
					Variables: map[string]string{
						"voipbin.conversation_message.text": "post-terminating user message.",
					},
				}, nil)

				// GetByReferenceID returns the StatusTerminating AIcall.
				m.db.EXPECT().AIcallGetByReferenceID(ctx, gomock.Any()).Return(existingTerminating, nil)

				// NO interruptPreviousPipecatcall (only happens on reuse path).
				// NO idle-expiry branch — short-circuited by Status == StatusTerminating.
				// NO UpdateStatus(StatusTerminated) on the old AIcall.

				// startAIcallByMessaging on the fresh path
				m.util.EXPECT().UUIDCreate().Return(newPCC)
				m.util.EXPECT().UUIDCreate().Return(newAIcallID)
				m.db.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(createdAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil)
				m.message.EXPECT().Create(ctx, uuid.Nil, createdAIcall.CustomerID, createdAIcall.ID, createdAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, gomock.Any(), nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// UpdateStatus -> Progressing on the new AIcall
				m.db.EXPECT().AIcallUpdate(ctx, newAIcallID, gomock.Any()).Return(nil)
				m.db.EXPECT().AIcallGet(ctx, newAIcallID).Return(progressingAIcall, nil)
				m.notify.EXPECT().PublishWebhookEvent(ctx, progressingAIcall.CustomerID, aicall.EventTypeStatusProgressing, progressingAIcall)

				// conversation message create
				m.message.EXPECT().Create(ctx, uuid.Nil, progressingAIcall.CustomerID, progressingAIcall.ID, progressingAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleUser, "post-terminating user message.", nil, "", gomock.Any()).Return(&message.Message{}, nil)

				// startPipecatcall
				m.message.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return([]*message.Message{}, nil)
				m.req.EXPECT().PipecatV1PipecatcallStart(
					ctx,
					progressingAIcall.PipecatcallID,
					progressingAIcall.CustomerID,
					progressingAIcall.ActiveflowID,
					pmpipecatcall.ReferenceTypeAICall,
					progressingAIcall.ID,
					gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(responsePC, nil)
				m.req.EXPECT().PipecatV1PipecatcallTerminateWithDelay(ctx, responsePC.HostID, responsePC.ID, defaultAITaskTimeout).Return(nil)
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("0c555555-0000-11f0-cccc-000000000002"),
					CustomerID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000002"),
				},
				ActiveflowID: uuid.FromStringOrNil("0c333333-0000-11f0-cccc-000000000003"),
				Status:       aicall.StatusProgressing,
			},
			expectIdleExpiredInc: false, // status == Terminating short-circuits idle-expiry
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			m := &mocks{
				util:    utilhandler.NewMockUtilHandler(mc),
				req:     requesthandler.NewMockRequestHandler(mc),
				notify:  notifyhandler.NewMockNotifyHandler(mc),
				db:      dbhandler.NewMockDBHandler(mc),
				ai:      aihandler.NewMockAIHandler(mc),
				team:    teamhandler.NewMockTeamHandler(mc),
				message: messagehandler.NewMockMessageHandler(mc),
			}

			h := &aicallHandler{
				utilHandler:    m.util,
				reqHandler:     m.req,
				notifyHandler:  m.notify,
				db:             m.db,
				aiHandler:      m.ai,
				teamHandler:    m.team,
				messageHandler: m.message,
			}
			ctx := context.Background()

			tt.mockSetup(ctx, m)

			// Snapshot the idle-expired counter only for the sub-case that
			// triggers it. Other sub-cases use t.Parallel(), so a strict
			// "no change" assertion would race against the idle-expired
			// sub-test's Inc(). Since promAIcallIdleExpiredTotal is uniquely
			// incremented by the idle-expired branch, we assert delta >= 1
			// for the expecting sub-case and skip the assertion otherwise.
			var beforeIdleExpired float64
			if tt.expectIdleExpiredInc {
				beforeIdleExpired = testutil.ToFloat64(promAIcallIdleExpiredTotal)
			}

			res, err := h.startReferenceTypeConversation(ctx, tt.ai, tt.assistanceType, tt.assistanceID, tt.activeflowID, tt.referenceID, nil, uuid.Nil)
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.expectErrSubstring != "" && !strings.Contains(err.Error(), tt.expectErrSubstring) {
					t.Errorf("expected error containing %q, got: %v", tt.expectErrSubstring, err)
				}
				if res != nil {
					t.Errorf("expected nil res on error, got: %v", res)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}

			if tt.expectIdleExpiredInc {
				afterIdleExpired := testutil.ToFloat64(promAIcallIdleExpiredTotal)
				delta := afterIdleExpired - beforeIdleExpired
				if delta < 1 {
					t.Errorf("expected idle-expired counter to increment by at least 1, got delta=%f", delta)
				}
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
		{
			name: "filters out notification role messages",

			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6754cd08-b62a-11f0-b4a3-13e4f1d01c60"),
				},
			},

			responseMessages: []*message.Message{
				{
					Role:    message.RoleAssistant,
					Content: "Hello, how can I help?",
				},
				{
					Role:    message.RoleNotification,
					Content: `{"type":"member_switched","from_member":{"name":"Reception"},"to_member":{"name":"Sales"}}`,
				},
				{
					Role:    message.RoleUser,
					Content: "I need help with billing.",
				},
			},
			expectRes: []map[string]any{
				{
					"role":    "user",
					"content": "I need help with billing.",
				},
				{
					"role":    "assistant",
					"content": "Hello, how can I help?",
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

			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)

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
				AIEngineModel: "openai.gpt-5-nano",
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "fa5c67b8-b62b-11f0-aab5-630972321af9",
				AISTTType:     ai.STTTypeDeepgram,
				STTLanguage:   "en-US",
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

			expectLLMType: pmpipecatcall.LLMType("openai.gpt-5-nano"),
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

			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)
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
				tt.aicall.STTLanguage,
				tt.expectTTSType,
				"",
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

func Test_startAIcallByRealtime(t *testing.T) {
	tests := []struct {
		name string

		ai              *ai.AI
		assistanceType  aicall.AssistanceType
		assistanceID    uuid.UUID
		activeflowID    uuid.UUID
		referenceType   aicall.ReferenceType
		referenceID     uuid.UUID
		confbridgeID    uuid.UUID

		isTask          bool
		teamParameter   map[string]any
		currentMemberID uuid.UUID

		// teamMembers + teamMemberAIs drive the team mock setup for
		// AssistanceTypeTeam cases (used by buildPromptSnapshots -> resolveAIForTeam).
		teamMembers   []team.Member
		teamMemberAIs map[uuid.UUID]*ai.AI

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
				InitPrompt:  "You are a helpful assistant.",
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
			activeflowID:   uuid.FromStringOrNil("b34140c8-b659-11f0-be3a-5fc8a6759b80"),
			referenceType:  aicall.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("b3662e38-b659-11f0-820a-833195d45f7e"),
			confbridgeID:   uuid.FromStringOrNil("b3864e5c-b659-11f0-ab17-6b281e446482"),

			isTask:         false,

			responseUUIDPipecatcallID: uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				ActiveflowID:  uuid.FromStringOrNil("b34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b3662e38-b659-11f0-820a-833195d45f7e"),
				ConfbridgeID:  uuid.FromStringOrNil("b3864e5c-b659-11f0-ab17-6b281e446482"),
				PipecatcallID: uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,

				STTLanguage: "en-US",

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID:   uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
							Prompt: "You are a helpful assistant.",
						},
					},
				},
			},
			expectVariables: map[string]string{
				"voipbin.aicall.ai_engine_model": "",
				"voipbin.aicall.ai_id":           "b30ecf94-b659-11f0-b8ef-13f90dff9ee8",
				"voipbin.aicall.confbridge_id":   "b3864e5c-b659-11f0-ab17-6b281e446482",

				"voipbin.aicall.id":              "b3af613e-b659-11f0-9a72-e3e004fae386",
				"voipbin.aicall.stt_language":        "en-US",
				"voipbin.aicall.pipecatcall_id":  "b3af613e-b659-11f0-9a72-e3e004fae386",
			},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				"You are a helpful assistant.",
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				ActiveflowID:  uuid.FromStringOrNil("b34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b3662e38-b659-11f0-820a-833195d45f7e"),
				ConfbridgeID:  uuid.FromStringOrNil("b3864e5c-b659-11f0-ab17-6b281e446482"),
				PipecatcallID: uuid.FromStringOrNil("b3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,

				STTLanguage: "en-US",

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID:   uuid.FromStringOrNil("b30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
							Prompt: "You are a helpful assistant.",
						},
					},
				},
			},
		},
		{
			name: "merge ai and team parameters with key collision - team overrides",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
					CustomerID: uuid.FromStringOrNil("c9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				Parameter: map[string]any{
					"shared_key": "ai_value",
					"ai_only":    "ai_data",
				},
				STTLanguage: "ko-KR",
			},
			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("c40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
			activeflowID:   uuid.FromStringOrNil("c34140c8-b659-11f0-be3a-5fc8a6759b80"),
			referenceType:  aicall.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("c3662e38-b659-11f0-820a-833195d45f7e"),
			confbridgeID:   uuid.FromStringOrNil("c3864e5c-b659-11f0-ab17-6b281e446482"),

			isTask:         false,
			teamParameter: map[string]any{
				"shared_key": "team_value",
				"team_only":  "team_data",
			},

			teamMembers: []team.Member{
				{
					ID:   uuid.FromStringOrNil("c5111111-b659-11f0-b8ef-13f90dff9ee8"),
					AIID: uuid.FromStringOrNil("c5222222-b659-11f0-b8ef-13f90dff9ee8"),
				},
			},
			teamMemberAIs: map[uuid.UUID]*ai.AI{
				uuid.FromStringOrNil("c5222222-b659-11f0-b8ef-13f90dff9ee8"): {
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("c5222222-b659-11f0-b8ef-13f90dff9ee8"),
						CustomerID: uuid.FromStringOrNil("c9be93b6-b659-11f0-b961-b32ce4769d7c"),
					},
				},
			},

			responseUUIDPipecatcallID: uuid.FromStringOrNil("c3af613e-b659-11f0-9a72-e3e004fae386"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("c3af613e-b659-11f0-9a72-e3e004fae386"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("c9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeTeam,
				AssistanceID:   uuid.FromStringOrNil("c40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
				Parameter: map[string]any{
					"shared_key": "team_value",
					"ai_only":    "ai_data",
					"team_only":  "team_data",
				},

				ActiveflowID:  uuid.FromStringOrNil("c34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c3662e38-b659-11f0-820a-833195d45f7e"),
				ConfbridgeID:  uuid.FromStringOrNil("c3864e5c-b659-11f0-ab17-6b281e446482"),
				PipecatcallID: uuid.FromStringOrNil("c3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,

				STTLanguage: "ko-KR",

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID:     uuid.FromStringOrNil("c5222222-b659-11f0-b8ef-13f90dff9ee8"),
							MemberID: uuid.FromStringOrNil("c5111111-b659-11f0-b8ef-13f90dff9ee8"),
						},
					},
				},
			},
			expectVariables: map[string]string{
				"voipbin.aicall.ai_engine_model": "",
				"voipbin.aicall.ai_id":           "c40ecf94-b659-11f0-b8ef-13f90dff9ee8",
				"voipbin.aicall.confbridge_id":   "c3864e5c-b659-11f0-ab17-6b281e446482",

				"voipbin.aicall.id":              "c3af613e-b659-11f0-9a72-e3e004fae386",
				"voipbin.aicall.stt_language":        "ko-KR",
				"voipbin.aicall.pipecatcall_id":  "c3af613e-b659-11f0-9a72-e3e004fae386",
			},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				`{"ai_only":"ai_data","shared_key":"team_value","team_only":"team_data"}`,
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("c9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeTeam,
				AssistanceID:   uuid.FromStringOrNil("c40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
				Parameter: map[string]any{
					"shared_key": "team_value",
					"ai_only":    "ai_data",
					"team_only":  "team_data",
				},

				ActiveflowID:  uuid.FromStringOrNil("c34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("c3662e38-b659-11f0-820a-833195d45f7e"),
				ConfbridgeID:  uuid.FromStringOrNil("c3864e5c-b659-11f0-ab17-6b281e446482"),
				PipecatcallID: uuid.FromStringOrNil("c3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,

				STTLanguage: "ko-KR",

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID:     uuid.FromStringOrNil("c5222222-b659-11f0-b8ef-13f90dff9ee8"),
							MemberID: uuid.FromStringOrNil("c5111111-b659-11f0-b8ef-13f90dff9ee8"),
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
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				teamHandler:    mockTeam,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			// buildPromptSnapshots -> resolveAIForTeam for team-typed calls
			if tt.assistanceType == aicall.AssistanceTypeTeam {
				mockTeam.EXPECT().Get(ctx, tt.assistanceID).Return(&team.Team{
					Members: tt.teamMembers,
				}, nil)
				for _, m := range tt.teamMembers {
					mockAI.EXPECT().Get(ctx, m.AIID).Return(tt.teamMemberAIs[m.AIID], nil)
				}
			}

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.expectAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.expectAIcall)

			// setActiveflowVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.expectAIcall.ActiveflowID, tt.expectVariables).Return(nil)

			// startInitMessages + buildPromptSnapshots both substitute the AI init prompt
			if tt.ai.InitPrompt != "" {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.ai.InitPrompt).Return(tt.ai.InitPrompt, nil).Times(2)
			}
			// parameter substitution
			if tt.expectAIcall.Parameter != nil {
				paramCount := 0
				for _, v := range tt.expectAIcall.Parameter {
					if _, ok := v.(string); ok {
						paramCount++
					}
				}
				if paramCount > 0 {
					mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, gomock.Any()).
						DoAndReturn(func(_ context.Context, _ uuid.UUID, input string) (string, error) {
							return input, nil
						}).Times(paramCount)
				}
			}
			for _, m := range tt.expectMessageTexts {
				mockMessage.EXPECT().Create(ctx, uuid.Nil, tt.expectAIcall.CustomerID, tt.expectAIcall.ID, tt.expectAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, m, nil, "", gomock.Any()).Return(&message.Message{}, nil)
			}

			res, err := h.startAIcallByRealtime(ctx, tt.ai, tt.assistanceType, tt.assistanceID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeID, tt.isTask, tt.teamParameter, tt.currentMemberID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startAIcallByMessaging(t *testing.T) {
	tests := []struct {
		name string

		ai              *ai.AI
		assistanceType  aicall.AssistanceType
		assistanceID    uuid.UUID
		activeflowID    uuid.UUID
		referenceType   aicall.ReferenceType
		referenceID     uuid.UUID

		isTask          bool
		teamParameter   map[string]any
		currentMemberID uuid.UUID

		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcallID      uuid.UUID

		expectAIcall       *aicall.AIcall
		expectVariables    map[string]string
		expectMessageTexts []string

		expectRes *aicall.AIcall
	}{
		{
			name: "normal - messaging path does not set TTS/STT/VAD",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
					CustomerID: uuid.FromStringOrNil("d9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				InitPrompt:  "You are a helpful assistant.",
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("d30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
			activeflowID:   uuid.FromStringOrNil("d34140c8-b659-11f0-be3a-5fc8a6759b80"),
			referenceType:  aicall.ReferenceTypeConversation,
			referenceID:    uuid.FromStringOrNil("d3662e38-b659-11f0-820a-833195d45f7e"),

			isTask:         false,

			responseUUIDPipecatcallID: uuid.FromStringOrNil("d3af613e-b659-11f0-9a72-e3e004fae386"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("d3af613e-b659-11f0-9a72-e3e004fae386"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("d9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("d30ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				AIEngineModel: ai.EngineModelOpenaiGPT5,

				ActiveflowID:  uuid.FromStringOrNil("d34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   uuid.FromStringOrNil("d3662e38-b659-11f0-820a-833195d45f7e"),
				PipecatcallID: uuid.FromStringOrNil("d3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,

				STTLanguage: "en-US",

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID:   uuid.FromStringOrNil("d30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
							Prompt: "You are a helpful assistant.",
						},
					},
				},
			},
			expectVariables: map[string]string{
				"voipbin.aicall.ai_engine_model": string(ai.EngineModelOpenaiGPT5),
				"voipbin.aicall.ai_id":           "d30ecf94-b659-11f0-b8ef-13f90dff9ee8",
				"voipbin.aicall.confbridge_id":   uuid.Nil.String(),

				"voipbin.aicall.id":              "d3af613e-b659-11f0-9a72-e3e004fae386",
				"voipbin.aicall.stt_language":        "en-US",
				"voipbin.aicall.pipecatcall_id":  "d3af613e-b659-11f0-9a72-e3e004fae386",
			},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				"You are a helpful assistant.",
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("d9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("d30ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				AIEngineModel: ai.EngineModelOpenaiGPT5,

				ActiveflowID:  uuid.FromStringOrNil("d34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   uuid.FromStringOrNil("d3662e38-b659-11f0-820a-833195d45f7e"),
				PipecatcallID: uuid.FromStringOrNil("d3af613e-b659-11f0-9a72-e3e004fae386"),

				Status:   aicall.StatusInitiating,

				STTLanguage: "en-US",

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID:   uuid.FromStringOrNil("d30ecf94-b659-11f0-b8ef-13f90dff9ee8"),
							Prompt: "You are a helpful assistant.",
						},
					},
				},
			},
		},
		{
			name: "with participant handler",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f10ecf94-b659-11f0-b8ef-13f90dff9ee8"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "21m00Tcm4TlvDq8ikWAM",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("f10ecf94-b659-11f0-b8ef-13f90dff9ee8"),
			activeflowID:   uuid.FromStringOrNil("f34140c8-b659-11f0-be3a-5fc8a6759b80"),
			referenceType:  aicall.ReferenceTypeConversation,
			referenceID:    uuid.FromStringOrNil("f3662e38-b659-11f0-820a-833195d45f7e"),

			isTask: false,

			responseUUIDPipecatcallID: uuid.FromStringOrNil("f3af613e-b659-11f0-9a72-e3e004fae386"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("f3af613e-b659-11f0-9a72-e3e004fae386"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("f10ecf94-b659-11f0-b8ef-13f90dff9ee8"),
				AIEngineModel:  ai.EngineModelOpenaiGPT5,
				ActiveflowID:   uuid.FromStringOrNil("f34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("f3662e38-b659-11f0-820a-833195d45f7e"),
				PipecatcallID:  uuid.FromStringOrNil("f3af613e-b659-11f0-9a72-e3e004fae386"),
				Status:         aicall.StatusInitiating,
				STTLanguage:    "en-US",
				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID: uuid.FromStringOrNil("f10ecf94-b659-11f0-b8ef-13f90dff9ee8"),
						},
					},
				},
			},
			expectVariables: map[string]string{
				"voipbin.aicall.ai_engine_model": string(ai.EngineModelOpenaiGPT5),
				"voipbin.aicall.ai_id":           "f10ecf94-b659-11f0-b8ef-13f90dff9ee8",
				"voipbin.aicall.confbridge_id":   uuid.Nil.String(),
				"voipbin.aicall.id":              "f3af613e-b659-11f0-9a72-e3e004fae386",
				"voipbin.aicall.stt_language":        "en-US",
				"voipbin.aicall.pipecatcall_id":  "f3af613e-b659-11f0-9a72-e3e004fae386",
			},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("f9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("f10ecf94-b659-11f0-b8ef-13f90dff9ee8"),
				AIEngineModel:  ai.EngineModelOpenaiGPT5,
				ActiveflowID:   uuid.FromStringOrNil("f34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType:  aicall.ReferenceTypeConversation,
				ReferenceID:    uuid.FromStringOrNil("f3662e38-b659-11f0-820a-833195d45f7e"),
				PipecatcallID:  uuid.FromStringOrNil("f3af613e-b659-11f0-9a72-e3e004fae386"),
				Status:         aicall.StatusInitiating,
				STTLanguage:    "en-US",
				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID: uuid.FromStringOrNil("f10ecf94-b659-11f0-b8ef-13f90dff9ee8"),
						},
					},
				},
			},
		},
		{
			name: "messaging path with task flag",

			ai: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
					CustomerID: uuid.FromStringOrNil("e9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				TTSType:     ai.TTSTypeCartesia,
				STTType:     ai.STTTypeDeepgram,
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("e40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
			activeflowID:   uuid.FromStringOrNil("e34140c8-b659-11f0-be3a-5fc8a6759b80"),
			referenceType:  aicall.ReferenceTypeTask,
			referenceID:    uuid.Nil,

			isTask:         true,

			responseUUIDPipecatcallID: uuid.FromStringOrNil("e3af613e-b659-11f0-9a72-e3e004fae386"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("e3af613e-b659-11f0-9a72-e3e004fae386"),

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("e9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("e40ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				AIEngineModel: ai.EngineModelOpenaiGPT5,

				ActiveflowID:  uuid.FromStringOrNil("e34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeTask,
				PipecatcallID: uuid.FromStringOrNil("e3af613e-b659-11f0-9a72-e3e004fae386"),

				Status: aicall.StatusInitiating,

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID: uuid.FromStringOrNil("e40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
						},
					},
				},
			},
			expectVariables: map[string]string{
				"voipbin.aicall.ai_engine_model": string(ai.EngineModelOpenaiGPT5),
				"voipbin.aicall.ai_id":           "e40ecf94-b659-11f0-b8ef-13f90dff9ee8",
				"voipbin.aicall.confbridge_id":   uuid.Nil.String(),

				"voipbin.aicall.id":              "e3af613e-b659-11f0-9a72-e3e004fae386",
				"voipbin.aicall.stt_language":        "",
				"voipbin.aicall.pipecatcall_id":  "e3af613e-b659-11f0-9a72-e3e004fae386",
			},
			expectMessageTexts: []string{
				defaultCommonAItaskSystemPrompt,
			},
			expectRes: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e3af613e-b659-11f0-9a72-e3e004fae386"),
					CustomerID: uuid.FromStringOrNil("e9be93b6-b659-11f0-b961-b32ce4769d7c"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("e40ecf94-b659-11f0-b8ef-13f90dff9ee8"),

				AIEngineModel: ai.EngineModelOpenaiGPT5,

				ActiveflowID:  uuid.FromStringOrNil("e34140c8-b659-11f0-be3a-5fc8a6759b80"),
				ReferenceType: aicall.ReferenceTypeTask,
				PipecatcallID: uuid.FromStringOrNil("e3af613e-b659-11f0-9a72-e3e004fae386"),

				Status: aicall.StatusInitiating,

				Metadata: map[string]any{
					aicall.MetaKeyPromptSnapshots: []aicall.PromptSnapshot{
						{
							AIID: uuid.FromStringOrNil("e40ecf94-b659-11f0-b8ef-13f90dff9ee8"),
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

			// startInitMessages + buildPromptSnapshots both substitute the init prompt
			if tt.ai.InitPrompt != "" {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, tt.ai.InitPrompt).Return(tt.ai.InitPrompt, nil).Times(2)
			}
			for _, m := range tt.expectMessageTexts {
				mockMessage.EXPECT().Create(ctx, uuid.Nil, tt.expectAIcall.CustomerID, tt.expectAIcall.ID, tt.expectAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, m, nil, "", gomock.Any()).Return(&message.Message{}, nil)
			}

			if tt.name == "with participant handler" {
				mockParticipant := participanthandler.NewMockParticipantHandler(mc)
				mockParticipant.EXPECT().Create(gomock.Any(), gomock.Any(), tt.ai.ID).Return(nil).Times(1)
				h.participantHandler = mockParticipant
			}

			res, err := h.startAIcallByMessaging(ctx, tt.ai, tt.assistanceType, tt.assistanceID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.isTask, tt.teamParameter, tt.currentMemberID)
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
		isTask bool

		responseSubstitutes []string

		expectMessageTexts []string
	}{
		{
			name: "has all",

			ai: &ai.AI{
				InitPrompt: "You are a helpful assistant.",
			},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("79e1da30-c055-11f0-9ca3-6ff0383cb80c"),
				},
				ActiveflowID: uuid.FromStringOrNil("7a0651d0-c055-11f0-a70e-8fe16492a013"),
				Parameter: map[string]any{
					"initial_system_prompt": "Bruce Wayne is Batman.",
				},
			},
			isTask: false,

			responseSubstitutes: []string{
				"You are a super helpful assistant.",
				"Bruce Wayne is Batman.",
			},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				"You are a super helpful assistant.",
				`{"initial_system_prompt":"Bruce Wayne is Batman."}`,
			},
		},
		{
			name: "has all and isTask true ",

			ai: &ai.AI{
				InitPrompt: "You are a helpful assistant.",
			},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("79c5557e-d49e-11f0-813f-fb8e8df71bb6"),
				},
				ActiveflowID: uuid.FromStringOrNil("79f79174-d49e-11f0-96e7-430b2e8fc74c"),
				Parameter: map[string]any{
					"initial_system_prompt": "Bruce Wayne is Batman.",
				},
			},
			isTask: true,

			responseSubstitutes: []string{
				"You are a super helpful assistant.",
				"Bruce Wayne is Batman.",
			},
			expectMessageTexts: []string{
				defaultCommonAItaskSystemPrompt,
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
			isTask: false,

			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
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
			isTask: false,

			responseSubstitutes: []string{"You are a super helpful assistant."},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				"You are a super helpful assistant.",
			},
		},
		{
			name: "has parameter",

			ai: &ai.AI{},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),
				},
				ActiveflowID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
				Parameter:    map[string]any{"greeting": "hello"},
			},
			isTask: false,

			responseSubstitutes: []string{"hello"},
			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
				`{"greeting":"hello"}`,
			},
		},
		{
			name: "empty map parameter is excluded from messages",

			ai: &ai.AI{},
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c3d4e5f6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
				},
				ActiveflowID: uuid.FromStringOrNil("d4e5f6a7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),
				Parameter:    map[string]any{},
			},
			isTask: false,

			expectMessageTexts: []string{
				defaultCommonAIcallSystemPrompt,
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
				mockMessage.EXPECT().Create(ctx, uuid.Nil, tt.aicall.CustomerID, tt.aicall.ID, tt.aicall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, m, nil, "", gomock.Any()).Return(&message.Message{}, nil)
			}

			if err := h.startInitMessages(ctx, tt.ai, tt.aicall, tt.isTask); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_StartTask(t *testing.T) {
	tests := []struct {
		name string

		assistanceType aicall.AssistanceType
		assistanceID   uuid.UUID
		activeflowID   uuid.UUID

		responseAI                *ai.AI
		responseUUIDPipecatcallID uuid.UUID
		responseUUIDAIcallID      uuid.UUID
		responseMessages          []*message.Message
		responsePipecatcall       *pmpipecatcall.Pipecatcall

		expectAIcall          *aicall.AIcall
		expectMessageContents []string
		expectLLMLessages     []map[string]any
	}{
		{
			name: "normal",

			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("c1d5f4e2-30de-11f0-8bbf-d773a2d31d73"),
			activeflowID:   uuid.FromStringOrNil("c1f4b4e4-30de-11f0-8bbf-d773a2d31d73"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c1d5f4e2-30de-11f0-8bbf-d773a2d31d73"),
					CustomerID: uuid.FromStringOrNil("da5752be-d798-11f0-9181-d3db413a82d0"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
				Parameter:  map[string]any{},

				InitPrompt: "",
			},
			responseUUIDPipecatcallID: uuid.FromStringOrNil("d9f74cd4-d798-11f0-9e0e-c3713b24ebdc"),
			responseUUIDAIcallID:      uuid.FromStringOrNil("da2a0df4-d798-11f0-a8c3-6348c9284ca2"),
			responseMessages: []*message.Message{
				{
					Role:    message.RoleSystem,
					Content: defaultCommonAItaskSystemPrompt,
				},
			},
			responsePipecatcall: &pmpipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d9f74cd4-d798-11f0-9e0e-c3713b24ebdc"),
				},
			},

			expectAIcall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("da2a0df4-d798-11f0-a8c3-6348c9284ca2"),
					CustomerID: uuid.FromStringOrNil("da5752be-d798-11f0-9181-d3db413a82d0"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("c1d5f4e2-30de-11f0-8bbf-d773a2d31d73"),
				ActiveflowID:   uuid.FromStringOrNil("c1f4b4e4-30de-11f0-8bbf-d773a2d31d73"),
				ReferenceType:  aicall.ReferenceTypeTask,
				ReferenceID:    uuid.Nil,
				PipecatcallID:  uuid.FromStringOrNil("d9f74cd4-d798-11f0-9e0e-c3713b24ebdc"),
				Status:         aicall.StatusInitiating,
			},
			expectMessageContents: []string{
				defaultCommonAItaskSystemPrompt,
			},
			expectLLMLessages: []map[string]any{
				{
					"role":    "system",
					"content": defaultCommonAItaskSystemPrompt,
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

			mockAI.EXPECT().Get(ctx, tt.assistanceID).Return(tt.responseAI, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDPipecatcallID)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAIcallID)
			mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDAIcallID).Return(tt.expectAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), aicall.EventTypeStatusInitializing, gomock.Any())

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.activeflowID, gomock.Any()).Return(nil)

			for _, m := range tt.expectMessageContents {
				mockMessage.EXPECT().Create(ctx, uuid.Nil, tt.expectAIcall.CustomerID, tt.expectAIcall.ID, tt.expectAIcall.ActiveflowID, message.DirectionOutgoing, message.RoleSystem, m, nil, "", gomock.Any()).Return(&message.Message{}, nil)
			}

			mockMessage.EXPECT().List(ctx, uint64(100), gomock.Any(), gomock.Any()).Return(tt.responseMessages, nil)
			mockReq.EXPECT().PipecatV1PipecatcallStart(
				ctx,
				tt.expectAIcall.PipecatcallID,
				tt.expectAIcall.CustomerID,
				tt.expectAIcall.ActiveflowID,
				pmpipecatcall.ReferenceTypeAICall,
				tt.expectAIcall.ID,
				pmpipecatcall.LLMType(tt.expectAIcall.AIEngineModel),
				tt.expectLLMLessages,
				pmpipecatcall.STTTypeNone,
				"",
				pmpipecatcall.TTSTypeNone,
				"",
				"",
			).Return(tt.responsePipecatcall, nil)

			mockReq.EXPECT().AIV1AIcallTerminateWithDelay(ctx, tt.expectAIcall.ID, defaultAITaskTimeout).Return(nil)

			res, err := h.StartTask(ctx, tt.assistanceType, tt.assistanceID, tt.activeflowID)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectAIcall) {
				t.Errorf("expected: %v, got: %v", tt.expectAIcall, res)
			}
		})
	}
}

func Test_resolveAI(t *testing.T) {
	tests := []struct {
		name string

		assistanceType aicall.AssistanceType
		assistanceID   uuid.UUID

		responseAI   *ai.AI
		responseTeam *team.Team
		errAI        error
		errTeam      error

		expectAI              *ai.AI
		expectTeamParameter   map[string]any
		expectCurrentMemberID uuid.UUID
		expectErr             bool
	}{
		{
			name: "AssistanceTypeAI returns ai and nil team parameter",

			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("d1a2b3c4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1a2b3c4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
				Parameter:   map[string]any{"key": "value"},
			},

			expectAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1a2b3c4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
				Parameter:   map[string]any{"key": "value"},
			},
			expectTeamParameter:   nil,
			expectCurrentMemberID: uuid.Nil,
			expectErr:             false,
		},
		{
			name: "AssistanceTypeTeam with parameter returns ai and team parameter",

			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("e2b3c4d5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),

			responseTeam: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e2b3c4d5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
				},
				StartMemberID: uuid.FromStringOrNil("f3c4d5e6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("f3c4d5e6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
						AIID: uuid.FromStringOrNil("a4d5e6f7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),
					},
				},
				Parameter: map[string]any{"team_key": "team_value"},
			},
			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4d5e6f7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
			},

			expectAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4d5e6f7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
			},
			expectTeamParameter:   map[string]any{"team_key": "team_value"},
			expectCurrentMemberID: uuid.FromStringOrNil("f3c4d5e6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
			expectErr:             false,
		},
		{
			name: "AssistanceTypeTeam start member not found returns error",

			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("b5e6f7a8-c9d0-11f0-e5f6-a7b8c9d0e1f2"),

			responseTeam: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5e6f7a8-c9d0-11f0-e5f6-a7b8c9d0e1f2"),
				},
				StartMemberID: uuid.FromStringOrNil("c6f7a8b9-d0e1-11f0-f6a7-b8c9d0e1f2a3"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("d7a8b9c0-e1f2-11f0-a7b8-c9d0e1f2a3b4"),
						AIID: uuid.FromStringOrNil("e8b9c0d1-f2a3-11f0-b8c9-d0e1f2a3b4c5"),
					},
				},
			},

			expectErr: true,
		},
		{
			name: "AssistanceTypeTeam team fetch fails returns error",

			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("f9c0d1e2-a3b4-11f0-c9d0-e1f2a3b4c5d6"),

			errTeam: fmt.Errorf("team not found"),

			expectErr: true,
		},
		{
			name: "AssistanceTypeAI ai fetch fails returns error",

			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("a0d1e2f3-b4c5-11f0-d0e1-f2a3b4c5d6e7"),

			errAI: fmt.Errorf("ai not found"),

			expectErr: true,
		},
		{
			name: "AssistanceTypeTeam empty members returns error",

			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("d1e2f3a4-b5c6-11f0-e1f2-a3b4c5d6e7f8"),

			responseTeam: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d1e2f3a4-b5c6-11f0-e1f2-a3b4c5d6e7f8"),
				},
				StartMemberID: uuid.FromStringOrNil("e2f3a4b5-c6d7-11f0-f2a3-b4c5d6e7f8a9"),
				Members:       []team.Member{},
			},

			expectErr: true,
		},
		{
			name: "AssistanceTypeTeam with nil parameter returns ai and nil team parameter",

			assistanceType: aicall.AssistanceTypeTeam,
			assistanceID:   uuid.FromStringOrNil("f3a4b5c6-d7e8-11f0-a3b4-c5d6e7f8a9b0"),

			responseTeam: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f3a4b5c6-d7e8-11f0-a3b4-c5d6e7f8a9b0"),
				},
				StartMemberID: uuid.FromStringOrNil("a4b5c6d7-e8f9-11f0-b4c5-d6e7f8a9b0c1"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("a4b5c6d7-e8f9-11f0-b4c5-d6e7f8a9b0c1"),
						AIID: uuid.FromStringOrNil("b5c6d7e8-f9a0-11f0-c5d6-e7f8a9b0c1d2"),
					},
				},
				Parameter: nil,
			},
			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5c6d7e8-f9a0-11f0-c5d6-e7f8a9b0c1d2"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},

			expectAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b5c6d7e8-f9a0-11f0-c5d6-e7f8a9b0c1d2"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			expectTeamParameter:   nil,
			expectCurrentMemberID: uuid.FromStringOrNil("a4b5c6d7-e8f9-11f0-b4c5-d6e7f8a9b0c1"),
			expectErr:             false,
		},
		{
			name: "unsupported assistance type returns error",

			assistanceType: aicall.AssistanceType("unknown"),
			assistanceID:   uuid.FromStringOrNil("c6d7e8f9-a0b1-11f0-d6e7-f8a9b0c1d2e3"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAI := aihandler.NewMockAIHandler(mc)
			mockTeam := teamhandler.NewMockTeamHandler(mc)

			h := &aicallHandler{
				aiHandler:   mockAI,
				teamHandler: mockTeam,
			}
			ctx := context.Background()

			switch tt.assistanceType {
			case aicall.AssistanceTypeAI:
				mockAI.EXPECT().Get(ctx, tt.assistanceID).Return(tt.responseAI, tt.errAI)

			case aicall.AssistanceTypeTeam:
				mockTeam.EXPECT().Get(ctx, tt.assistanceID).Return(tt.responseTeam, tt.errTeam)
				if tt.errTeam == nil && tt.responseTeam != nil {
					// check if start member exists to decide if AI Get will be called
					for _, m := range tt.responseTeam.Members {
						if m.ID == tt.responseTeam.StartMemberID {
							mockAI.EXPECT().Get(ctx, m.AIID).Return(tt.responseAI, tt.errAI)
							break
						}
					}
				}
			}

			resAI, resTeamParam, resCurrentMemberID, err := h.resolveAI(ctx, tt.assistanceType, tt.assistanceID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(resAI, tt.expectAI) {
				t.Errorf("expected AI: %v, got: %v", tt.expectAI, resAI)
			}

			if !reflect.DeepEqual(resTeamParam, tt.expectTeamParameter) {
				t.Errorf("expected team parameter: %v, got: %v", tt.expectTeamParameter, resTeamParam)
			}

			if resCurrentMemberID != tt.expectCurrentMemberID {
				t.Errorf("expected current member ID: %v, got: %v", tt.expectCurrentMemberID, resCurrentMemberID)
			}
		})
	}
}

func Test_resolveTeamMemberAI(t *testing.T) {
	tests := []struct {
		name string

		team     *team.Team
		memberID uuid.UUID

		responseAI *ai.AI
		errAI      error

		expectAI       *ai.AI
		expectMemberID uuid.UUID
		expectErr      bool
	}{
		{
			name: "member_found",

			team: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),
				},
				StartMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
						AIID: uuid.FromStringOrNil("c3d4e5f6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
					},
					{
						ID:   uuid.FromStringOrNil("d4e5f6a7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),
						AIID: uuid.FromStringOrNil("e5f6a7b8-c9d0-11f0-e5f6-a7b8c9d0e1f2"),
					},
				},
			},
			memberID: uuid.FromStringOrNil("d4e5f6a7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e5f6a7b8-c9d0-11f0-e5f6-a7b8c9d0e1f2"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
			},

			expectAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e5f6a7b8-c9d0-11f0-e5f6-a7b8c9d0e1f2"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5Dot1,
			},
			expectMemberID: uuid.FromStringOrNil("d4e5f6a7-b8c9-11f0-d4e5-f6a7b8c9d0e1"),
			expectErr:      false,
		},
		{
			name: "member_not_found_fallback_to_start_member",

			team: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),
				},
				StartMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
						AIID: uuid.FromStringOrNil("c3d4e5f6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
					},
				},
			},
			memberID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),

			responseAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c3d4e5f6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},

			expectAI: &ai.AI{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c3d4e5f6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
			},
			expectMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
			expectErr:      false,
		},
		{
			name: "neither_found",

			team: &team.Team{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11f0-a1b2-c3d4e5f6a7b8"),
				},
				StartMemberID: uuid.FromStringOrNil("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"),
				Members: []team.Member{
					{
						ID:   uuid.FromStringOrNil("b2c3d4e5-f6a7-11f0-b2c3-d4e5f6a7b8c9"),
						AIID: uuid.FromStringOrNil("c3d4e5f6-a7b8-11f0-c3d4-e5f6a7b8c9d0"),
					},
				},
			},
			memberID: uuid.FromStringOrNil("ffffffff-ffff-ffff-ffff-ffffffffffff"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				aiHandler: mockAI,
			}
			ctx := context.Background()

			if !tt.expectErr || tt.errAI != nil {
				// determine which member will be looked up
				for _, m := range tt.team.Members {
					if m.ID == tt.memberID {
						mockAI.EXPECT().Get(ctx, m.AIID).Return(tt.responseAI, tt.errAI)
						break
					}
				}
				// if memberID was not found, the fallback to start member will call Get
				found := false
				for _, m := range tt.team.Members {
					if m.ID == tt.memberID {
						found = true
						break
					}
				}
				if !found {
					for _, m := range tt.team.Members {
						if m.ID == tt.team.StartMemberID {
							mockAI.EXPECT().Get(ctx, m.AIID).Return(tt.responseAI, tt.errAI)
							break
						}
					}
				}
			}

			resAI, resMemberID, err := h.resolveTeamMemberAI(ctx, tt.team, tt.memberID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(resAI, tt.expectAI) {
				t.Errorf("expected AI: %v, got: %v", tt.expectAI, resAI)
			}

			if resMemberID != tt.expectMemberID {
				t.Errorf("expected member ID: %v, got: %v", tt.expectMemberID, resMemberID)
			}
		})
	}
}

func Test_mergeParameters(t *testing.T) {
	tests := []struct {
		name string

		aiParam   map[string]any
		teamParam map[string]any

		expectRes map[string]any
	}{
		{
			name:      "both_nil",
			aiParam:   nil,
			teamParam: nil,
			expectRes: nil,
		},
		{
			name:      "ai_only",
			aiParam:   map[string]any{"key1": "val1"},
			teamParam: nil,
			expectRes: map[string]any{"key1": "val1"},
		},
		{
			name:      "team_only",
			aiParam:   nil,
			teamParam: map[string]any{"key2": "val2"},
			expectRes: map[string]any{"key2": "val2"},
		},
		{
			name:      "both_no_overlap",
			aiParam:   map[string]any{"key1": "val1"},
			teamParam: map[string]any{"key2": "val2"},
			expectRes: map[string]any{"key1": "val1", "key2": "val2"},
		},
		{
			name:      "team_overrides_ai_on_collision",
			aiParam:   map[string]any{"key1": "ai_val"},
			teamParam: map[string]any{"key1": "team_val"},
			expectRes: map[string]any{"key1": "team_val"},
		},
		{
			name:      "both_empty_maps",
			aiParam:   map[string]any{},
			teamParam: map[string]any{},
			expectRes: nil,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			res := mergeParameters(tt.aiParam, tt.teamParam)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_resolveAIForTeam(t *testing.T) {
	teamID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	member1ID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")
	member2ID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")
	ai1ID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")
	ai2ID := uuid.FromStringOrNil("55555555-0000-0000-0000-000000000005")

	tests := []struct {
		name string

		teamID uuid.UUID

		mockSetup func(th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler)

		expectCount int
		expectKeys  []uuid.UUID
		expectErr   bool
	}{
		{
			name:   "both_members_succeed_returns_full_map",
			teamID: teamID,
			mockSetup: func(th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(&team.Team{
					Members: []team.Member{
						{ID: member1ID, AIID: ai1ID},
						{ID: member2ID, AIID: ai2ID},
					},
				}, nil)
				ah.EXPECT().Get(gomock.Any(), ai1ID).Return(&ai.AI{
					Identity: commonidentity.Identity{ID: ai1ID},
				}, nil)
				ah.EXPECT().Get(gomock.Any(), ai2ID).Return(&ai.AI{
					Identity: commonidentity.Identity{ID: ai2ID},
				}, nil)
			},
			expectCount: 2,
			expectKeys:  []uuid.UUID{member1ID, member2ID},
			expectErr:   false,
		},
		{
			name:   "one_member_ai_fetch_fails_returns_partial_map",
			teamID: teamID,
			mockSetup: func(th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(&team.Team{
					Members: []team.Member{
						{ID: member1ID, AIID: ai1ID},
						{ID: member2ID, AIID: ai2ID},
					},
				}, nil)
				ah.EXPECT().Get(gomock.Any(), ai1ID).Return(&ai.AI{
					Identity: commonidentity.Identity{ID: ai1ID},
				}, nil)
				ah.EXPECT().Get(gomock.Any(), ai2ID).Return(nil, errors.New("ai fetch failed"))
			},
			expectCount: 1,
			expectKeys:  []uuid.UUID{member1ID},
			expectErr:   false,
		},
		{
			name:   "teamhandler_get_fails_returns_error",
			teamID: teamID,
			mockSetup: func(th *teamhandler.MockTeamHandler, ah *aihandler.MockAIHandler) {
				th.EXPECT().Get(gomock.Any(), teamID).Return(nil, errors.New("team fetch failed"))
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockTeam := teamhandler.NewMockTeamHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			tt.mockSetup(mockTeam, mockAI)

			h := &aicallHandler{
				teamHandler: mockTeam,
				aiHandler:   mockAI,
			}
			ctx := context.Background()

			res, err := h.resolveAIForTeam(ctx, tt.teamID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if len(res) != tt.expectCount {
				t.Errorf("Wrong match. expect count: %d, got: %d", tt.expectCount, len(res))
			}
			for _, k := range tt.expectKeys {
				if _, ok := res[k]; !ok {
					t.Errorf("Wrong match. expected key %s not found in result map", k)
				}
			}
		})
	}
}
