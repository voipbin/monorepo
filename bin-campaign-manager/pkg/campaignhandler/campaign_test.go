package campaignhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	omoutdial "monorepo/bin-outdial-manager/models/outdial"

	qmqueue "monorepo/bin-queue-manager/models/queue"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
	"monorepo/bin-campaign-manager/pkg/outplanhandler"
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

			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			serviceLevel: 100,
			endHandle:    campaign.EndHandleStop,

			outplanID:      uuid.FromStringOrNil("7d568cbe-2928-4dbe-b41f-3b2afad1b6e3"),
			outdialID:      uuid.FromStringOrNil("fb4d2a07-187d-4274-85bf-70186d902873"),
			queueID:        uuid.FromStringOrNil("b5e1c926-6753-42ca-be72-e4a521d40bed"),
			nextCampaignID: uuid.FromStringOrNil("c6da6162-dfc5-495d-a5af-e99efc9a97f7"),

			responseOutplan: &outplan.Outplan{
				ID:         uuid.FromStringOrNil("7d568cbe-2928-4dbe-b41f-3b2afad1b6e3"),
				CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseOutdial: &omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("fb4d2a07-187d-4274-85bf-70186d902873"),
				CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseQueue: &qmqueue.Queue{
				ID:         uuid.FromStringOrNil("b5e1c926-6753-42ca-be72-e4a521d40bed"),
				CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseNextCampaign: &campaign.Campaign{
				ID:         uuid.FromStringOrNil("c6da6162-dfc5-495d-a5af-e99efc9a97f7"),
				CustomerID: uuid.FromStringOrNil("6634faca-f71b-40e5-97f4-dc393107aace"),
				TMDelete:   dbhandler.DefaultTimeStamp,
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

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.customerID, fmflow.TypeCampaign, "", "", gomock.Any(), true).Return(&fmflow.Flow{}, nil)
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
		campaignType campaign.Type
		serviceLevel int
		endHandle    campaign.EndHandle

		response  *campaign.Campaign
		expectRes *campaign.Campaign
	}{
		{
			name: "normal",

			id:           uuid.FromStringOrNil("dc1a10c1-65db-46a6-8fbd-07cf3113bac0"),
			campaignName: "update name",
			detail:       "update detail",
			campaignType: campaign.TypeCall,
			serviceLevel: 100,
			endHandle:    campaign.EndHandleContinue,

			response: &campaign.Campaign{
				ID:         uuid.FromStringOrNil("dc1a10c1-65db-46a6-8fbd-07cf3113bac0"),
				CustomerID: uuid.FromStringOrNil("1973d7a7-0a06-4be2-b855-73565b136f9e"),
			},
			expectRes: &campaign.Campaign{
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

			mockDB.EXPECT().CampaignUpdateBasicInfo(ctx, tt.id, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle).Return(nil)
			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)

			res, err := h.UpdateBasicInfo(ctx, tt.id, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle)
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
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response *campaign.Campaign
	}{
		{
			name: "test normal",

			id:             uuid.FromStringOrNil("1951cdde-9d6f-4aeb-8e64-f56fc67a5a4e"),
			outplanID:      uuid.FromStringOrNil("b4850013-42fe-4b18-9753-0e2871be2157"),
			outdialID:      uuid.FromStringOrNil("bc2031d2-53eb-4ee6-982e-b08ec0ffbde6"),
			queueID:        uuid.FromStringOrNil("12f560a9-9aed-4b5a-b748-06b6fe146ae4"),
			nextCampaignID: uuid.FromStringOrNil("cc7a0346-7ccb-11ee-a1fb-633ebf371fc3"),

			response: &campaign.Campaign{
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
			mockOutplan := outplanhandler.NewMockOutplanHandler(mc)
			h := &campaignHandler{
				db:             mockDB,
				notifyHandler:  mockNotify,
				reqHandler:     mockReq,
				outplanHandler: mockOutplan,
			}

			ctx := context.Background()

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)

			// validate
			if tt.outplanID != uuid.Nil {
				mockOutplan.EXPECT().Get(ctx, tt.outplanID).Return(&outplan.Outplan{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)
			}
			if tt.outdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialGet(ctx, tt.outdialID).Return(&omoutdial.Outdial{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)
			}
			if tt.queueID != uuid.Nil {
				mockReq.EXPECT().QueueV1QueueGet(ctx, tt.queueID).Return(&qmqueue.Queue{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)
			}
			if tt.nextCampaignID != uuid.Nil {
				mockDB.EXPECT().CampaignGet(ctx, tt.nextCampaignID).Return(&campaign.Campaign{CustomerID: tt.response.CustomerID, TMDelete: dbhandler.DefaultTimeStamp}, nil)

			}

			mockDB.EXPECT().CampaignUpdateResourceInfo(ctx, tt.id, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(nil)

			tmpActions, err := h.createFlowActions(ctx, tt.response.Actions, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			mockReq.EXPECT().FlowV1FlowUpdateActions(ctx, tt.response.FlowID, tmpActions).Return(&fmflow.Flow{}, nil)

			mockDB.EXPECT().CampaignGet(ctx, tt.id).Return(tt.response, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.response.CustomerID, campaign.EventTypeCampaignUpdated, tt.response)

			res, err := h.UpdateResourceInfo(ctx, tt.id, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
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
		{
			"has valid queue info",

			uuid.FromStringOrNil("ffce8382-cbd0-11ec-9cfd-af33d5b4a740"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&campaign.Campaign{
				ID:         uuid.FromStringOrNil("ffce8382-cbd0-11ec-9cfd-af33d5b4a740"),
				CustomerID: uuid.FromStringOrNil("fffd05d6-cbd0-11ec-85f8-c79ba4d71e60"),
				FlowID:     uuid.FromStringOrNil("0027a886-cbd1-11ec-8440-137d167ffeb1"),
				QueueID:    uuid.FromStringOrNil("0054fcf0-cbd1-11ec-978d-9b83e6ca7ad6"),
			},
			&fmflow.Flow{
				ID: uuid.FromStringOrNil("0027a886-cbd1-11ec-8440-137d167ffeb1"),
			},

			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
				{
					Type:   fmaction.TypeQueueJoin,
					Option: []byte(`{"queue_id":"0054fcf0-cbd1-11ec-978d-9b83e6ca7ad6"}`),
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
			mockReq.EXPECT().FlowV1FlowUpdateActions(ctx, tt.response.FlowID, tt.expectActions).Return(tt.responseFlow, nil)
			mockDB.EXPECT().CampaignUpdateActions(ctx, tt.id, tt.actions)
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

		outdialID  uuid.UUID
		campaignID uuid.UUID
		customerID uuid.UUID

		responseOutdial *omoutdial.Outdial
	}{
		{
			name: "normal",

			outdialID:  uuid.FromStringOrNil("caeec970-6cfa-11ee-8922-db4df7f7be10"),
			campaignID: uuid.FromStringOrNil("fbf0b198-6cfd-11ee-b521-37013565562b"),
			customerID: uuid.FromStringOrNil("d87b08d8-7614-11ee-bc88-3f7993d217a7"),

			responseOutdial: &omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("caeec970-6cfa-11ee-8922-db4df7f7be10"),
				CustomerID: uuid.FromStringOrNil("d87b08d8-7614-11ee-bc88-3f7993d217a7"),
				CampaignID: uuid.FromStringOrNil("fbf0b198-6cfd-11ee-b521-37013565562b"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
		{
			name: "campaign id is nil",

			outdialID:  uuid.FromStringOrNil("0c29e430-6cfe-11ee-a345-a3447618246d"),
			campaignID: uuid.Nil,
			customerID: uuid.FromStringOrNil("bc8a2ba8-7615-11ee-9eac-fb9471a8b634"),

			responseOutdial: &omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("0c29e430-6cfe-11ee-a345-a3447618246d"),
				CustomerID: uuid.FromStringOrNil("bc8a2ba8-7615-11ee-9eac-fb9471a8b634"),
				TMDelete:   dbhandler.DefaultTimeStamp,
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

			if res := h.isValidOutdialID(ctx, tt.outdialID, tt.campaignID, tt.customerID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}

		})
	}
}

func Test_isValidOutplanID(t *testing.T) {

	tests := []struct {
		name string

		outplanID  uuid.UUID
		customerID uuid.UUID

		responseOutplan *outplan.Outplan
	}{
		{
			name: "normal",

			outplanID:  uuid.FromStringOrNil("5feb6e54-6cfe-11ee-bc10-03bb387f94b5"),
			customerID: uuid.FromStringOrNil("2c806932-7615-11ee-b345-3f3d8312ef2b"),

			responseOutplan: &outplan.Outplan{
				ID:         uuid.FromStringOrNil("5feb6e54-6cfe-11ee-bc10-03bb387f94b5"),
				CustomerID: uuid.FromStringOrNil("2c806932-7615-11ee-b345-3f3d8312ef2b"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
		{
			name: "outplan id is nil",

			outplanID:  uuid.Nil,
			customerID: uuid.FromStringOrNil("2c806932-7615-11ee-b345-3f3d8312ef2b"),

			responseOutplan: nil,
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

			if res := h.isValidOutplanID(ctx, tt.outplanID, tt.customerID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_isValidQueueID(t *testing.T) {

	tests := []struct {
		name string

		queueID    uuid.UUID
		customerID uuid.UUID

		responseQueue *qmqueue.Queue
	}{
		{
			name: "normal",

			queueID:    uuid.FromStringOrNil("97519ade-6cff-11ee-8f11-ab52f82847cb"),
			customerID: uuid.FromStringOrNil("2cebe0e0-7615-11ee-aab6-272f6a805a85"),

			responseQueue: &qmqueue.Queue{
				ID:         uuid.FromStringOrNil("97519ade-6cff-11ee-8f11-ab52f82847cb"),
				CustomerID: uuid.FromStringOrNil("2cebe0e0-7615-11ee-aab6-272f6a805a85"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
		{
			name: "queue id is nil",

			queueID:    uuid.Nil,
			customerID: uuid.Nil,

			responseQueue: nil,
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

			if res := h.isValidQueueID(ctx, tt.queueID, tt.customerID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_isValidNextCampaignID(t *testing.T) {

	tests := []struct {
		name string

		nextCampaignID uuid.UUID
		customerID     uuid.UUID

		responseCampaign *campaign.Campaign
	}{
		{
			name: "normal",

			nextCampaignID: uuid.FromStringOrNil("b395c760-6cff-11ee-9dc5-c7265f1e5d16"),
			customerID:     uuid.FromStringOrNil("2d179f8c-7615-11ee-b87b-03f5ec3a4037"),

			responseCampaign: &campaign.Campaign{
				ID:         uuid.FromStringOrNil("b395c760-6cff-11ee-9dc5-c7265f1e5d16"),
				CustomerID: uuid.FromStringOrNil("2d179f8c-7615-11ee-b87b-03f5ec3a4037"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
		},
		{
			name: "next campaign id is nil",

			nextCampaignID: uuid.Nil,
			customerID:     uuid.FromStringOrNil("450cfc8a-7616-11ee-bcb0-7b4c6cb71529"),

			responseCampaign: nil,
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

			if res := h.isValidNextCampaignID(ctx, tt.nextCampaignID, tt.customerID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_updateReferencedResources(t *testing.T) {

	tests := []struct {
		name string

		campaign *campaign.Campaign

		responseOutdial *omoutdial.Outdial
	}{
		{
			name: "normal",

			campaign: &campaign.Campaign{
				ID:             uuid.FromStringOrNil("55c70eb8-6d00-11ee-af57-2f785264f30a"),
				OutplanID:      uuid.FromStringOrNil("55f43ff0-6d00-11ee-bbf1-97a90f12ce6b"),
				OutdialID:      uuid.FromStringOrNil("5623f40c-6d00-11ee-8d48-c715083940ba"),
				QueueID:        uuid.FromStringOrNil("56545c96-6d00-11ee-955f-af50e79460c9"),
				NextCampaignID: uuid.FromStringOrNil("60613bc8-6d00-11ee-ac28-6377edcc4a2f"),
			},

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

			if tt.campaign.OutdialID != uuid.Nil {
				mockReq.EXPECT().OutdialV1OutdialUpdateCampaignID(ctx, tt.campaign.OutdialID, tt.campaign.ID).Return(tt.responseOutdial, nil)
			}

			if res := h.updateReferencedResources(ctx, tt.campaign); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}

func Test_validateResources(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseOutplan      *outplan.Outplan
		responseOutdial      *omoutdial.Outdial
		responseQueue        *qmqueue.Queue
		responseNextCampaign *campaign.Campaign
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("aef448ca-6d00-11ee-a31d-4f74d98353e1"),
			customerID:     uuid.FromStringOrNil("2d3d77ac-7615-11ee-817f-c75f8810ad99"),
			outplanID:      uuid.FromStringOrNil("af229982-6d00-11ee-b767-5b0d2ca26cb4"),
			outdialID:      uuid.FromStringOrNil("af4a61c4-6d00-11ee-a129-1f00c9df4919"),
			queueID:        uuid.FromStringOrNil("af755fdc-6d00-11ee-98b2-275890b0ec69"),
			nextCampaignID: uuid.FromStringOrNil("afa1590c-6d00-11ee-9168-635f279cf425"),

			responseOutplan: &outplan.Outplan{
				ID:         uuid.FromStringOrNil("af229982-6d00-11ee-b767-5b0d2ca26cb4"),
				CustomerID: uuid.FromStringOrNil("2d3d77ac-7615-11ee-817f-c75f8810ad99"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseOutdial: &omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("5623f40c-6d00-11ee-8d48-c715083940ba"),
				CustomerID: uuid.FromStringOrNil("2d3d77ac-7615-11ee-817f-c75f8810ad99"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseQueue: &qmqueue.Queue{
				ID:         uuid.FromStringOrNil("af755fdc-6d00-11ee-98b2-275890b0ec69"),
				CustomerID: uuid.FromStringOrNil("2d3d77ac-7615-11ee-817f-c75f8810ad99"),
				TMDelete:   dbhandler.DefaultTimeStamp,
			},
			responseNextCampaign: &campaign.Campaign{
				ID:         uuid.FromStringOrNil("afa1590c-6d00-11ee-9168-635f279cf425"),
				CustomerID: uuid.FromStringOrNil("2d3d77ac-7615-11ee-817f-c75f8810ad99"),
				TMDelete:   dbhandler.DefaultTimeStamp,
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

			if res := h.validateResources(ctx, tt.id, tt.customerID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID); res != true {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}
