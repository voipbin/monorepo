package campaignhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/campaigncallhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

func Test_EventHandleActiveflowDeletedWithStoppableCampaign(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *campaign.Campaign
	}{
		{
			"campaign is stoppable",

			uuid.FromStringOrNil("8aca83f6-c3fb-11ec-b191-83f696719884"),

			&campaign.Campaign{
				ID:      uuid.FromStringOrNil("8aca83f6-c3fb-11ec-b191-83f696719884"),
				Execute: campaign.ExecuteStop,
				Status:  campaign.StatusStopping,
			},
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

			// isstoppable
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCampaigncall.EXPECT().GetsOngoingByCampaignID(ctx, tt.id, gomock.Any(), uint64(1)).Return([]*campaigncall.Campaigncall{}, nil)

			// updateStatusStop
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockDB.EXPECT().CampaignUpdateStatus(ctx, tt.id, campaign.StatusStop).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignStatusStop, tt.response)

			if err := h.EventHandleActiveflowDeleted(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_EventHandleReferenceCallHungupWithStoppableCampaign(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *campaign.Campaign
	}{
		{
			"campaign is stoppable",

			uuid.FromStringOrNil("86df1c8c-c3fd-11ec-a381-8fac03339669"),

			&campaign.Campaign{
				ID:      uuid.FromStringOrNil("86df1c8c-c3fd-11ec-a381-8fac03339669"),
				Execute: campaign.ExecuteStop,
				Status:  campaign.StatusStopping,
			},
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

			// isstoppable
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
			mockCampaigncall.EXPECT().GetsOngoingByCampaignID(ctx, tt.id, gomock.Any(), uint64(1)).Return([]*campaigncall.Campaigncall{}, nil)

			// updateStatusStop
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockDB.EXPECT().CampaignUpdateStatus(ctx, tt.id, campaign.StatusStop).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignStatusStop, tt.response)

			if err := h.EventHandleReferenceCallHungup(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
