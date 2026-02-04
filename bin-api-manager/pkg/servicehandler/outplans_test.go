package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	caoutplan "monorepo/bin-campaign-manager/models/outplan"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_OutplanCreate(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent

		outplanName string
		detail      string

		source *commonaddress.Address

		dialTimeout int
		tryInterval int

		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
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
			"test name",
			"test detail",
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			5,
			5,
			5,
			5,
			5,
			&caoutplan.Outplan{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
				},
			},
			&caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
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

			mockReq.EXPECT().CampaignV1OutplanCreate(ctx, tt.agent.CustomerID, tt.outplanName, tt.detail, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(tt.response, nil)
			res, err := h.OutplanCreate(ctx, tt.agent, tt.outplanName, tt.detail, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount0, tt.maxTryCount0, tt.maxTryCount0, tt.maxTryCount4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		id    uuid.UUID

		responseOutplan *caoutplan.Outplan
		expectRes       *caoutplan.WebhookMessage
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
			uuid.FromStringOrNil("bcbcffb4-c640-11ec-bdab-03b1d679601d"),

			&caoutplan.Outplan{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bcbcffb4-c640-11ec-bdab-03b1d679601d"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bcbcffb4-c640-11ec-bdab-03b1d679601d"),
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

			mockReq.EXPECT().CampaignV1OutplanGet(ctx, tt.id).Return(tt.responseOutplan, nil)
			mockReq.EXPECT().CampaignV1OutplanDelete(ctx, tt.id).Return(tt.responseOutplan, nil)
			res, err := h.OutplanDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanListByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		response  []caoutplan.Outplan
		expectRes []*caoutplan.WebhookMessage
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
			"2020-10-20T01:00:00.995000Z",
			10,

			[]caoutplan.Outplan{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6c3a509a-c641-11ec-97f3-97762ce0f584"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6c8f8c0e-c641-11ec-ae79-63f803cffc1f"),
					},
				},
			},
			[]*caoutplan.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6c3a509a-c641-11ec-97f3-97762ce0f584"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6c8f8c0e-c641-11ec-ae79-63f803cffc1f"),
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

			mockReq.EXPECT().CampaignV1OutplanList(ctx, tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.response, nil)
			res, err := h.OutplanGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanGet(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		outplanID uuid.UUID

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
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

			&caoutplan.Outplan{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&caoutplan.WebhookMessage{
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

			mockReq.EXPECT().CampaignV1OutplanGet(ctx, tt.outplanID).Return(tt.response, nil)
			res, err := h.OutplanGet(ctx, tt.agent, tt.outplanID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name        string
		agent       *amagent.Agent
		outplanID   uuid.UUID
		outplanName string
		detail      string

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
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

			&caoutplan.Outplan{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&caoutplan.WebhookMessage{
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

			mockReq.EXPECT().CampaignV1OutplanGet(ctx, tt.outplanID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1OutplanUpdateBasicInfo(ctx, tt.outplanID, tt.outplanName, tt.detail).Return(tt.response, nil)
			res, err := h.OutplanUpdateBasicInfo(ctx, tt.agent, tt.outplanID, tt.outplanName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateDialInfo(t *testing.T) {

	tests := []struct {
		name         string
		agent        *amagent.Agent
		outplanID    uuid.UUID
		source       *commonaddress.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
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

			uuid.FromStringOrNil("451a473e-c643-11ec-93c4-0bd1b9b41f16"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			5,
			5,
			5,
			5,
			5,

			&caoutplan.Outplan{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("451a473e-c643-11ec-93c4-0bd1b9b41f16"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&caoutplan.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("451a473e-c643-11ec-93c4-0bd1b9b41f16"),
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

			mockReq.EXPECT().CampaignV1OutplanGet(ctx, tt.outplanID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1OutplanUpdateDialInfo(ctx, tt.outplanID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(tt.response, nil)
			res, err := h.OutplanUpdateDialInfo(ctx, tt.agent, tt.outplanID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
