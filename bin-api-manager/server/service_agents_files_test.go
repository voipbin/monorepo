package server

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/lib/middleware"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsFiles(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseFiles []*smfile.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/service_agents/files?page_token=2020-09-20T03:23:20.995000Z&page_size=10",

			responseFiles: []*smfile.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("89f20424-c063-11ef-850f-ff10a10c813c"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("8a56f320-c063-11ef-9e55-37bada852d90"),
					},
				},
			},

			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"89f20424-c063-11ef-850f-ff10a10c813c","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null},{"id":"8a56f320-c063-11ef-9e55-37bada852d90","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ServiceAgentFileList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseFiles, nil)

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

func Test_PostServiceAgentsFiles(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		filename     string
		responseFile *smfile.WebhookMessage

		expectRes string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8df6606e-c064-11ef-a218-e33471dfe402"),
				},
			}),

			reqQuery: "/service_agents/files",
			filename: "testfile.txt",
			responseFile: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8e4c6f0e-c064-11ef-bb56-1378d1beb8d3"),
				},
			},

			expectRes: `{"id":"8e4c6f0e-c064-11ef-bb56-1378d1beb8d3","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", tt.filename)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// 10 MB
			testFileData := bytes.Repeat([]byte("a"), int(10<<20))
			_, err = part.Write(testFileData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			_ = writer.WriteField("type", "talk")
			_ = writer.Close()

			req, _ := http.NewRequest("POST", tt.reqQuery, body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			mockSvc.EXPECT().ServiceAgentFileCreate(req.Context(), tt.agent, gomock.Any(), smfile.Type("talk"), tt.filename, "Uploaded by agent", tt.filename).Return(tt.responseFile, nil)

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

func Test_PostServiceAgentsFiles_err(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string
		filename string
		filesize int
		fileType string
	}{
		{
			name: "file size over max size",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9a396eca-c064-11ef-80a5-83bf037694fc"),
				},
			}),

			reqQuery: "/service_agents/files",
			filename: "testfile.txt",
			filesize: int(constMaxFileSize) + 1,
			fileType: "talk",
		},
		{
			name: "empty type",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9a396eca-c064-11ef-80a5-83bf037694fc"),
				},
			}),

			reqQuery: "/service_agents/files",
			filename: "testfile.txt",
			filesize: int(10 << 20),
			fileType: "",
		},
		{
			name: "wrong type rag",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9a396eca-c064-11ef-80a5-83bf037694fc"),
				},
			}),

			reqQuery: "/service_agents/files",
			filename: "testfile.txt",
			filesize: int(10 << 20),
			fileType: "rag",
		},
		{
			name: "invalid type",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("9a396eca-c064-11ef-80a5-83bf037694fc"),
				},
			}),

			reqQuery: "/service_agents/files",
			filename: "testfile.txt",
			filesize: int(10 << 20),
			fileType: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			body := new(bytes.Buffer)
			writer := multipart.NewWriter(body)
			part, err := writer.CreateFormFile("file", tt.filename)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			// Set file size
			testFileData := bytes.Repeat([]byte("a"), tt.filesize)
			_, err = part.Write(testFileData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if tt.fileType != "" {
				_ = writer.WriteField("type", tt.fileType)
			}
			_ = writer.Close()

			req, _ := http.NewRequest("POST", tt.reqQuery, body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			r.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_GetServiceAgentsFilesId(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseFile *smfile.WebhookMessage

		expectFileID uuid.UUID
		expectRes    string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b842ed2e-c064-11ef-9fa3-1b01edf05df4"),
				},
			}),

			reqQuery: "/service_agents/files/b88b4e20-c064-11ef-87eb-97539ef68493",
			responseFile: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b88b4e20-c064-11ef-87eb-97539ef68493"),
				},
			},

			expectFileID: uuid.FromStringOrNil("b88b4e20-c064-11ef-87eb-97539ef68493"),
			expectRes:    `{"id":"b88b4e20-c064-11ef-87eb-97539ef68493","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ServiceAgentFileGet(req.Context(), tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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

func Test_GetServiceAgentsFilesIdFile(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseDownloadURL string

		expectFileID uuid.UUID
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b842ed2e-c064-11ef-9fa3-1b01edf05df4"),
				},
			}),

			reqQuery: "/service_agents/files/b88b4e20-c064-11ef-87eb-97539ef68493/file",

			responseDownloadURL: "https://storage.example.com/file.txt?token=valid",

			expectFileID: uuid.FromStringOrNil("b88b4e20-c064-11ef-87eb-97539ef68493"),
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
			mockSvc.EXPECT().ServiceAgentFileDownloadRedirect(req.Context(), tt.agent, tt.expectFileID).Return(tt.responseDownloadURL, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusTemporaryRedirect {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusTemporaryRedirect, w.Code)
			}

			if w.Result().Header["Location"][0] != tt.responseDownloadURL {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.responseDownloadURL, w.Result().Header["Location"][0])
			}
		})
	}
}

func Test_GetServiceAgentsFilesIdFile_NoAgent(t *testing.T) {

	tests := []struct {
		name     string
		reqQuery string
	}{
		{
			name:     "no agent in context",
			reqQuery: "/service_agents/files/b88b4e20-c064-11ef-87eb-97539ef68493/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware.RequestID())

			// No agent middleware - agent not set in context
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)

			r.ServeHTTP(w, req)
			assertErrorResponse(t, w, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED")
		})
	}
}

func Test_GetServiceAgentsFilesIdFile_InvalidUUID(t *testing.T) {

	tests := []struct {
		name     string
		agent    *auth.AuthIdentity
		reqQuery string
	}{
		{
			name: "invalid UUID",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b842ed2e-c064-11ef-9fa3-1b01edf05df4"),
				},
			}),
			reqQuery: "/service_agents/files/invalid-uuid/file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			r.ServeHTTP(w, req)
			// The openapi runtime rejects malformed openapi_types.UUID
			// path params with a 400 response BEFORE our handler runs,
			// so we cannot wrap this site in the canonical envelope.
			if w.Code != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusBadRequest, w.Code)
			}
		})
	}
}

func Test_GetServiceAgentsFilesIdFile_ServiceError(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		expectFileID uuid.UUID
	}{
		{
			name: "service returns error",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b842ed2e-c064-11ef-9fa3-1b01edf05df4"),
				},
			}),

			reqQuery: "/service_agents/files/b88b4e20-c064-11ef-87eb-97539ef68493/file",

			expectFileID: uuid.FromStringOrNil("b88b4e20-c064-11ef-87eb-97539ef68493"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)
			h := &server{
				serviceHandler: mockSvc,
			}

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(middleware.RequestID())

			r.Use(func(c *gin.Context) {
				c.Set("auth_identity", tt.agent)
			})
			openapi_server.RegisterHandlers(r, h)

			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
			mockSvc.EXPECT().ServiceAgentFileDownloadRedirect(gomock.Any(), tt.agent, tt.expectFileID).Return("", fmt.Errorf("%w: file not found", serviceerrors.ErrNotFound))

			r.ServeHTTP(w, req)
			// The legacy "file not found" message routes through the
			// translator's substring matcher to NOT_FOUND / 404.
			assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND")
		})
	}
}

func Test_DeleteServiceAgentsFilesId(t *testing.T) {

	tests := []struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery     string
		responseFile *smfile.WebhookMessage

		expectFileID uuid.UUID
		expectRes    string
	}{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b9025560-c064-11ef-a9ed-730006b7287a"),
				},
			}),

			reqQuery: "/service_agents/files/b92d7ca4-c064-11ef-92a7-93f60933d0ba",
			responseFile: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b92d7ca4-c064-11ef-92a7-93f60933d0ba"),
				},
			},

			expectFileID: uuid.FromStringOrNil("b92d7ca4-c064-11ef-92a7-93f60933d0ba"),
			expectRes:    `{"id":"b92d7ca4-c064-11ef-92a7-93f60933d0ba","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ServiceAgentFileDelete(req.Context(), tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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

// Test_serviceAgentsFilesPost_MissingAuthIdentity verifies
// PostServiceAgentsFiles emits the canonical UNAUTHENTICATED /
// AUTHENTICATION_REQUIRED envelope when auth_identity is missing from
// the gin context.
func Test_serviceAgentsFilesPost_MissingAuthIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	// Intentionally do not set auth_identity.
	openapi_server.RegisterHandlers(r, h)

	// Build a multipart body so the request shape matches a real
	// upload, even though auth check should reject before parsing.
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "sample.txt")
	_, _ = fw.Write([]byte("hello"))
	_ = mw.WriteField("type", "talk")
	_ = mw.Close()

	req, _ := http.NewRequest(http.MethodPost, "/service_agents/files", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED")
}
