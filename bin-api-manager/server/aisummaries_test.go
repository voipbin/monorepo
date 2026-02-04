package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	amsummary "monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostAisummaries(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseAIsummary *amsummary.WebhookMessage

		expectedLanguage      string
		expectedOnEndFlowID   uuid.UUID
		expectedReferenceType amsummary.ReferenceType
		expectedReferenceID   uuid.UUID
		expectedRes           string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aisummaries",
			reqBody:  []byte(`{"language":"en-US","on_end_flow_id":"1c723502-0ccd-11f0-8c43-17bfdea221c5","reference_type":"call","reference_id":"1ca46b4e-0ccd-11f0-8a52-2f20e29cf20a"}`),

			responseAIsummary: &amsummary.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("1cd2ab6c-0ccd-11f0-a783-67af79c5fb6d"),
				},
			},

			expectedLanguage:      "en-US",
			expectedOnEndFlowID:   uuid.FromStringOrNil("1c723502-0ccd-11f0-8c43-17bfdea221c5"),
			expectedReferenceType: amsummary.ReferenceTypeCall,
			expectedReferenceID:   uuid.FromStringOrNil("1ca46b4e-0ccd-11f0-8a52-2f20e29cf20a"),
			expectedRes:           `{"id":"1cd2ab6c-0ccd-11f0-a783-67af79c5fb6d","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().AISummaryCreate(
				req.Context(),
				&tt.agent,
				tt.expectedOnEndFlowID,
				tt.expectedReferenceType,
				tt.expectedReferenceID,
				tt.expectedLanguage,
			).Return(tt.responseAIsummary, nil)

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

func Test_GetAisummaries(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAIsummaries []*amsummary.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ef9f2bd8-0ccd-11f0-9c5b-0b861eb38bff"),
				},
			},

			reqQuery: "/aisummaries?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAIsummaries: []*amsummary.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("f01f38f0-0ccd-11f0-81ab-730c812c39fb"),
					},
					TMCreate: "2020-09-20T03:23:21.995000Z",
				},
			},
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"f01f38f0-0ccd-11f0-81ab-730c812c39fb","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995000Z"}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
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
			mockSvc.EXPECT().AISummaryGetsByCustomerID(req.Context(), &tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseAIsummaries, nil)

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

func Test_GetAisummariesId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAI *amsummary.WebhookMessage

		expectAIID uuid.UUID
		expectRes  string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aisummaries/f04d646e-0ccd-11f0-aa34-e39ff78be410",

			responseAI: &amsummary.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f04d646e-0ccd-11f0-aa34-e39ff78be410"),
				},
			},

			expectAIID: uuid.FromStringOrNil("f04d646e-0ccd-11f0-aa34-e39ff78be410"),
			expectRes:  `{"id":"f04d646e-0ccd-11f0-aa34-e39ff78be410","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().AISummaryGet(req.Context(), &tt.agent, tt.expectAIID).Return(tt.responseAI, nil)

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

func Test_DeleteAisummariesId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseAI *amsummary.WebhookMessage

		expectAIID uuid.UUID
		expectRes  string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/aisummaries/f07adf48-0ccd-11f0-9b39-932e754611a0",

			responseAI: &amsummary.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f07adf48-0ccd-11f0-9b39-932e754611a0"),
				},
			},

			expectAIID: uuid.FromStringOrNil("f07adf48-0ccd-11f0-9b39-932e754611a0"),
			expectRes:  `{"id":"f07adf48-0ccd-11f0-9b39-932e754611a0","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000"}`,
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().AISummaryDelete(req.Context(), &tt.agent, tt.expectAIID).Return(tt.responseAI, nil)

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
