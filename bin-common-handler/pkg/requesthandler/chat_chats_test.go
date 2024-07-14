package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	chatchat "monorepo/bin-chat-manager/models/chat"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_ChatV1ChatCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		chatType       chatchat.Type
		roomOwnerID    uuid.UUID
		participantIDs []uuid.UUID
		chatName       string
		detail         string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("95e79972-3697-11ed-85be-5b6792ca4a82"),
			chatchat.TypeNormal,
			uuid.FromStringOrNil("96137b14-3697-11ed-b9b4-d7dac3f2b181"),
			[]uuid.UUID{
				uuid.FromStringOrNil("964147e2-3697-11ed-a461-8342afde852e"),
				uuid.FromStringOrNil("966f1e74-3697-11ed-aebb-6703f26009c6"),
			},
			"test name",
			"test detail",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d50945c4-3697-11ed-9ffb-570b42b0ddd4"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"95e79972-3697-11ed-85be-5b6792ca4a82","type":"normal","room_owner_id":"96137b14-3697-11ed-b9b4-d7dac3f2b181","participant_ids":["964147e2-3697-11ed-a461-8342afde852e","966f1e74-3697-11ed-aebb-6703f26009c6"],"name":"test name","detail":"test detail"}`),
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d50945c4-3697-11ed-9ffb-570b42b0ddd4"),
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

			res, err := reqHandler.ChatV1ChatCreate(
				ctx,
				tt.customerID,
				tt.chatType,
				tt.roomOwnerID,
				tt.participantIDs,
				tt.chatName,
				tt.detail,
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

func Test_ChatV1ChatGet(t *testing.T) {

	tests := []struct {
		name string

		chatID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("6ccf608c-3698-11ed-9f4b-a7949f21d6b1"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6ccf608c-3698-11ed-9f4b-a7949f21d6b1"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/6ccf608c-3698-11ed-9f4b-a7949f21d6b1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("6ccf608c-3698-11ed-9f4b-a7949f21d6b1"),
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

			res, err := reqHandler.ChatV1ChatGet(ctx, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *rabbitmqhandler.Response

		expectURL     string
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  []chatchat.Chat
	}{
		{
			"normal",

			"2020-09-20 03:23:20.995000",
			10,
			map[string]string{
				"customer_id": "c6ebf88c-3698-11ed-a6e1-7f172e23f5ea",
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c7224cde-3698-11ed-b3c4-57ea8ad3e71d"}]`),
			},

			"/v1/chats?page_token=2020-09-20+03%3A23%3A20.995000&page_size=10",
			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      fmt.Sprintf("/v1/chats?page_token=%s&page_size=10&filter_customer_id=c6ebf88c-3698-11ed-a6e1-7f172e23f5ea", url.QueryEscape("2020-09-20 03:23:20.995000")),
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			[]chatchat.Chat{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c7224cde-3698-11ed-b3c4-57ea8ad3e71d"),
					},
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
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.ChatV1ChatGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}

func Test_ChatV1ChatDelete(t *testing.T) {

	tests := []struct {
		name string

		chatID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("fb7a2164-3698-11ed-acfd-5b96525f8ec9"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fb7a2164-3698-11ed-acfd-5b96525f8ec9"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/fb7a2164-3698-11ed-acfd-5b96525f8ec9",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("fb7a2164-3698-11ed-acfd-5b96525f8ec9"),
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

			res, err := reqHandler.ChatV1ChatDelete(ctx, tt.chatID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatUpdateBasicInfo(t *testing.T) {

	tests := []struct {
		name string

		chatID       uuid.UUID
		updateName   string
		updateDetail string

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("63e29b96-c515-11ec-ba52-ab7d7001913f"),
			"update name",
			"update detail",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"63e29b96-c515-11ec-ba52-ab7d7001913f"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/63e29b96-c515-11ec-ba52-ab7d7001913f",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"name":"update name","detail":"update detail"}`),
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("63e29b96-c515-11ec-ba52-ab7d7001913f"),
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

			res, err := reqHandler.ChatV1ChatUpdateBasicInfo(ctx, tt.chatID, tt.updateName, tt.updateDetail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatUpdateRoomOwnerID(t *testing.T) {

	tests := []struct {
		name string

		chatID            uuid.UUID
		updateRoomOwnerID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("873465c0-3699-11ed-b0cc-fbee9352367b"),
			uuid.FromStringOrNil("875e8c42-3699-11ed-9746-a772f54ed917"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"873465c0-3699-11ed-b0cc-fbee9352367b"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/873465c0-3699-11ed-b0cc-fbee9352367b/room_owner_id",
				Method:   rabbitmqhandler.RequestMethodPut,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"room_owner_id":"875e8c42-3699-11ed-9746-a772f54ed917"}`),
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("873465c0-3699-11ed-b0cc-fbee9352367b"),
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

			res, err := reqHandler.ChatV1ChatUpdateRoomOwnerID(ctx, tt.chatID, tt.updateRoomOwnerID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatAddParticipantID(t *testing.T) {

	tests := []struct {
		name string

		chatID        uuid.UUID
		participantID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("2658828e-369b-11ed-b9c9-5b03f9812b19"),
			uuid.FromStringOrNil("268b0e70-369b-11ed-ba58-cb3598f7f7d1"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2658828e-369b-11ed-b9c9-5b03f9812b19"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/2658828e-369b-11ed-b9c9-5b03f9812b19/participant_ids",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"participant_id":"268b0e70-369b-11ed-ba58-cb3598f7f7d1"}`),
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2658828e-369b-11ed-b9c9-5b03f9812b19"),
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

			res, err := reqHandler.ChatV1ChatAddParticipantID(ctx, tt.chatID, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_ChatV1ChatRemoveParticipantID(t *testing.T) {

	tests := []struct {
		name string

		chatID        uuid.UUID
		participantID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *chatchat.Chat
	}{
		{
			"normal",

			uuid.FromStringOrNil("355eda2a-369c-11ed-b5f8-bf163967cc10"),
			uuid.FromStringOrNil("358a2298-369c-11ed-aa26-0fc0067a8829"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"355eda2a-369c-11ed-b5f8-bf163967cc10"}`),
			},

			"bin-manager.chat-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/chats/355eda2a-369c-11ed-b5f8-bf163967cc10/participant_ids/358a2298-369c-11ed-aa26-0fc0067a8829",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},
			&chatchat.Chat{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("355eda2a-369c-11ed-b5f8-bf163967cc10"),
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

			res, err := reqHandler.ChatV1ChatRemoveParticipantID(ctx, tt.chatID, tt.participantID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
