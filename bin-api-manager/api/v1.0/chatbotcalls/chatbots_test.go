package chatbotcalls

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chatbotchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

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

func Test_chatbotcallsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery    string
		reqBody     request.ParamChatbotsGET
		resChatbots []*chatbotchatbotcall.WebhookMessage
		expectRes   string
	}

	tests := []test{
		{
			"1 item",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatbotcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamChatbotsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},

			[]*chatbotchatbotcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("fa136fec-eca6-4958-b9a8-21fd8d61b8aa"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
			},
			`{"result":[{"id":"fa136fec-eca6-4958-b9a8-21fd8d61b8aa","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:21.995000"}`,
		},
		{
			"more than 2 items",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatbotcalls?page_size=10&page_token=2020-09-20%2003:23:20.995000",
			request.ParamChatbotsGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2020-09-20 03:23:20.995000",
				},
			},
			[]*chatbotchatbotcall.WebhookMessage{
				{
					ID:       uuid.FromStringOrNil("f7576695-a944-4427-b7d6-1a776f83aa9a"),
					TMCreate: "2020-09-20T03:23:21.995000",
				},
				{
					ID:       uuid.FromStringOrNil("f34d51d0-4a74-40d7-9050-edc6fd1654f7"),
					TMCreate: "2020-09-20T03:23:22.995000",
				},
				{
					ID:       uuid.FromStringOrNil("227edc68-c2da-4ed8-bd28-08d8fab8c17c"),
					TMCreate: "2020-09-20T03:23:23.995000",
				},
			},
			`{"result":[{"id":"f7576695-a944-4427-b7d6-1a776f83aa9a","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:21.995000","tm_update":"","tm_delete":""},{"id":"f34d51d0-4a74-40d7-9050-edc6fd1654f7","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:22.995000","tm_update":"","tm_delete":""},{"id":"227edc68-c2da-4ed8-bd28-08d8fab8c17c","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"2020-09-20T03:23:23.995000","tm_update":"","tm_delete":""}],"next_page_token":"2020-09-20T03:23:23.995000"}`,
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
			mockSvc.EXPECT().ChatbotcallGetsByCustomerID(req.Context(), &tt.agent, tt.reqBody.PageSize, tt.reqBody.PageToken).Return(tt.resChatbots, nil)

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

func Test_chatbotcallsIDGET(t *testing.T) {

	tests := []struct {
		name          string
		agent         amagent.Agent
		chatbotcallID uuid.UUID

		reqQuery string

		responseChatbotcall *chatbotchatbotcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},
			uuid.FromStringOrNil("f199188b-8d78-4778-8891-8f276cd56de5"),

			"/v1.0/chatbotcalls/f199188b-8d78-4778-8891-8f276cd56de5",

			&chatbotchatbotcall.WebhookMessage{
				ID: uuid.FromStringOrNil("f199188b-8d78-4778-8891-8f276cd56de5"),
			},

			`{"id":"f199188b-8d78-4778-8891-8f276cd56de5","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().ChatbotcallGet(req.Context(), &tt.agent, tt.chatbotcallID).Return(tt.responseChatbotcall, nil)

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

func Test_chatbotcallsIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		chatbotcallID uuid.UUID

		responseChatbotcall *chatbotchatbotcall.WebhookMessage

		expectRes string
	}{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			"/v1.0/chatbotcalls/c1a95988-5382-4769-98a9-b404823a64bf",
			uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),

			&chatbotchatbotcall.WebhookMessage{
				ID: uuid.FromStringOrNil("c1a95988-5382-4769-98a9-b404823a64bf"),
			},

			`{"id":"c1a95988-5382-4769-98a9-b404823a64bf","customer_id":"00000000-0000-0000-0000-000000000000","chatbot_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","status":"","gender":"","language":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().ChatbotcallDelete(req.Context(), &tt.agent, tt.chatbotcallID).Return(tt.responseChatbotcall, nil)

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
