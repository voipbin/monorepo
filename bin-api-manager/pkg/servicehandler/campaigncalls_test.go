package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cacampaign "monorepo/bin-campaign-manager/models/campaign"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_CampaigncallGets(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseCampaigncalls []cacampaigncall.Campaigncall
		expectRes             []*cacampaigncall.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			size:  10,
			token: "2020-09-20 03:23:20.995000",

			responseCampaigncalls: []cacampaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3b598286-105d-11ee-a8e0-d3fe1d127d17"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3b7fc41e-105d-11ee-9b29-a77a519ca3b9"),
					},
				},
			},
			expectRes: []*cacampaigncall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3b598286-105d-11ee-a8e0-d3fe1d127d17"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("3b7fc41e-105d-11ee-9b29-a77a519ca3b9"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().CampaignV1CampaigncallList(ctx, tt.token, tt.size, gomock.Any()).Return(tt.responseCampaigncalls, nil)

			res, err := h.CampaigncallGets(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_campaigncallGetsByCampaignID(t *testing.T) {

	tests := []struct {
		name       string
		agent      *amagent.Agent
		campaignID uuid.UUID
		pageToken  string
		pageSize   uint64

		responseCampaign      *cacampaign.Campaign
		responseCampaigncalls []cacampaigncall.Campaigncall
		expectRes             []*cacampaigncall.WebhookMessage
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
			uuid.FromStringOrNil("8ac37eee-c86a-11ec-ba1f-9bdb87f79ae6"),
			"2020-10-20T01:00:00.995000",
			10,

			&cacampaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8ac37eee-c86a-11ec-ba1f-9bdb87f79ae6"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			[]cacampaigncall.Campaigncall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8b1cff64-c86a-11ec-87e0-e7907d8f52c9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8b4beac2-c86a-11ec-b1d9-6fdb61e23bd6"),
					},
				},
			},
			[]*cacampaigncall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8b1cff64-c86a-11ec-87e0-e7907d8f52c9"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8b4beac2-c86a-11ec-b1d9-6fdb61e23bd6"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.responseCampaign, nil)
			mockReq.EXPECT().CampaignV1CampaigncallList(ctx, tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.responseCampaigncalls, nil)
			res, err := h.CampaigncallGetsByCampaignID(ctx, tt.agent, tt.campaignID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallGet(t *testing.T) {

	tests := []struct {
		name           string
		agent          *amagent.Agent
		campaigncallID uuid.UUID

		response  *cacampaigncall.Campaigncall
		expectRes *cacampaigncall.WebhookMessage
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
			uuid.FromStringOrNil("33a06e3c-c86b-11ec-85e5-27fb8fd8a197"),

			&cacampaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33a06e3c-c86b-11ec-85e5-27fb8fd8a197"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaigncall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("33a06e3c-c86b-11ec-85e5-27fb8fd8a197"),
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

			mockReq.EXPECT().CampaignV1CampaigncallGet(ctx, tt.campaigncallID).Return(tt.response, nil)
			res, err := h.CampaigncallGet(ctx, tt.agent, tt.campaigncallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaigncallDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *amagent.Agent
		id    uuid.UUID

		responseCampaigncall *cacampaigncall.Campaigncall
		expectRes            *cacampaigncall.WebhookMessage
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
			uuid.FromStringOrNil("779eb134-c86b-11ec-bf08-0301553a89c1"),

			&cacampaigncall.Campaigncall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("779eb134-c86b-11ec-bf08-0301553a89c1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cacampaigncall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("779eb134-c86b-11ec-bf08-0301553a89c1"),
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

			mockReq.EXPECT().CampaignV1CampaigncallGet(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			mockReq.EXPECT().CampaignV1CampaigncallDelete(ctx, tt.id).Return(tt.responseCampaigncall, nil)
			res, err := h.CampaigncallDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
