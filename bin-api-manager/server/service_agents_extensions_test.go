package server

import (
	"errors"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_extensionsGET(t *testing.T) {

	tmCreate := time.Date(2026, 10, 5, 1, 2, 3, 456000000, time.UTC)
	tmCreateOlderKST := time.Date(2026, 10, 4, 10, 11, 12, 0, time.FixedZone("KST", 9*3600))

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery           string
		responseExtensions []*rmextension.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal, the page size defaults to 100",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/extensions",

			responseExtensions: []*rmextension.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7ea872bc-bbc5-11ef-83ae-dfcd9b190c58"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7efedf4e-bbc5-11ef-8d7d-ff69121f9899"),
					},
				},
			},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[{"id":"7ea872bc-bbc5-11ef-83ae-dfcd9b190c58","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"7efedf4e-bbc5-11ef-8d7d-ff69121f9899","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
		{
			name: "page_size and page_token are passed to the service handler and the next token is the last tm_create",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/extensions?page_size=20&page_token=2026-10-06T00:00:00.000000Z",

			responseExtensions: []*rmextension.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7ea872bc-bbc5-11ef-83ae-dfcd9b190c58"),
					},
					TMCreate: &tmCreate,
				},
			},

			expectPageToken: "2026-10-06T00:00:00.000000Z",
			expectPageSize:  20,
			expectRes:       `{"result":[{"id":"7ea872bc-bbc5-11ef-83ae-dfcd9b190c58","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":"2026-10-05T01:02:03.456Z","tm_update":null,"tm_delete":null}],"next_page_token":"2026-10-05T01:02:03.456000Z"}`,
		},
		{
			name: "the next token is the tm_create of the last item, in UTC",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/extensions",

			responseExtensions: []*rmextension.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7ea872bc-bbc5-11ef-83ae-dfcd9b190c58"),
					},
					TMCreate: &tmCreate,
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7efedf4e-bbc5-11ef-8d7d-ff69121f9899"),
					},
					TMCreate: &tmCreateOlderKST,
				},
			},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[{"id":"7ea872bc-bbc5-11ef-83ae-dfcd9b190c58","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":"2026-10-05T01:02:03.456Z","tm_update":null,"tm_delete":null},{"id":"7efedf4e-bbc5-11ef-8d7d-ff69121f9899","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":"2026-10-04T10:11:12+09:00","tm_update":null,"tm_delete":null}],"next_page_token":"2026-10-04T01:11:12.000000Z"}`,
		},
		{
			name: "a page size of exactly 100 is kept",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery:           "/service_agents/extensions?page_size=100",
			responseExtensions: []*rmextension.WebhookMessage{},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[],"next_page_token":""}`,
		},
		{
			name: "a page size just over the limit (101) is clamped to 100",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery:           "/service_agents/extensions?page_size=101",
			responseExtensions: []*rmextension.WebhookMessage{},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[],"next_page_token":""}`,
		},
		{
			name: "a page size over the limit is clamped to 100",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery:           "/service_agents/extensions?page_size=1000",
			responseExtensions: []*rmextension.WebhookMessage{},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[],"next_page_token":""}`,
		},
		{
			name: "a non-positive page size falls back to 100",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery:           "/service_agents/extensions?page_size=0",
			responseExtensions: []*rmextension.WebhookMessage{},

			expectPageToken: "",
			expectPageSize:  100,
			expectRes:       `{"result":[],"next_page_token":""}`,
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
			mockSvc.EXPECT().ServiceAgentExtensionList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseExtensions, nil)

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

func Test_extensionsIDGET_not_found(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	extensionID := uuid.FromStringOrNil("7f22ea24-bbc5-11ef-8c3f-139aa5535776")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/service_agents/extensions/"+extensionID.String(), nil)
	mockSvc.EXPECT().ServiceAgentExtensionGet(req.Context(), agent, extensionID).Return(nil,
		cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found."))

	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusNotFound, w.Code)
	}
	if !strings.Contains(w.Body.String(), `"reason":"EXTENSION_NOT_FOUND"`) {
		t.Errorf("Wrong match. expect: reason EXTENSION_NOT_FOUND, got: %s", w.Body.String())
	}
}

func Test_extensionsGET_permission_denied(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/service_agents/extensions", nil)
	mockSvc.EXPECT().ServiceAgentExtensionList(req.Context(), agent, uint64(100), "").Return(nil, serviceerrors.ErrPermissionDenied)

	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusForbidden, w.Code)
	}
}

// An unknown id (the registrar-manager error, wrapped with its cause), an id of another customer and a deleted
// extension (the service handler error) must produce the identical HTTP response body.
func Test_extensionsIDGET_not_found_bodies_are_identical(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})
	extensionID := uuid.FromStringOrNil("7f22ea24-bbc5-11ef-8c3f-139aa5535776")

	registrarErr := cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found.").Wrap(errors.New("db: not found"))
	handlerErr := cerrors.NotFound(commonoutline.ServiceNameRegistrarManager, "EXTENSION_NOT_FOUND", "The extension was not found.")

	body := func(serviceErr error) (int, string) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSvc := servicehandler.NewMockServiceHandler(mc)
		h := &server{serviceHandler: mockSvc}

		w := httptest.NewRecorder()
		_, r := gin.CreateTestContext(w)
		r.Use(func(c *gin.Context) {
			c.Set("auth_identity", agent)
		})
		openapi_server.RegisterHandlers(r, h)

		req, _ := http.NewRequest("GET", "/service_agents/extensions/"+extensionID.String(), nil)
		mockSvc.EXPECT().ServiceAgentExtensionGet(req.Context(), agent, extensionID).Return(nil, serviceErr)

		r.ServeHTTP(w, req)
		return w.Code, w.Body.String()
	}

	codeA, bodyA := body(registrarErr)
	codeB, bodyB := body(handlerErr)

	if codeA != http.StatusNotFound || codeB != http.StatusNotFound {
		t.Errorf("Wrong match. expect both: %d, got: %d and %d", http.StatusNotFound, codeA, codeB)
	}
	if bodyA != bodyB {
		t.Errorf("Wrong match. the response bodies differ.\nunknown id: %s\nother customer or deleted: %s", bodyA, bodyB)
	}
	if !strings.Contains(bodyA, `"reason":"EXTENSION_NOT_FOUND"`) {
		t.Errorf("Wrong match. expect: reason EXTENSION_NOT_FOUND, got: %s", bodyA)
	}
}

func Test_extensionsIDGET_invalid_id(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	// no expectation: the service handler must not be called for a malformed id.
	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("GET", "/service_agents/extensions/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
	}
	if !strings.Contains(w.Body.String(), `"reason":"INVALID_ID"`) {
		t.Errorf("Wrong match. expect: reason INVALID_ID, got: %s", w.Body.String())
	}
}

// A request without an authenticated identity must be rejected before the service handler is reached.
func Test_extensions_unauthenticated(t *testing.T) {
	tests := []struct {
		name     string
		reqQuery string
	}{
		{"list", "/service_agents/extensions"},
		{"get", "/service_agents/extensions/7f22ea24-bbc5-11ef-8c3f-139aa5535776"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation: any call to the service handler fails the test.
			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			r.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
			}
			if !strings.Contains(w.Body.String(), `"reason":"AUTHENTICATION_REQUIRED"`) {
				t.Errorf("Wrong match. expect: reason AUTHENTICATION_REQUIRED, got: %s", w.Body.String())
			}
		})
	}
}

func Test_extensionsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery          string
		responseExtension *rmextension.WebhookMessage

		expectedExtensionID uuid.UUID
		expectedRes         string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/extensions/7f22ea24-bbc5-11ef-8c3f-139aa5535776",
			responseExtension: &rmextension.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f22ea24-bbc5-11ef-8c3f-139aa5535776"),
				},
			},

			expectedExtensionID: uuid.FromStringOrNil("7f22ea24-bbc5-11ef-8c3f-139aa5535776"),
			expectedRes:         `{"id":"7f22ea24-bbc5-11ef-8c3f-139aa5535776","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","direct_hash":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().ServiceAgentExtensionGet(req.Context(), tt.agent, tt.expectedExtensionID).Return(tt.responseExtension, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}
