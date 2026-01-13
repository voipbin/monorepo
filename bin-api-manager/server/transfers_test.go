package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_transfersPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseTransfer *tmtransfer.WebhookMessage

		expectTransferType        tmtransfer.Type
		expectTransfererCallID    uuid.UUID
		expectTransfereeAddresses []commonaddress.Address
		expectRes                 string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			},

			reqQuery: "/transfers",
			reqBody:  []byte(`{"transfer_type":"attended","transferer_call_id":"204aaffe-dd3d-11ed-8c3a-5f454beaba92","transferee_addresses":[{"type":"tel","target":"+821100000001"}]}`),

			responseTransfer: &tmtransfer.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
				},
			},

			expectTransferType:     tmtransfer.TypeAttended,
			expectTransfererCallID: uuid.FromStringOrNil("204aaffe-dd3d-11ed-8c3a-5f454beaba92"),
			expectTransfereeAddresses: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
			},
			expectRes: `{"id":"72e68b78-8286-11ed-8875-378ced61c021","customer_id":"00000000-0000-0000-0000-000000000000","type":"","transferer_call_id":"00000000-0000-0000-0000-000000000000","transferee_addresses":null,"transferee_call_id":"00000000-0000-0000-0000-000000000000","groupcall_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TransferStart(req.Context(), &tt.agent, tt.expectTransferType, tt.expectTransfererCallID, tt.expectTransfereeAddresses.Return(tt.responseTransfer, nil)

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
