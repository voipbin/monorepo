package conferencecalls

import (
	"net/http"
	"net/http/httptest"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_conferencecallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery  string
		pageSize  uint64
		pageToken string

		responseConferencecalls []*cfconferencecall.WebhookMessage
		expectRes               string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/conferencecalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*cfconferencecall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("abd295d0-50cb-11ee-8248-c352b46cb94a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"abd295d0-50cb-11ee-8248-c352b46cb94a","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/conferencecalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			10,
			"2020-09-20 03:23:20.995000",

			[]*cfconferencecall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("ac00629e-50cb-11ee-979d-4f2d9dd53b6f"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("ac330f78-50cb-11ee-a312-5f2d51198045"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("ac5c2840-50cb-11ee-85d2-0321e8bcf0d4"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"ac00629e-50cb-11ee-979d-4f2d9dd53b6f","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"ac330f78-50cb-11ee-a312-5f2d51198045","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"ac5c2840-50cb-11ee-85d2-0321e8bcf0d4","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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

			mockSvc.EXPECT().ConferencecallGets(req.Context(), &tt.agent, tt.pageSize, tt.pageToken).Return(tt.responseConferencecalls, nil)

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

func Test_ConferencesIDGET(t *testing.T) {

	tests := []struct {
		name string

		agent amagent.Agent
		id    uuid.UUID

		requestURI string

		conference *cfconferencecall.WebhookMessage
	}{
		{
			"normal",

			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("c2de6db2-15b2-11ed-a8c9-df3874205c01"),

			"/v1.0/conferencecalls/c2de6db2-15b2-11ed-a8c9-df3874205c01",
			&cfconferencecall.WebhookMessage{
				ID: uuid.FromStringOrNil("c2de6db2-15b2-11ed-a8c9-df3874205c01"),
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

			req, _ := http.NewRequest("GET", tt.requestURI, nil)

			mockSvc.EXPECT().ConferencecallGet(req.Context(), &tt.agent, tt.id).Return(tt.conference, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conferencecallsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		id uuid.UUID

		requestURI string

		responseConferencecall *cfconferencecall.WebhookMessage
	}{
		{
			"simple test",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("23d576b4-15b4-11ed-b6f4-fbfaed3df462"),
			"/v1.0/conferencecalls/23d576b4-15b4-11ed-b6f4-fbfaed3df462",

			&cfconferencecall.WebhookMessage{
				ID: uuid.FromStringOrNil("23d576b4-15b4-11ed-b6f4-fbfaed3df462"),
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

			req, _ := http.NewRequest("DELETE", tt.requestURI, nil)

			mockSvc.EXPECT().ConferencecallKick(req.Context(), &tt.agent, tt.id).Return(tt.responseConferencecall, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}
