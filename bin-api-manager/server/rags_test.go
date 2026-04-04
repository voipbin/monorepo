package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/models/auth"
	rmrag "monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostRags(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseRag *rmrag.WebhookMessage

		expectedName        string
		expectedDescription string
		expectedRes         string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/rags",
			reqBody:  []byte(`{"name":"test rag","description":"test description"}`),

			responseRag: &rmrag.WebhookMessage{
				ID:         uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			expectedName:        "test rag",
			expectedDescription: "test description",
			expectedRes:         `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("POST", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().RagCreate(
				req.Context(),
				tt.agent,
				tt.expectedName,
				tt.expectedDescription,
				[]uuid.UUID{},
				[]string{},
			).Return(tt.responseRag, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetRags(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseRags []*rmrag.WebhookMessage

		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/rags?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseRags: []*rmrag.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6"),
					CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					TMCreate:   timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","tm_create":"2020-09-20T03:23:21.995Z"}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 items",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/rags?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseRags: []*rmrag.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("6a812daf-6ca6-4c34-892f-6e83dfd976f2"),
					CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					TMCreate:   timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					ID:         uuid.FromStringOrNil("aff6883a-b24f-4d93-ba09-32a276cedcb7"),
					CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					TMCreate:   timePtr("2020-09-20T03:23:22.995000Z"),
				},
				{
					ID:         uuid.FromStringOrNil("e9a4b1e2-100a-4433-a854-e4fb9b668681"),
					CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					TMCreate:   timePtr("2020-09-20T03:23:23.995000Z"),
				},
			},

			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6a812daf-6ca6-4c34-892f-6e83dfd976f2","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","tm_create":"2020-09-20T03:23:21.995Z"},{"id":"aff6883a-b24f-4d93-ba09-32a276cedcb7","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","tm_create":"2020-09-20T03:23:22.995Z"},{"id":"e9a4b1e2-100a-4433-a854-e4fb9b668681","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","tm_create":"2020-09-20T03:23:23.995Z"}],"next_page_token":"2020-09-20T03:23:23.995000Z"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().RagGets(req.Context(), tt.agent, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseRags, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_GetRagsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseRag *rmrag.WebhookMessage

		expectRagID uuid.UUID
		expectRes   string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/rags/07f52215-8366-4060-902f-a86857243351",

			responseRag: &rmrag.WebhookMessage{
				ID:         uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			expectRagID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
			expectRes:   `{"id":"07f52215-8366-4060-902f-a86857243351","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().RagGet(req.Context(), tt.agent, tt.expectRagID).Return(tt.responseRag, nil)

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

func Test_PutRagsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		reqBody  []byte

		responseRag *rmrag.WebhookMessage

		expectedRagID  uuid.UUID
		expectedFields map[rmrag.Field]any
		expectedRes    string
	}{
		{
			name: "normal with name and description",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/rags/2a2ec0ba-8004-11ec-aea5-439829c92a7c",
			reqBody:  []byte(`{"name":"updated name","description":"updated description"}`),

			responseRag: &rmrag.WebhookMessage{
				ID:         uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			expectedRagID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			expectedFields: map[rmrag.Field]any{
				rmrag.FieldName:        "updated name",
				rmrag.FieldDescription: "updated description",
			},
			expectedRes: `{"id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")
			mockSvc.EXPECT().RagUpdate(
				req.Context(),
				tt.agent,
				tt.expectedRagID,
				tt.expectedFields,
			).Return(tt.responseRag, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}

			if w.Body.String() != tt.expectedRes {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, w.Body)
			}
		})
	}
}

func Test_DeleteRagsId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseRag *rmrag.WebhookMessage

		expectRagID uuid.UUID
		expectRes   string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/rags/ab6f6c84-b9c2-4350-9978-4336b677603c",

			responseRag: &rmrag.WebhookMessage{
				ID:         uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
			},

			expectRagID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
			expectRes:   `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c"}`,
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
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("DELETE", tt.reqQuery, nil)
			mockSvc.EXPECT().RagDelete(req.Context(), tt.agent, tt.expectRagID).Return(tt.responseRag, nil)

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
