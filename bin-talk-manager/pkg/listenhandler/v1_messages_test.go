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
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/messagehandler"
)

func Test_processV1MessagesPost(t *testing.T) {
	parentID := uuid.FromStringOrNil("8fde8a00-53fc-11ed-a0b7-c5de94af9797")
	_ = parentID // Suppress unused variable warning if needed

	tests := []struct {
		name    string
		request *sock.Request

		createReq       messagehandler.MessageCreateRequest
		responseMessage *message.Message
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","type":"normal","text":"Hello world","medias":"[]"}`),
			},

			createReq: messagehandler.MessageCreateRequest{
				CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				ChatID:     uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				ParentID:   nil,
				OwnerType:  "agent",
				OwnerID:    uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				Type:       "normal",
				Text:       "Hello world",
				Medias:     "[]",
			},
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				},
				ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				ParentID: nil,
				Type:     message.TypeNormal,
				Text:     "Hello world",
				Medias:   "[]",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 201,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":"[]","metadata":"{\"reactions\":[]}","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			name: "with parent id",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","parent_id":"8fde8a00-53fc-11ed-a0b7-c5de94af9797","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","type":"normal","text":"Reply to message","medias":"[]"}`),
			},

			createReq: messagehandler.MessageCreateRequest{
				CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				ChatID:     uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				ParentID:   &parentID,
				OwnerType:  "agent",
				OwnerID:    uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				Type:       "normal",
				Text:       "Reply to message",
				Medias:     "[]",
			},
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("aaef9c20-75fe-11ed-c2d9-e7f006af9799"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				},
				ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				ParentID: &parentID,
				Type:     message.TypeNormal,
				Text:     "Reply to message",
				Medias:   "[]",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2021-11-23 18:00:00.000000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 201,
				DataType:   "application/json",
				Data:       []byte(`{"id":"aaef9c20-75fe-11ed-c2d9-e7f006af9799","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","parent_id":"8fde8a00-53fc-11ed-a0b7-c5de94af9797","type":"normal","text":"Reply to message","medias":"[]","metadata":"{\"reactions\":[]}","tm_create":"2021-11-23 18:00:00.000000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
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

			ctx := context.Background()
			mockMessage.EXPECT().MessageCreate(ctx, tt.createReq).Return(tt.responseMessage, nil)

			res, err := h.v1MessagesPost(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesPost_error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		expectRes *sock.Response
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/messages",
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
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","type":"normal","text":"test"}`),
			},
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil chat id",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","chat_id":"","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","type":"normal","text":"test"}`),
			},
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil owner id",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","owner_type":"agent","owner_id":"","type":"normal","text":"test"}`),
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
			}

			ctx := context.Background()
			res, err := h.v1MessagesPost(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string

		responseMessages []*message.Message
		expectRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages?page_size=10&page_token=2021-11-23%2017:55:39.712000",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},

			pageSize:  10,
			pageToken: "2021-11-23 17:55:39.712000",

			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID:         uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
						CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
					},
					Owner: commonidentity.Owner{
						OwnerType: "agent",
						OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
					},
					ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
					Type:     message.TypeNormal,
					Text:     "Hello world",
					Medias:   "[]",
					Metadata: `{"reactions":[]}`,
					TMCreate: "2021-11-23 17:55:39.712000",
					TMUpdate: "9999-01-01 00:00:00.000000",
					TMDelete: "9999-01-01 00:00:00.000000",
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":"[]","metadata":"{\"reactions\":[]}","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}]`),
			},
		},
		{
			name: "default page size",
			request: &sock.Request{
				URI:      "/v1/messages",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{}`),
			},

			pageSize:  50,
			pageToken: "",

			responseMessages: []*message.Message{},
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &listenHandler{
				sockHandler:    mockSock,
				messageHandler: mockMessage,
				utilHandler:    mockUtil,
			}

			ctx := context.Background()

			// Set expectations for filter parsing
			mockUtil.EXPECT().ParseFiltersFromRequestBody(tt.request.Data).Return(map[string]any{}, nil)
			mockMessage.EXPECT().MessageList(ctx, gomock.Any(), tt.pageToken, tt.pageSize).Return(tt.responseMessages, nil)

			res, err := h.v1MessagesGet(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesIDGet(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		messageID       uuid.UUID
		responseMessage *message.Message
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				},
				ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				Type:     message.TypeNormal,
				Text:     "Hello world",
				Medias:   "[]",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "9999-01-01 00:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":"[]","metadata":"{\"reactions\":[]}","tm_create":"2021-11-23 17:55:39.712000","tm_update":"9999-01-01 00:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
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

			ctx := context.Background()
			mockMessage.EXPECT().MessageGet(ctx, tt.messageID).Return(tt.responseMessage, nil)

			res, err := h.v1MessagesIDGet(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesIDDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		messageID       uuid.UUID
		responseMessage *message.Message
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
					CustomerID: uuid.FromStringOrNil("5e4a0680-804e-11ec-8477-2fea5968d85b"),
				},
				Owner: commonidentity.Owner{
					OwnerType: "agent",
					OwnerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
				},
				ChatID:   uuid.FromStringOrNil("6ebc6880-31da-11ed-8e95-a3bc92af9795"),
				Type:     message.TypeNormal,
				Text:     "Hello world",
				Medias:   "[]",
				Metadata: `{"reactions":[]}`,
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "2021-11-23 18:00:00.000000",
				TMDelete: "2021-11-23 18:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":"[]","metadata":"{\"reactions\":[]}","tm_create":"2021-11-23 17:55:39.712000","tm_update":"2021-11-23 18:00:00.000000","tm_delete":"2021-11-23 18:00:00.000000"}`),
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

			ctx := context.Background()
			mockMessage.EXPECT().MessageDelete(ctx, tt.messageID).Return(tt.responseMessage, nil)

			res, err := h.v1MessagesIDDelete(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func Test_processV1MessagesID_unsupported_method(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		request   *sock.Request
// 		expectRes *sock.Response
// 	}{
// 		{
// 			name: "PUT method",
// 			request: &sock.Request{
// 				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798",
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
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)
// 
// 			h := &listenHandler{
// 				sockHandler:    mockSock,
// 				messageHandler: mockMessage,
// 			}
// 
// 			ctx := context.Background()
// 			res, err := h.v1MessagesIDGet(ctx, *tt.request)
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
