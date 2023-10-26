package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"
	qmqueue "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"

	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaigncall"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/campaigncallhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/campaign-manager.git/pkg/outplanhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		campaignType campaign.Type
		campaignName string
		detail       string

		serviceLevel int
		endHandle    campaign.EndHandle

		flowID         uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseFlow         *fmflow.Flow
		responseOutplan      *outplan.Outplan
		responseOutdial      *omoutdial.Outdial
		responseQueue        *qmqueue.Queue
		responseNextCampaign *campaign.Campaign

		responseCampaign *campaign.Campaign
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("dc55d2f4-c453-11ec-a621-8be3afeb72f9"),
			customerID:   uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
			campaignType: campaign.TypeCall,
			campaignName: "test name",
			detail:       "test detail",

			serviceLevel: 100,
			endHandle:    campaign.EndHandleStop,

			flowID:         uuid.FromStringOrNil("069bf798-72d9-11ee-b88e-63bfd43a9549"),
			outplanID:      uuid.FromStringOrNil("7d568cbe-2928-4dbe-b41f-3b2afad1b6e3"),
			outdialID:      uuid.FromStringOrNil("fb4d2a07-187d-4274-85bf-70186d902873"),
			queueID:        uuid.FromStringOrNil("b5e1c926-6753-42ca-be72-e4a521d40bed"),
			nextCampaignID: uuid.FromStringOrNil("c6da6162-dfc5-495d-a5af-e99efc9a97f7"),

			responseFlow: &fmflow.Flow{
				ID:       uuid.FromStringOrNil("069bf798-72d9-11ee-b88e-63bfd43a9549"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseOutplan: &outplan.Outplan{
				ID:       uuid.FromStringOrNil("7d568cbe-2928-4dbe-b41f-3b2afad1b6e3"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseOutdial: &omoutdial.Outdial{
				ID:       uuid.FromStringOrNil("fb4d2a07-187d-4274-85bf-70186d902873"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseQueue: &qmqueue.Queue{
				ID:       uuid.FromStringOrNil("b5e1c926-6753-42ca-be72-e4a521d40bed"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseNextCampaign: &campaign.Campaign{
				ID:       uuid.FromStringOrNil("c6da6162-dfc5-495d-a5af-e99efc9a97f7"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseCampaign: &campaign.Campaign{
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
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			h := &campaignHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				outplanHandler: mockOutplan,
			}
			ctx := context.Background()

			// validate
			if tt.flowID != uuid.Nil {
				mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			}
			if tt.outplanID != uuid.Nil {
				mockOutplan.EXPECT().Get(ctx, tt.outplanID).Return(tt.responseOutplan, nil)
			}
			if tt.outdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)
			}
			if tt.queueID != uuid.Nil {
				mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			}
			if tt.nextCampaignID != uuid.Nil {
				mockDB.EXPECT().CampaignGet(ctx, tt.nextCampaignID).Return(tt.responseNextCampaign, nil)
			}

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
				tt.serviceLevel,
				tt.endHandle,
				tt.flowID,
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
			mockReq.EXPECT().FlowV1FlowDelete(ctx, tt.responseCampaign.FlowID).Return(&fmflow.Flow{}, nil)

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
			"2020-10-10 03:30:17.000000",
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

		id             uuid.UUID
		flowID         uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseCampaign *campaign.Campaign
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("1951cdde-9d6f-4aeb-8e64-f56fc67a5a4e"),
			flowID:         uuid.FromStringOrNil("1dc8d7c4-72d9-11ee-afb6-f3345fb127a4"),
			outplanID:      uuid.FromStringOrNil("b4850013-42fe-4b18-9753-0e2871be2157"),
			outdialID:      uuid.FromStringOrNil("bc2031d2-53eb-4ee6-982e-b08ec0ffbde6"),
			queueID:        uuid.FromStringOrNil("12f560a9-9aed-4b5a-b748-06b6fe146ae4"),
			nextCampaignID: uuid.FromStringOrNil("1de84dca-72d9-11ee-a479-f739bc72acc2"),

			responseCampaign: &campaign.Campaign{
				ID:         uuid.FromStringOrNil("1951cdde-9d6f-4aeb-8e64-f56fc67a5a4e"),
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
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			h := &campaignHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				outplanHandler: mockOutplan,
			}

			ctx := context.Background()

			// validate
			mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(&fmflow.Flow{TMDelete: dbhandler.DefaultTimeStamp}, nil)
			mockOutplan.EXPECT().Get(ctx, tt.outplanID).Return(&outplan.Outplan{TMDelete: dbhandler.DefaultTimeStamp}, nil)
			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(&omoutdial.Outdial{TMDelete: dbhandler.DefaultTimeStamp}, nil)
			mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(&qmqueue.Queue{TMDelete: dbhandler.DefaultTimeStamp}, nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.nextCampaignID).Return(&campaign.Campaign{TMDelete: dbhandler.DefaultTimeStamp}, nil)

			mockDB.EXPECT().CampaignUpdateResourceInfo(ctx, tt.id, tt.flowID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(nil)

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCampaign.CustomerID, campaign.EventTypeCampaignUpdated, tt.responseCampaign)

			res, err := h.UpdateResourceInfo(ctx, tt.id, tt.flowID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.responseCampaign) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.responseCampaign, res)
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
				mockReq.EXPECT().CampaignV1CampaignExecute(ctx, tt.id, 1000).Return(nil)
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

			mockDB.EXPECT().CampaignUpdateExecute(ctx, tt.id, campaign.ExecuteStop).Return(nil)

			// isstoppable
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime())
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

func Test_isValidOutdialID(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		outdialID uuid.UUID

		responseOutdial *omoutdial.Outdial
	}{
		{
			"normal",

			uuid.FromStringOrNil("fbf0b198-6cfd-11ee-b521-37013565562b"),
			uuid.FromStringOrNil("caeec970-6cfa-11ee-8922-db4df7f7be10"),

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("caeec970-6cfa-11ee-8922-db4df7f7be10"),
				CampaignID: uuid.FromStringOrNil("fbf0b198-6cfd-11ee-b521-37013565562b"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
		{
			"campaign id is nil",

			uuid.Nil,
			uuid.FromStringOrNil("0c29e430-6cfe-11ee-a345-a3447618246d"),

			&omoutdial.Outdial{
				ID:       uuid.FromStringOrNil("0c29e430-6cfe-11ee-a345-a3447618246d"),
				TMDelete: dbhandler.DefaultTimeStamp,
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

			mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)

			if res := h.isValidOutdialID(ctx, tt.id, tt.outdialID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}

		})
	}
}

func Test_isValidOutplanID(t *testing.T) {

	tests := []struct {
		name string

		outplanID uuid.UUID

		responseOutplan *outplan.Outplan
	}{
		{
			"normal",

			uuid.FromStringOrNil("5feb6e54-6cfe-11ee-bc10-03bb387f94b5"),

			&outplan.Outplan{
				ID:       uuid.FromStringOrNil("5feb6e54-6cfe-11ee-bc10-03bb387f94b5"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
		{
			"outplan id is nil",

			uuid.Nil,

			nil,
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
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
				outplanHandler:      mockOutplan,
			}
			ctx := context.Background()

			if tt.outplanID != uuid.Nil {
				mockOutplan.EXPECT().Get(ctx, tt.outplanID).Return(tt.responseOutplan, nil)
			}

			if res := h.isValidOutplanID(ctx, tt.outplanID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_isValidQueueID(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID

		responseQueue *qmqueue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("97519ade-6cff-11ee-8f11-ab52f82847cb"),

			&qmqueue.Queue{
				ID:       uuid.FromStringOrNil("97519ade-6cff-11ee-8f11-ab52f82847cb"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
		{
			"queue id is nil",

			uuid.Nil,

			nil,
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

			if tt.queueID != uuid.Nil {
				mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			}

			if res := h.isValidQueueID(ctx, tt.queueID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_isValidNextCampaignID(t *testing.T) {

	tests := []struct {
		name string

		nextCampaignID uuid.UUID

		responseCampaign *campaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("b395c760-6cff-11ee-9dc5-c7265f1e5d16"),

			&campaign.Campaign{
				ID:       uuid.FromStringOrNil("b395c760-6cff-11ee-9dc5-c7265f1e5d16"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
		{
			"next campaign id is nil",

			uuid.Nil,

			nil,
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

			if tt.nextCampaignID != uuid.Nil {
				mockDB.EXPECT().CampaignGet(ctx, tt.nextCampaignID).Return(tt.responseCampaign, nil)
			}

			if res := h.isValidNextCampaignID(ctx, tt.nextCampaignID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_updateResources(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		outdialID uuid.UUID

		responseOutdial *omoutdial.Outdial
	}{
		{
			name: "normal",

			id:        uuid.FromStringOrNil("2652d520-72de-11ee-85a1-b31c21b06322"),
			outdialID: uuid.FromStringOrNil("268701ba-72de-11ee-a0fd-435a39acbd16"),

			responseOutdial: &omoutdial.Outdial{
				ID:       uuid.FromStringOrNil("5623f40c-6d00-11ee-8d48-c715083940ba"),
				TMDelete: dbhandler.DefaultTimeStamp,
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

			if tt.outdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialUpdateCampaignID(ctx, tt.outdialID, tt.id).Return(tt.responseOutdial, nil)
			}

			if res := h.updateResources(ctx, tt.id, tt.outdialID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_validateResources(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		flowID         uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseFlow         *fmflow.Flow
		responseOutplan      *outplan.Outplan
		responseOutdial      *omoutdial.Outdial
		responseQueue        *qmqueue.Queue
		responseNextCampaign *campaign.Campaign
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("aef448ca-6d00-11ee-a31d-4f74d98353e1"),
			flowID:         uuid.FromStringOrNil("74801612-72da-11ee-8394-db4cf6a52125"),
			outplanID:      uuid.FromStringOrNil("af229982-6d00-11ee-b767-5b0d2ca26cb4"),
			outdialID:      uuid.FromStringOrNil("af4a61c4-6d00-11ee-a129-1f00c9df4919"),
			queueID:        uuid.FromStringOrNil("af755fdc-6d00-11ee-98b2-275890b0ec69"),
			nextCampaignID: uuid.FromStringOrNil("afa1590c-6d00-11ee-9168-635f279cf425"),

			responseFlow: &fmflow.Flow{
				ID:       uuid.FromStringOrNil("74801612-72da-11ee-8394-db4cf6a52125"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseOutplan: &outplan.Outplan{
				ID:       uuid.FromStringOrNil("af229982-6d00-11ee-b767-5b0d2ca26cb4"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseOutdial: &omoutdial.Outdial{
				ID:       uuid.FromStringOrNil("5623f40c-6d00-11ee-8d48-c715083940ba"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseQueue: &qmqueue.Queue{
				ID:       uuid.FromStringOrNil("af755fdc-6d00-11ee-98b2-275890b0ec69"),
				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseNextCampaign: &campaign.Campaign{
				ID:       uuid.FromStringOrNil("afa1590c-6d00-11ee-9168-635f279cf425"),
				TMDelete: dbhandler.DefaultTimeStamp,
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
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			h := &campaignHandler{
				db:                  mockDB,
				notifyHandler:       mockNotify,
				reqHandler:          mockReq,
				campaigncallHandler: mockCampaigncall,
				outplanHandler:      mockOutplan,
			}
			ctx := context.Background()

			if tt.flowID != uuid.Nil {
				mockReq.EXPECT().FlowV1FlowGet(ctx, tt.flowID).Return(tt.responseFlow, nil)
			}

			if tt.outplanID != uuid.Nil {
				mockOutplan.EXPECT().Get(ctx, tt.outplanID).Return(tt.responseOutplan, nil)
			}

			if tt.outdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(tt.responseOutdial, nil)
			}

			if tt.queueID != uuid.Nil {
				mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			}

			if tt.nextCampaignID != uuid.Nil {
				mockDB.EXPECT().CampaignGet(ctx, tt.nextCampaignID).Return(tt.responseNextCampaign, nil)
			}

			if res := h.validateResources(ctx, tt.id, tt.flowID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}
