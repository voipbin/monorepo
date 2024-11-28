package tags

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	tmtag "monorepo/bin-tag-manager/models/tag"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

func Test_TagsPOST(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent
		reqBody  request.BodyTagsPOST
		reqQuery string

		tagName string
		detail  string

		res *tmtag.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			request.BodyTagsPOST{
				Name:   "test1 name",
				Detail: "test1 detail",
			},
			"/v1.0/tags",

			"test1 name",
			"test1 detail",

			&tmtag.WebhookMessage{
				ID: uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagCreate(req.Context(), &tt.customer, tt.tagName, tt.detail).Return(tt.res, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestTagsGET(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent
		reqQuery string

		pageSize  uint64
		pageToken string

		resAgents []*tmtag.WebhookMessage
		expectRes string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			"/v1.0/tags?page_size=11&page_token=2020-09-20T03:23:20.995000",

			11,
			"2020-09-20T03:23:20.995000",

			[]*tmtag.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 results",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			"/v1.0/tags?page_size=10&page_token=2020-09-20T03:23:20.995000",

			10,
			"2020-09-20T03:23:20.995000",

			[]*tmtag.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
					TMCreate: "2020-09-20T03:23:21.995002",
				},
			},

			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"2c1abc5c-500d-11ec-8896-9bca824c5a63","name":"","detail":"","tm_create":"2020-09-20T03:23:21.995002","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995002"}`,
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().TagGets(req.Context(), &tt.customer, tt.pageSize, tt.pageToken).Return(tt.resAgents, nil)
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

func TestTagsDelete(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent
		agentID  uuid.UUID
		reqQuery string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			"/v1.0/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagDelete(req.Context(), &tt.customer, tt.agentID).Return(&tmtag.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestTagsIDGet(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent
		agentID  uuid.UUID
		reqQuery string

		response *tmtag.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			"/v1.0/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			&tmtag.WebhookMessage{
				ID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagGet(req.Context(), &tt.customer, tt.agentID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_TagsIDPut(t *testing.T) {

	type test struct {
		name     string
		customer amagent.Agent
		id       uuid.UUID
		reqBody  []byte
		reqQuery string

		tagName string
		detail  string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},
			uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			[]byte(`{"name":"update name", "detail": "update detail"}`),
			"/v1.0/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			"update name",
			"update detail",
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
				c.Set("agent", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagUpdate(req.Context(), &tt.customer, tt.id, tt.tagName, tt.detail).Return(&tmtag.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
