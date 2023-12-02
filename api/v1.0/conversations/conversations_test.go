package conversations

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmedia "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"

	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/common"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/request"
	"gitlab.com/voipbin/bin-manager/api-manager.git/api/models/response"
	"gitlab.com/voipbin/bin-manager/api-manager.git/lib/middleware"
	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/servicehandler"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0", middleware.Authorized)
	ApplyRoutes(v1)
}

func Test_conversationsGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		size  uint64
		token string

		resCustomers []*cvconversation.WebhookMessage
		expectRes    *response.BodyConversationsGET
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/conversations?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			20,
			"2020-09-20 03:23:20.995000",

			[]*cvconversation.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("120bc6da-ed2e-11ec-839d-cb324c315bf3"),
				},
			},
			&response.BodyConversationsGET{
				Result: []*cvconversation.WebhookMessage{
					{
						ID: uuid.FromStringOrNil("120bc6da-ed2e-11ec-839d-cb324c315bf3"),
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

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationGetsByCustomerID(req.Context(), &tt.agent, tt.size, tt.token).Return(tt.resCustomers, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationsIDGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id uuid.UUID

		expectRes *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/conversations/4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d",

			uuid.FromStringOrNil("4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d"),

			&cvconversation.WebhookMessage{
				ID: uuid.FromStringOrNil("4d9ae3d4-ed2e-11ec-a384-1b8e70bb589d"),
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

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationGet(req.Context(), &tt.agent, tt.id).Return(tt.expectRes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationsIDMessagesGet(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id    uuid.UUID
		size  uint64
		token string

		resMessages []*cvmessage.WebhookMessage
		expectRes   *response.BodyConversationsIDMessagesGET
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/conversations/a09b01e0-ed2e-11ec-bdf1-8fa58d1092ad/messages?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			uuid.FromStringOrNil("a09b01e0-ed2e-11ec-bdf1-8fa58d1092ad"),
			20,
			"2020-09-20 03:23:20.995000",

			[]*cvmessage.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("120bc6da-ed2e-11ec-839d-cb324c315bf3"),
				},
			},
			&response.BodyConversationsIDMessagesGET{
				Result: []*cvmessage.WebhookMessage{
					{
						ID: uuid.FromStringOrNil("120bc6da-ed2e-11ec-839d-cb324c315bf3"),
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

			// create request
			req, _ := http.NewRequest("GET", tt.target, nil)

			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationMessageGetsByConversationID(req.Context(), &tt.agent, tt.id, tt.size, tt.token).Return(tt.resMessages, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			res, err := json.Marshal(tt.expectRes)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(w.Body.Bytes(), res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", res, w.Body.Bytes())
			}
		})
	}
}

func Test_conversationsIDMessagesPost(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id  uuid.UUID
		req request.BodyConversationsIDMessagesPOST

		expectRes *cvmessage.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/conversations/5950b02c-ed2f-11ec-9093-d3dcc91a72fa/messages",

			uuid.FromStringOrNil("5950b02c-ed2f-11ec-9093-d3dcc91a72fa"),
			request.BodyConversationsIDMessagesPOST{
				Text: "hello world.",
			},

			&cvmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("44757534-ed2f-11ec-b41b-b36583f1d5a7"),
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
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("POST", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationMessageSend(req.Context(), &tt.agent, tt.id, tt.req.Text, []cvmedia.Media{}).Return(tt.expectRes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}

func Test_conversationsIDPut(t *testing.T) {

	type test struct {
		name   string
		agent  amagent.Agent
		target string

		id  uuid.UUID
		req request.BodyConversationsIDPUT

		expectRes *cvconversation.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				ID: uuid.FromStringOrNil("cdb5213a-8003-11ec-84ca-9fa226fcda9f"),
			},
			"/v1.0/conversations/0e288b58-007d-11ee-b0ac-8be49d249ca9",

			uuid.FromStringOrNil("0e288b58-007d-11ee-b0ac-8be49d249ca9"),
			request.BodyConversationsIDPUT{
				Name:   "test name",
				Detail: "test detail",
			},

			&cvconversation.WebhookMessage{
				ID: uuid.FromStringOrNil("0e288b58-007d-11ee-b0ac-8be49d249ca9"),
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
			body, err := json.Marshal(tt.req)
			if err != nil {
				t.Errorf("Wong match. expect: ok, got: %v", err)
			}
			req, _ := http.NewRequest("PUT", tt.target, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().ConversationUpdate(req.Context(), &tt.agent, tt.id, tt.req.Name, tt.req.Detail).Return(tt.expectRes, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

		})
	}
}
