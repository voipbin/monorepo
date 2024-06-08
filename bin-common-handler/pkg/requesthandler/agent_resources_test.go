package requesthandler

import (
	"context"
	amresource "monorepo/bin-agent-manager/models/resource"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
)

func Test_AgentV1ResourceGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []amresource.Resource
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/resources?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/resources?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"f1634af6-25bc-11ef-8945-4b79f4685b17"}]`),
			},
			[]amresource.Resource{
				{
					ID: uuid.FromStringOrNil("f1634af6-25bc-11ef-8945-4b79f4685b17"),
				},
			},
		},
		{
			"2 results",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/resources?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/resources?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"f1a5fbda-25bc-11ef-8942-8724bf1df5fa"},{"id":"f1f7bf2e-25bc-11ef-a049-1f189fdc2dff"}]`),
			},
			[]amresource.Resource{
				{
					ID: uuid.FromStringOrNil("f1a5fbda-25bc-11ef-8942-8724bf1df5fa"),
				},
				{
					ID: uuid.FromStringOrNil("f1f7bf2e-25bc-11ef-a049-1f189fdc2dff"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			reqHandler := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}

			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AgentV1ResourceGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentV1ResourceGet(t *testing.T) {

	tests := []struct {
		name string

		resourceID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *amresource.Resource
	}{
		{
			"normal",

			uuid.FromStringOrNil("4fd34046-25bd-11ef-9596-c767575e5fd8"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/resources/4fd34046-25bd-11ef-9596-c767575e5fd8",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"4fd34046-25bd-11ef-9596-c767575e5fd8"}`),
			},
			&amresource.Resource{
				ID: uuid.FromStringOrNil("4fd34046-25bd-11ef-9596-c767575e5fd8"),
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

			res, err := reqHandler.AgentV1ResourceGet(ctx, tt.resourceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentV1ResourceDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *amresource.Resource
	}{
		{
			"normal",

			uuid.FromStringOrNil("8e27fa94-25bd-11ef-b883-a7d793e988d4"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:    "/v1/resources/8e27fa94-25bd-11ef-b883-a7d793e988d4",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8e27fa94-25bd-11ef-b883-a7d793e988d4"}`),
			},
			&amresource.Resource{
				ID: uuid.FromStringOrNil("8e27fa94-25bd-11ef-b883-a7d793e988d4"),
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

			res, err := reqHandler.AgentV1ResourceDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
