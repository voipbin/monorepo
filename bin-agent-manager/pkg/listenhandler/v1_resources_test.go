package listenhandler

import (
	"monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-agent-manager/pkg/agenthandler"
	"monorepo/bin-agent-manager/pkg/resourcehandler"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_ProcessV1ResourcesGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		pageSize  uint64
		pageToken string

		responseFilters   map[string]string
		responseResources []*resource.Resource
		expectRes         *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/resources?page_size=10&page_token=2021-11-23%2017:55:39.712000&filter_reference_id=c20a594c-24dc-11ef-b9f0-6f86b22d7f85",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			10,
			"2021-11-23 17:55:39.712000",

			map[string]string{
				"reference_id": "c20a594c-24dc-11ef-b9f0-6f86b22d7f85",
			},
			[]*resource.Resource{
				{
					ID: uuid.FromStringOrNil("c260610c-24dc-11ef-ac24-0301eb0f82cb"),
				},
				{
					ID: uuid.FromStringOrNil("c2a5c90e-24dc-11ef-8d74-634bd94f8f16"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c260610c-24dc-11ef-ac24-0301eb0f82cb","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","data":null,"tm_create":"","tm_update":"","tm_delete":""},{"id":"c2a5c90e-24dc-11ef-8d74-634bd94f8f16","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","data":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockResource := resourcehandler.NewMockResourceHandler(mc)

			h := &listenHandler{
				utilHandler:     mockUtil,
				rabbitSock:      mockSock,
				agentHandler:    mockAgent,
				resourceHandler: mockResource,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockResource.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseResources, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1ResourcesIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id uuid.UUID

		responseResource *resource.Resource
		expectRes        *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/resources/796ffbe6-24dd-11ef-9b2d-03bd83bc441f",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("796ffbe6-24dd-11ef-9b2d-03bd83bc441f"),

			&resource.Resource{
				ID: uuid.FromStringOrNil("796ffbe6-24dd-11ef-9b2d-03bd83bc441f"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"796ffbe6-24dd-11ef-9b2d-03bd83bc441f","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","data":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)
			mockResource := resourcehandler.NewMockResourceHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				agentHandler:    mockAgent,
				resourceHandler: mockResource,
			}

			mockResource.EXPECT().Get(gomock.Any(), tt.id).Return(tt.responseResource, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestProcessV1ResourcesIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		id uuid.UUID

		responseAgent *resource.Resource
		expectRes     *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/resources/c50adda0-24dd-11ef-9c53-5bf4b80843f2",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("c50adda0-24dd-11ef-9c53-5bf4b80843f2"),

			&resource.Resource{
				ID: uuid.FromStringOrNil("c50adda0-24dd-11ef-9c53-5bf4b80843f2"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c50adda0-24dd-11ef-9c53-5bf4b80843f2","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","data":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockAgent := agenthandler.NewMockAgentHandler(mc)
			mockResource := resourcehandler.NewMockResourceHandler(mc)

			h := &listenHandler{
				rabbitSock:      mockSock,
				agentHandler:    mockAgent,
				resourceHandler: mockResource,
			}

			mockResource.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseAgent, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
