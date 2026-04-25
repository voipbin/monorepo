package server

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	rmextension "monorepo/bin-registrar-manager/models/extension"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func TestExtensionsPOST(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseExtension *rmextension.WebhookMessage

		expectExtension string
		expectPassword  string
		expectName      string
		expectDetail    string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/extensions",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","extension":"test","password":"password"}`),

			responseExtension: &rmextension.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a6405fe0-d8e1-11ef-945f-9fe89b72b04d"),
				},
			},

			expectExtension: "test",
			expectPassword:  "password",
			expectName:      "test name",
			expectDetail:    "test detail",
			expectRes:       `{"id":"a6405fe0-d8e1-11ef-945f-9fe89b72b04d","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			req, _ := http.NewRequest("POST", "/extensions", bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ExtensionCreate(req.Context(), tt.agent, tt.expectExtension, tt.expectPassword, tt.expectName, tt.expectDetail).Return(tt.responseExtension, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_ExtensionsGET(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseExntesions []*rmextension.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/extensions?page_size=20&page_token=2020-09-20T03:23:20.995000Z",

			responseExntesions: []*rmextension.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2ce09268-d8e2-11ef-af37-7baa30593c20"),
					},
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"2ce09268-d8e2-11ef-af37-7baa30593c20","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ExtensionList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseExntesions, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func Test_ExtensionsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseExntesion *rmextension.WebhookMessage

		expectExntesionID uuid.UUID
		expectRes         string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/extensions/2fbb29c0-6fb0-11eb-b2ef-4303769ecba5",

			responseExntesion: &rmextension.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				},
			},

			expectExntesionID: uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
			expectRes:         `{"id":"2fbb29c0-6fb0-11eb-b2ef-4303769ecba5","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			req, _ := http.NewRequest("GET", fmt.Sprintf("/extensions/%s", tt.responseExntesion.ID), nil)
			mockSvc.EXPECT().ExtensionGet(req.Context(), tt.agent, tt.responseExntesion.ID).Return(tt.responseExntesion, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func TestExtensionsIDPUT(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseExtension *rmextension.WebhookMessage

		extensionID    uuid.UUID
		expectName     string
		expectDetail   string
		expectPassword string
		expectRes      string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/extensions/67492c7a-6fb0-11eb-8b3f-d7eb268910df",
			reqBody:  []byte(`{"name":"test name","detail":"test detail","password":"update password"}`),

			responseExtension: &rmextension.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("67492c7a-6fb0-11eb-8b3f-d7eb268910df"),
				},
			},

			extensionID:    uuid.FromStringOrNil("67492c7a-6fb0-11eb-8b3f-d7eb268910df"),
			expectName:     "test name",
			expectDetail:   "test detail",
			expectPassword: "update password",
			expectRes:      `{"id":"67492c7a-6fb0-11eb-8b3f-d7eb268910df","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			req, _ := http.NewRequest("PUT", "/extensions/"+tt.extensionID.String(), bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ExtensionUpdate(req.Context(), tt.agent, tt.extensionID, tt.expectName, tt.expectDetail, tt.expectPassword).Return(tt.responseExtension, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

func TestExtensionsIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseExtension *rmextension.WebhookMessage

		expectExtensionID uuid.UUID
		expectRes         string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/extensions/be0c2b70-6fb0-11eb-849d-3f923b334d3b",

			responseExtension: &rmextension.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("be0c2b70-6fb0-11eb-849d-3f923b334d3b"),
				},
			},

			expectExtensionID: uuid.FromStringOrNil("be0c2b70-6fb0-11eb-849d-3f923b334d3b"),
			expectRes:         `{"id":"be0c2b70-6fb0-11eb-849d-3f923b334d3b","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().ExtensionDelete(req.Context(), tt.agent, tt.expectExtensionID).Return(tt.responseExtension, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, w.Body)
			}
		})
	}
}

// Test_extensionsIDPut_InvalidID verifies PutExtensionsId rejects a malformed
// UUID in the path with INVALID_ARGUMENT / INVALID_ID before the
// servicehandler is consulted.
func Test_extensionsIDPut_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest(http.MethodPut, "/extensions/not-a-uuid", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID", commonoutline.ServiceNameAPIManager)
}
