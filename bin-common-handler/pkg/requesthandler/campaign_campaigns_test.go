package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	cacampaign "monorepo/bin-campaign-manager/models/campaign"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
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
		actions        []fmaction.Action
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("1d8334ff-afa2-4687-9b9a-038df4f27cf9"),
			uuid.FromStringOrNil("857f154e-7f4d-11ec-b669-a7aa025fbeaf"),
			cacampaign.TypeCall,
			"test name",
			"test detail",
			100,
			cacampaign.EndHandleStop,
			[]fmaction.Action{},
			uuid.FromStringOrNil("7db3f543-e9f4-4e87-aec9-b66713d2b4da"),
			uuid.FromStringOrNil("b07a3fb5-59df-450f-a3bf-779faea8baaf"),
			uuid.FromStringOrNil("6d23319a-74f9-4251-bdbf-650926b7ceb6"),
			uuid.FromStringOrNil("01f7ce4d-69bc-4d6a-aafa-6b4cdf43a4d1"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1d8334ff-afa2-4687-9b9a-038df4f27cf9"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"id":"1d8334ff-afa2-4687-9b9a-038df4f27cf9","customer_id":"857f154e-7f4d-11ec-b669-a7aa025fbeaf","type":"call","name":"test name","detail":"test detail","service_level":100,"end_handle":"stop","actions":[],"outplan_id":"7db3f543-e9f4-4e87-aec9-b66713d2b4da","outdial_id":"b07a3fb5-59df-450f-a3bf-779faea8baaf","queue_id":"6d23319a-74f9-4251-bdbf-650926b7ceb6","next_campaign_id":"01f7ce4d-69bc-4d6a-aafa-6b4cdf43a4d1"}`),
			},
			&cacampaign.Campaign{
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
				tt.actions,
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  []cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("4b1deb60-a784-4207-b1d8-a96df6bae951"),
			"2020-09-20 03:23:20.995000",
			10,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2bf5c9ab-25bd-4bdf-a637-56b882785da9"}]`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      fmt.Sprintf("/v1/campaigns?page_token=%s&page_size=10&customer_id=4b1deb60-a784-4207-b1d8-a96df6bae951", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   sock.RequestMethodGet,
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("8633f201-cf6d-42e7-af63-d63fbc36f637"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8633f201-cf6d-42e7-af63-d63fbc36f637"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/8633f201-cf6d-42e7-af63-d63fbc36f637",
				Method:   sock.RequestMethodGet,
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("22d9075d-08bd-4eb0-b868-3b102f0bcb39"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"22d9075d-08bd-4eb0-b868-3b102f0bcb39"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/22d9075d-08bd-4eb0-b868-3b102f0bcb39",
				Method:   sock.RequestMethodDelete,
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			"normal",

			uuid.FromStringOrNil("00089b80-3c19-42f1-80d3-f6ff450b1562"),
			DelayNow,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/00089b80-3c19-42f1-80d3-f6ff450b1562/execute",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
			},
		},
		{
			"5 seconds delay",

			uuid.FromStringOrNil("d7bc51db-e61b-460b-b13e-2d4f453151cd"),
			5000,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/d7bc51db-e61b-460b-b13e-2d4f453151cd/execute",
				Method:   sock.RequestMethodPost,
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

		campaignID         uuid.UUID
		updateName         string
		updateDetail       string
		campaignType       cacampaign.Type
		updateServiceLevel int
		updateEndHandle    cacampaign.EndHandle

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			name: "normal",

			campaignID:         uuid.FromStringOrNil("1692450e-c50f-11ec-8e6c-07b184583eb1"),
			updateName:         "update name",
			updateDetail:       "update detail",
			campaignType:       cacampaign.TypeCall,
			updateServiceLevel: 100,
			updateEndHandle:    cacampaign.EndHandleContinue,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1692450e-c50f-11ec-8e6c-07b184583eb1"}`),
			},

			expectTarget: "bin-manager.campaign-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/campaigns/1692450e-c50f-11ec-8e6c-07b184583eb1",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail","type":"call","service_level":100,"end_handle":"continue"}`),
			},
			expectResult: &cacampaign.Campaign{
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

			res, err := reqHandler.CampaignV1CampaignUpdateBasicInfo(ctx, tt.campaignID, tt.updateName, tt.updateDetail, tt.campaignType, tt.updateServiceLevel, tt.updateEndHandle)
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("f08f88a9-1e97-4da3-8052-3506ec5d73ae"),
			cacampaign.StatusRun,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f08f88a9-1e97-4da3-8052-3506ec5d73ae"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/f08f88a9-1e97-4da3-8052-3506ec5d73ae/status",
				Method:   sock.RequestMethodPut,
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("4a334640-35f9-4742-8428-97d386804c8b"),
			100,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4a334640-35f9-4742-8428-97d386804c8b"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/4a334640-35f9-4742-8428-97d386804c8b/service_level",
				Method:   sock.RequestMethodPut,
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

func Test_CampaignV1CampaignUpdateActions(t *testing.T) {

	tests := []struct {
		name string

		campaignID uuid.UUID
		actions    []fmaction.Action

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("381d05c3-5cc2-4296-89c9-80aa751e2d2c"),
			[]fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
				},
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"381d05c3-5cc2-4296-89c9-80aa751e2d2c"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/381d05c3-5cc2-4296-89c9-80aa751e2d2c/actions",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}]}`),
			},
			&cacampaign.Campaign{
				ID: uuid.FromStringOrNil("381d05c3-5cc2-4296-89c9-80aa751e2d2c"),
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

			res, err := reqHandler.CampaignV1CampaignUpdateActions(ctx, tt.campaignID, tt.actions)
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
		outplanID      uuid.UUID
		outdialID      uuid.UUID
		queueID        uuid.UUID
		nextCampaignID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			name: "normal",

			campaignID:     uuid.FromStringOrNil("e559c128-c6b3-11ec-8f7c-67e43d0d08d8"),
			outplanID:      uuid.FromStringOrNil("e5907394-c6b3-11ec-9dfa-17e8177ec4c1"),
			outdialID:      uuid.FromStringOrNil("e5bcde16-c6b3-11ec-b955-b75320ec1cc2"),
			queueID:        uuid.FromStringOrNil("e5e5f206-c6b3-11ec-bc99-17af712a37b1"),
			nextCampaignID: uuid.FromStringOrNil("eeff5402-7cd0-11ee-bcb6-9b5f97f1f8a9"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"e559c128-c6b3-11ec-8f7c-67e43d0d08d8"}`),
			},

			expectTarget: "bin-manager.campaign-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/campaigns/e559c128-c6b3-11ec-8f7c-67e43d0d08d8/resource_info",
				Method:   sock.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"outplan_id":"e5907394-c6b3-11ec-9dfa-17e8177ec4c1","outdial_id":"e5bcde16-c6b3-11ec-b955-b75320ec1cc2","queue_id":"e5e5f206-c6b3-11ec-bc99-17af712a37b1","next_campaign_id":"eeff5402-7cd0-11ee-bcb6-9b5f97f1f8a9"}`),
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

			res, err := reqHandler.CampaignV1CampaignUpdateResourceInfo(ctx, tt.campaignID, tt.outplanID, tt.outdialID, tt.queueID, tt.nextCampaignID)
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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cacampaign.Campaign
	}{
		{
			"normal",

			uuid.FromStringOrNil("42a6943c-c6b4-11ec-a70b-cb75b0197d55"),
			uuid.FromStringOrNil("2bed4c36-c6b4-11ec-92e6-1b01011d10cf"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"42a6943c-c6b4-11ec-a70b-cb75b0197d55"}`),
			},

			"bin-manager.campaign-manager.request",
			&sock.Request{
				URI:      "/v1/campaigns/42a6943c-c6b4-11ec-a70b-cb75b0197d55/next_campaign_id",
				Method:   sock.RequestMethodPut,
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
