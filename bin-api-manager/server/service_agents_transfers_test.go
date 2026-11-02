package server

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_serviceAgentsTransfersPOST(t *testing.T) {
	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
		},
	})
	callID := uuid.FromStringOrNil("896d70be-3f9f-11ef-bfa4-532d59e85e8c")
	extensionID := "c5e7f18c-fc5a-4520-8326-e534e2ca0b8f"

	tests := []struct {
		name string

		reqBody string

		callService      bool
		expectType       tmtransfer.Type
		expectAddresses  []commonaddress.Address
		responseRes      *tmtransfer.WebhookMessage
		responseErr      error
		expectStatus     int
		expectBodyPrefix string
	}{
		{
			name:    "blind, only type and target reach the service",
			reqBody: `{"transfer_type":"blind","transferer_call_id":"` + callID.String() + `","transferee_addresses":[{"type":"extension","target":"` + extensionID + `","target_name":"9999","name":"x","detail":"y"}]}`,

			callService:     true,
			expectType:      tmtransfer.TypeBlind,
			expectAddresses: []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID}},
			responseRes: &tmtransfer.WebhookMessage{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021")},
				Type:     tmtransfer.TypeBlind,
			},
			expectStatus:     http.StatusOK,
			expectBodyPrefix: `{"id":"72e68b78-8286-11ed-8875-378ced61c021"`,
		},
		{
			name:    "attended",
			reqBody: `{"transfer_type":"attended","transferer_call_id":"` + callID.String() + `","transferee_addresses":[{"type":"extension","target":"` + extensionID + `"}]}`,

			callService:      true,
			expectType:       tmtransfer.TypeAttended,
			expectAddresses:  []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID}},
			responseRes:      &tmtransfer.WebhookMessage{Type: tmtransfer.TypeAttended},
			expectStatus:     http.StatusOK,
			expectBodyPrefix: `{"id":"00000000-0000-0000-0000-000000000000"`,
		},
		{
			name:    "permission denied",
			reqBody: `{"transfer_type":"blind","transferer_call_id":"` + callID.String() + `","transferee_addresses":[{"type":"extension","target":"` + extensionID + `"}]}`,

			callService:     true,
			expectType:      tmtransfer.TypeBlind,
			expectAddresses: []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID}},
			responseErr:     serviceerrors.ErrPermissionDenied,
			expectStatus:    http.StatusForbidden,
		},
		{
			name:    "invalid argument from the service",
			reqBody: `{"transfer_type":"blind","transferer_call_id":"` + callID.String() + `","transferee_addresses":[{"type":"extension","target":"` + extensionID + `"}]}`,

			callService:     true,
			expectType:      tmtransfer.TypeBlind,
			expectAddresses: []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID}},
			responseErr:     cerrors.InvalidArgument("api-manager", "CALL_NOT_TRANSFERABLE", "The call is not transferable."),
			expectStatus:    http.StatusBadRequest,
		},
		{
			name:    "backend failure",
			reqBody: `{"transfer_type":"blind","transferer_call_id":"` + callID.String() + `","transferee_addresses":[{"type":"extension","target":"` + extensionID + `"}]}`,

			callService:     true,
			expectType:      tmtransfer.TypeBlind,
			expectAddresses: []commonaddress.Address{{Type: commonaddress.TypeExtension, Target: extensionID}},
			responseErr:     errors.New("rpc failed"),
			expectStatus:    http.StatusInternalServerError,
		},
		{
			name:         "malformed json is rejected before the service",
			reqBody:      `{"transfer_type":`,
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid transferer call id is rejected before the service",
			reqBody:      `{"transfer_type":"blind","transferer_call_id":"not-a-uuid","transferee_addresses":[{"type":"extension","target":"` + extensionID + `"}]}`,
			expectStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
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

			req, _ := http.NewRequest("POST", "/service_agents/transfers", bytes.NewBufferString(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.callService {
				mockSvc.EXPECT().ServiceAgentTransferStart(req.Context(), agent, tt.expectType, callID, tt.expectAddresses).Return(tt.responseRes, tt.responseErr)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
			if tt.expectBodyPrefix != "" && !bytes.HasPrefix(w.Body.Bytes(), []byte(tt.expectBodyPrefix)) {
				t.Errorf("Wrong body.\nexpect prefix: %s\ngot: %s", tt.expectBodyPrefix, w.Body.String())
			}
		})
	}
}

func Test_serviceAgentsTransfersPOST_NoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &server{serviceHandler: servicehandler.NewMockServiceHandler(mc)}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	openapi_server.RegisterHandlers(r, h)

	req, _ := http.NewRequest("POST", "/service_agents/transfers", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("Wrong status. expect: %d, got: %d", http.StatusUnauthorized, w.Code)
	}
}
