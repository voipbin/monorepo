package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmanalysis "monorepo/bin-timeline-manager/models/analysis"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostTimelineAnalyses(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseAnalysis *tmanalysis.WebhookMessage

		expectedActiveflowID uuid.UUID
		expectedReanalyze    bool
		expectedRes          string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/timeline-analyses",
			reqBody:  []byte(`{"activeflow_id":"11110000-0000-0000-0000-000000000002","reanalyze":true}`),

			responseAnalysis: &tmanalysis.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000003"),
				},
				ActiveflowID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000002"),
				Status:       tmanalysis.StatusProgressing,
			},

			expectedActiveflowID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000002"),
			expectedReanalyze:    true,
			expectedRes:          `{"id":"11110000-0000-0000-0000-000000000003","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"11110000-0000-0000-0000-000000000002","status":"progressing","tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
		{
			name: "reanalyze omitted defaults to false",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000011"),
				},
			}),

			reqQuery: "/timeline-analyses",
			reqBody:  []byte(`{"activeflow_id":"11110000-0000-0000-0000-000000000012"}`),

			responseAnalysis: &tmanalysis.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000013"),
				},
				ActiveflowID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000012"),
				Status:       tmanalysis.StatusCompleted,
			},

			expectedActiveflowID: uuid.FromStringOrNil("11110000-0000-0000-0000-000000000012"),
			expectedReanalyze:    false,
			expectedRes:          `{"id":"11110000-0000-0000-0000-000000000013","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"11110000-0000-0000-0000-000000000012","status":"completed","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TimelineAnalysisCreate(
				req.Context(),
				tt.agent,
				tt.expectedActiveflowID,
				tt.expectedReanalyze,
			).Return(tt.responseAnalysis, nil)

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

func Test_GetTimelineAnalyses(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAnalyses []*tmanalysis.WebhookMessage

		expectedPageSize     uint64
		expectedPageToken    string
		expectedActiveflowID uuid.UUID
		expectedStatus       tmanalysis.Status
		expectedRes          string
	}{
		{
			name: "with filters",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("22220000-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/timeline-analyses?page_size=10&page_token=2020-09-20T03:23:20.995000Z&activeflow_id=22220000-0000-0000-0000-000000000009&status=completed",

			responseAnalyses: []*tmanalysis.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("22220000-0000-0000-0000-000000000002"),
					},
					Status:   tmanalysis.StatusCompleted,
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectedPageSize:     10,
			expectedPageToken:    "2020-09-20T03:23:20.995000Z",
			expectedActiveflowID: uuid.FromStringOrNil("22220000-0000-0000-0000-000000000009"),
			expectedStatus:       tmanalysis.StatusCompleted,
			expectedRes:          `{"result":[{"id":"22220000-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","status":"completed","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "no filters",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("22220000-0000-0000-0000-000000000011"),
				},
			}),

			reqQuery: "/timeline-analyses?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseAnalyses: []*tmanalysis.WebhookMessage{},

			expectedPageSize:     10,
			expectedPageToken:    "2020-09-20T03:23:20.995000Z",
			expectedActiveflowID: uuid.Nil,
			expectedStatus:       tmanalysis.Status(""),
			expectedRes:          `{"result":[],"next_page_token":""}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().TimelineAnalysisGetsByCustomerID(
				req.Context(),
				tt.agent,
				tt.expectedPageSize,
				tt.expectedPageToken,
				tt.expectedActiveflowID,
				tt.expectedStatus,
			).Return(tt.responseAnalyses, nil)

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

func Test_GetTimelineAnalysesId(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAnalysis *tmanalysis.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("33330000-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/timeline-analyses/33330000-0000-0000-0000-000000000002",

			responseAnalysis: &tmanalysis.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("33330000-0000-0000-0000-000000000002"),
				},
				Status: tmanalysis.StatusCompleted,
			},

			expectedID:  uuid.FromStringOrNil("33330000-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"33330000-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","status":"completed","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().TimelineAnalysisGet(req.Context(), tt.agent, tt.expectedID).Return(tt.responseAnalysis, nil)

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

func Test_DeleteTimelineAnalysesId(t *testing.T) {
	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseAnalysis *tmanalysis.WebhookMessage

		expectedID  uuid.UUID
		expectedRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44440000-0000-0000-0000-000000000001"),
				},
			}),

			reqQuery: "/timeline-analyses/44440000-0000-0000-0000-000000000002",

			responseAnalysis: &tmanalysis.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("44440000-0000-0000-0000-000000000002"),
				},
				Status: tmanalysis.StatusCompleted,
			},

			expectedID:  uuid.FromStringOrNil("44440000-0000-0000-0000-000000000002"),
			expectedRes: `{"id":"44440000-0000-0000-0000-000000000002","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","status":"completed","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().TimelineAnalysisDelete(req.Context(), tt.agent, tt.expectedID).Return(tt.responseAnalysis, nil)

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
