package listenhandler

import (
	"context"
	"encoding/json"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-talk-manager/models/talk"
	"monorepo/bin-talk-manager/pkg/talkhandler"
)

func Test_processV1TalkChatsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		customerID uuid.UUID
		talkType   talk.Type

		responseTalk *talk.Talk
		expectRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chats",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"normal"}`),
			},

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			talkType:   talk.TypeNormal,

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type:     talk.TypeNormal,
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 201,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"normal","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			name: "group talk",
			request: &sock.Request{
				URI:      "/v1/chats",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"group"}`),
			},

			customerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
			talkType:   talk.TypeGroup,

			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type:     talk.TypeGroup,
				TMCreate: "2021-11-23 18:00:00.000000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 201,
				DataType:   "application/json",
				Data:       []byte(`{"id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"group","tm_create":"2021-11-23 18:00:00.000000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTalk := talkhandler.NewMockTalkHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				talkHandler: mockTalk,
			}

			ctx := context.Background()
			mockTalk.EXPECT().TalkCreate(ctx, tt.customerID, tt.talkType).Return(tt.responseTalk, nil)

			res, err := h.v1ChatsPost(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalkChatsPost_error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/chats",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{invalid json`),
			},
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil customer id",
			request: &sock.Request{
				URI:      "/v1/chats",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"","type":"normal"}`),
			},
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTalk := talkhandler.NewMockTalkHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				talkHandler: mockTalk,
			}

			ctx := context.Background()
			res, err := h.v1ChatsPost(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalkChatsGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string

		responseTalks []*talk.Talk
		expectRes     *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chats?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},

			pageSize:  10,
			pageToken: "2021-11-23 17:55:39.712000",

			responseTalks: []*talk.Talk{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
						CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					},
					Type:     talk.TypeNormal,
					TMCreate: "2021-11-23 17:55:39.712000",
					TMUpdate: "9999-01-01 00:00:00.000000",
					TMDelete: "9999-01-01 00:00:00.000000",
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"normal","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
		{
			name: "default page size",
			request: &sock.Request{
				URI:      "/v1/chats",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},

			pageSize:  50,
			pageToken: "",

			responseTalks: []*talk.Talk{},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTalk := talkhandler.NewMockTalkHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				talkHandler: mockTalk,
			}

			ctx := context.Background()
			mockTalk.EXPECT().TalkList(ctx, nil, tt.pageToken, tt.pageSize).Return(tt.responseTalks, nil)

			res, err := h.v1ChatsGet(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalkChatsIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		talkID       uuid.UUID
		responseTalk *talk.Talk
		expectRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chats/6ebc6880-31da-11ed-8e95-a3bc92af9795",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			talkID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type:     talk.TypeNormal,
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"normal","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTalk := talkhandler.NewMockTalkHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				talkHandler: mockTalk,
			}

			ctx := context.Background()
			mockTalk.EXPECT().TalkGet(ctx, tt.talkID).Return(tt.responseTalk, nil)

			res, err := h.v1ChatsIDGet(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1TalkChatsIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		talkID       uuid.UUID
		responseTalk *talk.Talk
		expectRes    *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/chats/6ebc6880-31da-11ed-8e95-a3bc92af9795",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			talkID: uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
			responseTalk: &talk.Talk{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Type:     talk.TypeNormal,
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "2021-11-23 18:00:00.000000",
				TMDelete: "2021-11-23 18:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","type":"normal","tm_create":"2021-11-23 17:55:39.712000","tm_update":"2021-11-23 18:00:00.000000","tm_delete":"2021-11-23 18:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTalk := talkhandler.NewMockTalkHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				talkHandler: mockTalk,
			}

			ctx := context.Background()
			mockTalk.EXPECT().TalkDelete(ctx, tt.talkID).Return(tt.responseTalk, nil)

			res, err := h.v1ChatsIDDelete(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func Test_processV1TalkChatsID_unsupported_method(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		request   *sock.Request
// 		expectRes *sock.Response
// 	}{
// 		{
// 			name: "PUT method",
// 			request: &sock.Request{
// 				URI:      "/v1/chats/6ebc6880-31da-11ed-8e95-a3bc92af9795",
// 				Method:   sock.RequestMethodPut,
// 				DataType: "application/json",
// 			},
// 			expectRes: &sock.Response{
// 				StatusCode: 405,
// 				DataType:   "application/json",
// 				Data:       json.RawMessage("{}"),
// 			},
// 		},
// 	}
// 
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()
// 
// 			mockSock := sockhandler.NewMockSockHandler(mc)
// 			mockTalk := talkhandler.NewMockTalkHandler(mc)
// 
// 			h := &listenHandler{
// 				sockHandler: mockSock,
// 				talkHandler: mockTalk,
// 			}
// 
// 			ctx := context.Background()
// 			res, err := h.processV1TalkChatsID(ctx, *tt.request)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 
// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}
// 		})
// 	}
// }
