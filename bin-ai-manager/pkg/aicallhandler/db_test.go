package aicallhandler

import (
	"context"
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

		ai            *ai.AI
		activeflowID  uuid.UUID
		referenceType aicall.ReferenceType
		referenceID   uuid.UUID
		confbridgeID  uuid.UUID
		pipecatcallID uuid.UUID
		gender        aicall.Gender
		language      string

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
				EngineType:  ai.EngineTypeNone,
				EngineModel: ai.EngineModelOpenaiGPT4,
				EngineData: map[string]any{
					"key1": "value1",
				},
				TTSType:    ai.TTSTypeElevenLabs,
				TTSVoiceID: "d4c7e1f2-5e8b-4d3a-9f0a-1c2b3d4e5f60",
				STTType:    ai.STTTypeDeepgram,
			},
			activeflowID:  uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
			referenceType: aicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
			confbridgeID:  uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
			pipecatcallID: uuid.FromStringOrNil("b063584e-b462-11f0-82f0-9b410ef3ab1e"),
			gender:        aicall.GenderFemale,
			language:      "en-US",

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
				AIID:          uuid.FromStringOrNil("81b311ee-a707-11ed-b499-f3284ac97a08"),
				AIEngineType:  ai.EngineTypeNone,
				AIEngineModel: ai.EngineModelOpenaiGPT4,
				AIEngineData: map[string]any{
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
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				Status:        aicall.StatusInitiating,
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

			res, err := h.Create(ctx, tt.ai, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeID, tt.pipecatcallID, tt.gender, tt.language)
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
