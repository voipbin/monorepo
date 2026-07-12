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

func Test_PutContactAddressesId(t *testing.T) {

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
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/contact_addresses/a1b2c3d4-5066-11ec-ab34-23643cfdc1c5",
			reqBody:  []byte(`{"target":"+121****9999"}`),

			responseAddress: &cmcontact.Address{
				ID:        uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
				ContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
				Address:   commonaddress.Address{Type: "tel", Target: "+121****9999"},
			},

			expectAddressID: uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
			expectFields:    map[string]any{"target": "+121****9999"},
			expectRes:       `{"type":"tel","target":"+121****9999","id":"a1b2c3d4-5066-11ec-ab34-23643cfdc1c5","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"3147612c-5066-11ec-ab34-23643cfdc1c5","is_primary":false,"tm_create":null}`,
		},
		{
			name: "update name and detail",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			}),

			reqQuery: "/contact_addresses/a1b2c3d4-5066-11ec-ab34-23643cfdc1c5",
			reqBody:  []byte(`{"name":"Main Office","detail":"Primary contact number"}`),

			responseAddress: &cmcontact.Address{
				ID:        uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
				ContactID: uuid.FromStringOrNil("3147612c-5066-11ec-ab34-23643cfdc1c5"),
				Address:   commonaddress.Address{Type: "tel", Target: "+121****9999", Name: "Main Office", Detail: "Primary contact number"},
			},

			expectAddressID: uuid.FromStringOrNil("a1b2c3d4-5066-11ec-ab34-23643cfdc1c5"),
			expectFields:    map[string]any{"name": "Main Office", "detail": "Primary contact number"},
			expectRes:       `{"type":"tel","target":"+121****9999","name":"Main Office","detail":"Primary contact number","id":"a1b2c3d4-5066-11ec-ab34-23643cfdc1c5","customer_id":"00000000-0000-0000-0000-000000000000","contact_id":"3147612c-5066-11ec-ab34-23643cfdc1c5","is_primary":false,"tm_create":null}`,
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

			mockSvc.EXPECT().ContactAddressUpdateIndependent(req.Context(), tt.agent, tt.expectAddressID, tt.expectFields).Return(tt.responseAddress, nil)

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
