package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/campaigncallhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		campaignType campaign.Type
		campaignName string
		detail       string

		actions      []fmaction.Action
		serviceLevel int
		endHandle    campaign.EndHandle

		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseCampaign *campaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("dc55d2f4-c453-11ec-a621-8be3afeb72f9"),
			uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
			campaign.TypeCall,
			"test name",
			"test detail",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			100,
			campaign.EndHandleStop,

			uuid.FromStringOrNil("7d568cbe-2928-4dbe-b41f-3b2afad1b6e3"),
			uuid.FromStringOrNil("fb4d2a07-187d-4274-85bf-70186d902873"),
			uuid.FromStringOrNil("b5e1c926-6753-42ca-be72-e4a521d40bed"),
			uuid.FromStringOrNil("c6da6162-dfc5-495d-a5af-e99efc9a97f7"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("dc55d2f4-c453-11ec-a621-8be3afeb72f9"),
				CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
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

			mockReq.EXPECT().FMV1FlowCreate(ctx, tt.customerID, fmflow.TypeCampaign, "", "", gomock.Any(), true).Return(&fmflow.Flow{}, nil)
			mockDB.EXPECT().CampaignCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, gomock.Any()).Return(tt.responseCampaign, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaign.CustomerID, campaign.EventTypeCampaignCreated, tt.responseCampaign)

			_, err := h.Create(
				ctx,
				tt.id,
				tt.customerID,
				tt.campaignType,
				tt.campaignName,
				tt.detail,
				tt.actions,
				tt.serviceLevel,
				tt.endHandle,
				tt.outplanID,
				tt.outdialID,
				tt.queueID,
				tt.nextCampaignID,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCampaign *campaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("ef3feb86-db79-4dab-a55d-41d65a231c10"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("ef3feb86-db79-4dab-a55d-41d65a231c10"),
				CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
				FlowID:     uuid.FromStringOrNil("60e0f90a-db73-4aaf-add8-6b7cd8edc82c"),
				Status:     campaign.StatusStop,
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
			mockDB.EXPECT().CampaignDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, gomock.Any()).Return(tt.responseCampaign, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaign.CustomerID, campaign.EventTypeCampaignDeleted, tt.responseCampaign)
			mockReq.EXPECT().FMV1FlowDelete(ctx, tt.responseCampaign.FlowID).Return(&fmflow.Flow{}, nil)

			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsByCustomerID(t *testing.T) {

	tests := []struct {
		name       string
		customerID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			"test normal",
			uuid.FromStringOrNil("938cdf96-7f4c-11ec-94d3-8ba7d397d7fb"),
			"2020-10-10T03:30:17.000000",
			10,
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
			mockDB.EXPECT().CampaignGetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit).Return(nil, nil)

			_, err := h.GetsByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		campaignName string
		detail       string

		response  *campaign.Campaign
		expectRes *campaign.Campaign
	}{
		{
			"test normal",

			uuid.FromStringOrNil("dc1a10c1-65db-46a6-8fbd-07cf3113bac0"),
			"update name",
			"update detail",

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("dc1a10c1-65db-46a6-8fbd-07cf3113bac0"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
			},
			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("dc1a10c1-65db-46a6-8fbd-07cf3113bac0"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
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

			mockDB.EXPECT().CampaignUpdateBasicInfo(ctx, tt.id, tt.campaignName, tt.detail).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)

			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.campaignName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_UpdateResourceInfo(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		outplanID uuid.UUID
		outdialID uuid.UUID
		queueID   uuid.UUID

		response *campaign.Campaign
	}{
		{
			"test normal",

			uuid.FromStringOrNil("1951cdde-9d6f-4aeb-8e64-f56fc67a5a4e"),
			uuid.FromStringOrNil("b4850013-42fe-4b18-9753-0e2871be2157"),
			uuid.FromStringOrNil("bc2031d2-53eb-4ee6-982e-b08ec0ffbde6"),
			uuid.FromStringOrNil("12f560a9-9aed-4b5a-b748-06b6fe146ae4"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("1951cdde-9d6f-4aeb-8e64-f56fc67a5a4e"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				Actions: []fmaction.Action{
					{
						Type: fmaction.TypeAnswer,
					},
				},
				FlowID: uuid.FromStringOrNil("f52090d7-7325-418e-bacd-b4a82692f6b5"),
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

			mockDB.EXPECT().CampaignUpdateResourceInfo(ctx, tt.id, tt.outplanID, tt.outdialID, tt.queueID).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)

			tmpActions, err := h.createFlowActions(ctx, tt.response.Actions, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			mockReq.EXPECT().FMV1FlowUpdateActions(ctx, tt.response.FlowID, tmpActions).Return(&fmflow.Flow{}, nil)

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)

			res, err := h.UpdateResourceInfo(ctx, tt.id, tt.outplanID, tt.outdialID, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.response, res)
			}
		})
	}
}

func Test_UpdateNextCampaignID(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		nextCampaignID uuid.UUID

		response *campaign.Campaign
	}{
		{
			"test normal",

			uuid.FromStringOrNil("bfd09fa5-4c2c-46ea-aee9-a01a386e154a"),
			uuid.FromStringOrNil("2861e6ce-844b-42e5-bc5a-625c2123f662"),

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("bfd09fa5-4c2c-46ea-aee9-a01a386e154a"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
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

			mockDB.EXPECT().CampaignUpdateNextCampaignID(ctx, tt.id, tt.nextCampaignID).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)
			if tt.response.Execute == campaign.ExecuteRun {
				mockReq.EXPECT().CAV1CampaignExecute(ctx, tt.id, 1000).Return(nil)
			}

			res, err := h.UpdateNextCampaignID(ctx, tt.id, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.response, res)
			}
		})
	}
}

func Test_UpdateServiceLevel(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		serviceLevel int

		response *campaign.Campaign
	}{
		{
			"test normal",

			uuid.FromStringOrNil("d4e36568-c3f4-11ec-9151-8357f70ffbc4"),
			100,

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("d4e36568-c3f4-11ec-9151-8357f70ffbc4"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
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

			mockDB.EXPECT().CampaignUpdateServiceLevel(ctx, tt.id, tt.serviceLevel).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)

			res, err := h.UpdateServiceLevel(ctx, tt.id, tt.serviceLevel)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.response, res)
			}
		})
	}
}

func Test_UpdateActions(t *testing.T) {

	tests := []struct {
		name string

		id      uuid.UUID
		actions []fmaction.Action

		response      *campaign.Campaign
		responseFlow  *fmflow.Flow
		expectActions []fmaction.Action
	}{
		{
			"test normal",

			uuid.FromStringOrNil("d4e36568-c3f4-11ec-9151-8357f70ffbc4"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("d4e36568-c3f4-11ec-9151-8357f70ffbc4"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
				FlowID:     uuid.FromStringOrNil("8840b1c4-c3f5-11ec-8961-bbf3aed170d6"),
			},
			&fmflow.Flow{
				ID: uuid.FromStringOrNil("8840b1c4-c3f5-11ec-8961-bbf3aed170d6"),
			},

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
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
			mockReq.EXPECT().FMV1FlowUpdateActions(ctx, tt.response.FlowID, tt.expectActions).Return(tt.responseFlow, nil)
			mockDB.EXPECT().CampaignUpdateActions(ctx, tt.id, tt.expectActions)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)

			res, err := h.UpdateActions(ctx, tt.id, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.response) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.response, res)
			}
		})
	}
}

func Test_createFlowActions(t *testing.T) {

	tests := []struct {
		name string

		actions []fmaction.Action
		queueID uuid.UUID

		expectRes []fmaction.Action
	}{
		{
			"has no queue id",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			uuid.Nil,

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
		},

		{
			"has queue id",

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("8de92286-c3f6-11ec-bade-ff667a1ea0af"),

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeQueueJoin,
					Option: []byte(`{"queue_id":"8de92286-c3f6-11ec-bade-ff667a1ea0af"}`),
				},
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

			res, err := h.createFlowActions(ctx, tt.actions, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_updateExecuteStopAndCampaignIsStoppable(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *campaign.Campaign
	}{
		{
			"campaign is stoppable",

			uuid.FromStringOrNil("c58bf240-c3f6-11ec-99bd-83480d2667d8"),

			&campaign.Campaign{
				ID:      uuid.FromStringOrNil("c58bf240-c3f6-11ec-99bd-83480d2667d8"),
				Execute: campaign.ExecuteStop,
				Status:  campaign.StatusStopping,
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
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignUpdateExecute(ctx, tt.id, campaign.ExecuteStop).Return(nil)

			// isstoppable
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockCampaigncall.EXPECT().GetsOngoingByCampaignID(ctx, tt.id, gomock.Any(), uint64(1)).Return([]*campaigncall.Campaigncall{}, nil)

			// updateStatusStop
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockDB.EXPECT().CampaignUpdateStatus(ctx, tt.id, campaign.StatusStop).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignStatusStop, tt.response)

			if err := h.updateExecuteStop(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_updateExecuteStopAndCampaignIsNotStoppable(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *campaign.Campaign
	}{
		{
			"campaign is not stoppable",

			uuid.FromStringOrNil("19cb128e-c3fa-11ec-b6ab-6f645ada73ce"),

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("19cb128e-c3fa-11ec-b6ab-6f645ada73ce"),
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
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignUpdateExecute(ctx, tt.id, campaign.ExecuteStop).Return(nil)

			// is stoppable
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)

			// UpdateStatusStopping
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockDB.EXPECT().CampaignUpdateStatus(ctx, tt.id, campaign.StatusStopping).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignStatusStopping, tt.response)

			if err := h.updateExecuteStop(ctx, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
