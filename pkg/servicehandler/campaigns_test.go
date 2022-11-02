package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_CampaignCreate(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer

		campaignName string
		detail       string

		campaignType   cacampaign.Type
		serviceLevel   int
		endHandle      cacampaign.EndHandle
		actions        []fmaction.Action
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"test name",
			"test detail",

			cacampaign.TypeCall,
			100,
			cacampaign.EndHandleStop,
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},
			uuid.FromStringOrNil("a44727da-c653-11ec-a0b7-a7d6b873b66d"),
			uuid.FromStringOrNil("a4aafd28-c653-11ec-9b79-47790e39b9be"),
			uuid.FromStringOrNil("a4e4ccce-c653-11ec-b64b-1b6af5c458a8"),
			uuid.FromStringOrNil("a51c8010-c653-11ec-953a-43eabdb60873"),

			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("c5edb1ce-c653-11ec-bb63-1f0413e1ebdc"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("c5edb1ce-c653-11ec-bb63-1f0413e1ebdc"),
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

			mockReq.EXPECT().CampaignV1CampaignCreate(ctx, uuid.Nil, tt.customer.ID, tt.campaignType, tt.campaignName, tt.detail, tt.serviceLevel, tt.endHandle, tt.actions, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(tt.response, nil)
			res, err := h.CampaignCreate(ctx, tt.customer, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle, tt.actions, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []cacampaign.Campaign
		expectRes []*cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]cacampaign.Campaign{
				{
					ID: uuid.FromStringOrNil("bf203708-c654-11ec-910b-63ef1793516d"),
				},
				{
					ID: uuid.FromStringOrNil("bf4959a8-c654-11ec-bc10-53da5a6de123"),
				},
			},
			[]*cacampaign.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("bf203708-c654-11ec-910b-63ef1793516d"),
				},
				{
					ID: uuid.FromStringOrNil("bf4959a8-c654-11ec-bc10-53da5a6de123"),
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

			mockReq.EXPECT().CampaignV1CampaignGetsByCustomerID(ctx, tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.CampaignGetsByCustomerID(ctx, tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignGet(t *testing.T) {

	tests := []struct {
		name       string
		customer   *cscustomer.Customer
		campaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			res, err := h.CampaignGet(ctx, tt.customer, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignDelete(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		id       uuid.UUID

		responseCampaign *cacampaign.Campaign
		expectRes        *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("bcbcffb4-c640-11ec-bdab-03b1d679601d"),

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("32d63a4e-c655-11ec-8288-2707fffc29b5"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("32d63a4e-c655-11ec-8288-2707fffc29b5"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.id).Return(tt.responseCampaign, nil)
			mockReq.EXPECT().CampaignV1CampaignDelete(ctx, tt.id).Return(tt.responseCampaign, nil)
			res, err := h.CampaignDelete(ctx, tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name         string
		customer     *cscustomer.Customer
		campaignID   uuid.UUID
		campaignName string
		detail       string

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
			"test name",
			"test detail",

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateBasicInfo(ctx, tt.campaignID, tt.campaignName, tt.detail).Return(tt.response, nil)
			res, err := h.CampaignUpdateBasicInfo(ctx, tt.customer, tt.campaignID, tt.campaignName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateStatus(t *testing.T) {

	tests := []struct {
		name       string
		customer   *cscustomer.Customer
		campaignID uuid.UUID
		status     cacampaign.Status

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
			cacampaign.StatusRun,

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateStatus(ctx, tt.campaignID, tt.status).Return(tt.response, nil)
			res, err := h.CampaignUpdateStatus(ctx, tt.customer, tt.campaignID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateServiceLevel(t *testing.T) {

	tests := []struct {
		name         string
		customer     *cscustomer.Customer
		campaignID   uuid.UUID
		serviceLevel int

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
			100,

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("6d1e3e5e-c655-11ec-bc77-cf50387b8fe7"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateServiceLevel(ctx, tt.campaignID, tt.serviceLevel).Return(tt.response, nil)
			res, err := h.CampaignUpdateServiceLevel(ctx, tt.customer, tt.campaignID, tt.serviceLevel)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateActions(t *testing.T) {

	tests := []struct {
		name       string
		customer   *cscustomer.Customer
		campaignID uuid.UUID
		actions    []fmaction.Action

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("eb889654-c655-11ec-a97a-636c4c1455d8"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("eb889654-c655-11ec-a97a-636c4c1455d8"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("eb889654-c655-11ec-a97a-636c4c1455d8"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateActions(ctx, tt.campaignID, tt.actions).Return(tt.response, nil)
			res, err := h.CampaignUpdateActions(ctx, tt.customer, tt.campaignID, tt.actions)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateResourceInfo(t *testing.T) {

	tests := []struct {
		name       string
		customer   *cscustomer.Customer
		campaignID uuid.UUID
		outplanID  uuid.UUID
		outdialID  uuid.UUID
		queueID    uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("6589627a-c6b6-11ec-80ec-eb94b8bc76e7"),
			uuid.FromStringOrNil("65c34a94-c6b6-11ec-b153-6be43b327a5e"),
			uuid.FromStringOrNil("65ef10ca-c6b6-11ec-93a6-af4ca7079371"),
			uuid.FromStringOrNil("661dcbc2-c6b6-11ec-934c-c3c128f1d3b9"),

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("6589627a-c6b6-11ec-80ec-eb94b8bc76e7"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("6589627a-c6b6-11ec-80ec-eb94b8bc76e7"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateResourceInfo(ctx, tt.campaignID, tt.outplanID, tt.outdialID, tt.queueID).Return(tt.response, nil)
			res, err := h.CampaignUpdateResourceInfo(ctx, tt.customer, tt.campaignID, tt.outplanID, tt.outdialID, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CampaignUpdateNextCampaignID(t *testing.T) {

	tests := []struct {
		name           string
		customer       *cscustomer.Customer
		campaignID     uuid.UUID
		nextCampaignID uuid.UUID

		response  *cacampaign.Campaign
		expectRes *cacampaign.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("916c14be-c6b6-11ec-83a5-8f67784590f9"),
			uuid.FromStringOrNil("919d20e0-c6b6-11ec-bdc6-9f571a70547e"),

			&cacampaign.Campaign{
				ID:         uuid.FromStringOrNil("916c14be-c6b6-11ec-83a5-8f67784590f9"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&cacampaign.WebhookMessage{
				ID: uuid.FromStringOrNil("916c14be-c6b6-11ec-83a5-8f67784590f9"),
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

			mockReq.EXPECT().CampaignV1CampaignGet(ctx, tt.campaignID).Return(tt.response, nil)
			mockReq.EXPECT().CampaignV1CampaignUpdateNextCampaignID(ctx, tt.campaignID, tt.nextCampaignID).Return(tt.response, nil)
			res, err := h.CampaignUpdateNextCampaignID(ctx, tt.customer, tt.campaignID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
