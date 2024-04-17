package transfers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_transfersPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		requestBody request.BodyTransfersPOST
		trans       *tmtransfer.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
			},

			"/v1.0/transfers",
			request.BodyTransfersPOST{
				TransferType:     "attended",
				TransfererCallID: uuid.FromStringOrNil("204aaffe-dd3d-11ed-8c3a-5f454beaba92"),
				TransfereeAddresses: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000001",
					},
				},
			},
			&tmtransfer.WebhookMessage{
				ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("agent", tt.agent)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TransferStart(req.Context(), &tt.agent, tt.requestBody.TransferType, tt.requestBody.TransfererCallID, tt.requestBody.TransfereeAddresses).Return(tt.trans, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
