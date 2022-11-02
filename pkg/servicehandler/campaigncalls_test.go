package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	cacampaigncall "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_CampaigncallGetsByCampaignID(t *testing.T) {

	tests := []struct {
		name       string
		customer   *cscustomer.Customer
		campaignID uuid.UUID
		pageToken  string
		pageSize   uint64

		response         []cacampaigncall.Campaigncall
		responseCampaign *cacampaign.Campaign
		expectRes        []*cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("8ac37eee-c86a-11ec-ba1f-9bdb87f79ae6"),
			"2020-10-20T01:00:00.995000",
			10,

			[]cacampaigncall.Campaigncall{
				{
					ID: uuid.FromStringOrNil("8b1cff64-c86a-11ec-87e0-e7907d8f52c9"),
				},
				{
					ID: uuid.FromStringOrNil("8b4beac2-c86a-11ec-b1d9-6fdb61e23bd6"),
				},
			},
			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("8ac37eee-c86a-11ec-ba1f-9bdb87f79ae6"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			[]*cacampaigncall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("8b1cff64-c86a-11ec-87e0-e7907d8f52c9"),
				},
				{
					ID: uuid.FromStringOrNil("8b4beac2-c86a-11ec-b1d9-6fdb61e23bd6"),
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
			mockReq.EXPECT().CampaignV1CampaigncallGetsByCampaignID(ctx, tt.campaignID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.CampaigncallGetsByCampaignID(ctx, tt.customer, tt.campaignID, tt.pageSize, tt.pageToken)
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
		customer       *cscustomer.Customer
		campaigncallID uuid.UUID

		response  *cacampaigncall.Campaigncall
		expectRes *cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("33a06e3c-c86b-11ec-85e5-27fb8fd8a197"),

			&cacampaigncall.Campaigncall{
				ID:         uuid.FromStringOrNil("33a06e3c-c86b-11ec-85e5-27fb8fd8a197"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaigncall.WebhookMessage{
				ID: uuid.FromStringOrNil("33a06e3c-c86b-11ec-85e5-27fb8fd8a197"),
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
			res, err := h.CampaigncallGet(ctx, tt.customer, tt.campaigncallID)
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

		customer *cscustomer.Customer
		id       uuid.UUID

		responseCampaigncall *cacampaigncall.Campaigncall
		expectRes            *cacampaigncall.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("779eb134-c86b-11ec-bf08-0301553a89c1"),

			&cacampaigncall.Campaigncall{
				ID:         uuid.FromStringOrNil("779eb134-c86b-11ec-bf08-0301553a89c1"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaigncall.WebhookMessage{
				ID: uuid.FromStringOrNil("779eb134-c86b-11ec-bf08-0301553a89c1"),
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
			res, err := h.CampaigncallDelete(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
