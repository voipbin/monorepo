package extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	rmextension "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestExtensionsPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		ext      string
		password string
		extName  string
		detail   string

		requestBody request.BodyExtensionsPOST
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"test",
			"password",
			"test name",
			"test detail",

			request.BodyExtensionsPOST{
				Name:      "test name",
				Detail:    "test detail",
				Extension: "test",
				Password:  "password",
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

			req, _ := http.NewRequest("POST", "/v1.0/extensions", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ExtensionCreate(req.Context(), &tt.agent, tt.ext, tt.password, tt.extName, tt.detail).Return(&rmextension.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_ExtensionsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		expectExt []*rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/extensions",
			[]*rmextension.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				},
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

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ExtensionGets(req.Context(), &tt.agent, uint64(10), "").Return(tt.expectExt, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_ExtensionsIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent
		ext   *rmextension.Extension

		expectExt *rmextension.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			&rmextension.Extension{
				ID:         uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				CustomerID: uuid.FromStringOrNil("580a7a44-7ff8-11ec-916e-d35fe5e74591"),
				Name:       "test name",
				Detail:     "test detail",
				Extension:  "test",
				Password:   "password",
			},
			&rmextension.WebhookMessage{
				ID:        uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
				Name:      "test name",
				Detail:    "test detail",
				Extension: "test",
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

			req, _ := http.NewRequest("GET", fmt.Sprintf("/v1.0/extensions/%s", tt.ext.ID), nil)
			mockSvc.EXPECT().ExtensionGet(req.Context(), &tt.agent, tt.ext.ID).Return(tt.expectExt, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDPUT(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		extID    uuid.UUID
		extName  string
		detail   string
		password string

		requestBody request.BodyExtensionsIDPUT
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			uuid.FromStringOrNil("67492c7a-6fb0-11eb-8b3f-d7eb268910df"),
			"test name",
			"test detail",
			"update password",

			request.BodyExtensionsIDPUT{
				Name:     "test name",
				Detail:   "test detail",
				Password: "update password",
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

			req, _ := http.NewRequest("PUT", "/v1.0/extensions/"+tt.extID.String(), bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ExtensionUpdate(req.Context(), &tt.agent, tt.extID, tt.extName, tt.detail, tt.password).Return(&rmextension.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestExtensionsIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent
		extID uuid.UUID
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("be0c2b70-6fb0-11eb-849d-3f923b334d3b"),
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

			req, _ := http.NewRequest("DELETE", fmt.Sprintf("/v1.0/extensions/%s", tt.extID), nil)
			mockSvc.EXPECT().ExtensionDelete(req.Context(), &tt.agent, tt.extID).Return(&rmextension.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
