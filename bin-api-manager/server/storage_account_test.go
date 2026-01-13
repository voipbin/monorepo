package server

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	smaccount "monorepo/bin-storage-manager/models/account"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_storageAccountGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseStorageAccount *smaccount.WebhookMessage

		expectRes string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ab2f092e-004e-11ee-b834-b7077f22c1eb"),
				},
			},

			reqQuery: "/storage_account",

			responseStorageAccount: &smaccount.WebhookMessage{
				ID: uuid.FromStringOrNil("c85cf9d0-1bef-11ef-a736-e75259c323b2"),
			},

			expectRes: `{"id":"c85cf9d0-1bef-11ef-a736-e75259c323b2","customer_id":"00000000-0000-0000-0000-000000000000","total_file_count":0,"total_file_size":0,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			// create request
			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().StorageAccountGetByCustomerID(req.Context(), &tt.agent.Return(tt.responseStorageAccount, nil)

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
