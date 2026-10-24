package requesthandler

import (
	"context"
	"testing"

	wcwidget "monorepo/bin-webchat-manager/models/widget"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

const webchatQueue = "bin-manager.webchat-manager.request"

func Test_WebchatV1WidgetWrappers(t *testing.T) {
	widgetID := uuid.FromStringOrNil("3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18")
	customerID := uuid.FromStringOrNil("7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30")
	sessionFlowID := uuid.FromStringOrNil("b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52")
	messageFlowID := uuid.FromStringOrNil("f5a7c9e1-3b8d-4a2c-8e6c-8e0a2c4e6f74")

	widgetWant := &wcwidget.Widget{Identity: identity.Identity{ID: widgetID, CustomerID: customerID}, Name: "site chat", Status: wcwidget.StatusActive}
	const widgetResponse = `{"id":"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18","customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","name":"site chat","status":"active"}`

	theme := &wcwidget.ThemeConfig{PrimaryColor: "#112233", HeaderTitle: "Hello"}

	cases := []rpcCase{
		{
			name:           "create_with_theme",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","name":"site chat","session_flow_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","message_flow_id":"f5a7c9e1-3b8d-4a2c-8e6c-8e0a2c4e6f74","session_idle_timeout":900,"theme_config":{"primary_color":"#112233","header_title":"Hello"}}`,
			responseData:   widgetResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetCreate(context.Background(), customerID, "site chat", sessionFlowID, messageFlowID, 900, theme)
			},
			expectRes: widgetWant,
		},
		{
			name:           "create_without_theme_or_timeout",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeJSON,
			expectData:     `{"customer_id":"7c9e1a3b-5d4f-4d86-a028-4a6c8e0a2d30","name":"site chat","session_flow_id":"b1d3f5a7-9c6e-4f08-9a4a-6c8e0a2c4e52","message_flow_id":"f5a7c9e1-3b8d-4a2c-8e6c-8e0a2c4e6f74"}`,
			responseData:   widgetResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetCreate(context.Background(), customerID, "site chat", sessionFlowID, messageFlowID, 0, nil)
			},
			expectRes: widgetWant,
		},
		{
			name:           "get",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets/3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeNone,
			responseData:   widgetResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetGet(context.Background(), widgetID)
			},
			expectRes: widgetWant,
		},
		{
			name:           "list",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets?page_token=token-1&page_size=10",
			expectMethod:   sock.RequestMethodGet,
			expectDataType: ContentTypeJSON,
			expectData:     `{"status":"active"}`,
			responseData:   `[` + widgetResponse + `]`,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetList(context.Background(), "token-1", 10, map[wcwidget.Field]any{wcwidget.FieldStatus: wcwidget.StatusActive})
			},
			expectRes: []*wcwidget.Widget{widgetWant},
		},
		{
			name:           "update",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets/3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18",
			expectMethod:   sock.RequestMethodPut,
			expectDataType: ContentTypeJSON,
			expectData:     `{"name":"renamed","session_flow_id":"00000000-0000-0000-0000-000000000000","message_flow_id":"00000000-0000-0000-0000-000000000000","session_idle_timeout":60}`,
			responseData:   widgetResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetUpdate(context.Background(), widgetID, "renamed", uuid.Nil, uuid.Nil, 60, nil)
			},
			expectRes: widgetWant,
		},
		{
			name:           "delete",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets/3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18",
			expectMethod:   sock.RequestMethodDelete,
			expectDataType: ContentTypeNone,
			responseData:   widgetResponse,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetDelete(context.Background(), widgetID)
			},
			expectRes: widgetWant,
		},
		{
			name:           "direct_hash_regenerate",
			queue:          webchatQueue,
			expectURI:      "/v1/widgets/3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18/direct-hash-regenerate",
			expectMethod:   sock.RequestMethodPost,
			expectDataType: ContentTypeNone,
			responseData:   `{"id":"3a5c7e9b-1d2f-4b64-8c06-2e4a6c8e0b18","direct_hash":"h-2"}`,
			call: func(h *requestHandler) (any, error) {
				return h.WebchatV1WidgetDirectHashRegenerate(context.Background(), widgetID)
			},
			expectRes: &wcwidget.Widget{Identity: identity.Identity{ID: widgetID}, Hash: "h-2"},
		},
	}

	runRPCCases(t, cases)

	runMarshalErrorCase(t, "list_marshal_error", func(h *requestHandler, ctx context.Context) (any, error) {
		return h.WebchatV1WidgetList(ctx, "", 10, map[wcwidget.Field]any{wcwidget.FieldName: make(chan int)})
	})
}
