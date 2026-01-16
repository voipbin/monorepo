package listenhandler

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/messagehandler"
)

func Test_processV1MessagesGet(t *testing.T) {

	tests := []struct {
		name string

		pageSize   uint64
		pageToken  string
		filters    map[message.Field]any
		resultData []*message.Message

		request  *sock.Request
		response *sock.Response
	}{
		{
			"normal",

			10,
			"2021-03-01 03:30:17.000000",
			map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("197609d6-a29b-11ec-b884-5b8a227db58a"),
			},
			[]*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("eeafd418-7a4e-11eb-8750-9bb0ca1d7926"),
						CustomerID: uuid.FromStringOrNil("197609d6-a29b-11ec-b884-5b8a227db58a"),
					},
				},
			},

			&sock.Request{
				URI:    "/v1/messages?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"197609d6-a29b-11ec-b884-5b8a227db58a"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"eeafd418-7a4e-11eb-8750-9bb0ca1d7926","customer_id":"197609d6-a29b-11ec-b884-5b8a227db58a","type":"","source":null,"targets":null,"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
		{
			"2 results",

			10,
			"2021-03-01 03:30:17.000000",
			map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("75dd760a-a29b-11ec-ba70-cb282aa1d594"),
			},
			[]*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("760f30fa-a29b-11ec-87e7-2fc5bdc8739b"),
						CustomerID: uuid.FromStringOrNil("75dd760a-a29b-11ec-ba70-cb282aa1d594"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("7639e39a-a29b-11ec-b393-5b239b119501"),
						CustomerID: uuid.FromStringOrNil("75dd760a-a29b-11ec-ba70-cb282aa1d594"),
					},
				},
			},

			&sock.Request{
				URI:    "/v1/messages?page_size=10&page_token=2021-03-01%2003%3A30%3A17.000000",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"75dd760a-a29b-11ec-ba70-cb282aa1d594"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"760f30fa-a29b-11ec-87e7-2fc5bdc8739b","customer_id":"75dd760a-a29b-11ec-ba70-cb282aa1d594","type":"","source":null,"targets":null,"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"7639e39a-a29b-11ec-b393-5b239b119501","customer_id":"75dd760a-a29b-11ec-ba70-cb282aa1d594","type":"","source":null,"targets":null,"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
	
			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
			}

			mockMessage.EXPECT().List(gomock.Any(), tt.pageToken, tt.pageSize, gomock.Any()).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1MessagesPost(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		source       *commonaddress.Address
		destinations []commonaddress.Address
		Text         string

		responseSend *message.Message

		request  *sock.Request
		response *sock.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("5f00c9bc-f176-11ec-bda0-af0b8c9491f5"),
			uuid.FromStringOrNil("fdca8fb4-a22b-11ec-8894-7bfd708fa894"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			[]commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			"hello, world",

			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f00c9bc-f176-11ec-bda0-af0b8c9491f5"),
				},
			},

			&sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"5f00c9bc-f176-11ec-bda0-af0b8c9491f5","customer_id":"fdca8fb4-a22b-11ec-8894-7bfd708fa894", "source":{"type": "tel", "target": "+821100000001"}, "destinations": [{"type": "tel", "target": "+821100000002"}], "text": "hello, world"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5f00c9bc-f176-11ec-bda0-af0b8c9491f5","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMessageHandler := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessageHandler,
			}

			mockMessageHandler.EXPECT().Send(gomock.Any(), tt.id, tt.customerID, tt.source, tt.destinations, tt.Text).Return(tt.responseSend, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1MessagesIDGet(t *testing.T) {

	tests := []struct {
		name       string
		id         uuid.UUID
		resultData *message.Message

		request  *sock.Request
		response *sock.Response
	}{
		{
			"1 number",
			uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740"),
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("73071e00-a29a-11ec-a43a-079fe08ce740"),
				},
			},
			&sock.Request{
				URI:    "/v1/messages/73071e00-a29a-11ec-a43a-079fe08ce740",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"73071e00-a29a-11ec-a43a-079fe08ce740","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
			}

			mockMessage.EXPECT().Get(gomock.Any(), tt.id).Return(tt.resultData, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}
		})
	}
}

func Test_processV1MessagesIDDelete(t *testing.T) {

	tests := []struct {
		name           string
		id             uuid.UUID
		responseDelete *message.Message

		request  *sock.Request
		response *sock.Response
	}{
		{
			"normal",
			uuid.FromStringOrNil("63772a08-a2ee-11ec-8c6d-9714fb1cc108"),
			&message.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("63772a08-a2ee-11ec-8c6d-9714fb1cc108"),
				},
			},
			&sock.Request{
				URI:    "/v1/messages/63772a08-a2ee-11ec-8c6d-9714fb1cc108",
				Method: sock.RequestMethodDelete,
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"63772a08-a2ee-11ec-8c6d-9714fb1cc108","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"provider_name":"","provider_reference_id":"","text":"","medias":null,"direction":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
			}

			mockMessage.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseDelete, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.response, res) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.response, res)
			}

		})
	}
}
