package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	tmtag "monorepo/bin-tag-manager/models/tag"

	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_TagsPOST(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseTag *tmtag.WebhookMessage

		expectName   string
		expectDetail string
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/tags",
			reqBody:  []byte(`{"name":"test1 name", "detail": "test1 detail"}`),

			responseTag: &tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("bd8cee04-4f21-11ec-9955-db7041b6d997"),
				},
			},

			expectName:   "test1 name",
			expectDetail: "test1 detail",
			expectRes:    `{"id":"bd8cee04-4f21-11ec-9955-db7041b6d997","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagCreate(req.Context(), &tt.agent, tt.expectName, tt.expectDetail).Return(tt.responseTag, nil)

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

func TestTagsGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTags []*tmtag.WebhookMessage

		expectPageSize  uint64
		expectPageToken string

		expectRes string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/tags?page_size=11&page_token=2020-09-20T03:23:20.995000Z",

			responseTags: []*tmtag.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},

			expectPageSize:  11,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "more than 2 results",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/tags?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseTags: []*tmtag.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("bafb72ae-f983-11ea-9b02-67e734510d1a"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995000Z"),
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2c1abc5c-500d-11ec-8896-9bca824c5a63"),
					},
					TMCreate: timePtr("2020-09-20T03:23:21.995002Z"),
				},
			},

			expectPageSize:  10,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"bafb72ae-f983-11ea-9b02-67e734510d1a","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":"2020-09-20T03:23:21.995Z","tm_update":null,"tm_delete":null},{"id":"2c1abc5c-500d-11ec-8896-9bca824c5a63","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":"2020-09-20T03:23:21.995002Z","tm_update":null,"tm_delete":null}],"next_page_token":"2020-09-20T03:23:21.995002Z"}`,
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
			mockSvc.EXPECT().TagList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseTags, nil)

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

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTag *tmtag.WebhookMessage

		expectTagID uuid.UUID
		expectRes   string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			responseTag: &tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
				},
			},

			expectTagID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectRes:   `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagDelete(req.Context(), &tt.agent, tt.expectTagID).Return(tt.responseTag, nil)

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

func TestTagsIDGet(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseTag *tmtag.WebhookMessage

		expectTagID uuid.UUID
		expectRes   string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			responseTag: &tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
				},
			},

			expectTagID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectRes:   `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagGet(req.Context(), &tt.agent, tt.expectTagID).Return(tt.responseTag, nil)

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

func Test_TagsIDPut(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqBody  []byte
		reqQuery string

		responseTag *tmtag.WebhookMessage

		expectTagID  uuid.UUID
		expectName   string
		expectDetail string
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqBody:  []byte(`{"name":"update name", "detail": "update detail"}`),
			reqQuery: "/tags/c07ff34e-500d-11ec-8393-2bc7870b7eff",

			responseTag: &tmtag.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
				},
			},

			expectTagID:  uuid.FromStringOrNil("c07ff34e-500d-11ec-8393-2bc7870b7eff"),
			expectName:   "update name",
			expectDetail: "update detail",
			expectRes:    `{"id":"c07ff34e-500d-11ec-8393-2bc7870b7eff","customer_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","tm_create":null,"tm_update":null,"tm_delete":null}`,
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

			req, _ := http.NewRequest("PUT", tt.reqQuery, bytes.NewBuffer(tt.reqBody))
			req.Header.Set("Content-Type", "application/json")

			mockSvc.EXPECT().TagUpdate(req.Context(), &tt.agent, tt.expectTagID, tt.expectName, tt.expectDetail).Return(tt.responseTag, nil)

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
