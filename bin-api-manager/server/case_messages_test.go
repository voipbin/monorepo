package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostCasesIdMessages(t *testing.T) {
	customerID := uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c")
	agentID := uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c")
	caseID := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         agentID,
			CustomerID: customerID,
		},
	})

	tests := []struct {
		name string

		agent *auth.AuthIdentity

		reqQuery string
		reqBody  map[string]string

		responseMessage *cvmessage.WebhookMessage
		responseErr     error

		expectStatus int
	}{
		{
			name:  "normal",
			agent: agent,

			reqQuery: "/cases/11111111-0000-0000-0000-000000000001/messages",
			reqBody: map[string]string{
				"source":      "+15551234567",
				"destination": "+15559876543",
				"text":        "hello",
			},

			responseMessage: &cvmessage.WebhookMessage{},

			expectStatus: http.StatusOK,
		},
		{
			name:  "case closed maps to a 4xx",
			agent: agent,

			reqQuery: "/cases/11111111-0000-0000-0000-000000000001/messages",
			reqBody: map[string]string{
				"source":      "+15551234567",
				"destination": "+15559876543",
				"text":        "hello",
			},

			responseErr: serviceerrors.ErrCaseClosed,

			expectStatus: http.StatusConflict,
		},
		{
			name:         "unauthenticated",
			agent:        nil,
			reqQuery:     "/cases/11111111-0000-0000-0000-000000000001/messages",
			reqBody:      map[string]string{},
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{serviceHandler: mockSvc}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				if tt.agent != nil {
					c.Set("auth_identity", tt.agent)
				}
			})
			openapi_server.RegisterHandlers(r, h)

			bodyBytes, _ := json.Marshal(tt.reqBody)
			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			if tt.agent != nil {
				mockSvc.EXPECT().
					CaseMessageSend(req.Context(), tt.agent, caseID, tt.reqBody["source"], tt.reqBody["destination"], tt.reqBody["text"]).
					Return(tt.responseMessage, tt.responseErr)
			}

			r.ServeHTTP(w, req)
			if w.Code != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d, body: %s", tt.expectStatus, w.Code, w.Body.String())
			}
		})
	}
}
