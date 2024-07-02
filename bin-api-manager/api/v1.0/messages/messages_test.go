package messages

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	mmmessage "monorepo/bin-message-manager/models/message"

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

func Test_MessagesGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent
		uri   string
		req   request.ParamMessagesGET

		responseGets []*mmmessage.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27db3e44-a2e9-11ec-a397-43b3625cf0d3"),
				},
			},
			"/v1.0/messages?page_size=10&page_token=2021-03-02%2003%3A23%3A20.995000",
			request.ParamMessagesGET{
				Pagination: request.Pagination{
					PageSize:  10,
					PageToken: "2021-03-02 03:23:20.995000",
				},
			},
			[]*mmmessage.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("282cf482-a2e9-11ec-a87d-6f5255677379"),
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

			req, _ := http.NewRequest("GET", tt.uri, nil)
			mockSvc.EXPECT().MessageGets(req.Context(), &tt.agent, tt.req.PageSize, tt.req.PageToken).Return(tt.responseGets, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_MessagesIDGET(t *testing.T) {

	tests := []struct {
		name      string
		agent     amagent.Agent
		messageID uuid.UUID
		uri       string

		responseGet *mmmessage.WebhookMessage
		expectBody  []byte
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55882316-a2e9-11ec-aaeb-3b99d4cc0a71"),
				},
			},
			uuid.FromStringOrNil("55b11820-a2e9-11ec-bc5e-936e4fe1096f"),
			"/v1.0/messages/55b11820-a2e9-11ec-bc5e-936e4fe1096f",
			&mmmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("55b11820-a2e9-11ec-bc5e-936e4fe1096f"),
			},
			[]byte(`{"id":"55b11820-a2e9-11ec-bc5e-936e4fe1096f","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"text":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`),
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

			req, _ := http.NewRequest("GET", tt.uri, nil)
			mockSvc.EXPECT().MessageGet(req.Context(), &tt.agent, tt.messageID).Return(tt.responseGet, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			resBytes, err := io.ReadAll(w.Body)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(resBytes, tt.expectBody) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectBody, resBytes)
			}
		})
	}
}

func Test_messagesIDDELETE(t *testing.T) {

	type test struct {
		name      string
		agent     amagent.Agent
		messageID uuid.UUID
		uri       string

		responseDelete *mmmessage.WebhookMessage
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9d9efc88-a2e9-11ec-a22a-9369e8662ddd"),
				},
			},
			uuid.FromStringOrNil("9dd8042e-a2e9-11ec-b9b1-5740852cabef"),
			"/v1.0/messages/9dd8042e-a2e9-11ec-b9b1-5740852cabef",
			&mmmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("9dd8042e-a2e9-11ec-b9b1-5740852cabef"),
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

			req, _ := http.NewRequest("DELETE", tt.uri, nil)
			mockSvc.EXPECT().MessageDelete(req.Context(), &tt.agent, tt.messageID).Return(tt.responseDelete, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_messagesPOST(t *testing.T) {

	type test struct {
		name        string
		agent       amagent.Agent
		uri         string
		requestBody request.BodyMessagesPOST
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c96bf1c2-a2e9-11ec-a8e3-a716ee72ed9d"),
				},
			},
			"/v1.0/messages",
			request.BodyMessagesPOST{
				Source: &commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destinations: []commonaddress.Address{
					{
						Type:   commonaddress.TypeTel,
						Target: "+821100000002",
					},
				},
				Text: "hello world",
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
			req, _ := http.NewRequest("POST", tt.uri, bytes.NewBuffer(body))

			mockSvc.EXPECT().MessageSend(req.Context(), &tt.agent, tt.requestBody.Source, tt.requestBody.Destinations, tt.requestBody.Text).Return(&mmmessage.WebhookMessage{}, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}
