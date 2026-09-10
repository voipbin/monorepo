package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostMcpservers(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseMcpServer *ammcpserver.WebhookMessage

		expectedName         string
		expectedDetail       string
		expectedURL          string
		expectedAuthType     ammcpserver.AuthType
		expectedAPIKeyHeader string
		expectedSecret       string
		expectedRes          string
	}{
		{
			name: "normal with bearer auth",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/mcpservers",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","url":"https://mcp.example.com/mcp","auth_type":"bearer","secret":"test-secret"}`),

			responseMcpServer: &ammcpserver.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				},
			},

			expectedName:         "test name",
			expectedDetail:       "test detail",
			expectedURL:          "https://mcp.example.com/mcp",
			expectedAuthType:     ammcpserver.AuthTypeBearer,
			expectedAPIKeyHeader: "",
			expectedSecret:       "test-secret",
			expectedRes:          `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","name":"","url":"","has_secret":false}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().McpServerCreate(
				req.Context(),
				tt.agent,
				tt.expectedName,
				tt.expectedDetail,
				tt.expectedURL,
				tt.expectedAuthType,
				tt.expectedAPIKeyHeader,
				tt.expectedSecret,
			).Return(tt.responseMcpServer, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
			}
		})
	}
}

func Test_GetMcpserversId(t *testing.T) {

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	id := uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/mcpservers/"+id.String(), nil)
	mockSvc.EXPECT().McpServerGet(req.Context(), agent, id).Return(&ammcpserver.WebhookMessage{
		Identity: commonidentity.Identity{ID: id},
	}, nil)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

// Test_PutMcpserversId_SecretOmitted pins the wire-level half of the
// design's *string PUT secret semantics: when the JSON body omits
// "secret" entirely, the servicehandler must receive a nil pointer (not
// a pointer to an empty string) so an existing secret is left untouched
// rather than being cleared.
func Test_PutMcpserversId_SecretOmitted(t *testing.T) {

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	id := uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// "secret" key is absent from the body entirely.
	body := []byte(`{"name":"renamed"}`)
	req, _ := http.NewRequest("PUT", "/mcpservers/"+id.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	mockSvc.EXPECT().McpServerUpdate(
		req.Context(),
		agent,
		id,
		"renamed",
		"",
		"",
		ammcpserver.Status(""),
		ammcpserver.AuthType(""),
		"",
		(*string)(nil),
	).Return(&ammcpserver.WebhookMessage{Identity: commonidentity.Identity{ID: id}}, nil)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

// Test_PutMcpserversId_SecretExplicitClear pins the other half: a body
// with "secret":"" must reach the servicehandler as a non-nil pointer to
// an empty string (explicit clear), distinguishable from the omitted case
// above by the pointer's nilness, not its pointee.
func Test_PutMcpserversId_SecretExplicitClear(t *testing.T) {

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	id := uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	body := []byte(`{"secret":""}`)
	req, _ := http.NewRequest("PUT", "/mcpservers/"+id.String(), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	emptySecret := ""
	mockSvc.EXPECT().McpServerUpdate(
		req.Context(),
		agent,
		id,
		"",
		"",
		"",
		ammcpserver.Status(""),
		ammcpserver.AuthType(""),
		"",
		&emptySecret,
	).Return(&ammcpserver.WebhookMessage{Identity: commonidentity.Identity{ID: id}}, nil)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

func Test_DeleteMcpserversId(t *testing.T) {

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	id := uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("DELETE", "/mcpservers/"+id.String(), nil)
	mockSvc.EXPECT().McpServerDelete(req.Context(), agent, id).Return(&ammcpserver.WebhookMessage{
		Identity: commonidentity.Identity{ID: id},
	}, nil)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d (body: %s)", http.StatusOK, w.Code, w.Body.String())
	}
}

// Test_GetMcpserversId_ResponseExcludesSecret is a redundant, explicit
// guard (in addition to the WebhookMessage json:"-" tags themselves)
// against ever regressing to include the encrypted secret material or a
// plaintext secret field in a GET response body.
func Test_GetMcpserversId_ResponseExcludesSecret(t *testing.T) {

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	id := uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{
		serviceHandler: mockSvc,
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/mcpservers/"+id.String(), nil)
	mockSvc.EXPECT().McpServerGet(req.Context(), agent, id).Return(&ammcpserver.WebhookMessage{
		Identity:  commonidentity.Identity{ID: id},
		HasSecret: true,
	}, nil)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	for _, forbidden := range []string{"secret_ciphertext", "secret_nonce", "key_version", `"secret":`} {
		if bytes.Contains([]byte(body), []byte(forbidden)) {
			t.Errorf("Response body leaks a forbidden secret-related field %q. body: %s", forbidden, body)
		}
	}
}
