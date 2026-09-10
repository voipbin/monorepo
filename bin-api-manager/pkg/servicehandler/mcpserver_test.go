package servicehandler

import (
	"context"
	"reflect"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/dbhandler"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_McpServerCreate(t *testing.T) {

	tests := []struct {
		name string

		agent        *auth.AuthIdentity
		reqName      string
		reqDetail    string
		reqURL       string
		reqAuthType  ammcpserver.AuthType
		reqAPIHeader string
		reqSecret    string

		response  *ammcpserver.McpServer
		expectRes *ammcpserver.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			"test name",
			"test detail",
			"https://mcp.example.com/mcp",
			ammcpserver.AuthTypeBearer,
			"",
			"test-secret",

			&ammcpserver.McpServer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&ammcpserver.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1McpServerCreate(ctx, tt.agent.CustomerID, tt.reqName, tt.reqDetail, tt.reqURL, tt.reqAuthType, tt.reqAPIHeader, tt.reqSecret).Return(tt.response, nil)

			res, err := h.McpServerCreate(ctx, tt.agent, tt.reqName, tt.reqDetail, tt.reqURL, tt.reqAuthType, tt.reqAPIHeader, tt.reqSecret)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_McpServerGet(t *testing.T) {

	tests := []struct {
		name string

		agent *auth.AuthIdentity
		id    uuid.UUID

		response  *ammcpserver.McpServer
		expectRes *ammcpserver.WebhookMessage
	}{
		{
			"normal, same customer",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),

			&ammcpserver.McpServer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&ammcpserver.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1McpServerGet(ctx, tt.id).Return(tt.response, nil)

			res, err := h.McpServerGet(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_McpServerGet_IDOR pins that a customer cannot read another
// customer's McpServer row by guessing/enumerating its id -- the
// permission check must be evaluated against the FETCHED row's own
// CustomerID, not the caller's, since the id is client-supplied and not
// itself scoped to the caller's customer.
func Test_McpServerGet_IDOR(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"), // caller's own customer
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
	targetID := uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63")

	// The row belongs to a DIFFERENT customer than the caller.
	mockReq.EXPECT().AIV1McpServerGet(ctx, targetID).Return(&ammcpserver.McpServer{
		Identity: commonidentity.Identity{
			ID:         targetID,
			CustomerID: uuid.FromStringOrNil("aaaaaaaa-8e5f-11ee-97b2-cfe7337b701c"), // NOT agent.CustomerID
		},
	}, nil)

	res, err := h.McpServerGet(ctx, agent, targetID)
	if err == nil {
		t.Errorf("Wrong match. expect: permission denied error, got: nil (res: %v)", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result on IDOR rejection, got: %v", res)
	}
}

func Test_McpServerUpdate(t *testing.T) {

	tests := []struct {
		name string

		agent  *auth.AuthIdentity
		id     uuid.UUID
		secret *string

		getResponse    *ammcpserver.McpServer
		updateResponse *ammcpserver.McpServer
		expectRes      *ammcpserver.WebhookMessage
	}{
		{
			name: "secret nil leaves the existing secret untouched",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			id:     uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
			secret: nil,

			getResponse: &ammcpserver.McpServer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			updateResponse: &ammcpserver.McpServer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &ammcpserver.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1McpServerGet(ctx, tt.id).Return(tt.getResponse, nil)
			mockReq.EXPECT().AIV1McpServerUpdate(ctx, tt.id, "", "", "", ammcpserver.Status(""), ammcpserver.AuthType(""), "", tt.secret).Return(tt.updateResponse, nil)

			res, err := h.McpServerUpdate(ctx, tt.agent, tt.id, "", "", "", ammcpserver.Status(""), ammcpserver.AuthType(""), "", tt.secret)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_McpServerUpdate_IDOR mirrors Test_McpServerGet_IDOR for the write
// path: updating a different customer's row must be rejected before any
// AIV1McpServerUpdate RPC is issued.
func Test_McpServerUpdate_IDOR(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})
	targetID := uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63")

	mockReq.EXPECT().AIV1McpServerGet(ctx, targetID).Return(&ammcpserver.McpServer{
		Identity: commonidentity.Identity{
			ID:         targetID,
			CustomerID: uuid.FromStringOrNil("aaaaaaaa-8e5f-11ee-97b2-cfe7337b701c"),
		},
	}, nil)
	// No AIV1McpServerUpdate expectation set -- gomock's strict controller
	// fails the test if the update RPC is ever issued after the IDOR check
	// should have short-circuited.

	res, err := h.McpServerUpdate(ctx, agent, targetID, "new name", "", "", ammcpserver.Status(""), ammcpserver.AuthType(""), "", nil)
	if err == nil {
		t.Errorf("Wrong match. expect: permission denied error, got: nil (res: %v)", res)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil result on IDOR rejection, got: %v", res)
	}
}

func Test_McpServerDelete(t *testing.T) {

	tests := []struct {
		name string

		agent *auth.AuthIdentity
		id    uuid.UUID

		getResponse    *ammcpserver.McpServer
		deleteResponse *ammcpserver.McpServer
		expectRes      *ammcpserver.WebhookMessage
	}{
		{
			"normal",
			auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&ammcpserver.McpServer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&ammcpserver.McpServer{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&ammcpserver.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1McpServerGet(ctx, tt.id).Return(tt.getResponse, nil)
			mockReq.EXPECT().AIV1McpServerDelete(ctx, tt.id).Return(tt.deleteResponse, nil)

			res, err := h.McpServerDelete(ctx, tt.agent, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

// Test_McpServerCreate_DirectAccessNotSupported pins that a direct
// (unauthenticated-customer / SIP-direct-access) identity cannot register
// an MCP server, matching every other admin-resource create path in this
// package (e.g. AICreate).
func Test_McpServerCreate_DirectAccessNotSupported(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	agent := auth.NewDirectIdentity(&auth.DirectScope{
		CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
	})

	res, err := h.McpServerCreate(ctx, agent, "name", "detail", "https://mcp.example.com/mcp", ammcpserver.AuthTypeNone, "", "")
	if err != serviceerrors.ErrDirectAccessNotSupported {
		t.Errorf("Wrong match. expect: %v, got: %v", serviceerrors.ErrDirectAccessNotSupported, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}
}

func Test_McpServerGetsByCustomerID(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := serviceHandler{
		reqHandler: mockReq,
		dbHandler:  mockDB,
	}
	ctx := context.Background()

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
		Permission: amagent.PermissionCustomerAdmin,
	})

	mockReq.EXPECT().AIV1McpServerList(ctx, gomock.Any(), uint64(100), gomock.Any()).Return([]*ammcpserver.McpServer{
		{
			Identity: commonidentity.Identity{
				ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}, nil)

	res, err := h.McpServerGetsByCustomerID(ctx, agent, 100, "2020-09-20T03:23:20.995000Z")
	if err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	if len(res) != 1 {
		t.Errorf("Wrong match. expect: 1 result, got: %d", len(res))
	}
}
