package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"

	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
)

func Test_campaignRun(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *campaign.Campaign
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("8ff1d160-6110-43d4-a2da-d132f8696aaf"),

			response: &campaign.Campaign{
				ID:         uuid.FromStringOrNil("bfd09fa5-4c2c-46ea-aee9-a01a386e154a"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				OutplanID:  uuid.FromStringOrNil("c9af1a74-2dc8-4053-a181-5b47bebab2c4"),
				OutdialID:  uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
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

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)

			// validate resource
			if tt.response.OutplanID != uuid.Nil {
				mockOutplan.EXPECT().Get(ctx, tt.response.OutplanID).Return(&outplan.Outplan{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)
			}
			if tt.response.OutdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.response.OutdialID).Return(&omoutdial.Outdial{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)
			}
			if tt.response.QueueID != uuid.Nil {
				mockReq.EXPECT().QueueV1QueueGet(ctx, tt.response.QueueID).Return(&qmqueue.Queue{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)
			}

			mockDB.EXPECT().CampaignUpdateStatusAndExecute(ctx, tt.id, campaign.StatusRun, campaign.ExecuteRun).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignStatusRun, tt.response)
			mockReq.EXPECT().CampaignV1CampaignExecute(ctx, tt.id, 1000).Return(nil)

			res, err := h.campaignRun(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

// func Test_isRunable(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		campaign *campaign.Campaign

// 		expectRes bool
// 	}{
// 		{
// 			"normal",

// 			&campaign.Campaign{
// 				ID:         uuid.FromStringOrNil("621847e2-c43f-11ec-a7d9-9f0a9ddc8347"),
// 				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
// 				OutplanID:  uuid.FromStringOrNil("c9af1a74-2dc8-4053-a181-5b47bebab2c4"),
// 				OutdialID:  uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
// 			},
// 			true,
// 		},
// 		{
// 			"campaign has no outdial id",

// 			&campaign.Campaign{
// 				ID:         uuid.FromStringOrNil("91b8e236-c43f-11ec-84e3-4f39221f60e9"),
// 				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
// 				OutplanID:  uuid.FromStringOrNil("c9af1a74-2dc8-4053-a181-5b47bebab2c4"),
// 			},
// 			false,
// 		},
// 		{
// 			"campaign has no outplan id",

// 			&campaign.Campaign{
// 				ID:         uuid.FromStringOrNil("621847e2-c43f-11ec-a7d9-9f0a9ddc8347"),
// 				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
// 				OutdialID:  uuid.FromStringOrNil("c7268f48-1a01-47ee-8cb1-ea2a34c53bff"),
// 			},
// 			false,
// 		},
// 		{
// 			"campaign has no outplan id and outdial id",

// 			&campaign.Campaign{
// 				ID:         uuid.FromStringOrNil("bf98bb0e-c43f-11ec-9e71-276c5b3e6078"),
// 				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
// 			},
// 			false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			h := &campaignHandler{
// 				db:            mockDB,
// 				notifyHandler: mockNotify,
// 				reqHandler:    mockReq,
// 			}

// 			ctx := context.Background()

// 			res := h.isRunable(ctx, tt.campaign)

// 			if reflect.DeepEqual(res, tt.expectRes) != true {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
