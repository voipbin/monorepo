package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"

	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

func Test_campaignRun(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCampaign *campaign.Campaign
		responseOutplan  *outplan.Outplan
		responseOutdial  *omoutdial.Outdial
		responseQueue    *qmqueue.Queue
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("8ff1d160-6110-43d4-a2da-d132f8696aaf"),

			responseCampaign: &campaign.Campaign{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("bfd09fa5-4c2c-46ea-aee9-a01a386e154a"),
					CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				},
				OutplanID: uuid.FromStringOrNil("c9af1a74-2dc8-4053-a181-5b47bebab2c4"),
				OutdialID: uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
			},
			responseOutplan: &outplan.Outplan{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c9af1a74-2dc8-4053-a181-5b47bebab2c4"),
					CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				},
			},
			responseOutdial: &omoutdial.Outdial{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
					CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				},
			},
			responseQueue: &qmqueue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
					CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				},
				TMDelete: nil,
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
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			h := &campaignHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				outplanHandler: mockOutplan,
			}
			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)

			// validate resource
			if tt.responseCampaign.OutplanID != uuid.Nil {
				mockOutplan.EXPECT().Get(ctx, tt.responseCampaign.OutplanID).Return(tt.responseOutplan, nil)
			}
			if tt.responseCampaign.OutdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.responseCampaign.OutdialID).Return(tt.responseOutdial, nil)
			}
			if tt.responseCampaign.QueueID != uuid.Nil {
				mockReq.EXPECT().QueueV1QueueGet(ctx, tt.responseCampaign.QueueID).Return(tt.responseQueue, nil)
			}

			mockDB.EXPECT().CampaignUpdateStatusAndExecute(ctx, tt.id, campaign.StatusRun, campaign.ExecuteRun).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaign.CustomerID, campaign.EventTypeCampaignStatusRun, tt.responseCampaign)
			mockReq.EXPECT().CampaignV1CampaignExecute(ctx, tt.id, 1000).Return(nil)

			res, err := h.campaignRun(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseCampaign) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseCampaign, res)
			}
		})
	}
}
