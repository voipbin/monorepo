package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	wcmessage "monorepo/bin-webchat-manager/models/message"
)

func Test_WebchatV1MessageWrappers(t *testing.T) {
	messageID := uuid.FromStringOrNil("d4e5f6a7-b8c9-4d0e-9f2a-3b4c5d6e7f80")
	customerID := uuid.FromStringOrNil("e5f6a7b8-c9d0-4e1f-8a3b-4c5d6e7f8091")
	sessionID := uuid.FromStringOrNil("f6a7b8c9-d0e1-4f2a-9b4c-5d6e7f8091a2")
	senderID := uuid.FromStringOrNil("a7b8c9d0-e1f2-4a3b-8c5d-6e7f8091a2b3")

	messageWant := &wcmessage.Message{
		Identity:  identity.Identity{ID: messageID, CustomerID: customerID},
		SessionID: sessionID,
		Direction: wcmessage.DirectionInbound,
		Text:      "hello",
	}
	const messageResponse = `{"id":"d4e5f6a7-b8c9-4d0e-9f2a-3b4c5d6e7f80","customer_id":"e5f6a7b8-c9d0-4e1f-8a3b-4c5d6e7f8091","session_id":"f6a7b8c9-d0e1-4f2a-9b4c-5d6e7f8091a2","direction":"inbound","text":"hello"}`

	cases := []rpcCase{
		{
			name:           "create_outbound_with_sender",
			queue:          webchatQueue,
			expectURI:      "/v1/messages",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"e5f6a7b8-c9d0-4e1f-8a3b-4c5d6e7f8091","session_id":"f6a7b8c9-d0e1-4f2a-9b4c-5d6e7f8091a2","direction":"outbound","sender_id":"a7b8c9d0-e1f2-4a3b-8c5d-6e7f8091a2b3","text":"hi there"}`,
			responseData:   messageResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1MessageCreate(context.Background(), customerID, sessionID, wcmessage.DirectionOutbound, senderID, "hi there")
			},
			expectRes: messageWant,
		},
		{
			name:           "create_inbound_without_sender",
			queue:          webchatQueue,
			expectURI:      "/v1/messages",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"e5f6a7b8-c9d0-4e1f-8a3b-4c5d6e7f8091","session_id":"f6a7b8c9-d0e1-4f2a-9b4c-5d6e7f8091a2","direction":"inbound","sender_id":"00000000-0000-0000-0000-000000000000","text":"hello"}`,
			responseData:   messageResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1MessageCreate(context.Background(), customerID, sessionID, wcmessage.DirectionInbound, uuid.Nil, "hello")
			},
			expectRes: messageWant,
		},
		{
			name:           "get",
			queue:          webchatQueue,
			expectURI:      "/v1/messages/d4e5f6a7-b8c9-4d0e-9f2a-3b4c5d6e7f80",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   messageResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1MessageGet(context.Background(), messageID)
			},
			expectRes: messageWant,
		},
		{
			name:           "list_with_filter",
			queue:          webchatQueue,
			expectURI:      "/v1/messages?page_token=token-1&page_size=20",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"direction":"inbound"}`,
			responseData:   `[` + messageResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1MessageList(context.Background(), "token-1", 20, map[wcmessage.Field]any{wcmessage.FieldDirection: "inbound"})
			},
			expectRes: []*wcmessage.Message{messageWant},
		},
		{
			name:           "list_without_filter",
			queue:          webchatQueue,
			expectURI:      "/v1/messages?page_token=&page_size=0",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `null`,
			responseData:   `[]`,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1MessageList(context.Background(), "", 0, nil)
			},
			expectRes: []*wcmessage.Message{},
		},
		{
			name:           "delete",
			queue:          webchatQueue,
			expectURI:      "/v1/messages/d4e5f6a7-b8c9-4d0e-9f2a-3b4c5d6e7f80",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   messageResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1MessageDelete(context.Background(), messageID)
			},
			expectRes: messageWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.WebchatV1MessageList(ctx, "", 10, map[wcmessage.Field]any{wcmessage.FieldDirection: make(chan int)})
	})
}
