package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	wcsession "monorepo/bin-webchat-manager/models/session"
)

func Test_WebchatV1SessionWrappers(t *testing.T) {
	sessionID := uuid.FromStringOrNil("a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d")
	customerID := uuid.FromStringOrNil("b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e")
	widgetID := uuid.FromStringOrNil("c3d4e5f6-a7b8-4c9d-8e1f-2a3b4c5d6e7f")

	sessionWant := &wcsession.Session{
		Identity: identity.Identity{ID: sessionID, CustomerID: customerID},
		WidgetID: widgetID,
		Status:   wcsession.StatusActive,
		PageURL:  "https://example.com/pricing",
	}
	const sessionResponse = `{"id":"a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d","customer_id":"b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e","widget_id":"c3d4e5f6-a7b8-4c9d-8e1f-2a3b4c5d6e7f","status":"active","page_url":"https://example.com/pricing"}`
	endedWant := &wcsession.Session{
		Identity: identity.Identity{ID: sessionID, CustomerID: customerID},
		Status:   wcsession.StatusEnded,
	}
	const endedResponse = `{"id":"a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d","customer_id":"b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e","status":"ended"}`

	cases := []rpcCase{
		{
			name:           "create_with_pinned_id",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"id":"a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d","customer_id":"b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e","widget_id":"c3d4e5f6-a7b8-4c9d-8e1f-2a3b4c5d6e7f","page_url":"https://example.com/pricing","referrer":"https://example.com/"}`,
			responseData:   sessionResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionCreate(context.Background(), sessionID, customerID, widgetID, "https://example.com/pricing", "https://example.com/")
			},
			expectRes: sessionWant,
		},
		{
			name:           "create_without_id_page_or_referrer",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"id":"00000000-0000-0000-0000-000000000000","customer_id":"b2c3d4e5-f6a7-4b8c-9d0e-1f2a3b4c5d6e","widget_id":"c3d4e5f6-a7b8-4c9d-8e1f-2a3b4c5d6e7f"}`,
			responseData:   sessionResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionCreate(context.Background(), uuid.Nil, customerID, widgetID, "", "")
			},
			expectRes: sessionWant,
		},
		{
			name:           "get",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions/a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   sessionResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionGet(context.Background(), sessionID)
			},
			expectRes: sessionWant,
		},
		{
			name:           "list_with_filter",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions?page_token=token-1&page_size=20",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"status":"active"}`,
			responseData:   `[` + sessionResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionList(context.Background(), "token-1", 20, map[wcsession.Field]any{wcsession.FieldStatus: "active"})
			},
			expectRes: []*wcsession.Session{sessionWant},
		},
		{
			name:           "list_without_filter",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions?page_token=&page_size=0",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `null`,
			responseData:   `[]`,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionList(context.Background(), "", 0, nil)
			},
			expectRes: []*wcsession.Session{},
		},
		{
			name:           "delete",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions/a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   sessionResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionDelete(context.Background(), sessionID)
			},
			expectRes: sessionWant,
		},
		{
			name:           "end",
			queue:          webchatQueue,
			expectURI:      "/v1/sessions/a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d/end",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeNone,
			responseData:   endedResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1SessionEnd(context.Background(), sessionID)
			},
			expectRes: endedWant,
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.WebchatV1SessionList(ctx, "", 10, map[wcsession.Field]any{wcsession.FieldStatus: make(chan int)})
	})
}
