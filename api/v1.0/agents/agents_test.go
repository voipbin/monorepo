package agents

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func TestAgentsPOST(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name     string
		customer cscustomer.Customer
		req      request.BodyAgentsPOST

		username   string
		password   string
		agentName  string
		detail     string
		ringMethod amagent.RingMethod
		permission amagent.Permission
		tagIDs     []uuid.UUID
		addresses  []commonaddress.Address

		res *amagent.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("580a7a44-7ff8-11ec-916e-d35fe5e74591"),
			},
			request.BodyAgentsPOST{
				Username:   "test1",
				Password:   "password1",
				Name:       "test1 name",
				Detail:     "test1 detail",
				RingMethod: "ringall",
				Permission: 0,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
			},

			"test1",
			"password1",
			"test1 name",
			"test1 detail",
			amagent.RingMethodRingAll,
			0,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&amagent.WebhookMessage{
				ID: uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
			},
		},
		{
			"have webhook",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("580a7a44-7ff8-11ec-916e-d35fe5e74591"),
			},
			request.BodyAgentsPOST{
				Username:   "test1",
				Password:   "password1",
				Name:       "test1 name",
				Detail:     "test1 detail",
				RingMethod: "ringall",
				Permission: 0,
				TagIDs:     []uuid.UUID{},
				Addresses:  []commonaddress.Address{},
			},

			"test1",
			"password1",
			"test1 name",
			"test1 detail",
			amagent.RingMethodRingAll,
			0,
			[]uuid.UUID{},
			[]commonaddress.Address{},

			&amagent.WebhookMessage{
				ID: uuid.FromStringOrNil("3071bee2-79af-11ec-9f30-83b56e9d88b5"),
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}

			req, _ := http.NewRequest("POST", "/v1.0/agents", bytes.NewBuffer(body))

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().AgentCreate(&tt.customer, tt.username, tt.password, tt.agentName, tt.detail, tt.ringMethod, tt.permission, tt.tagIDs, tt.addresses).Return(tt.res, nil)

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
		customer cscustomer.Customer
		reqQuery string

		pageSize  uint64
		pageToken string
		tagIDs    []uuid.UUID
		status    amagent.Status

		resAgents []*amagent.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("b045eb94-8002-11ec-a30e-ab41a9c5ed95"),
			},
			"/v1.0/agents?page_size=11&page_token=2020-09-20T03:23:20.995000",

			11,
			"2020-09-20T03:23:20.995000",
			[]uuid.UUID{},
			amagent.StatusNone,

			[]*amagent.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
		},
		{
			"1 tag id and status",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("b045eb94-8002-11ec-a30e-ab41a9c5ed95"),
			},
			"/v1.0/agents?page_size=10&page_token=&tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3&status=available",

			10,
			"",
			[]uuid.UUID{
				uuid.FromStringOrNil("b79599f2-4f2a-11ec-b49d-df70a67f68d3"),
			},
			amagent.StatusAvailable,

			[]*amagent.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
		},
		{
			"more than 2 tag ids",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("b045eb94-8002-11ec-a30e-ab41a9c5ed95"),
			},
			"/v1.0/agents?tag_ids=b79599f2-4f2a-11ec-b49d-df70a67f68d3,39fa07ce-4fb8-11ec-8e5b-db7c7886455c",

			10,
			"",
			[]uuid.UUID{
				uuid.FromStringOrNil("b79599f2-4f2a-11ec-b49d-df70a67f68d3"),
				uuid.FromStringOrNil("39fa07ce-4fb8-11ec-8e5b-db7c7886455c"),
			},
			"",

			[]*amagent.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
				},
				{
					ID: uuid.FromStringOrNil("39fa07ce-4fb8-11ec-8e5b-db7c7886455c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().AgentGets(&tt.customer, tt.pageSize, tt.pageToken, tt.tagIDs, tt.status).Return(tt.resAgents, nil)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func TestAgentsIDStatusPUT(t *testing.T) {

	// create mock
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)

	type test struct {
		name string

		customer cscustomer.Customer
		reqQuery string
		reqBody  []byte

		agentID uuid.UUID
		status  amagent.Status
	}

	tests := []test{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("09e38a62-8003-11ec-8085-7f8bfbbc02de"),
			},
			"/v1.0/agents/a8ba6662-540a-11ec-9a9f-b31de1a77615/status",
			[]byte(`{"status":"available"}`),

			uuid.FromStringOrNil("a8ba6662-540a-11ec-9a9f-b31de1a77615"),
			amagent.StatusAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			mockSvc.EXPECT().AgentUpdateStatus(&tt.customer, tt.agentID, tt.status).Return(&amagent.WebhookMessage{}, nil)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
