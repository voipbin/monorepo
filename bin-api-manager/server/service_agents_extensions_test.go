package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	rmextension "monorepo/bin-registrar-manager/models/extension"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_extensionsGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery           string
		responseExtensions []*rmextension.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

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

			expectRes: `{"result":[{"id":"7ea872bc-bbc5-11ef-83ae-dfcd9b190c58","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","tm_create":null,"tm_update":null,"tm_delete":null},{"id":"7efedf4e-bbc5-11ef-8d7d-ff69121f9899","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentExtensionList(req.Context(), &tt.agent).Return(tt.responseExtensions, nil)

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

func Test_extensionsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery          string
		responseExtension *rmextension.WebhookMessage

		expectedExtensionID uuid.UUID
		expectedRes         string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/service_agents/extensions/7f22ea24-bbc5-11ef-8c3f-139aa5535776",
			responseExtension: &rmextension.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7f22ea24-bbc5-11ef-8c3f-139aa5535776"),
				},
			},

			expectedExtensionID: uuid.FromStringOrNil("7f22ea24-bbc5-11ef-8c3f-139aa5535776"),
			expectedRes:         `{"id":"7f22ea24-bbc5-11ef-8c3f-139aa5535776","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","extension":"","domain_name":"","username":"","password":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentExtensionGet(req.Context(), &tt.agent, tt.expectedExtensionID).Return(tt.responseExtension, nil)

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
