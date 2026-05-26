package aicallhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		ai              *ai.AI
		assistanceType  aicall.AssistanceType
		assistanceID    uuid.UUID
		activeflowID    uuid.UUID
		referenceType   aicall.ReferenceType
		referenceID     uuid.UUID
		confbridgeID    uuid.UUID
		pipecatcallID   uuid.UUID
		currentMemberID uuid.UUID
		parameter       map[string]any

		responseUUIDID uuid.UUID
		responseAIcall *aicall.AIcall

		expectAIcall *aicall.AIcall
	}{
		{
			name: "have all",

			ai: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("81b311ee-a707-11ed-b499-f3284ac97a08"),
					CustomerID: uuid.FromStringOrNil("81880ddc-a707-11ed-be35-87b2fee31bb7"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				Parameter: map[string]any{
					"key1": "value1",
				},
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "d4c7e1f2-5e8b-4d3a-9f0a-1c2b3d4e5f60",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "en-US",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("81b311ee-a707-11ed-b499-f3284ac97a08"),
			activeflowID:   uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
			referenceType:  aicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
			confbridgeID:  uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
			pipecatcallID: uuid.FromStringOrNil("b063584e-b462-11f0-82f0-9b410ef3ab1e"),

			parameter: map[string]any{
				"key1": "value1",
			},

			responseUUIDID: uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
				},
			},

			expectAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
					CustomerID: uuid.FromStringOrNil("81880ddc-a707-11ed-be35-87b2fee31bb7"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("81b311ee-a707-11ed-b499-f3284ac97a08"),
				AIEngineModel:  ai.EngineModelOpenaiGPT5,
				Parameter: map[string]any{
					"key1": "value1",
				},
				AITTSType:     ai.TTSTypeElevenLabs,
				AITTSVoiceID:  "d4c7e1f2-5e8b-4d3a-9f0a-1c2b3d4e5f60",
				AISTTType:     ai.STTTypeDeepgram,
				ActiveflowID:  uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
				ConfbridgeID:  uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
				PipecatcallID: uuid.FromStringOrNil("b063584e-b462-11f0-82f0-9b410ef3ab1e"),

				STTLanguage:   "en-US",
				Status:        aicall.StatusInitiating,
			},
		},
		{
			name: "nil parameter",

			ai: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-11ed-a1b2-c3d4e5f6a7b8"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ed-b2c3-d4e5f6a7b8c9"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "voice-id-123",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "ja-JP",
			},
			assistanceType: aicall.AssistanceTypeAI,
			assistanceID:   uuid.FromStringOrNil("a1b2c3d4-e5f6-11ed-a1b2-c3d4e5f6a7b8"),
			activeflowID:   uuid.FromStringOrNil("c3d4e5f6-a7b8-11ed-c3d4-e5f6a7b8c9d0"),
			referenceType:  aicall.ReferenceTypeCall,
			referenceID:    uuid.FromStringOrNil("d4e5f6a7-b8c9-11ed-d4e5-f6a7b8c9d0e1"),
			confbridgeID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-11ed-e5f6-a7b8c9d0e1f2"),
			pipecatcallID:  uuid.FromStringOrNil("f6a7b8c9-d0e1-11ed-f6a7-b8c9d0e1f2a3"),

			parameter:      nil,

			responseUUIDID: uuid.FromStringOrNil("a7b8c9d0-e1f2-11ed-a7b8-c9d0e1f2a3b4"),
			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a7b8c9d0-e1f2-11ed-a7b8-c9d0e1f2a3b4"),
				},
			},

			expectAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a7b8c9d0-e1f2-11ed-a7b8-c9d0e1f2a3b4"),
					CustomerID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ed-b2c3-d4e5f6a7b8c9"),
				},
				AssistanceType: aicall.AssistanceTypeAI,
				AssistanceID:   uuid.FromStringOrNil("a1b2c3d4-e5f6-11ed-a1b2-c3d4e5f6a7b8"),
				AIEngineModel:  ai.EngineModelOpenaiGPT5,
				AITTSType:      ai.TTSTypeElevenLabs,
				AITTSVoiceID:   "voice-id-123",
				AISTTType:      ai.STTTypeDeepgram,
				ActiveflowID:   uuid.FromStringOrNil("c3d4e5f6-a7b8-11ed-c3d4-e5f6a7b8c9d0"),
				ReferenceType:  aicall.ReferenceTypeCall,
				ReferenceID:    uuid.FromStringOrNil("d4e5f6a7-b8c9-11ed-d4e5-f6a7b8c9d0e1"),
				ConfbridgeID:   uuid.FromStringOrNil("e5f6a7b8-c9d0-11ed-e5f6-a7b8c9d0e1f2"),
				PipecatcallID:  uuid.FromStringOrNil("f6a7b8c9-d0e1-11ed-f6a7-b8c9d0e1f2a3"),

				STTLanguage:    "ja-JP",
				Status:         aicall.StatusInitiating,
			},
		},
		{
			name: "with current member id",

			ai: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("c1d2e3f4-a5b6-11ed-c1d2-e3f4a5b6c7d8"),
					CustomerID: uuid.FromStringOrNil("d2e3f4a5-b6c7-11ed-d2e3-f4a5b6c7d8e9"),
				},
				EngineModel: ai.EngineModelOpenaiGPT5,
				TTSType:     ai.TTSTypeElevenLabs,
				TTSVoiceID:  "voice-id-456",
				STTType:     ai.STTTypeDeepgram,
				STTLanguage: "ko-KR",
			},
			assistanceType:  aicall.AssistanceTypeTeam,
			assistanceID:    uuid.FromStringOrNil("c1d2e3f4-a5b6-11ed-c1d2-e3f4a5b6c7d8"),
			activeflowID:    uuid.FromStringOrNil("e3f4a5b6-c7d8-11ed-e3f4-a5b6c7d8e9f0"),
			referenceType:   aicall.ReferenceTypeCall,
			referenceID:     uuid.FromStringOrNil("f4a5b6c7-d8e9-11ed-f4a5-b6c7d8e9f0a1"),
			confbridgeID:    uuid.FromStringOrNil("a5b6c7d8-e9f0-11ed-a5b6-c7d8e9f0a1b2"),
			pipecatcallID:   uuid.FromStringOrNil("b6c7d8e9-f0a1-11ed-b6c7-d8e9f0a1b2c3"),
			currentMemberID: uuid.FromStringOrNil("c7d8e9f0-a1b2-11ed-c7d8-e9f0a1b2c3d4"),

			parameter:       nil,

			responseUUIDID: uuid.FromStringOrNil("d8e9f0a1-b2c3-11ed-d8e9-f0a1b2c3d4e5"),
			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d8e9f0a1-b2c3-11ed-d8e9-f0a1b2c3d4e5"),
				},
			},

			expectAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d8e9f0a1-b2c3-11ed-d8e9-f0a1b2c3d4e5"),
					CustomerID: uuid.FromStringOrNil("d2e3f4a5-b6c7-11ed-d2e3-f4a5b6c7d8e9"),
				},
				AssistanceType:  aicall.AssistanceTypeTeam,
				AssistanceID:    uuid.FromStringOrNil("c1d2e3f4-a5b6-11ed-c1d2-e3f4a5b6c7d8"),
				AIEngineModel:   ai.EngineModelOpenaiGPT5,
				AITTSType:       ai.TTSTypeElevenLabs,
				AITTSVoiceID:    "voice-id-456",
				AISTTType:       ai.STTTypeDeepgram,
				ActiveflowID:    uuid.FromStringOrNil("e3f4a5b6-c7d8-11ed-e3f4-a5b6c7d8e9f0"),
				ReferenceType:   aicall.ReferenceTypeCall,
				ReferenceID:     uuid.FromStringOrNil("f4a5b6c7-d8e9-11ed-f4a5-b6c7d8e9f0a1"),
				ConfbridgeID:    uuid.FromStringOrNil("a5b6c7d8-e9f0-11ed-a5b6-c7d8e9f0a1b2"),
				PipecatcallID:   uuid.FromStringOrNil("b6c7d8e9-f0a1-11ed-b6c7-d8e9f0a1b2c3"),
				CurrentMemberID: uuid.FromStringOrNil("c7d8e9f0-a1b2-11ed-c7d8-e9f0a1b2c3d4"),

				STTLanguage:     "ko-KR",
				Status:          aicall.StatusInitiating,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUIDID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			res, err := h.Create(ctx, tt.ai, tt.assistanceType, tt.assistanceID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeID, tt.pipecatcallID, tt.currentMemberID, tt.parameter, nil)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcall, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAIcall *aicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("5154c3f6-a709-11ed-b011-c7644d9b5fc9"),

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("5154c3f6-a709-11ed-b011-c7644d9b5fc9"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcall, res)
			}
		})
	}
}

func Test_GetByReferenceID(t *testing.T) {

	tests := []struct {
		name string

		referenceID uuid.UUID

		responseAIcall *aicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a665825e-a709-11ed-967e-538651691d20"),

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a964a98a-a709-11ed-ad69-3ff036631417"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallGetByReferenceID(ctx, tt.referenceID).Return(tt.responseAIcall, nil)

			res, err := h.GetByReferenceID(ctx, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcall, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAIcall *aicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("301029b6-578f-41c4-905a-906e4e8ebbb3"),

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("301029b6-578f-41c4-905a-906e4e8ebbb3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, nil)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcall, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		size    uint64
		token   string
		filters map[aicall.Field]any

		responseAIcalls []*aicall.AIcall
	}{
		{
			name: "normal",

			size:  10,
			token: "2023-01-03T21:35:02.809Z",
			filters: map[aicall.Field]any{
				aicall.FieldDeleted:    false,
				aicall.FieldCustomerID: uuid.FromStringOrNil("1694a6ac-b485-11ee-9900-ff6bfeb9a3cc"),
			},

			responseAIcalls: []*aicall.AIcall{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("16f0c4f0-b485-11ee-81fd-6fe39701733c"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallList(ctx, tt.size, tt.token, tt.filters).Return(tt.responseAIcalls, nil)

			res, err := h.List(ctx, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcalls) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcalls, res)
			}
		})
	}
}

func Test_UpdateCurrentMemberID(t *testing.T) {

	tests := []struct {
		name string

		id              uuid.UUID
		currentMemberID uuid.UUID

		responseUpdateErr error
		responseAIcall    *aicall.AIcall
		responseGetErr    error

		expectFields map[aicall.Field]any
		expectRes    *aicall.AIcall
		expectErr    bool
	}{
		{
			name: "normal",

			id:              uuid.FromStringOrNil("a1b2c3d4-e5f6-11ed-a1b2-c3d4e5f6a7b8"),
			currentMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ed-b2c3-d4e5f6a7b8c9"),

			responseUpdateErr: nil,
			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11ed-a1b2-c3d4e5f6a7b8"),
				},
				CurrentMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ed-b2c3-d4e5f6a7b8c9"),
			},
			responseGetErr: nil,

			expectFields: map[aicall.Field]any{
				aicall.FieldCurrentMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ed-b2c3-d4e5f6a7b8c9"),
			},
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a1b2c3d4-e5f6-11ed-a1b2-c3d4e5f6a7b8"),
				},
				CurrentMemberID: uuid.FromStringOrNil("b2c3d4e5-f6a7-11ed-b2c3-d4e5f6a7b8c9"),
			},
			expectErr: false,
		},
		{
			name: "update error",

			id:              uuid.FromStringOrNil("c3d4e5f6-a7b8-11ed-c3d4-e5f6a7b8c9d0"),
			currentMemberID: uuid.FromStringOrNil("d4e5f6a7-b8c9-11ed-d4e5-f6a7b8c9d0e1"),

			responseUpdateErr: fmt.Errorf("update failed"),

			expectFields: map[aicall.Field]any{
				aicall.FieldCurrentMemberID: uuid.FromStringOrNil("d4e5f6a7-b8c9-11ed-d4e5-f6a7b8c9d0e1"),
			},
			expectErr: true,
		},
		{
			name: "get error after update",

			id:              uuid.FromStringOrNil("e5f6a7b8-c9d0-11ed-e5f6-a7b8c9d0e1f2"),
			currentMemberID: uuid.FromStringOrNil("f6a7b8c9-d0e1-11ed-f6a7-b8c9d0e1f2a3"),

			responseUpdateErr: nil,
			responseAIcall:    nil,
			responseGetErr:    fmt.Errorf("get failed"),

			expectFields: map[aicall.Field]any{
				aicall.FieldCurrentMemberID: uuid.FromStringOrNil("f6a7b8c9-d0e1-11ed-f6a7-b8c9d0e1f2a3"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &aicallHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdate(ctx, tt.id, tt.expectFields).Return(tt.responseUpdateErr)
			if tt.responseUpdateErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, tt.responseGetErr)
			}

			res, err := h.UpdateCurrentMemberID(ctx, tt.id, tt.currentMemberID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdateActiveflowID(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		activeflowID uuid.UUID

		responseUpdateErr error
		responseAIcall    *aicall.AIcall
		responseGetErr    error

		expectFields map[aicall.Field]any
		expectRes    *aicall.AIcall
		expectErr    bool
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("21a4d6f0-2342-11ef-9f0a-cb0d8f0c3a01"),
			activeflowID: uuid.FromStringOrNil("21d6f3aa-2342-11ef-bd13-7b3a4f7c2d11"),

			responseUpdateErr: nil,
			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("21a4d6f0-2342-11ef-9f0a-cb0d8f0c3a01"),
				},
				ActiveflowID: uuid.FromStringOrNil("21d6f3aa-2342-11ef-bd13-7b3a4f7c2d11"),
			},
			responseGetErr: nil,

			expectFields: map[aicall.Field]any{
				aicall.FieldActiveflowID: uuid.FromStringOrNil("21d6f3aa-2342-11ef-bd13-7b3a4f7c2d11"),
			},
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("21a4d6f0-2342-11ef-9f0a-cb0d8f0c3a01"),
				},
				ActiveflowID: uuid.FromStringOrNil("21d6f3aa-2342-11ef-bd13-7b3a4f7c2d11"),
			},
			expectErr: false,
		},
		{
			name: "update error",

			id:           uuid.FromStringOrNil("220f1c2c-2342-11ef-bf21-3f7d2a4b5e22"),
			activeflowID: uuid.FromStringOrNil("2243b15e-2342-11ef-9c3a-bb6e9d8f2c33"),

			responseUpdateErr: fmt.Errorf("update failed"),

			expectFields: map[aicall.Field]any{
				aicall.FieldActiveflowID: uuid.FromStringOrNil("2243b15e-2342-11ef-9c3a-bb6e9d8f2c33"),
			},
			expectErr: true,
		},
		{
			name: "get error after update",

			id:           uuid.FromStringOrNil("22725c4a-2342-11ef-aebc-7f4c9d0e3a44"),
			activeflowID: uuid.FromStringOrNil("22a3f0d2-2342-11ef-b6c4-cb1d2e3f4a55"),

			responseUpdateErr: nil,
			responseAIcall:    nil,
			responseGetErr:    fmt.Errorf("get failed"),

			expectFields: map[aicall.Field]any{
				aicall.FieldActiveflowID: uuid.FromStringOrNil("22a3f0d2-2342-11ef-b6c4-cb1d2e3f4a55"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &aicallHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdate(ctx, tt.id, tt.expectFields).Return(tt.responseUpdateErr)
			if tt.responseUpdateErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, tt.responseGetErr)
			}

			res, err := h.UpdateActiveflowID(ctx, tt.id, tt.activeflowID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdatePipecatcallIDAndActiveflowID(t *testing.T) {

	tests := []struct {
		name string

		id            uuid.UUID
		pipecatcallID uuid.UUID
		activeflowID  uuid.UUID

		responseUpdateErr error
		responseAIcall    *aicall.AIcall
		responseGetErr    error

		expectFields map[aicall.Field]any
		expectRes    *aicall.AIcall
		expectErr    bool
	}{
		{
			name: "normal",

			id:            uuid.FromStringOrNil("3a01b6c0-2342-11ef-9f0a-cb0d8f0c3a01"),
			pipecatcallID: uuid.FromStringOrNil("3a3454ce-2342-11ef-bd13-7b3a4f7c2d11"),
			activeflowID:  uuid.FromStringOrNil("3a6660d4-2342-11ef-9c3a-bb6e9d8f2c33"),

			responseUpdateErr: nil,
			responseAIcall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3a01b6c0-2342-11ef-9f0a-cb0d8f0c3a01"),
				},
				PipecatcallID: uuid.FromStringOrNil("3a3454ce-2342-11ef-bd13-7b3a4f7c2d11"),
				ActiveflowID:  uuid.FromStringOrNil("3a6660d4-2342-11ef-9c3a-bb6e9d8f2c33"),
			},
			responseGetErr: nil,

			expectFields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("3a3454ce-2342-11ef-bd13-7b3a4f7c2d11"),
				aicall.FieldActiveflowID:  uuid.FromStringOrNil("3a6660d4-2342-11ef-9c3a-bb6e9d8f2c33"),
			},
			expectRes: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3a01b6c0-2342-11ef-9f0a-cb0d8f0c3a01"),
				},
				PipecatcallID: uuid.FromStringOrNil("3a3454ce-2342-11ef-bd13-7b3a4f7c2d11"),
				ActiveflowID:  uuid.FromStringOrNil("3a6660d4-2342-11ef-9c3a-bb6e9d8f2c33"),
			},
			expectErr: false,
		},
		{
			name: "update error",

			id:            uuid.FromStringOrNil("3aa3a8a4-2342-11ef-bf21-3f7d2a4b5e22"),
			pipecatcallID: uuid.FromStringOrNil("3ad5d6dc-2342-11ef-9c3a-bb6e9d8f2c33"),
			activeflowID:  uuid.FromStringOrNil("3b07ce28-2342-11ef-aebc-7f4c9d0e3a44"),

			responseUpdateErr: fmt.Errorf("update failed"),

			expectFields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("3ad5d6dc-2342-11ef-9c3a-bb6e9d8f2c33"),
				aicall.FieldActiveflowID:  uuid.FromStringOrNil("3b07ce28-2342-11ef-aebc-7f4c9d0e3a44"),
			},
			expectErr: true,
		},
		{
			name: "get error after update",

			id:            uuid.FromStringOrNil("3b3a82a8-2342-11ef-aebc-7f4c9d0e3a44"),
			pipecatcallID: uuid.FromStringOrNil("3b6dadea-2342-11ef-b6c4-cb1d2e3f4a55"),
			activeflowID:  uuid.FromStringOrNil("3b9f9c5c-2342-11ef-b6c4-cb1d2e3f4a55"),

			responseUpdateErr: nil,
			responseAIcall:    nil,
			responseGetErr:    fmt.Errorf("get failed"),

			expectFields: map[aicall.Field]any{
				aicall.FieldPipecatcallID: uuid.FromStringOrNil("3b6dadea-2342-11ef-b6c4-cb1d2e3f4a55"),
				aicall.FieldActiveflowID:  uuid.FromStringOrNil("3b9f9c5c-2342-11ef-b6c4-cb1d2e3f4a55"),
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &aicallHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdate(ctx, tt.id, tt.expectFields).Return(tt.responseUpdateErr)
			if tt.responseUpdateErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, tt.responseGetErr)
			}

			res, err := h.UpdatePipecatcallIDAndActiveflowID(ctx, tt.id, tt.pipecatcallID, tt.activeflowID)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
