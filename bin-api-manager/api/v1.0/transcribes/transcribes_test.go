package transcribes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

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

func Test_transcribesPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		requestBody request.BodyTranscribesPOST
		trans       *tmtranscribe.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
			},

			"/v1.0/transcribes",
			request.BodyTranscribesPOST{
				ReferenceType: request.TranscribeReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("4ecc56ec-8285-11ed-9958-8b0a60b665bf"),
				Language:      "en-US",
				Direction:     tmtranscribe.DirectionBoth,
			},
			&tmtranscribe.WebhookMessage{
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
			mockSvc.EXPECT().TranscribeStart(req.Context(), &tt.agent, tt.requestBody.ReferenceType, tt.requestBody.ReferenceID, tt.requestBody.Language, tt.requestBody.Direction).Return(tt.trans, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_transcribesGET(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		reqQuery    string
		requestBody request.ParamTranscribesGET

		responseTranscribes []*tmtranscribe.WebhookMessage
		expectRes           string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("4e72f3ea-8285-11ed-a55b-6bf44eeb8a87"),
			},

			"/v1.0/transcribes?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamTranscribesGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*tmtranscribe.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("6e812ad0-828a-11ed-bfe8-9f9b344a834b"),
				},
			},
			`{"result":[{"id":"6e812ad0-828a-11ed-bfe8-9f9b344a834b","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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

			req, _ := http.NewRequest("GET", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TranscribeGets(req.Context(), &tt.agent, tt.requestBody.PageSize, tt.requestBody.PageToken).Return(tt.responseTranscribes, nil)

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

func Test_transcribesIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTranscribes *tmtranscribe.WebhookMessage
		expectRes           string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("c7631adc-828a-11ed-bfb9-87aeb6847454"),
			},

			"/v1.0/transcribes/cced3564-828a-11ed-902f-6b70b24b6821",

			&tmtranscribe.WebhookMessage{
				ID: uuid.FromStringOrNil("cced3564-828a-11ed-902f-6b70b24b6821"),
			},
			`{"id":"cced3564-828a-11ed-902f-6b70b24b6821","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TranscribeGet(req.Context(), &tt.agent, tt.responseTranscribes.ID).Return(tt.responseTranscribes, nil)

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
		expectRes           string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("9534352c-828b-11ed-985f-5b1e2478a83f"),
			},

			"/v1.0/transcribes/9563c0da-828b-11ed-9ca3-d735336f3293",

			&tmtranscribe.WebhookMessage{
				ID: uuid.FromStringOrNil("9563c0da-828b-11ed-9ca3-d735336f3293"),
			},
			`{"id":"9563c0da-828b-11ed-9ca3-d735336f3293","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TranscribeDelete(req.Context(), &tt.agent, tt.responseTranscribes.ID).Return(tt.responseTranscribes, nil)

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
		expectRes           string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("c5e620e0-828b-11ed-ba7c-4be64b7a1acc"),
			},

			"/v1.0/transcribes/c61977a6-828b-11ed-b4c5-f73135cd3f5a/stop",

			&tmtranscribe.WebhookMessage{
				ID: uuid.FromStringOrNil("c61977a6-828b-11ed-b4c5-f73135cd3f5a"),
			},
			`{"id":"c61977a6-828b-11ed-b4c5-f73135cd3f5a","customer_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","language":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`,
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

			req, _ := http.NewRequest("POST", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().TranscribeStop(req.Context(), &tt.agent, tt.responseTranscribes.ID).Return(tt.responseTranscribes, nil)

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
