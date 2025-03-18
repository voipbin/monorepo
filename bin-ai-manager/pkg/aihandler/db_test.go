package aihandler

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
	"monorepo/bin-ai-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		aiName      string
		detail      string
		engineType  ai.EngineType
		engineModel ai.EngineModel
		engineData  map[string]any
		initPrompt  string

		responseUUID uuid.UUID
		responseAI   *ai.AI

		expectAI *ai.AI
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("8db73654-a70d-11ed-ae15-6726993338d8"),
			aiName:      "test name",
			detail:      "test detail",
			engineType:  ai.EngineTypeNone,
			engineModel: ai.EngineModelOpenaiGPT4Turbo,
			engineData: map[string]any{
				"key1": "val1",
			},
			initPrompt: "test init prompt",

			responseUUID: uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
				},
			},

			expectAI: &ai.AI{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
					CustomerID: uuid.FromStringOrNil("8db73654-a70d-11ed-ae15-6726993338d8"),
				},
				Name:        "test name",
				Detail:      "test detail",
				EngineType:  ai.EngineTypeNone,
				EngineModel: ai.EngineModelOpenaiGPT4Turbo,
				EngineData: map[string]any{
					"key1": "val1",
				},
				InitPrompt: "test init prompt",
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

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().AICreate(ctx, tt.expectAI).Return(nil)
			mockDB.EXPECT().AIGet(ctx, tt.responseUUID).Return(tt.responseAI, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAI.CustomerID, ai.EventTypeCreated, tt.responseAI)

			res, err := h.Create(ctx, tt.customerID, tt.aiName, tt.detail, tt.engineType, tt.engineModel, tt.engineData, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAI, res)
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

		responseAIs []*ai.AI
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("132be434-f839-11ed-ae95-efa657af10fb"),
			size:       10,
			token:      "2023-01-03 21:35:02.809",
			filters: map[string]string{
				"deleted": "false",
			},

			responseAIs: []*ai.AI{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("31b00c64-f839-11ed-8f59-ab874a16ee9c"),
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

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIGets(ctx, tt.customerID, tt.size, tt.token, tt.filters).Return(tt.responseAIs, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAIs) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAIs, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAI *ai.AI
	}{
		{
			"normal",

			uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),

			&ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),
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

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseAI, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAI, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseAI *ai.AI
	}{
		{
			"normal",

			uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),

			&ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),
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

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AIDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseAI, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAI.CustomerID, ai.EventTypeDeleted, tt.responseAI)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAI, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		aiName      string
		detail      string
		engineType  ai.EngineType
		engineModel ai.EngineModel
		engineData  map[string]any
		initPrompt  string

		responseAI *ai.AI
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("fd49c1d6-f82e-11ed-8893-dfb489cd9bb9"),
			aiName:      "new name",
			detail:      "new detail",
			engineType:  ai.EngineTypeNone,
			engineModel: ai.EngineModelOpenaiGPT3Dot5Turbo,
			engineData: map[string]any{
				"key1": "val1",
			},
			initPrompt: "new init prompt",

			responseAI: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("fd49c1d6-f82e-11ed-8893-dfb489cd9bb9"),
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

			h := &aiHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AISetInfo(ctx, tt.id, tt.aiName, tt.detail, tt.engineType, tt.engineModel, tt.engineData, tt.initPrompt).Return(nil)
			mockDB.EXPECT().AIGet(ctx, tt.id).Return(tt.responseAI, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseAI.CustomerID, ai.EventTypeUpdated, tt.responseAI)

			res, err := h.Update(ctx, tt.id, tt.aiName, tt.detail, tt.engineType, tt.engineModel, tt.engineData, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseAI) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseAI, res)
			}
		})
	}
}
