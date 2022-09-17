package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_AgentV1TagCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		tagName    string
		deail      string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *amtag.Tag
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"test tag1",
			"test tag1 detail",

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/tags",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"7fdb8e66-7fe7-11ec-ac90-878b581c2615","name":"test tag1","detail":"test tag1 detail"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c9e3cab8-4e7e-11ec-a153-472a8569b57b"}`),
			},
			&amtag.Tag{
				ID: uuid.FromStringOrNil("c9e3cab8-4e7e-11ec-a153-472a8569b57b"),
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

			res, err := reqHandler.AgentV1TagCreate(ctx, tt.customerID, tt.tagName, tt.deail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentV1TagGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *amtag.Tag
	}{
		{
			"normal",

			uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/tags/7ab80df4-4c72-11ec-b095-17146a0e7e4c",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7ab80df4-4c72-11ec-b095-17146a0e7e4c"}`),
			},
			&amtag.Tag{
				ID: uuid.FromStringOrNil("7ab80df4-4c72-11ec-b095-17146a0e7e4c"),
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

			res, err := reqHandler.AgentV1TagGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentV1TagGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []amtag.Tag
	}{
		{
			"normal",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"2020-09-20 03:23:20.995000",
			10,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/tags?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d3ce27ac-4c72-11ec-b790-6b79445cbb01"}]`),
			},
			[]amtag.Tag{
				{
					ID: uuid.FromStringOrNil("d3ce27ac-4c72-11ec-b790-6b79445cbb01"),
				},
			},
		},
		{
			"2 agents",

			uuid.FromStringOrNil("7fdb8e66-7fe7-11ec-ac90-878b581c2615"),
			"2020-09-20 03:23:20.995000",
			10,

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/tags?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10&customer_id=7fdb8e66-7fe7-11ec-ac90-878b581c2615",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"8ea7c836-4e7f-11ec-9ae4-a3350cf9fcb1"},{"id":"8ec9ab2c-4e7f-11ec-9027-af8f1460c6a8"}]`),
			},
			[]amtag.Tag{
				{
					ID: uuid.FromStringOrNil("8ea7c836-4e7f-11ec-9ae4-a3350cf9fcb1"),
				},
				{
					ID: uuid.FromStringOrNil("8ec9ab2c-4e7f-11ec-9027-af8f1460c6a8"),
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

			res, err := reqHandler.AgentV1TagGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_AMTagDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *amtag.Tag
	}{
		{
			"normal",

			uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/tags/f4b44b28-4e79-11ec-be3c-73450ec23a51",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f4b44b28-4e79-11ec-be3c-73450ec23a51"}`),
			},
			&amtag.Tag{
				ID: uuid.FromStringOrNil("f4b44b28-4e79-11ec-be3c-73450ec23a51"),
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

			res, err := reqHandler.AgentV1TagDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AgentV1TagUpdate(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		agentName string
		detail    string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request

		response  *rabbitmqhandler.Response
		expectRes *amtag.Tag
	}{
		{
			"normal",

			uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
			"update name",
			"update detail",

			"bin-manager.agent-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/tags/1e60cb12-4e7b-11ec-9d7b-532466c1faf1",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1e60cb12-4e7b-11ec-9d7b-532466c1faf1"}`),
			},
			&amtag.Tag{
				ID: uuid.FromStringOrNil("1e60cb12-4e7b-11ec-9d7b-532466c1faf1"),
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

			res, err := reqHandler.AgentV1TagUpdate(ctx, tt.id, tt.agentName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
