package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ExtensionCreate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		ext      string
		password string
		extName  string
		detail   string

		response  *rmextension.Extension
		expectRes *rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			"test",
			"password",
			"test",
			"test detail",

			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("4037dd90-6fa4-11eb-b51b-771a2747271b"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&rmextension.WebhookMessage{
				ID:         uuid.FromStringOrNil("4037dd90-6fa4-11eb-b51b-771a2747271b"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionCreate(ctx, tt.agent.CustomerID, tt.ext, tt.password, tt.extName, tt.detail).Return(tt.response, nil)

			res, err := h.ExtensionCreate(ctx, tt.agent, tt.ext, tt.password, tt.extName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionUpdate(t *testing.T) {

	type test struct {
		name  string
		agent *amagent.Agent

		id       uuid.UUID
		extName  string
		detail   string
		password string

		responseExtension *rmextension.Extension
		expectRes         *rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
			"update name",
			"update detail",
			"update password",

			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "update name",
				Detail:     "update detail",
				Extension:  "test",
				Password:   "update password",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "2020-09-20 03:23:23.995000",
			},
			&rmextension.WebhookMessage{
				ID:         uuid.FromStringOrNil("50c1e4ca-6fa5-11eb-8a12-67425d88ba43"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Name:       "update name",
				Detail:     "update detail",
				Extension:  "test",
				Password:   "update password",
				TMCreate:   "2020-09-20 03:23:20.995000",
				TMUpdate:   "2020-09-20 03:23:23.995000"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.id).Return(tt.responseExtension, nil)
			mockReq.EXPECT().RegistrarV1ExtensionUpdate(ctx, tt.id, tt.extName, tt.detail, tt.password).Return(tt.responseExtension, nil)
			res, err := h.ExtensionUpdate(ctx, tt.agent, tt.id, tt.extName, tt.detail, tt.password)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionDelete(t *testing.T) {

	type test struct {
		name        string
		agent       *amagent.Agent
		extensionID uuid.UUID

		response  *rmextension.Extension
		expectRes *rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("aa1fda4e-6fa6-11eb-8385-a3288e16c056"),

			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("aa1fda4e-6fa6-11eb-8385-a3288e16c056"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),

				Name:   "test",
				Detail: "test detail",

				Extension: "test",
				Password:  "password",
			},
			&rmextension.WebhookMessage{
				ID:         uuid.FromStringOrNil("aa1fda4e-6fa6-11eb-8385-a3288e16c056"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),

				Name:   "test",
				Detail: "test detail",

				Extension: "test",
				Password:  "password",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.extensionID).Return(tt.response, nil)
			mockReq.EXPECT().RegistrarV1ExtensionDelete(ctx, tt.extensionID).Return(tt.response, nil)

			res, err := h.ExtensionDelete(ctx, tt.agent, tt.extensionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionGet(t *testing.T) {

	type test struct {
		name        string
		agent       *amagent.Agent
		extensionID uuid.UUID

		response  *rmextension.Extension
		expectRes *rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("27a0b1ba-6fab-11eb-9aec-6b59dbde86d8"),

			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("27a0b1ba-6fab-11eb-9aec-6b59dbde86d8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),

				Name:   "test",
				Detail: "test detail",
			},
			&rmextension.WebhookMessage{
				ID:         uuid.FromStringOrNil("27a0b1ba-6fab-11eb-9aec-6b59dbde86d8"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),

				Name:   "test",
				Detail: "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGet(ctx, tt.extensionID).Return(tt.response, nil)
			res, err := h.ExtensionGet(ctx, tt.agent, tt.extensionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ExtensionGets(t *testing.T) {

	type test struct {
		name          string
		agent         *amagent.Agent
		pageToken     string
		pageSize      uint64
		expectFilters map[string]string

		response  []rmextension.Extension
		expectRes []*rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20T01:00:00.995000",
			10,
			map[string]string{
				"customer_id": "5f621078-8e5f-11ee-97b2-cfe7337b701c",
				"deleted":     "false",
			},

			[]rmextension.Extension{
				{
					ID: uuid.FromStringOrNil("cd569cb4-4ff5-11ee-931b-177bc147dd69"),
				},
				{
					ID: uuid.FromStringOrNil("cda81aa8-4ff5-11ee-a7c5-ab71c50d7ed0"),
				},
			},
			[]*rmextension.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("cd569cb4-4ff5-11ee-931b-177bc147dd69"),
				},
				{
					ID: uuid.FromStringOrNil("cda81aa8-4ff5-11ee-a7c5-ab71c50d7ed0"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ExtensionGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.response, nil)

			res, err := h.ExtensionGets(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
