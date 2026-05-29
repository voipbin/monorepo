package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	amaiaudit "monorepo/bin-ai-manager/models/aiaudit"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIV1AIAuditCreate(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		aicallID   uuid.UUID
		language   string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []*amaiaudit.AIAudit
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			aicallID:   uuid.FromStringOrNil("e8604e8a-ef52-11ef-88be-43d681e412f7"),
			language:   "en-US",

			response: &sock.Response{
				StatusCode: 202,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d0f1a2b3-0000-0000-0000-000000000001"}]`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aiaudits",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a","aicall_id":"e8604e8a-ef52-11ef-88be-43d681e412f7","language":"en-US"}`),
			},
			expectRes: []*amaiaudit.AIAudit{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),
					},
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIAuditCreate(ctx, tt.customerID, tt.aicallID, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIAuditList(t *testing.T) {
	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[amaiaudit.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []*amaiaudit.AIAudit
	}{
		{
			name: "normal",

			pageToken: "2026-01-15T09:30:00.000000Z",
			pageSize:  10,
			filters: map[amaiaudit.Field]any{
				amaiaudit.FieldDeleted:    false,
				amaiaudit.FieldCustomerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d0f1a2b3-0000-0000-0000-000000000001"},{"id":"d0f1a2b3-0000-0000-0000-000000000002"}]`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      fmt.Sprintf("/v1/aiaudits?page_token=%s&page_size=10", url.QueryEscape("2026-01-15T09:30:00.000000Z")),
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a","deleted":false}`),
			},
			expectRes: []*amaiaudit.AIAudit{
				{Identity: identity.Identity{ID: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001")}},
				{Identity: identity.Identity{ID: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000002")}},
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIAuditList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIAuditGet(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaiaudit.AIAudit
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0f1a2b3-0000-0000-0000-000000000001"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/aiaudits/d0f1a2b3-0000-0000-0000-000000000001",
				Method: sock.RequestMethodGet,
			},
			expectRes: &amaiaudit.AIAudit{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIAuditGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIAuditDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaiaudit.AIAudit
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0f1a2b3-0000-0000-0000-000000000001"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:    "/v1/aiaudits/d0f1a2b3-0000-0000-0000-000000000001",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &amaiaudit.AIAudit{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.AIV1AIAuditDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIAuditCreateWithDelay(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		aicallID   uuid.UUID
		language   string
		delay      int

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			aicallID:   uuid.FromStringOrNil("e8604e8a-ef52-11ef-88be-43d681e412f7"),
			language:   "en-US",
			delay:      1000,

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aiaudits",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a","aicall_id":"e8604e8a-ef52-11ef-88be-43d681e412f7","language":"en-US"}`),
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

			mockSock.EXPECT().RequestPublishWithDelay(tt.expectTarget, tt.expectRequest, tt.delay).Return(nil)

			err := reqHandler.AIV1AIAuditCreateWithDelay(ctx, tt.customerID, tt.aicallID, tt.language, tt.delay)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}
		})
	}
}
