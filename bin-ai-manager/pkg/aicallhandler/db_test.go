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
		gender        aicall.Gender
		language      string

		responseUUID   uuid.UUID
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
			},
			activeflowID:  uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
			referenceType: aicall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
			confbridgeID:  uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
			gender:        aicall.GenderFemale,
			language:      "en-US",

			responseUUID: uuid.FromStringOrNil("820745c0-a707-11ed-9b12-9bce1a08774b"),
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
				ActiveflowID:  uuid.FromStringOrNil("fef51c0a-fba4-11ed-b222-673487fcf35b"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("81deff70-a707-11ed-9bf5-6b5e777ccc90"),
				ConfbridgeID:  uuid.FromStringOrNil("df491e7a-c10d-4d9e-a17b-e6ffb2a752e9"),
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AIcallCreate(ctx, tt.expectAIcall).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.responseUUID).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusInitializing, tt.responseAIcall)

			res, err := h.Create(ctx, tt.ai, tt.activeflowID, tt.referenceType, tt.referenceID, tt.confbridgeID, tt.gender, tt.language)
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

func Test_GetByTranscribeID(t *testing.T) {

	tests := []struct {
		name string

		transcribeID uuid.UUID

		responseAIcall *aicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("c590415a-a709-11ed-b130-eba649c97eab"),

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("c5b76622-a709-11ed-8d54-63813a022d9a"),
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

			mockDB.EXPECT().AIcallGetByTranscribeID(ctx, tt.transcribeID).Return(tt.responseAIcall, nil)

			res, err := h.GetByTranscribeID(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcall, res)
			}
		})
	}
}

func Test_UpdateStatusStart(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		transcribeID uuid.UUID

		responseAIcall *aicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("3447ddd8-a70a-11ed-8b76-43164266fbb2"),
			uuid.FromStringOrNil("3470b852-a70a-11ed-9d3f-7feaeeaa417b"),

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("3447ddd8-a70a-11ed-8b76-43164266fbb2"),
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

			mockDB.EXPECT().AIcallUpdateStatusProgressing(ctx, tt.id, tt.transcribeID).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusProgressing, tt.responseAIcall)

			res, err := h.UpdateStatusStartProgressing(ctx, tt.id, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcall, res)
			}
		})
	}
}

func Test_UpdateStatusEnd(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAIcall *aicall.AIcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("a3c338ec-a70a-11ed-b305-9bd0df7c9474"),

			&aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a3c338ec-a70a-11ed-b305-9bd0df7c9474"),
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

			mockDB.EXPECT().AIcallUpdateStatusEnd(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, tt.id).Return(tt.responseAIcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAIcall.CustomerID, aicall.EventTypeStatusEnd, tt.responseAIcall)

			res, err := h.UpdateStatusEnd(ctx, tt.id)
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

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string
		filters    map[string]string

		responseAIcalls []*aicall.AIcall
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("1694a6ac-b485-11ee-9900-ff6bfeb9a3cc"),
			size:       10,
			token:      "2023-01-03 21:35:02.809",
			filters: map[string]string{
				"deleted": "false",
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

			mockDB.EXPECT().AIcallGets(ctx, tt.customerID, tt.size, tt.token, tt.filters).Return(tt.responseAIcalls, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIcalls) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIcalls, res)
			}
		})
	}
}
