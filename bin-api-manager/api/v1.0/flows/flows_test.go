package flows

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmflow "monorepo/bin-flow-manager/models/flow"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonidentity "monorepo/bin-common-handler/models/identity"

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

func Test_flowsPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  request.BodyFlowsPOST
		resFlow  *fmflow.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/flows",
			request.BodyFlowsPOST{
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			&fmflow.WebhookMessage{
				ID:     uuid.FromStringOrNil("264b18d4-82fa-11eb-919b-9f55a7f6ace1"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
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

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().FlowCreate(req.Context(), &tt.agent, tt.reqBody.Name, tt.reqBody.Detail, tt.reqBody.Actions, true).Return(tt.resFlow, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_flowsIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseFlow *fmflow.WebhookMessage
		expectRes    string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/flows/2375219e-0b87-11eb-90f9-036ec16f126b",

			&fmflow.WebhookMessage{
				ID:     uuid.FromStringOrNil("2375219e-0b87-11eb-90f9-036ec16f126b"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			`{"id":"2375219e-0b87-11eb-90f9-036ec16f126b","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"test name","detail":"test detail","actions":[{"id":"00000000-0000-0000-0000-000000000000","next_id":"00000000-0000-0000-0000-000000000000","type":"answer"}],"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().FlowGet(req.Context(), &tt.agent, tt.responseFlow.ID).Return(tt.responseFlow, nil)

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

func Test_flowsIDPUT(t *testing.T) {

	tests := []struct {
		name         string
		agent        amagent.Agent
		expectFlowID uuid.UUID

		reqQuery string
		reqBody  request.BodyFlowsIDPUT

		responseFlow *fmflow.WebhookMessage

		expectFlow *fmflow.Flow
		expectRes  string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),

			"/v1.0/flows/d213a09e-6790-11eb-8cea-bb3b333200ed",
			request.BodyFlowsIDPUT{
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},

			&fmflow.WebhookMessage{
				ID: uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
			},
			&fmflow.Flow{
				ID:     uuid.FromStringOrNil("d213a09e-6790-11eb-8cea-bb3b333200ed"),
				Name:   "test name",
				Detail: "test detail",
				Actions: []fmaction.Action{
					{
						Type: "answer",
					},
				},
			},
			`{"id":"d213a09e-6790-11eb-8cea-bb3b333200ed","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().FlowUpdate(req.Context(), &tt.agent, tt.expectFlow).Return(tt.responseFlow, nil)

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

func Test_flowsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery     string
		responseFlow *fmflow.WebhookMessage

		expectFlowID uuid.UUID
		expectRes    string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/flows/d466f900-67cb-11eb-b2ff-1f9adc48f842",
			&fmflow.WebhookMessage{
				ID: uuid.FromStringOrNil("d466f900-67cb-11eb-b2ff-1f9adc48f842"),
			},

			uuid.FromStringOrNil("d466f900-67cb-11eb-b2ff-1f9adc48f842"),
			`{"id":"d466f900-67cb-11eb-b2ff-1f9adc48f842","customer_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","actions":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().FlowDelete(req.Context(), &tt.agent, tt.expectFlowID).Return(tt.responseFlow, nil)

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
