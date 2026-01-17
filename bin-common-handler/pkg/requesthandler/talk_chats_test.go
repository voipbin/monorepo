package requesthandler

import (
	"context"
	"reflect"
	"testing"

	talktalk "monorepo/bin-talk-manager/models/talk"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_TalkV1ChatGet(t *testing.T) {

	type test struct {
		name string

		talkID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *talktalk.Talk
	}

	tests := []test{
		{
			name: "normal",

			talkID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/talk_chats/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"normal"}`),
			},
			expectRes: &talktalk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type: talktalk.TypeNormal,
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

			res, err := reqHandler.TalkV1ChatGet(ctx, tt.talkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_TalkV1ChatCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		talkType   talktalk.Type

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *talktalk.Talk
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
			talkType:   talktalk.TypeNormal,

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/talk_chats",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"normal"}`),
			},

			response: &sock.Response{
				StatusCode: 201,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"normal"}`),
			},
			expectRes: &talktalk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type: talktalk.TypeNormal,
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

			res, err := reqHandler.TalkV1ChatCreate(ctx, tt.customerID, tt.talkType)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1ChatDelete(t *testing.T) {

	tests := []struct {
		name string

		talkID uuid.UUID

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes *talktalk.Talk
	}{
		{
			name: "normal",

			talkID: uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/talk_chats/72179880-ec5f-11ec-920e-c77279756b6d",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"normal"}`),
			},
			expectRes: &talktalk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
					CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
				},
				Type: talktalk.TypeNormal,
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

			res, err := reqHandler.TalkV1ChatDelete(ctx, tt.talkID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TalkV1ChatList(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64

		expectQueue   string
		expectRequest *sock.Request

		response  *sock.Response
		expectRes []*talktalk.Talk
	}{
		{
			name: "normal",

			pageToken: "2020-09-20 03:23:20.995000",
			pageSize:  10,

			expectQueue: "bin-manager.talk-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/talk_chats?page_token=2020-09-20%2003%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeNone,
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   ContentTypeJSON,
				Data:       []byte(`[{"id":"72179880-ec5f-11ec-920e-c77279756b6d","customer_id":"550e8400-e29b-41d4-a716-446655440000","type":"normal"}]`),
			},
			expectRes: []*talktalk.Talk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("72179880-ec5f-11ec-920e-c77279756b6d"),
						CustomerID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440000"),
					},
					Type: talktalk.TypeNormal,
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

			res, err := reqHandler.TalkV1ChatList(ctx, tt.pageToken, tt.pageSize)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
