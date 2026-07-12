package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	amagent "monorepo/bin-agent-manager/models/agent"
	openapi_server "monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cmcontact "monorepo/bin-contact-manager/models/contact"
)

func Test_PutServiceAgentsContactAddressesId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAddress *cmcontact.Address

		expectAddressID uuid.UUID
		expectFields    map[string]any
		expectRes       string
	}{
		{
			name: "update target",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58b5afa0-8004-11ec-aea5-5f3d4e3c86d1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				Permission: amagent.PermissionAll,
			}),

			reqQuery: "/service_agents/contact_addresses/a1b2c3d4-0001-11ec-0001-000000000001",
			reqBody:  []byte(`{"target":"+121****9999"}`),

			responseAddress: &cmcontact.Address{
				ID:        uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
				ContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
				Address:   commonaddress.Address{Type: "tel", Target: "+121****9999"},
			},

			expectAddressID: uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
			expectFields:    map[string]any{"target": "+121****9999"},
			expectRes:       `{"type":"tel","target":"+121****9999","id":"a1b2c3d4-0001-11ec-0001-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","is_primary":false,"tm_create":null}`,
		},
		{
			name: "update name and detail",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("58b5afa0-8004-11ec-aea5-5f3d4e3c86d1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8004-11ec-aea5-d3a320e3b3c0"),
				},
				Permission: amagent.PermissionAll,
			}),

			reqQuery: "/service_agents/contact_addresses/a1b2c3d4-0001-11ec-0001-000000000001",
			reqBody:  []byte(`{"name":"Main Office","detail":"Primary contact number"}`),

			responseAddress: &cmcontact.Address{
				ID:        uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
				ContactID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
				Address:   commonaddress.Address{Type: "tel", Target: "+121****9999", Name: "Main Office", Detail: "Primary contact number"},
			},

			expectAddressID: uuid.FromStringOrNil("a1b2c3d4-0001-11ec-0001-000000000001"),
			expectFields:    map[string]any{"name": "Main Office", "detail": "Primary contact number"},
			expectRes:       `{"type":"tel","target":"+121****9999","name":"Main Office","detail":"Primary contact number","id":"a1b2c3d4-0001-11ec-0001-000000000001","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","is_primary":false,"tm_create":null}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ServiceAgentContactAddressUpdateIndependent(req.Context(), tt.agent, tt.expectAddressID, tt.expectFields).Return(tt.responseAddress, nil)

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
