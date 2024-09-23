package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaign"
	"monorepo/bin-campaign-manager/pkg/campaignhandler"
)

func Test_v1CampaignsPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		id             uuid.UUID
		customerID     uuid.UUID
		campaignType   campaign.Type
		campaignName   string
		detail         string
		actions        []fmaction.Action
		serviceLevel   int
		endHandle      campaign.EndHandle
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigns",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"3653adb2-c454-11ec-8c9f-7bcd6924ee69","customer_id":"60f76478-c454-11ec-969b-13906a5eea9d","type":"call","name":"test name","detail":"test detail","actions":[{"type":"answer"}],"service_level":100,"end_handle":"stop","outplan_id":"612a9186-c454-11ec-8afa-4327f7ed3a4e","outdial_id":"615dce2a-c454-11ec-aefb-a32f5d072685","queue_id":"61893ca4-c454-11ec-a2a2-3f4cd3113b05","next_campaign_id":"61b3c7f8-c454-11ec-9d0e-07f30d37566d"}`),
			},

			uuid.FromStringOrNil("3653adb2-c454-11ec-8c9f-7bcd6924ee69"),
			uuid.FromStringOrNil("60f76478-c454-11ec-969b-13906a5eea9d"),
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

			uuid.FromStringOrNil("612a9186-c454-11ec-8afa-4327f7ed3a4e"),
			uuid.FromStringOrNil("615dce2a-c454-11ec-aefb-a32f5d072685"),
			uuid.FromStringOrNil("61893ca4-c454-11ec-a2a2-3f4cd3113b05"),
			uuid.FromStringOrNil("61b3c7f8-c454-11ec-9d0e-07f30d37566d"),

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("3653adb2-c454-11ec-8c9f-7bcd6924ee69"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3653adb2-c454-11ec-8c9f-7bcd6924ee69","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().Create(gomock.Any(), tt.id, tt.customerID, tt.campaignType, tt.campaignName, tt.detail, tt.actions, tt.serviceLevel, tt.endHandle, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageToken  string
		pageSize   uint64
		customerID uuid.UUID

		responseCampaigns []*campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigns?page_token=2020-10-10%2003:30:17.000000&page_size=10&customer_id=1a2f447a-c459-11ec-8299-636f031f01c1",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			"2020-10-10 03:30:17.000000",
			10,
			uuid.FromStringOrNil("1a2f447a-c459-11ec-8299-636f031f01c1"),

			[]*campaign.Campaign{
				{
					ID: uuid.FromStringOrNil("3653adb2-c454-11ec-8c9f-7bcd6924ee69"),
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"3653adb2-c454-11ec-8c9f-7bcd6924ee69","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().GetsByCustomerID(gomock.Any(), tt.customerID, tt.pageToken, tt.pageSize).Return(tt.responseCampaigns, nil)
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

func Test_v1CampaignsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigns/edb1a7ca-c459-11ec-b591-733bb55d7160",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("edb1a7ca-c459-11ec-b591-733bb55d7160"),

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("edb1a7ca-c459-11ec-b591-733bb55d7160"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"edb1a7ca-c459-11ec-b591-733bb55d7160","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().Get(gomock.Any(), tt.campaignID).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigns/5a797d38-c45a-11ec-95be-bb5e6cfb1d96",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("5a797d38-c45a-11ec-95be-bb5e6cfb1d96"),

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("5a797d38-c45a-11ec-95be-bb5e6cfb1d96"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5a797d38-c45a-11ec-95be-bb5e6cfb1d96","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().Delete(gomock.Any(), tt.campaignID).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID   uuid.UUID
		campaignName string
		detail       string
		campaignType campaign.Type
		serviceLevel int
		endHandle    campaign.EndHandle

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/campaigns/40b95d6c-c466-11ec-88ac-734fd1ce5539",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail","type":"call","service_level":100,"end_handle":"continue"}`),
			},

			campaignID:   uuid.FromStringOrNil("40b95d6c-c466-11ec-88ac-734fd1ce5539"),
			campaignName: "update name",
			detail:       "update detail",
			campaignType: campaign.TypeCall,
			serviceLevel: 100,
			endHandle:    campaign.EndHandleContinue,

			responseCampaign: &campaign.Campaign{
				ID: uuid.FromStringOrNil("40b95d6c-c466-11ec-88ac-734fd1ce5539"),
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"40b95d6c-c466-11ec-88ac-734fd1ce5539","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().UpdateBasicInfo(gomock.Any(), tt.campaignID, tt.campaignName, tt.detail, tt.campaignType, tt.serviceLevel, tt.endHandle).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDExecutePost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/campaigns/741b122e-c45a-11ec-9a3f-9ba245ef6dec/execute",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("741b122e-c45a-11ec-9a3f-9ba245ef6dec"),

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("741b122e-c45a-11ec-9a3f-9ba245ef6dec"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().Execute(gomock.Any(), tt.campaignID)
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

func Test_v1CampaignsIDStatus(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID
		status     campaign.Status

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"stopping",
			&sock.Request{
				URI:      "/v1/campaigns/088b70c0-c45b-11ec-b93c-87920bba8787/status",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"status":"run"}`),
			},

			uuid.FromStringOrNil("088b70c0-c45b-11ec-b93c-87920bba8787"),
			campaign.StatusRun,

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("088b70c0-c45b-11ec-b93c-87920bba8787"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"088b70c0-c45b-11ec-b93c-87920bba8787","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
		{
			"stop",
			&sock.Request{
				URI:      "/v1/campaigns/d26b0c58-c45a-11ec-b42d-3b261e615304/status",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"status":"stop"}`),
			},

			uuid.FromStringOrNil("d26b0c58-c45a-11ec-b42d-3b261e615304"),
			campaign.StatusStop,

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("d26b0c58-c45a-11ec-b42d-3b261e615304"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d26b0c58-c45a-11ec-b42d-3b261e615304","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().UpdateStatus(gomock.Any(), tt.campaignID, tt.status).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDServiceLevelPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID   uuid.UUID
		serviceLevel int

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"stopping",
			&sock.Request{
				URI:      "/v1/campaigns/088b70c0-c45b-11ec-b93c-87920bba8787/service_level",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"service_level":100}`),
			},

			uuid.FromStringOrNil("088b70c0-c45b-11ec-b93c-87920bba8787"),
			100,

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("088b70c0-c45b-11ec-b93c-87920bba8787"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"088b70c0-c45b-11ec-b93c-87920bba8787","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().UpdateServiceLevel(gomock.Any(), tt.campaignID, tt.serviceLevel).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDActionsPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID uuid.UUID
		actions    []fmaction.Action

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"stopping",
			&sock.Request{
				URI:      "/v1/campaigns/045cdfc4-c45c-11ec-915c-5b6e9c81d305/actions",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"actions":[{"type":"answer"}]}`),
			},

			uuid.FromStringOrNil("045cdfc4-c45c-11ec-915c-5b6e9c81d305"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("045cdfc4-c45c-11ec-915c-5b6e9c81d305"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"045cdfc4-c45c-11ec-915c-5b6e9c81d305","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().UpdateActions(gomock.Any(), tt.campaignID, tt.actions).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDResourceInfoPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID     uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			name: "stopping",
			request: &sock.Request{
				URI:      "/v1/campaigns/e74223b2-c6af-11ec-9f40-1f88a3e01636/resource_info",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"outplan_id":"010e228c-c6b0-11ec-87b5-7bb69124c874","outdial_id":"013945a2-c6b0-11ec-ba93-2300004f60a7","queue_id":"0169cb32-c6b0-11ec-b616-4fe9ae9e95da","next_campaign_id":"0468ac9c-7cce-11ee-9d09-7feca9bc6422"}`),
			},

			campaignID:     uuid.FromStringOrNil("e74223b2-c6af-11ec-9f40-1f88a3e01636"),
			outplanID:      uuid.FromStringOrNil("010e228c-c6b0-11ec-87b5-7bb69124c874"),
			outdialID:      uuid.FromStringOrNil("013945a2-c6b0-11ec-ba93-2300004f60a7"),
			queueID:        uuid.FromStringOrNil("0169cb32-c6b0-11ec-b616-4fe9ae9e95da"),
			nextCampaignID: uuid.FromStringOrNil("0468ac9c-7cce-11ee-9d09-7feca9bc6422"),

			responseCampaign: &campaign.Campaign{
				ID: uuid.FromStringOrNil("e74223b2-c6af-11ec-9f40-1f88a3e01636"),
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e74223b2-c6af-11ec-9f40-1f88a3e01636","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().UpdateResourceInfo(gomock.Any(), tt.campaignID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID).Return(tt.responseCampaign, nil)
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

func Test_v1CampaignsIDNextCampaignIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		campaignID     uuid.UUID
		nextCampaignID uuid.UUID

		responseCampaign *campaign.Campaign

		expectRes *sock.Response
	}{
		{
			"stopping",
			&sock.Request{
				URI:      "/v1/campaigns/e1f5109e-c6b0-11ec-a87d-1f8fe2380e97/next_campaign_id",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"next_campaign_id":"e98f6ab6-c6b0-11ec-b69c-df65d271a9d5"}`),
			},

			uuid.FromStringOrNil("e1f5109e-c6b0-11ec-a87d-1f8fe2380e97"),
			uuid.FromStringOrNil("e98f6ab6-c6b0-11ec-b69c-df65d271a9d5"),

			&campaign.Campaign{
				ID: uuid.FromStringOrNil("e1f5109e-c6b0-11ec-a87d-1f8fe2380e97"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e1f5109e-c6b0-11ec-a87d-1f8fe2380e97","customer_id":"00000000-0000-0000-0000-000000000000","type":"","execute":"","name":"","detail":"","status":"","service_level":0,"end_handle":"","flow_id":"00000000-0000-0000-0000-000000000000","actions":null,"outplan_id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","next_campaign_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCampaign := campaignhandler.NewMockCampaignHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				campaignHandler: mockCampaign,
			}

			mockCampaign.EXPECT().UpdateNextCampaignID(gomock.Any(), tt.campaignID, tt.nextCampaignID).Return(tt.responseCampaign, nil)
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
