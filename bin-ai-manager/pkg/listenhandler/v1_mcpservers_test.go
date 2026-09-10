package listenhandler

import (
	stderrors "errors"
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-ai-manager/pkg/mcpserverhandler"
)

func Test_processV1McpServersGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMcpServers []*mcpserver.McpServer

		expectPageSize  uint64
		expectPageToken string
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/mcp_servers?page_size=10&page_token=2020-05-03T21:35:02.809Z&filter_customer_id=24676972-7f49-11ec-bc89-b7d33e9d3ea8&filter_deleted=false",
				Method: sock.RequestMethodGet,
			},

			responseMcpServers: []*mcpserver.McpServer{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("0b61dcbe-a770-11ed-bab4-2fc1dac66672"),
					},
					SecretCiphertext: []byte("must-not-leak"),
					SecretNonce:      []byte("must-not-leak"),
					KeyVersion:       7,
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-05-03T21:35:02.809Z",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"0b61dcbe-a770-11ed-bab4-2fc1dac66672","customer_id":"00000000-0000-0000-0000-000000000000","has_secret":false,"tm_create":null,"tm_update":null,"tm_delete":null}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			mockMcpServer.EXPECT().List(gomock.Any(), tt.expectPageSize, tt.expectPageToken, gomock.Any()).Return(tt.responseMcpServers, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			// SECURITY-CRITICAL: the marshaled response must never contain the
			// encrypted secret material.
			if data := string(res.Data); reflect.DeepEqual(res, tt.expectRes) {
				for _, forbidden := range []string{"secret_ciphertext", "secret_nonce", "key_version"} {
					if got := data; got != "" && contains(got, forbidden) {
						t.Errorf("Response leaked forbidden field %q. data: %s", forbidden, got)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}

func Test_processV1McpServersGet_error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		responseErr error

		expectPageSize  uint64
		expectPageToken string
	}{
		{
			name: "handler returns error",
			request: &sock.Request{
				URI:    "/v1/mcp_servers?page_size=10&page_token=",
				Method: sock.RequestMethodGet,
			},
			responseErr: stderrors.New("mcp server list failed"),

			expectPageSize:  10,
			expectPageToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			mockMcpServer.EXPECT().List(gomock.Any(), tt.expectPageSize, tt.expectPageToken, gomock.Any()).Return(nil, tt.responseErr)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil || res.StatusCode < 400 {
				t.Errorf("Wrong match. expect: error status code, got: %v", res)
			}
		})
	}
}

func Test_processV1McpServersPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMcpServer *mcpserver.McpServer

		expectCustomerID   uuid.UUID
		expectName         string
		expectDetail       string
		expectURL          string
		expectStatus       mcpserver.Status
		expectAuthType     mcpserver.AuthType
		expectAPIKeyHeader string
		expectSecret       string
		expectRes          *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/mcp_servers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"58e7502c-a770-11ed-9b86-7fabe2dba847","name":"test mcp server","detail":"test detail","url":"https://mcp.example.com","auth_type":"bearer","api_key_header":"","secret":"top-secret"}`),
			},

			responseMcpServer: &mcpserver.McpServer{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("59230ca2-a770-11ed-b5dd-2783587ed477"),
				},
			},

			expectCustomerID:   uuid.FromStringOrNil("58e7502c-a770-11ed-9b86-7fabe2dba847"),
			expectName:         "test mcp server",
			expectDetail:       "test detail",
			expectURL:          "https://mcp.example.com",
			expectStatus:       mcpserver.StatusActive,
			expectAuthType:     mcpserver.AuthTypeBearer,
			expectAPIKeyHeader: "",
			expectSecret:       "top-secret",

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"59230ca2-a770-11ed-b5dd-2783587ed477","customer_id":"00000000-0000-0000-0000-000000000000","has_secret":false,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			mockMcpServer.EXPECT().Create(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectName,
				tt.expectDetail,
				tt.expectURL,
				tt.expectStatus,
				tt.expectAuthType,
				tt.expectAPIKeyHeader,
				tt.expectSecret,
			).Return(tt.responseMcpServer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1McpServersPost_error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/mcp_servers",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{invalid`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil || res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

func Test_processV1McpServersIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMcpServer *mcpserver.McpServer

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/mcp_servers/de740384-a770-11ed-afab-5f9c8a447889",
				Method: sock.RequestMethodGet,
			},

			&mcpserver.McpServer{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),
				},
			},

			uuid.FromStringOrNil("de740384-a770-11ed-afab-5f9c8a447889"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de740384-a770-11ed-afab-5f9c8a447889","customer_id":"00000000-0000-0000-0000-000000000000","has_secret":false,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			mockMcpServer.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseMcpServer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1McpServersIDGet_error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "invalid id",
			request: &sock.Request{
				URI:    "/v1/mcp_servers/not-a-uuid",
				Method: sock.RequestMethodGet,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// "not-a-uuid" doesn't match the route regexp, so the request
			// never reaches processV1McpServersIDGet; dispatch itself 404s.
			if res == nil || res.StatusCode != 404 {
				t.Errorf("Wrong match. expect: 404, got: %v", res)
			}
		})
	}
}

func Test_processV1McpServersIDPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMcpServer *mcpserver.McpServer

		expectID            uuid.UUID
		expectName          string
		expectDetail        string
		expectURL           string
		expectStatus        mcpserver.Status
		expectAuthType      mcpserver.AuthType
		expectAPIKeyHeader  string
		expectSecret        *string
		expectRes           *sock.Response
	}{
		{
			name: "normal - secret provided",
			request: &sock.Request{
				URI:      "/v1/mcp_servers/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"updated mcp server","detail":"updated detail","url":"https://mcp.example.com","status":"active","auth_type":"api_key","api_key_header":"X-API-Key","secret":"new-secret"}`),
			},

			responseMcpServer: &mcpserver.McpServer{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
				},
			},

			expectID:           uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			expectName:         "updated mcp server",
			expectDetail:       "updated detail",
			expectURL:          "https://mcp.example.com",
			expectStatus:       mcpserver.StatusActive,
			expectAuthType:     mcpserver.AuthTypeAPIKey,
			expectAPIKeyHeader: "X-API-Key",
			expectSecret:       strPtr("new-secret"),

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa4d3b6a-f82f-11ed-9176-d32f5705e10c","customer_id":"00000000-0000-0000-0000-000000000000","has_secret":false,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
		{
			name: "normal - secret omitted means nil pointer (unchanged)",
			request: &sock.Request{
				URI:      "/v1/mcp_servers/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"name":"updated mcp server","detail":"updated detail","url":"https://mcp.example.com","status":"active","auth_type":"bearer"}`),
			},

			responseMcpServer: &mcpserver.McpServer{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
				},
			},

			expectID:           uuid.FromStringOrNil("fa4d3b6a-f82f-11ed-9176-d32f5705e10c"),
			expectName:         "updated mcp server",
			expectDetail:       "updated detail",
			expectURL:          "https://mcp.example.com",
			expectStatus:       mcpserver.StatusActive,
			expectAuthType:     mcpserver.AuthTypeBearer,
			expectAPIKeyHeader: "",
			expectSecret:       nil,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"fa4d3b6a-f82f-11ed-9176-d32f5705e10c","customer_id":"00000000-0000-0000-0000-000000000000","has_secret":false,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			mockMcpServer.EXPECT().Update(
				gomock.Any(),
				tt.expectID,
				tt.expectName,
				tt.expectDetail,
				tt.expectURL,
				tt.expectStatus,
				tt.expectAuthType,
				tt.expectAPIKeyHeader,
				tt.expectSecret,
			).Return(tt.responseMcpServer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1McpServersIDPut_error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "invalid json",
			request: &sock.Request{
				URI:      "/v1/mcp_servers/fa4d3b6a-f82f-11ed-9176-d32f5705e10c",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{invalid`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res == nil || res.StatusCode != 400 {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

func Test_processV1McpServersIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		responseMcpServer *mcpserver.McpServer

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/mcp_servers/de99e522-a770-11ed-a0ab-5b39ee2db203",
				Method: sock.RequestMethodDelete,
			},

			&mcpserver.McpServer{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),
				},
			},

			uuid.FromStringOrNil("de99e522-a770-11ed-a0ab-5b39ee2db203"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"de99e522-a770-11ed-a0ab-5b39ee2db203","customer_id":"00000000-0000-0000-0000-000000000000","has_secret":false,"tm_create":null,"tm_update":null,"tm_delete":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			mockMcpServer.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseMcpServer, nil)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_processV1McpServersIDDelete_error(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "invalid id",
			request: &sock.Request{
				URI:    "/v1/mcp_servers/not-a-uuid",
				Method: sock.RequestMethodDelete,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockMcpServer := mcpserverhandler.NewMockMcpServerHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				mcpServerHandler: mockMcpServer,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// "not-a-uuid" doesn't match the route regexp, so the request
			// never reaches processV1McpServersIDDelete; dispatch 404s.
			if res == nil || res.StatusCode != 404 {
				t.Errorf("Wrong match. expect: 404, got: %v", res)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
