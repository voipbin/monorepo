package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	omoutdial "gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdial"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_OutdialCreate(t *testing.T) {

	tests := []struct {
		name string

		customer    *cscustomer.Customer
		campaignID  uuid.UUID
		outdialName string
		detail      string
		data        string

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("62b784eb-63e0-48e6-b6e1-2904eafd842d"),
			"test name",
			"test detail",
			"test data",

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("58515568-030d-4fcd-a11d-e606d439eaef"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Data:       "test data",
			},
			&omoutdial.WebhookMessage{
				ID:         uuid.FromStringOrNil("58515568-030d-4fcd-a11d-e606d439eaef"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
				Name:       "test",
				Detail:     "test detail",
				Data:       "test data",
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

			mockReq.EXPECT().OMV1OutdialCreate(gomock.Any(), tt.customer.ID, tt.campaignID, tt.outdialName, tt.detail, tt.data).Return(tt.response, nil)
			res, err := h.OutdialCreate(tt.customer, tt.campaignID, tt.outdialName, tt.detail, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialGets(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		pageToken string
		pageSize  uint64

		response  []omoutdial.Outdial
		expectRes []*omoutdial.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			"2020-10-20T01:00:00.995000",
			10,

			[]omoutdial.Outdial{
				{
					ID: uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
				},
				{
					ID: uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
				},
			},
			[]*omoutdial.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("ccda6eb2-0c5c-11eb-ae7e-a3ae4bcd3975"),
				},
				{
					ID: uuid.FromStringOrNil("d950aef4-0c5c-11eb-82dd-3b31d4ba2ea4"),
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

			mockReq.EXPECT().OMV1OutdialGetsByCustomerID(gomock.Any(), tt.customer.ID, tt.pageToken, tt.pageSize).Return(tt.response, nil)
			res, err := h.OutdialGetsByCustomerID(tt.customer, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialDelete(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		outdialID uuid.UUID

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			uuid.FromStringOrNil("92d41af7-4249-41a8-b86a-cb2ce21f214a"),

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("92d41af7-4249-41a8-b86a-cb2ce21f214a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdial.WebhookMessage{
				ID:         uuid.FromStringOrNil("92d41af7-4249-41a8-b86a-cb2ce21f214a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
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

			mockReq.EXPECT().OMV1OutdialGet(gomock.Any(), tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OMV1OutdialDelete(gomock.Any(), tt.outdialID).Return(tt.response, nil)

			res, err := h.OutdialDelete(tt.customer, tt.outdialID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialUpdate(t *testing.T) {

	tests := []struct {
		name        string
		customer    *cscustomer.Customer
		outdialID   uuid.UUID
		outdialName string
		detail      string

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
			"test name",
			"test detail",

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdial.WebhookMessage{
				ID:         uuid.FromStringOrNil("178f8cfa-b46f-4a66-aa95-85b9dd65500a"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
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

			mockReq.EXPECT().OMV1OutdialGet(gomock.Any(), tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OMV1OutdialUpdateBasicInfo(gomock.Any(), tt.outdialID, tt.outdialName, tt.detail).Return(tt.response, nil)
			res, err := h.OutdialUpdateBasicInfo(tt.customer, tt.outdialID, tt.outdialName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialUpdateCampaignID(t *testing.T) {

	tests := []struct {
		name       string
		customer   *cscustomer.Customer
		outdialID  uuid.UUID
		campaignID uuid.UUID

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("a7b05592-2d89-4440-a53d-a8dff4acc581"),
			uuid.FromStringOrNil("78f711a7-3b75-4c47-a796-cff180370aa1"),

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("a7b05592-2d89-4440-a53d-a8dff4acc581"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdial.WebhookMessage{
				ID:         uuid.FromStringOrNil("a7b05592-2d89-4440-a53d-a8dff4acc581"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
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

			mockReq.EXPECT().OMV1OutdialGet(gomock.Any(), tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OMV1OutdialUpdateCampaignID(gomock.Any(), tt.outdialID, tt.campaignID).Return(tt.response, nil)
			res, err := h.OutdialUpdateCampaignID(tt.customer, tt.outdialID, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_OutdialUpdateData(t *testing.T) {

	tests := []struct {
		name      string
		customer  *cscustomer.Customer
		outdialID uuid.UUID
		data      string

		response  *omoutdial.Outdial
		expectRes *omoutdial.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},

			uuid.FromStringOrNil("e46bbea3-4b82-4b11-a9bb-8be3e152ae92"),
			"test data",

			&omoutdial.Outdial{
				ID:         uuid.FromStringOrNil("e46bbea3-4b82-4b11-a9bb-8be3e152ae92"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
			},
			&omoutdial.WebhookMessage{
				ID:         uuid.FromStringOrNil("e46bbea3-4b82-4b11-a9bb-8be3e152ae92"),
				CustomerID: uuid.FromStringOrNil("1e7f44c4-7fff-11ec-98ef-c70700134988"),
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

			mockReq.EXPECT().OMV1OutdialGet(gomock.Any(), tt.outdialID).Return(tt.response, nil)
			mockReq.EXPECT().OMV1OutdialUpdateData(gomock.Any(), tt.outdialID, tt.data).Return(tt.response, nil)
			res, err := h.OutdialUpdateData(tt.customer, tt.outdialID, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}
