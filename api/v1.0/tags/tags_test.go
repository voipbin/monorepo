package tags

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/tag"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestTagsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     user.User
		reqBody  request.BodyTagsPOST
		reqQuery string

		tagName string
		detail  string

		res *tag.Tag
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			request.BodyTagsPOST{
				Name:   "test1 name",
				Detail: "test1 detail",
			},
			"/v1.0/tags",

			"test1 name",
			"test1 detail",

			&tag.Tag{
				ID: uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagCreate(&tt.user, tt.tagName, tt.detail).Return(tt.res, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestAgentsGET(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     user.User
		reqQuery string

		pageSize  uint64
		pageToken string

		resAgents []*tag.Tag
		expectRes string
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			"/v1.0/tags?page_size=11&page_token=2020-09-20T03:23:20.995000",

			11,
			"2020-09-20T03:23:20.995000",

			[]*tag.Tag{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","user_id":0,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 results",
			user.User{
				ID: 1,
			},
			"/v1.0/tags?page_size=10&page_token=2020-09-20T03:23:20.995000",

			10,
			"2020-09-20T03:23:20.995000",

			[]*tag.Tag{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
					TMCreate: "2020-09-20T03:23:21.995002",
				},
			},

			`{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","user_id":0,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"2c1abc5c-500d-11ec-8896-9bca824c5a63","user_id":0,"name":"","detail":"","tm_create":"2020-09-20T03:23:21.995002","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995002"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			mockSvc.EXPECT().TagGets(&tt.user, tt.pageSize, tt.pageToken).Return(tt.resAgents, nil)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
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

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     user.User
		agentID  uuid.UUID
		reqQuery string
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			"/v1.0/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagDelete(&tt.user, tt.agentID).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestTagsIDGet(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     user.User
		agentID  uuid.UUID
		reqQuery string

		response *tag.Tag
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			"/v1.0/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			&tag.Tag{
				ID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagGet(&tt.user, tt.agentID).Return(tt.response, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestTagsIDPut(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		user     user.User
		agentID  uuid.UUID
		reqBody  []byte
		reqQuery string

		tagName string
		detail  string
	}

	tests := []test{
		{
			"normal",
			user.User{
				ID: 1,
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

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("user", tt.user)
			})
			setupServer(r)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagUpdate(&tt.user, tt.agentID, tt.tagName, tt.detail).Return(nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
