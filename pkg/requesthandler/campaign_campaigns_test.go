package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cacampaign "gitlab.com/voipbin/bin-manager/campaign-manager.git/models/campaign"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CampaignV1CampaignCreate(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		campaignType   cacampaign.Type
		campaignName   string
		campaignDetail string
		serviceLevel   int
		endHandle      cacampaign.EndHandle
		flowID         uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			name: "normal",

			id:             uuid.FromStringOrNil("1d8334ff-afa2-4687-9b9a-038df4f27cf9"),
			customerID:     uuid.FromStringOrNil("857f154e-7f4d-11ec-b669-a7aa025fbeaf"),
			campaignType:   cacampaign.TypeCall,
			campaignName:   "test name",
			campaignDetail: "test detail",
			serviceLevel:   100,
			endHandle:      cacampaign.EndHandleStop,
			flowID:         uuid.FromStringOrNil("83ed4b38-741a-11ee-b952-a320162d3e6a"),
			outplanID:      uuid.FromStringOrNil("7db3f543-e9f4-4e87-aec9-b66713d2b4da"),
			outdialID:      uuid.FromStringOrNil("b07a3fb5-59df-450f-a3bf-779faea8baaf"),
			queueID:        uuid.FromStringOrNil("6d23319a-74f9-4251-bdbf-650926b7ceb6"),
			nextCampaignID: uuid.FromStringOrNil("01f7ce4d-69bc-4d6a-aafa-6b4cdf43a4d1"),

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1d8334ff-afa2-4687-9b9a-038df4f27cf9"}`),
			},

			expectTarget: "bin-manager.campaign-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/campaigns",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"1d8334ff-afa2-4687-9b9a-038df4f27cf9","customer_id":"857f154e-7f4d-11ec-b669-a7aa025fbeaf","type":"call","name":"test name","detail":"test detail","service_level":100,"end_handle":"stop","actions":null,"flow_id":"83ed4b38-741a-11ee-b952-a320162d3e6a","outplan_id":"7db3f543-e9f4-4e87-aec9-b66713d2b4da","outdial_id":"b07a3fb5-59df-450f-a3bf-779faea8baaf","queue_id":"6d23319a-74f9-4251-bdbf-650926b7ceb6","next_campaign_id":"01f7ce4d-69bc-4d6a-aafa-6b4cdf43a4d1"}`),
			},
			expectResult: &cacampaign.Campaign{
				ID: uuid.FromStringOrNil("1d8334ff-afa2-4687-9b9a-038df4f27cf9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignCreate(
				ctx,
				tt.id,
				tt.customerID,
				tt.campaignType,
				tt.campaignName,
				tt.campaignDetail,
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

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong matchdfdsfd.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("4b1deb60-a784-4207-b1d8-a96df6bae951"),
			"2020-09-20 03:23:20.995000",
			10,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2bf5c9ab-25bd-4bdf-a637-56b882785da9"}]`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/campaigns?page_token=%s&page_size=10&customer_id=4b1deb60-a784-4207-b1d8-a96df6bae951", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]cacampaign.Campaign{
				{
					ID: uuid.FromStringOrNil("2bf5c9ab-25bd-4bdf-a637-56b882785da9"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignGetsByCustomerID(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_CampaignV1CampaignGet(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("8633f201-cf6d-42e7-af63-d63fbc36f637"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8633f201-cf6d-42e7-af63-d63fbc36f637"}`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/8633f201-cf6d-42e7-af63-d63fbc36f637",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("8633f201-cf6d-42e7-af63-d63fbc36f637"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignGet(ctx, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignDelete(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("22d9075d-08bd-4eb0-b868-3b102f0bcb39"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"22d9075d-08bd-4eb0-b868-3b102f0bcb39"}`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/22d9075d-08bd-4eb0-b868-3b102f0bcb39",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("22d9075d-08bd-4eb0-b868-3b102f0bcb39"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignDelete(ctx, tt.campaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignExecute(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID
		delay      int

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
	}{
		{
			"normal",

			uuid.FromStringOrNil("00089b80-3c19-42f1-80d3-f6ff450b1562"),
			DelayNow,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/00089b80-3c19-42f1-80d3-f6ff450b1562/execute",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
		},
		{
			"5 seconds delay",

			uuid.FromStringOrNil("d7bc51db-e61b-460b-b13e-2d4f453151cd"),
			5000,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/d7bc51db-e61b-460b-b13e-2d4f453151cd/execute",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			if tt.delay == 0 {
				mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)
			} else {
				mockSock.EXPECT().PublishExchangeDelayedRequest(gomock.Any(), tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)
			}

			if err := reqHandler.CampaignV1CampaignExecute(ctx, tt.campaignID, tt.delay); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_CampaignV1CampaignUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		campaignID   uuid.UUID
		updateName   string
		updateDetail string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("1692450e-c50f-11ec-8e6c-07b184583eb1"),
			"update name",
			"update detail",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1692450e-c50f-11ec-8e6c-07b184583eb1"}`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/1692450e-c50f-11ec-8e6c-07b184583eb1",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("1692450e-c50f-11ec-8e6c-07b184583eb1"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignUpdateBasicInfo(ctx, tt.campaignID, tt.updateName, tt.updateDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignUpdateStatus(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID
		status     cacampaign.Status

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("f08f88a9-1e97-4da3-8052-3506ec5d73ae"),
			cacampaign.StatusRun,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f08f88a9-1e97-4da3-8052-3506ec5d73ae"}`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/f08f88a9-1e97-4da3-8052-3506ec5d73ae/status",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"status":"run"}`),
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("f08f88a9-1e97-4da3-8052-3506ec5d73ae"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignUpdateStatus(ctx, tt.campaignID, tt.status)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignUpdateServiceLevel(t *testing.T) {

	tests := []struct {
		name string

		campaignID   uuid.UUID
		serviceLevel int

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("4a334640-35f9-4742-8428-97d386804c8b"),
			100,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4a334640-35f9-4742-8428-97d386804c8b"}`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/4a334640-35f9-4742-8428-97d386804c8b/service_level",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"service_level":100}`),
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("4a334640-35f9-4742-8428-97d386804c8b"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignUpdateServiceLevel(ctx, tt.campaignID, tt.serviceLevel)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignUpdateResourceInfo(t *testing.T) {

	tests := []struct {
		name string

		campaignID     uuid.UUID
		flowID         uuid.UUID
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			name: "normal",

			campaignID:     uuid.FromStringOrNil("e559c128-c6b3-11ec-8f7c-67e43d0d08d8"),
			flowID:         uuid.FromStringOrNil("e685d2ce-741a-11ee-a0a2-fb244e48d247"),
			outplanID:      uuid.FromStringOrNil("e5907394-c6b3-11ec-9dfa-17e8177ec4c1"),
			outdialID:      uuid.FromStringOrNil("e5bcde16-c6b3-11ec-b955-b75320ec1cc2"),
			queueID:        uuid.FromStringOrNil("e5e5f206-c6b3-11ec-bc99-17af712a37b1"),
			nextCampaignID: uuid.FromStringOrNil("e6a7f796-741a-11ee-a025-f7cd4aa93a52"),

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e559c128-c6b3-11ec-8f7c-67e43d0d08d8"}`),
			},

			expectTarget: "bin-manager.campaign-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/campaigns/e559c128-c6b3-11ec-8f7c-67e43d0d08d8/resource_info",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"flow_id":"e685d2ce-741a-11ee-a0a2-fb244e48d247","outplan_id":"e5907394-c6b3-11ec-9dfa-17e8177ec4c1","outdial_id":"e5bcde16-c6b3-11ec-b955-b75320ec1cc2","queue_id":"e5e5f206-c6b3-11ec-bc99-17af712a37b1","next_campaign_id":"e6a7f796-741a-11ee-a025-f7cd4aa93a52"}`),
			},
			expectResult: &cacampaign.Campaign{
				ID: uuid.FromStringOrNil("e559c128-c6b3-11ec-8f7c-67e43d0d08d8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignUpdateResourceInfo(ctx, tt.campaignID, tt.flowID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CampaignV1CampaignUpdateNextCampaignID(t *testing.T) {

	tests := []struct {
		name string

		campaignID     uuid.UUID
		nextCampaignID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("42a6943c-c6b4-11ec-a70b-cb75b0197d55"),
			uuid.FromStringOrNil("2bed4c36-c6b4-11ec-92e6-1b01011d10cf"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"42a6943c-c6b4-11ec-a70b-cb75b0197d55"}`),
			},

			"bin-manager.campaign-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/campaigns/42a6943c-c6b4-11ec-a70b-cb75b0197d55/next_campaign_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"next_campaign_id":"2bed4c36-c6b4-11ec-92e6-1b01011d10cf"}`),
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("42a6943c-c6b4-11ec-a70b-cb75b0197d55"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CampaignV1CampaignUpdateNextCampaignID(ctx, tt.campaignID, tt.nextCampaignID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
