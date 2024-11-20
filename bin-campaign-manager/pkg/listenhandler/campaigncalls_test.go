package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/campaigncallhandler"
)

func Test_v1CampaigncallsGet_campaignID(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken  string
		pageSize   uint64
		campaignID uuid.UUID

		responseCampaigns []*campaigncall.Campaigncall

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigncalls?page_token=2020-10-10%2003:30:17.000000&page_size=10&campaign_id=d448b208-c849-11ec-8ea9-130b51c62f3e",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10 03:30:17.000000",
			10,
			uuid.FromStringOrNil("d448b208-c849-11ec-8ea9-130b51c62f3e"),

			[]*campaigncall.Campaigncall{
				{
					ID: uuid.FromStringOrNil("d4852cba-c849-11ec-986c-e360df927fc5"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d4852cba-c849-11ec-986c-e360df927fc5","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				campaigncallHandler: mockCampaigncall,
			}

			mockCampaigncall.EXPECT().GetsByCampaignID(gomock.Any(), tt.campaignID, tt.pageToken, tt.pageSize).Return(tt.responseCampaigns, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1CampaigncallsGet_customerID(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken  string
		pageSize   uint64
		customerID uuid.UUID

		responseCampaigns []*campaigncall.Campaigncall

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/campaigncalls?page_token=2020-10-10%2003:30:17.000000&page_size=10&customer_id=a5b9fbf6-6e31-11ee-98a3-33e454942a48",
				Method: sock.RequestMethodGet,
			},

			"2020-10-10 03:30:17.000000",
			10,
			uuid.FromStringOrNil("a5b9fbf6-6e31-11ee-98a3-33e454942a48"),

			[]*campaigncall.Campaigncall{
				{
					ID: uuid.FromStringOrNil("a5fa0f84-6e31-11ee-a6f2-0bb6f14c4687"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"a5fa0f84-6e31-11ee-a6f2-0bb6f14c4687","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				campaigncallHandler: mockCampaigncall,
			}

			mockCampaigncall.EXPECT().GetsByCustomerID(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.responseCampaigns, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1CampaigncallsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaigncallID uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigncalls/5f7dd038-c84a-11ec-9943-936e5cfdeb4c",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("5f7dd038-c84a-11ec-9943-936e5cfdeb4c"),

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("5f7dd038-c84a-11ec-9943-936e5cfdeb4c"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5f7dd038-c84a-11ec-9943-936e5cfdeb4c","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				campaigncallHandler: mockCampaigncall,
			}

			mockCampaigncall.EXPECT().Get(gomock.Any(), tt.campaigncallID).Return(tt.responseCampaigncall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1CampaigncallsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaigncallID uuid.UUID

		responseCampaigncall *campaigncall.Campaigncall

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigncalls/ef345db4-d31a-11ee-b584-2f258c86723e",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("ef345db4-d31a-11ee-b584-2f258c86723e"),

			&campaigncall.Campaigncall{
				ID: uuid.FromStringOrNil("ef345db4-d31a-11ee-b584-2f258c86723e"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"ef345db4-d31a-11ee-b584-2f258c86723e","customer_id":"00000000-0000-0000-0000-000000000000","campaign_id":"00000000-0000-0000-0000-000000000000","outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","outdial_target_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","result":"","source":null,"destination":null,"destination_index":0,"try_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaigncall := campaigncallhandler.NewMockCampaigncallHandler(mc)

			h := &listenHandler{
				sockHandler:         mockSock,
				campaigncallHandler: mockCampaigncall,
			}

			mockCampaigncall.EXPECT().Delete(gomock.Any(), tt.campaigncallID).Return(tt.responseCampaigncall, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
