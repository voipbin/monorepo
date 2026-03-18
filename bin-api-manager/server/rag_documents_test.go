package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	rmdocument "monorepo/bin-rag-manager/models/document"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostRagDocuments(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseDoc *rmdocument.WebhookMessage

		expectedRagID         uuid.UUID
		expectedName          string
		expectedDocType       rmdocument.DocType
		expectedSourceURL     string
		expectedStorageFileID uuid.UUID
		expectedRes           string
	}{
		{
			name: "url doc type",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/rag-documents",
			reqBody:  []byte(`{"rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","name":"test doc","doc_type":"url","source_url":"https://example.com/doc.html"}`),

			responseDoc: &rmdocument.WebhookMessage{
				ID:         uuid.FromStringOrNil("dbceb866-4506-4e86-9851-a82d4d3ced88"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},

			expectedRagID:         uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			expectedName:          "test doc",
			expectedDocType:       rmdocument.DocTypeURL,
			expectedSourceURL:     "https://example.com/doc.html",
			expectedStorageFileID: uuid.Nil,
			expectedRes:           `{"id":"dbceb866-4506-4e86-9851-a82d4d3ced88","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","storage_file_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().RagDocumentCreate(
				req.Context(),
				&tt.agent,
				tt.expectedRagID,
				tt.expectedName,
				tt.expectedDocType,
				tt.expectedSourceURL,
				tt.expectedStorageFileID,
			).Return(tt.responseDoc, nil)

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

func Test_GetRagDocuments(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseDocs []*rmdocument.WebhookMessage

		expectedRagID     uuid.UUID
		expectedPageSize  uint64
		expectedPageToken string
		expectedRes       string
	}

	tests := []test{
		{
			name: "1 item",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/rag-documents?page_size=10&page_token=2020-09-20T03:23:20.995000Z",

			responseDocs: []*rmdocument.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6"),
					CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					TMCreate:   timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},
			expectedRagID:     uuid.Nil,
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"4a918c83-50b9-4fb4-8a22-afd1a1fd2dc6","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","storage_file_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995Z"}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
		},
		{
			name: "filter by rag_id",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/rag-documents?page_size=10&page_token=2020-09-20T03:23:20.995000Z&rag_id=a1b2c3d4-e5f6-7890-abcd-ef1234567890",

			responseDocs: []*rmdocument.WebhookMessage{
				{
					ID:         uuid.FromStringOrNil("6a812daf-6ca6-4c34-892f-6e83dfd976f2"),
					CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
					RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
					TMCreate:   timePtr("2020-09-20T03:23:21.995000Z"),
				},
			},
			expectedRagID:     uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			expectedPageSize:  10,
			expectedPageToken: "2020-09-20T03:23:20.995000Z",
			expectedRes:       `{"result":[{"id":"6a812daf-6ca6-4c34-892f-6e83dfd976f2","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","storage_file_id":"00000000-0000-0000-0000-000000000000","tm_create":"2020-09-20T03:23:21.995Z"}],"next_page_token":"2020-09-20T03:23:21.995000Z"}`,
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
			mockSvc.EXPECT().RagDocumentGets(req.Context(), &tt.agent, tt.expectedRagID, tt.expectedPageSize, tt.expectedPageToken).Return(tt.responseDocs, nil)

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

func Test_GetRagDocumentsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseDoc *rmdocument.WebhookMessage

		expectDocID uuid.UUID
		expectRes   string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/rag-documents/07f52215-8366-4060-902f-a86857243351",

			responseDoc: &rmdocument.WebhookMessage{
				ID:         uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},

			expectDocID: uuid.FromStringOrNil("07f52215-8366-4060-902f-a86857243351"),
			expectRes:   `{"id":"07f52215-8366-4060-902f-a86857243351","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","storage_file_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().RagDocumentGet(req.Context(), &tt.agent, tt.expectDocID).Return(tt.responseDoc, nil)

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

func Test_DeleteRagDocumentsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseDoc *rmdocument.WebhookMessage

		expectDocID uuid.UUID
		expectRes   string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/rag-documents/ab6f6c84-b9c2-4350-9978-4336b677603c",

			responseDoc: &rmdocument.WebhookMessage{
				ID:         uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
				CustomerID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				RagID:      uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
			},

			expectDocID: uuid.FromStringOrNil("ab6f6c84-b9c2-4350-9978-4336b677603c"),
			expectRes:   `{"id":"ab6f6c84-b9c2-4350-9978-4336b677603c","customer_id":"2a2ec0ba-8004-11ec-aea5-439829c92a7c","rag_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","storage_file_id":"00000000-0000-0000-0000-000000000000"}`,
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
			mockSvc.EXPECT().RagDocumentDelete(req.Context(), &tt.agent, tt.expectDocID).Return(tt.responseDoc, nil)

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
