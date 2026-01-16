package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_OutdialCreate(t *testing.T) {

	tests := []struct {
		name string

		agent       *amagent.Agent
		campaignID  uuid.UUID
		outdialName string
		detail      string
		data        string

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("62b784eb-63e0-48e6-b6e1-2904eafd842d"),
			"test name",
			"test detail",
			"test data",

			&omoutdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58515568-030d-4fcd-a11d-e606d439eaef"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:   "test",
				Detail: "test detail",
				Data:   "test data",
			},
			&omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58515568-030d-4fcd-a11d-e606d439eaef"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Name:   "test",
				Detail: "test detail",
				Data:   "test data",
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

			mockReq.EXPECT().OutdialV1OutdialCreate(ctx, tt.agent.CustomerID, tt.campaignID, tt.outdialName, tt.detail, tt.data).Return(tt.response, nil)
			res, err := h.OutdialCreate(ctx, tt.agent, tt.campaignID, tt.outdialName, tt.detail, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialList(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response  []omoutdial.Outdial
		expectRes []*omoutdial.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]omoutdial.Outdial{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					},
				},
			},
			[]*omoutdial.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
					},
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

			mockReq.EXPECT().OutdialV1OutdialList(ctx, tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.response, nil)
			res, err := h.OutdialGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialDelete(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		outdialID uuid.UUID

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("92d41af7-4249-41a8-b86a-cb2ce21f214a"),

			&omoutdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("92d41af7-4249-41a8-b86a-cb2ce21f214a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("92d41af7-4249-41a8-b86a-cb2ce21f214a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialDelete(ctx, tt.outdialID).Return(tt.response, nil)

			res, err := h.OutdialDelete(ctx, tt.agent, tt.outdialID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialUpdate(t *testing.T) {

	tests := []struct {
		name        string
		agent       *amagent.Agent
		outdialID   uuid.UUID
		outdialName string
		detail      string

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
			"test name",
			"test detail",

			&omoutdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialUpdateBasicInfo(ctx, tt.outdialID, tt.outdialName, tt.detail).Return(tt.response, nil)
			res, err := h.OutdialUpdateBasicInfo(ctx, tt.agent, tt.outdialID, tt.outdialName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialUpdateCampaignID(t *testing.T) {

	tests := []struct {
		name       string
		agent      *amagent.Agent
		outdialID  uuid.UUID
		campaignID uuid.UUID

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("a7b05592-2d89-4440-a53d-a8dff4acc581"),
			uuid.FromStringOrNil("78f711a7-3b75-4c47-a796-cff180370aa1"),

			&omoutdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a7b05592-2d89-4440-a53d-a8dff4acc581"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a7b05592-2d89-4440-a53d-a8dff4acc581"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialUpdateCampaignID(ctx, tt.outdialID, tt.campaignID).Return(tt.response, nil)
			res, err := h.OutdialUpdateCampaignID(ctx, tt.agent, tt.outdialID, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialUpdateData(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		outdialID uuid.UUID
		data      string

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("e46bbea3-4b82-4b11-a9bb-8be3e152ae92"),
			"test data",

			&omoutdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e46bbea3-4b82-4b11-a9bb-8be3e152ae92"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&omoutdial.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e46bbea3-4b82-4b11-a9bb-8be3e152ae92"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OutdialV1OutdialUpdateData(ctx, tt.outdialID, tt.data).Return(tt.response, nil)
			res, err := h.OutdialUpdateData(ctx, tt.agent, tt.outdialID, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
