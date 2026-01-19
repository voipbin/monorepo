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
	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/reactionhandler"
)

func Test_processV1MessagesIDReactionsPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		messageID       uuid.UUID
		reaction        string
		ownerType       string
		ownerID         uuid.UUID
		responseMessage *message.Message
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","emoji":"👍"}`),
			},

			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			reaction:  "👍",
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
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
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"), TMCreate: "2021-11-23 17:55:39.712000"}}},
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "2021-11-23 17:56:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":[],"metadata":{"reactions":[{"emoji":"👍","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","tm_create":"2021-11-23 17:55:39.712000"}]},"tm_create":"2021-11-23 17:55:39.712000","tm_update":"2021-11-23 17:56:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			name: "heart reaction",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"owner_type":"customer","owner_id":"8ede8b40-86ef-11ed-d4fb-e9e028af9801","emoji":"❤️"}`),
			},

			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			reaction:  "❤️",
			ownerType: "customer",
			ownerID:   uuid.FromStringOrNil("8ede8b40-86ef-11ed-d4fb-e9e028af9801"),
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
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "❤️", OwnerType: "customer", OwnerID: uuid.FromStringOrNil("8ede8b40-86ef-11ed-d4fb-e9e028af9801"), TMCreate: "2021-11-23 18:00:00.000000"}}},
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "2021-11-23 18:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":[],"metadata":{"reactions":[{"emoji":"❤️","owner_type":"customer","owner_id":"8ede8b40-86ef-11ed-d4fb-e9e028af9801","tm_create":"2021-11-23 18:00:00.000000"}]},"tm_create":"2021-11-23 17:55:39.712000","tm_update":"2021-11-23 18:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReaction := reactionhandler.NewMockReactionHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reactionHandler: mockReaction,
			}

			ctx := context.Background()
			mockReaction.EXPECT().ReactionAdd(ctx, tt.messageID, tt.reaction, tt.ownerType, tt.ownerID).Return(tt.responseMessage, nil)

			res, err := h.v1MessagesIDReactionsPost(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesIDReactionsPost_error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		messageID uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{invalid json`),
			},
			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil owner id",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"owner_type":"agent","owner_id":"","emoji":"👍"}`),
			},
			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
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
			mockReaction := reactionhandler.NewMockReactionHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reactionHandler: mockReaction,
			}

			ctx := context.Background()
			res, err := h.v1MessagesIDReactionsPost(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesIDReactionsDelete(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		messageID       uuid.UUID
		reaction        string
		ownerType       string
		ownerID         uuid.UUID
		responseMessage *message.Message
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{"owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","emoji":"👍"}`),
			},

			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			reaction:  "👍",
			ownerType: "agent",
			ownerID:   uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"),
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
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{}},
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "2021-11-23 17:56:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":[],"metadata":{"reactions":[]},"tm_create":"2021-11-23 17:55:39.712000","tm_update":"2021-11-23 17:56:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
		{
			name: "remove heart reaction",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{"owner_type":"customer","owner_id":"8ede8b40-86ef-11ed-d4fb-e9e028af9801","emoji":"❤️"}`),
			},

			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			reaction:  "❤️",
			ownerType: "customer",
			ownerID:   uuid.FromStringOrNil("8ede8b40-86ef-11ed-d4fb-e9e028af9801"),
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
				Medias:   []message.Media{},
				Metadata: message.Metadata{Reactions: []message.Reaction{{Emoji: "👍", OwnerType: "agent", OwnerID: uuid.FromStringOrNil("7fcd7990-42eb-11ed-9fa6-b4cd93af9796"), TMCreate: "2021-11-23 17:55:39.712000"}}},
				TMCreate: "2021-11-23 17:55:39.712000",
				TMUpdate: "2021-11-23 18:00:00.000000",
				TMDelete: "9999-01-01 00:00:00.000000",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"9ade9b10-64ed-11ed-b1c8-d6ef95af9798","customer_id":"5e4a0680-804e-11ec-8477-2fea5968d85b","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","chat_id":"6ebc6880-31da-11ed-8e95-a3bc92af9795","type":"normal","text":"Hello world","medias":[],"metadata":{"reactions":[{"emoji":"👍","owner_type":"agent","owner_id":"7fcd7990-42eb-11ed-9fa6-b4cd93af9796","tm_create":"2021-11-23 17:55:39.712000"}]},"tm_create":"2021-11-23 17:55:39.712000","tm_update":"2021-11-23 18:00:00.000000","tm_delete":"9999-01-01 00:00:00.000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockReaction := reactionhandler.NewMockReactionHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reactionHandler: mockReaction,
			}

			ctx := context.Background()
			mockReaction.EXPECT().ReactionRemove(ctx, tt.messageID, tt.reaction, tt.ownerType, tt.ownerID).Return(tt.responseMessage, nil)

			res, err := h.v1MessagesIDReactionsDelete(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1MessagesIDReactionsDelete_error(t *testing.T) {
	tests := []struct {
		name      string
		request   *sock.Request
		messageID uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{invalid json`),
			},
			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
			expectRes: &sock.Response{
				StatusCode: 400,
				DataType:   "application/json",
				Data:       json.RawMessage("{}"),
			},
		},
		{
			name: "nil owner id",
			request: &sock.Request{
				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
				Data:     []byte(`{"owner_type":"agent","owner_id":"","emoji":"👍"}`),
			},
			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
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
			mockReaction := reactionhandler.NewMockReactionHandler(mc)

			h := &listenHandler{
				sockHandler:     mockSock,
				reactionHandler: mockReaction,
			}

			ctx := context.Background()
			res, err := h.v1MessagesIDReactionsDelete(ctx, *tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

// func Test_processV1MessagesIDReactions_unsupported_method(t *testing.T) {
// 	tests := []struct {
// 		name      string
// 		request   *sock.Request
// 		messageID uuid.UUID
// 		expectRes *sock.Response
// 	}{
// 		{
// 			name: "GET method",
// 			request: &sock.Request{
// 				URI:      "/v1/messages/9ade9b10-64ed-11ed-b1c8-d6ef95af9798/reactions",
// 				Method:   sock.RequestMethodGet,
// 				DataType: "application/json",
// 			},
// 			messageID: uuid.FromStringOrNil("9ade9b10-64ed-11ed-b1c8-d6ef95af9798"),
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
// 			mockReaction := reactionhandler.NewMockReactionHandler(mc)
// 
// 			h := &listenHandler{
// 				sockHandler:     mockSock,
// 				reactionHandler: mockReaction,
// 			}
// 
// 			ctx := context.Background()
// 			res, err := h.processV1TalkMessagesIDReactions(ctx, *tt.request)
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
