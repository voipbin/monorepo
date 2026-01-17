package requesthandler

import (
	"context"
	"reflect"
	"testing"

	talkmessage "monorepo/bin-talk-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_TalkV1TalkMessageGet(t *testing.T) {

	tests := []struct {
		name string

		messageID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *talkmessage.Message
	}{
		{
			name: "normal",

			messageID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Hello"}`),
			},
			expectRes: &talkmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
				},
				ChatID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
				Type:   talkmessage.TypeNormal,
				Text:   "Hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TalkV1TalkMessageGet(ctx, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1TalkMessageCreate(t *testing.T) {

	parentID := uuid.FromStringOrNil("880e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name string

		chatID    uuid.UUID
		parentID  *uuid.UUID
		ownerType string
		ownerID   uuid.UUID
		msgType   talkmessage.Type
		text      string

		expectQueue string

		response  *sock.Response
		expectRes *talkmessage.Message
	}{
		{
			name: "normal without parent",

			chatID:    uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
			parentID:  nil,
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			msgType:   talkmessage.TypeNormal,
			text:      "Hello",

			expectQueue: "bin-manager.talk-manager.request",

			response: &sock.Response{
				StatusCode: 201,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Hello"}`),
			},
			expectRes: &talkmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
				},
				ChatID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
				Type:   talkmessage.TypeNormal,
				Text:   "Hello",
			},
		},
		{
			name: "with parent (threading)",

			chatID:    uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
			parentID:  &parentID,
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			msgType:   talkmessage.TypeNormal,
			text:      "Reply",

			expectQueue: "bin-manager.talk-manager.request",

			response: &sock.Response{
				StatusCode: 201,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","parent_id":"880e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Reply"}`),
			},
			expectRes: &talkmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
				},
				ChatID:   uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
				ParentID: &parentID,
				Type:     talkmessage.TypeNormal,
				Text:     "Reply",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, gomock.Any()).Return(tt.response, nil)

			res, err := reqHandler.TalkV1TalkMessageCreate(ctx, tt.chatID, tt.parentID, tt.ownerType, tt.ownerID, tt.msgType, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1TalkMessageDelete(t *testing.T) {

	tests := []struct {
		name string

		messageID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *talkmessage.Message
	}{
		{
			name: "normal",

			messageID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Hello"}`),
			},
			expectRes: &talkmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
				},
				ChatID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
				Type:   talkmessage.TypeNormal,
				Text:   "Hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TalkV1TalkMessageDelete(ctx, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1TalkMessageList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes []*talkmessage.Message
	}{
		{
			name: "normal",

			pageToken: "2020-09-20 03:23:20.995000",
			pageSize:  10,

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/messages?page_token=2020-09-20%2003%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`[{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Hello"}]`),
			},
			expectRes: []*talkmessage.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
					},
					ChatID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
					Type:   talkmessage.TypeNormal,
					Text:   "Hello",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, gomock.Any()).Return(tt.response, nil)

			res, err := reqHandler.TalkV1TalkMessageList(ctx, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1TalkMessageReactionCreate(t *testing.T) {

	tests := []struct {
		name string

		messageID uuid.UUID
		ownerType string
		ownerID   uuid.UUID
		emoji     string

		expectQueue string

		response  *sock.Response
		expectRes *talkmessage.Message
	}{
		{
			name: "normal",

			messageID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
			emoji:     "👍",

			expectQueue: "bin-manager.talk-manager.request",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000","chat_id":"770e8400-e29b-41d4-a716-446655440000","type":"normal","text":"Hello","reactions":[{"emoji":"👍","owner_type":"agent","owner_id":"660e8400-e29b-41d4-a716-446655440000"}]}`),
			},
			expectRes: &talkmessage.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("660e8400-e29b-41d4-a716-446655440000"),
				},
				ChatID: uuid.FromStringOrNil("770e8400-e29b-41d4-a716-446655440000"),
				Type:   talkmessage.TypeNormal,
				Text:   "Hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, gomock.Any()).Return(tt.response, nil)

			res, err := reqHandler.TalkV1TalkMessageReactionCreate(ctx, tt.messageID, tt.ownerType, tt.ownerID, tt.emoji)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
