package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	caoutplan "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/outplan"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_OutplanCreate(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer

		outplanName string
		detail      string

		source *cmaddress.Address

		dialTimeout int
		tryInterval int

		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"test name",
			"test detail",
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			5,
			5,
			5,
			5,
			5,
			&caoutplan.Outplan{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
			},
			&caoutplan.WebhookMessage{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
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

			mockReq.EXPECT().CAV1OutplanCreate(gomock.Any(), tt.customer.ID, tt.outplanName, tt.detail, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(tt.response, nil)
			res, err := h.OutplanCreate(tt.customer, tt.outplanName, tt.detail, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount0, tt.maxTryCount0, tt.maxTryCount0, tt.maxTryCount4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanDelete(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		id       uuid.UUID

		responseOutplan *caoutplan.Outplan
		expectRes       *caoutplan.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("bcbcffb4-c640-11ec-bdab-03b1d679601d"),

			&caoutplan.Outplan{
				ID:         uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&caoutplan.WebhookMessage{
				ID: uuid.FromStringOrNil("5e602408-e819-4aa0-aac6-24072a224dff"),
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

			mockReq.EXPECT().CAV1OutplanGet(gomock.Any(), tt.id).Return(tt.responseOutplan, nil)
			mockReq.EXPECT().CAV1OutplanDelete(gomock.Any(), tt.id).Return(tt.responseOutplan, nil)
			res, err := h.OutplanDelete(tt.customer, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []caoutplan.Outplan
		expectRes []*caoutplan.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]caoutplan.Outplan{
				{
					ID: uuid.FromStringOrNil("6c3a509a-c641-11ec-97f3-97762ce0f584"),
				},
				{
					ID: uuid.FromStringOrNil("6c8f8c0e-c641-11ec-ae79-63f803cffc1f"),
				},
			},
			[]*caoutplan.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6c3a509a-c641-11ec-97f3-97762ce0f584"),
				},
				{
					ID: uuid.FromStringOrNil("6c8f8c0e-c641-11ec-ae79-63f803cffc1f"),
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

			mockReq.EXPECT().CAV1OutplanGetsByCustomerID(gomock.Any(), tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.OutplanGetsByCustomerID(tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanGet(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		outplanID uuid.UUID

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),

			&caoutplan.Outplan{
				ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&caoutplan.WebhookMessage{
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

			mockReq.EXPECT().CAV1OutplanGet(gomock.Any(), tt.outplanID).Return(tt.response, nil)
			res, err := h.OutplanGet(tt.customer, tt.outplanID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name        string
		customer    *cscustomer.Customer
		outplanID   uuid.UUID
		outplanName string
		detail      string

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
			"test name",
			"test detail",

			&caoutplan.Outplan{
				ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&caoutplan.WebhookMessage{
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

			mockReq.EXPECT().CAV1OutplanGet(gomock.Any(), tt.outplanID).Return(tt.response, nil)
			mockReq.EXPECT().CAV1OutplanUpdateBasicInfo(gomock.Any(), tt.outplanID, tt.outplanName, tt.detail).Return(tt.response, nil)
			res, err := h.OutplanUpdateBasicInfo(tt.customer, tt.outplanID, tt.outplanName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutplanUpdateDialInfo(t *testing.T) {

	tests := []struct {
		name         string
		customer     *cscustomer.Customer
		outplanID    uuid.UUID
		source       *cmaddress.Address
		dialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		response  *caoutplan.Outplan
		expectRes *caoutplan.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("451a473e-c643-11ec-93c4-0bd1b9b41f16"),
			&cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			5,
			5,
			5,
			5,
			5,

			&caoutplan.Outplan{
				ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&caoutplan.WebhookMessage{
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

			mockReq.EXPECT().CAV1OutplanGet(gomock.Any(), tt.outplanID).Return(tt.response, nil)
			mockReq.EXPECT().CAV1OutplanUpdateDialInfo(gomock.Any(), tt.outplanID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(tt.response, nil)
			res, err := h.OutplanUpdateDialInfo(tt.customer, tt.outplanID, tt.source, tt.dialTimeout, tt.tryInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
