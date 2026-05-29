package requesthandler

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"testing"

	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_AIV1AIPromptProposalCreate(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		aiID       uuid.UUID
		auditIDs   []uuid.UUID
		language   string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaipromptproposal.AIPromptProposal
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			aiID:       uuid.FromStringOrNil("e8604e8a-ef52-11ef-88be-43d681e412f7"),
			auditIDs:   []uuid.UUID{uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000010")},
			language:   "en-US",

			response: &sock.Response{
				StatusCode: 202,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0f1a2b3-0000-0000-0000-000000000001"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aipromptproposals",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a","ai_id":"e8604e8a-ef52-11ef-88be-43d681e412f7","audit_ids":["d0f1a2b3-0000-0000-0000-000000000010"],"language":"en-US"}`),
			},
			expectRes: &amaipromptproposal.AIPromptProposal{
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

			res, err := reqHandler.AIV1AIPromptProposalCreate(ctx, tt.customerID, tt.aiID, tt.auditIDs, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIPromptProposalList(t *testing.T) {
	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[amaipromptproposal.Field]any

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     []*amaipromptproposal.AIPromptProposal
	}{
		{
			name: "normal",

			pageToken: "2026-01-15T09:30:00.000000Z",
			pageSize:  10,
			filters: map[amaipromptproposal.Field]any{
				amaipromptproposal.FieldDeleted:    false,
				amaipromptproposal.FieldCustomerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d0f1a2b3-0000-0000-0000-000000000001"},{"id":"d0f1a2b3-0000-0000-0000-000000000002"}]`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      fmt.Sprintf("/v1/aipromptproposals?page_token=%s&page_size=10", url.QueryEscape("2026-01-15T09:30:00.000000Z")),
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a","deleted":false}`),
			},
			expectRes: []*amaipromptproposal.AIPromptProposal{
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

			res, err := reqHandler.AIV1AIPromptProposalList(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIPromptProposalGet(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaipromptproposal.AIPromptProposal
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
				URI:    "/v1/aipromptproposals/d0f1a2b3-0000-0000-0000-000000000001",
				Method: sock.RequestMethodGet,
			},
			expectRes: &amaipromptproposal.AIPromptProposal{
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

			res, err := reqHandler.AIV1AIPromptProposalGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIPromptProposalAccept(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		id         uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaipromptproposal.AIPromptProposal
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			id:         uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0f1a2b3-0000-0000-0000-000000000001"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aipromptproposals/d0f1a2b3-0000-0000-0000-000000000001/accept",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a"}`),
			},
			expectRes: &amaipromptproposal.AIPromptProposal{
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

			res, err := reqHandler.AIV1AIPromptProposalAccept(ctx, tt.customerID, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIPromptProposalReject(t *testing.T) {
	tests := []struct {
		name string

		customerID uuid.UUID
		id         uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaipromptproposal.AIPromptProposal
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("ccf7720e-4838-4f97-bb61-3021e14c185a"),
			id:         uuid.FromStringOrNil("d0f1a2b3-0000-0000-0000-000000000001"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"d0f1a2b3-0000-0000-0000-000000000001"}`),
			},

			expectTarget: string(outline.QueueNameAIRequest),
			expectRequest: &sock.Request{
				URI:      "/v1/aipromptproposals/d0f1a2b3-0000-0000-0000-000000000001/reject",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"ccf7720e-4838-4f97-bb61-3021e14c185a"}`),
			},
			expectRes: &amaipromptproposal.AIPromptProposal{
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

			res, err := reqHandler.AIV1AIPromptProposalReject(ctx, tt.customerID, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AIV1AIPromptProposalDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *amaipromptproposal.AIPromptProposal
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
				URI:    "/v1/aipromptproposals/d0f1a2b3-0000-0000-0000-000000000001",
				Method: sock.RequestMethodDelete,
			},
			expectRes: &amaipromptproposal.AIPromptProposal{
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

			res, err := reqHandler.AIV1AIPromptProposalDelete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
