package chatbots

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	chatbotchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
	cspermission "gitlab.com/voipbin/bin-manager/customer-manager.git/models/permission"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_chatbotsPOST(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer

		reqQuery string
		reqBody  request.BodyChatbotsPOST

		response *chatbotchatbot.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				PermissionIDs: []uuid.UUID{
					cspermission.PermissionAdmin.ID,
				},
			},

			"/v1.0/chatbots",
			request.BodyChatbotsPOST{
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeChatGPT,
			},

			&chatbotchatbot.WebhookMessage{
				ID: uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
			},

			`{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			// create body
			body, err := json.Marshal(tt.reqBody)
			if err != nil {
				t.Errorf("Could not marshal the request. err: %v", err)
			}

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().ChatbotCreate(
				req.Context(),
				&tt.customer,
				tt.reqBody.Name,
				tt.reqBody.Detail,
				tt.reqBody.EngineType,
			).Return(tt.response, nil)

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

func Test_chatbotsGET(t *testing.T) {

	type test struct {
		name     string
		customer cscustomer.Customer

		reqQuery    string
		reqBody     request.ParamChatbotsGET
		resChatbots []*chatbotchatbot.WebhookMessage
		expectRes   string
	}

	tests := []test{
		{
			"1 item",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatbots?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamChatbotsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*chatbotchatbot.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatbots?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamChatbotsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},
			[]*chatbotchatbot.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("6a812daf-6ca6-4c34-892f-6e83dfd976f2"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("aff6883a-b24f-4d93-ba09-32a276cedcb7"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("e9a4b1e2-100a-4433-a854-e4fb9b668681"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"6a812daf-6ca6-4c34-892f-6e83dfd976f2","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"aff6883a-b24f-4d93-ba09-32a276cedcb7","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"e9a4b1e2-100a-4433-a854-e4fb9b668681","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ChatbotGetsByCustomerID(req.Context(), &tt.customer, tt.reqBody.PageSize, tt.reqBody.PageToken).Return(tt.resChatbots, nil)

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

func Test_chatbotsIDGET(t *testing.T) {

	tests := []struct {
		name      string
		customer  cscustomer.Customer
		chatbotID uuid.UUID

		reqQuery string

		responseChatbot *chatbotchatbot.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),

			"/v1.0/chatbots/07f52215-8366-4060-902f-a86857243351",

			&chatbotchatbot.WebhookMessage{
				ID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
			},

			`{"id":"07f52215-8366-4060-902f-a86857243351","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ChatbotGet(req.Context(), &tt.customer, tt.chatbotID).Return(tt.responseChatbot, nil)

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

func Test_chatbotsIDDELETE(t *testing.T) {

	tests := []struct {
		name     string
		customer cscustomer.Customer

		reqQuery  string
		chatbotID uuid.UUID

		responseChatbot *chatbotchatbot.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			cscustomer.Customer{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatbots/ab6f6c84-b9c2-4350-9978-4336b677603c",
			uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),

			&chatbotchatbot.WebhookMessage{
				ID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
			},

			`{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","engine_type":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
				c.Set("customer", tt.customer)
			})
			setupServer(r)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().ChatbotDelete(req.Context(), &tt.customer, tt.chatbotID).Return(tt.responseChatbot, nil)

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
