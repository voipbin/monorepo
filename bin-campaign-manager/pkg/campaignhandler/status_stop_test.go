package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func Test_UpdateStatusStopping(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCampaign *campaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("6341e026-c442-11ec-8a25-ef389fb6d478"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("6341e026-c442-11ec-8a25-ef389fb6d478"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				OutplanID:  uuid.FromStringOrNil("c9af1a74-2dc8-4053-a181-5b47bebab2c4"),
				OutdialID:  uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
				Status:     campaign.StatusRun,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaignHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockDB.EXPECT().CampaignUpdateStatus(ctx, tt.id, campaign.StatusStopping).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaign.CustomerID, campaign.EventTypeCampaignStatusStopping, tt.responseCampaign)

			res, err := h.campaignStopping(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseCampaign) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseCampaign, res)
			}
		})
	}
}

func Test_updateStatusStop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *campaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("dd70296f-fadd-4fb0-bfc4-017944ec4597"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("dd70296f-fadd-4fb0-bfc4-017944ec4597"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				Status:     campaign.StatusStopping,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaignHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				reqHandler:    mockReq,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockDB.EXPECT().CampaignUpdateStatus(ctx, tt.id, campaign.StatusStop).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignStatusStop, tt.response)

			res, err := h.campaignStopNow(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_isStoppable(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response             *campaign.Campaign
		responseCampaigncall []*campaigncall.Campaigncall

		expectRes bool
	}{
		{
			"campaign is stoppable",

			uuid.FromStringOrNil("dd70296f-fadd-4fb0-bfc4-017944ec4597"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("dd70296f-fadd-4fb0-bfc4-017944ec4597"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				Execute:    campaign.ExecuteStop,
				Status:     campaign.StatusStopping,
			},
			[]*campaigncall.Campaigncall{},

			true,
		},
		{
			"campaign's exeucte is running",

			uuid.FromStringOrNil("8a6cd8d4-c444-11ec-b7a9-87b9a6605375"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("8a6cd8d4-c444-11ec-b7a9-87b9a6605375"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				Execute:    campaign.ExecuteRun,
				Status:     campaign.StatusStopping,
			},
			[]*campaigncall.Campaigncall{},

			false,
		},
		{
			"campaign's status is run but has no campaigncall and execute is stop",

			uuid.FromStringOrNil("8a6cd8d4-c444-11ec-b7a9-87b9a6605375"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("8a6cd8d4-c444-11ec-b7a9-87b9a6605375"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				Execute:    campaign.ExecuteStop,
				Status:     campaign.StatusRun,
			},
			[]*campaigncall.Campaigncall{},

			true,
		},
		{
			"campaign has campaign calls",

			uuid.FromStringOrNil("8a6cd8d4-c444-11ec-b7a9-87b9a6605375"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("8a6cd8d4-c444-11ec-b7a9-87b9a6605375"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				Execute:    campaign.ExecuteStop,
				Status:     campaign.StatusStopping,
			},
			[]*campaigncall.Campaigncall{
				{
					ID: uuid.FromStringOrNil("b708b214-c444-11ec-943a-ff8f38547200"),
				},
			},

			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				util:                mockUtil,
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)

			if tt.response.Execute == campaign.ExecuteStop {
				mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
				mockCampaigncall.EXPECT().GetsOngoingByCampaignID(ctx, tt.id, gomock.Any(), uint64(1)).Return(tt.responseCampaigncall, nil)
			}

			res := h.isStoppable(ctx, tt.id)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
