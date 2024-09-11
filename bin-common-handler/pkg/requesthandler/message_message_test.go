package requesthandler

import (
	"context"
	"reflect"
	"testing"

	mmmessage "monorepo/bin-message-manager/models/message"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_MessageV1MessageGets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		pageToken  string
		pageSize   uint64

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []mmmessage.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("2970f4e8-a2b1-11ec-b21d-a7848e946530"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=2970f4e8-a2b1-11ec-b21d-a7848e946530",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"29c761de-a2b1-11ec-98af-e7185fc83700"}]`),
			},
			[]mmmessage.Message{
				{
					ID: uuid.FromStringOrNil("29c761de-a2b1-11ec-98af-e7185fc83700"),
				},
			},
		},
		{
			"2 messages",

			uuid.FromStringOrNil("6f0e7d2c-a2b1-11ec-88c4-af58c97aff78"),
			"2020-09-20T03:23:20.995000",
			10,

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&customer_id=6f0e7d2c-a2b1-11ec-88c4-af58c97aff78",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6f3a1e50-a2b1-11ec-a108-33dea3457c3b"},{"id":"6f68ee1a-a2b1-11ec-934e-fb86dd720e70"}]`),
			},
			[]mmmessage.Message{
				{
					ID: uuid.FromStringOrNil("6f3a1e50-a2b1-11ec-a108-33dea3457c3b"),
				},
				{
					ID: uuid.FromStringOrNil("6f68ee1a-a2b1-11ec-934e-fb86dd720e70"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.MessageV1MessageGets(ctx, tt.customerID, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageV1MessageGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *mmmessage.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("c6132bfe-a2b1-11ec-a9fd-0f7e4afd00d8"),

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages/c6132bfe-a2b1-11ec-a9fd-0f7e4afd00d8",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c6132bfe-a2b1-11ec-a9fd-0f7e4afd00d8"}`),
			},
			&mmmessage.Message{
				ID: uuid.FromStringOrNil("c6132bfe-a2b1-11ec-a9fd-0f7e4afd00d8"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.MessageV1MessageGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageV1MessageDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *mmmessage.Message
	}{
		{
			"normal",

			uuid.FromStringOrNil("b8c3c122-a2c3-11ec-89ee-ebb21f6a7e14"),

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages/b8c3c122-a2c3-11ec-89ee-ebb21f6a7e14",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b8c3c122-a2c3-11ec-89ee-ebb21f6a7e14"}`),
			},
			&mmmessage.Message{
				ID: uuid.FromStringOrNil("b8c3c122-a2c3-11ec-89ee-ebb21f6a7e14"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.MessageV1MessageDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_MessageV1MessageSend(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		source       *address.Address
		destinations []address.Address
		text         string

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *mmmessage.Message
	}{
		{
			"1 destination",

			uuid.FromStringOrNil("dde92b9a-f179-11ec-adc4-931faecc6a89"),
			uuid.FromStringOrNil("96ed3008-a2b2-11ec-b585-bf3e19b7355a"),
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			[]address.Address{
				{
					Type:   address.TypeTel,
					Target: "+821100000002",
				},
			},
			"hello world",

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"dde92b9a-f179-11ec-adc4-931faecc6a89","customer_id":"96ed3008-a2b2-11ec-b585-bf3e19b7355a","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""}],"text":"hello world"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dde92b9a-f179-11ec-adc4-931faecc6a89"}`),
			},
			&mmmessage.Message{
				ID: uuid.FromStringOrNil("dde92b9a-f179-11ec-adc4-931faecc6a89"),
			},
		},
		{
			"has no id",

			uuid.Nil,
			uuid.FromStringOrNil("96ed3008-a2b2-11ec-b585-bf3e19b7355a"),
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			[]address.Address{
				{
					Type:   address.TypeTel,
					Target: "+821100000002",
				},
			},
			"hello world",

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"00000000-0000-0000-0000-000000000000","customer_id":"96ed3008-a2b2-11ec-b585-bf3e19b7355a","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""}],"text":"hello world"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"283d6350-f17a-11ec-9277-436d6d821637"}`),
			},
			&mmmessage.Message{
				ID: uuid.FromStringOrNil("283d6350-f17a-11ec-9277-436d6d821637"),
			},
		},
		{
			"2 destinations",

			uuid.FromStringOrNil("e930839a-f179-11ec-acb5-3f10a4a1c047"),
			uuid.FromStringOrNil("333d1508-a2c3-11ec-872d-8796fdc672b5"),
			&address.Address{
				Type:   address.TypeTel,
				Target: "+821100000001",
			},
			[]address.Address{
				{
					Type:   address.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   address.TypeTel,
					Target: "+821100000003",
				},
			},
			"hello world",

			"bin-manager.message-manager.request",
			&sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"e930839a-f179-11ec-acb5-3f10a4a1c047","customer_id":"333d1508-a2c3-11ec-872d-8796fdc672b5","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""},{"type":"tel","target":"+821100000003","target_name":"","name":"","detail":""}],"text":"hello world"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"3378aa5a-a2c3-11ec-abb8-97d85696ccd9"}`),
			},
			&mmmessage.Message{
				ID: uuid.FromStringOrNil("3378aa5a-a2c3-11ec-abb8-97d85696ccd9"),
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
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.MessageV1MessageSend(ctx, tt.id, tt.customerID, tt.source, tt.destinations, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
