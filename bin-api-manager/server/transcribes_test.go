package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_transcribesPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseTranscribe *tmtranscribe.WebhookMessage

		expectReferenceType string
		expectReferenceID   uuid.UUID
		expectLanguage      string
		expectDirection     tmtranscribe.Direction
		expectOnEndFlowID   uuid.UUID
		expectRes           string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			},

			reqQuery: "/transcribes",
			reqBody:  []byte(`{"reference_type":"call","reference_id":"4ecc56ec-8285-11ed-9958-8b0a60b665bf","language":"en-US","direction":"both","on_end_flow_id":"199a8a78-0944-11f0-b57c-dbf18b86df64"}`),

			responseTranscribe: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("72e68b78-8286-11ed-8875-378ced61c021"),
				},
			},

			expectReferenceType: "call",
			expectReferenceID:   uuid.FromStringOrNil("4ecc56ec-8285-11ed-9958-8b0a60b665bf"),
			expectLanguage:      "en-US",
			expectDirection:     tmtranscribe.DirectionBoth,
			expectOnEndFlowID:   uuid.FromStringOrNil("199a8a78-0944-11f0-b57c-dbf18b86df64"),
			expectRes:           `{"id":"72e68b78-8286-11ed-8875-378ced61c021","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			mockSvc.EXPECT().TranscribeStart(
				req.Context(),
				&tt.agent,
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectLanguage,
				tt.expectDirection,
				tt.expectOnEndFlowID,
			).Return(tt.responseTranscribe, nil)

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

func Test_transcribesGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTranscribes []*tmtranscribe.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
				},
			},

			reqQuery: "/transcribes?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseTranscribes: []*tmtranscribe.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
					},
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TranscribeList(req.Context(), &tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseTranscribes, nil)

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

func Test_transcribesIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTranscribe *tmtranscribe.WebhookMessage

		expectTranscribeID uuid.UUID
		expectRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c7631adc-828a-11ed-bfb9-87aeb6847454"),
				},
			},

			reqQuery: "/transcribes/cced3564-828a-11ed-902f-6b70b24b6821",

			responseTranscribe: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("cced3564-828a-11ed-902f-6b70b24b6821"),
				},
			},

			expectTranscribeID: uuid.FromStringOrNil("cced3564-828a-11ed-902f-6b70b24b6821"),
			expectRes:          `{"id":"cced3564-828a-11ed-902f-6b70b24b6821","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TranscribeGet(req.Context(), &tt.agent, tt.expectTranscribeID).Return(tt.responseTranscribe, nil)

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

func Test_transcribesIDDelete(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTranscribes *tmtranscribe.WebhookMessage

		expectTranscribeID uuid.UUID
		expectRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9534352c-828b-11ed-985f-5b1e2478a83f"),
				},
			},

			reqQuery: "/transcribes/9563c0da-828b-11ed-9ca3-d735336f3293",

			responseTranscribes: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9563c0da-828b-11ed-9ca3-d735336f3293"),
				},
			},

			expectTranscribeID: uuid.FromStringOrNil("9563c0da-828b-11ed-9ca3-d735336f3293"),
			expectRes:          `{"id":"9563c0da-828b-11ed-9ca3-d735336f3293","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TranscribeDelete(req.Context(), &tt.agent, tt.expectTranscribeID).Return(tt.responseTranscribes, nil)

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

func Test_transcribesIDStopPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTranscribes *tmtranscribe.WebhookMessage

		expectTranscribeID uuid.UUID
		expectRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c5e620e0-828b-11ed-ba7c-4be64b7a1acc"),
				},
			},

			reqQuery: "/transcribes/c61977a6-828b-11ed-b4c5-f73135cd3f5a/stop",

			responseTranscribes: &tmtranscribe.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c61977a6-828b-11ed-b4c5-f73135cd3f5a"),
				},
			},

			expectTranscribeID: uuid.FromStringOrNil("c61977a6-828b-11ed-b4c5-f73135cd3f5a"),
			expectRes:          `{"id":"c61977a6-828b-11ed-b4c5-f73135cd3f5a","customer_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","on_end_flow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TranscribeStop(req.Context(), &tt.agent, tt.expectTranscribeID).Return(tt.responseTranscribes, nil)

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
