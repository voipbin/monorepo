package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/request"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	mmmessage "monorepo/bin-message-manager/models/message"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_MessagesGET(t *testing.T) {

	tests := []struct {
		name     string
		agent    amagent.Agent
		reqQuery string

		responseGets []*mmmessage.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27db3e44-a2e9-11ec-a397-43b3625cf0d3"),
				},
			},

			reqQuery: "/messages?page_size=10&page_token=2021-03-02%2003%3A23%3A20.995000",

			responseGets: []*mmmessage.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("282cf482-a2e9-11ec-a87d-6f5255677379"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2021-03-02 03:23:20.995000",
			expectRes:       `{"result":[{"id":"282cf482-a2e9-11ec-a87d-6f5255677379","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"text":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().MessageGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseGets, nil)

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

func Test_MessagesIDGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseMessage *mmmessage.WebhookMessage

		expectMessageID uuid.UUID
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("55882316-a2e9-11ec-aaeb-3b99d4cc0a71"),
				},
			},

			reqQuery: "/messages/55b11820-a2e9-11ec-bc5e-936e4fe1096f",

			responseMessage: &mmmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("55b11820-a2e9-11ec-bc5e-936e4fe1096f"),
			},

			expectMessageID: uuid.FromStringOrNil("55b11820-a2e9-11ec-bc5e-936e4fe1096f"),
			expectRes:       string(`{"id":"55b11820-a2e9-11ec-bc5e-936e4fe1096f","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"text":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().MessageGet(req.Context(), &tt.agent, tt.expectMessageID).Return(tt.responseMessage, nil)

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

func Test_messagesIDDELETE(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseMessage *mmmessage.WebhookMessage

		expectMessageID uuid.UUID
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9d9efc88-a2e9-11ec-a22a-9369e8662ddd"),
				},
			},

			reqQuery: "/messages/9dd8042e-a2e9-11ec-b9b1-5740852cabef",

			responseMessage: &mmmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("9dd8042e-a2e9-11ec-b9b1-5740852cabef"),
			},

			expectMessageID: uuid.FromStringOrNil("9dd8042e-a2e9-11ec-b9b1-5740852cabef"),
			expectRes:       `{"id":"9dd8042e-a2e9-11ec-b9b1-5740852cabef","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"text":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().MessageDelete(req.Context(), &tt.agent, tt.expectMessageID).Return(tt.responseMessage, nil)

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

func Test_messagesPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		requestBody request.BodyMessagesPOST

		responseMessage *mmmessage.WebhookMessage

		expectSource       commonaddress.Address
		expectDestinations []commonaddress.Address
		expectText         string
		expectRes          string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c96bf1c2-a2e9-11ec-a8e3-a716ee72ed9d"),
				},
			},

			reqQuery: "/messages",
			reqBody:  []byte(`{"source":{"type":"tel","target":"+821100000001"},"destinations":[{"type":"tel","target":"+821100000002"}],"text":"hello world"}`),

			requestBody: request.BodyMessagesPOST{
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

			responseMessage: &mmmessage.WebhookMessage{
				ID: uuid.FromStringOrNil("d0b1f3f4-a2e9-11ec-8b3b-4b3b3b3b3b3b"),
			},

			expectSource: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
			},
			expectText: "hello world",
			expectRes:  `{"id":"d0b1f3f4-a2e9-11ec-8b3b-4b3b3b3b3b3b","customer_id":"00000000-0000-0000-0000-000000000000","type":"","source":null,"targets":null,"text":"","direction":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create mock
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)

			r.Use(func(c *gin.Context) {
				c.Set("agent", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))

			mockSvc.EXPECT().MessageSend(req.Context(), &tt.agent, tt.requestBody.Source, tt.requestBody.Destinations, tt.requestBody.Text).Return(tt.responseMessage, nil)

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
