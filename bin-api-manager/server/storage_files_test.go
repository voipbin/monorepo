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
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostStorageFiles(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files",
			filename: "testfile.txt",
			responseFile: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("39ae35ca-1710-11ef-bae6-afeb7c57c901"),
				},
			},

			expectRes: `{"id":"39ae35ca-1710-11ef-bae6-afeb7c57c901","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			_ = writer.WriteField("type", "rag")
			_ = writer.Close()

			req, _ := http.NewRequest("POST", tt.reqQuery, body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			mockSvc.EXPECT().StorageFileCreate(req.Context(), tt.agent, gomock.Any(), smfile.Type("rag"), "", "", tt.filename).Return(tt.responseFile, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

func Test_PostStorageFiles_err(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files",
			filename: "testfile.txt",
			filesize: int(constMaxFileSize) + 1,
			fileType: "rag",
		},
		{
			name: "empty type",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files",
			filename: "testfile.txt",
			filesize: int(10 << 20),
			fileType: "",
		},
		{
			name: "wrong type talk",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files",
			filename: "testfile.txt",
			filesize: int(10 << 20),
			fileType: "talk",
		},
		{
			name: "invalid type",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files",
			filename: "testfile.txt",
			filesize: int(10 << 20),
			fileType: "invalid",
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

func Test_GetStorageFiles(t *testing.T) {

	type test struct {
		name  string
		agent *auth.AuthIdentity

		reqQuery string

		responseExtension []*smfile.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files?page_size=20&page_token=2020-09-20T03:23:20.995000Z",
			responseExtension: []*smfile.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
					},
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20T03:23:20.995000Z",
			expectRes:       `{"result":[{"id":"2fbb29c0-6fb0-11eb-b2ef-4303769ecba5","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}],"next_page_token":""}`,
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
			mockSvc.EXPECT().StorageFileList(req.Context(), tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseExtension, nil)

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

func Test_GetStorageFilesId(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files/e1eb02c2-1715-11ef-b15f-f3c445db0e34",
			responseFile: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e1eb02c2-1715-11ef-b15f-f3c445db0e34"),
				},
			},

			expectFileID: uuid.FromStringOrNil("e1eb02c2-1715-11ef-b15f-f3c445db0e34"),
			expectRes:    `{"id":"e1eb02c2-1715-11ef-b15f-f3c445db0e34","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().StorageFileGet(req.Context(), tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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

func Test_DeleteStorageFilesId(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files/22bad83e-1718-11ef-8e93-63a03937356b",
			responseFile: &smfile.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("22bad83e-1718-11ef-8e93-63a03937356b"),
				},
			},

			expectFileID: uuid.FromStringOrNil("22bad83e-1718-11ef-8e93-63a03937356b"),
			expectRes:    `{"id":"22bad83e-1718-11ef-8e93-63a03937356b","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","type":"","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":null,"tm_create":null,"tm_update":null,"tm_delete":null}`,
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
			mockSvc.EXPECT().StorageFileDelete(req.Context(), tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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

func Test_GetStorageFilesIdFile(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files/e1eb02c2-1715-11ef-b15f-f3c445db0e34/file",

			responseDownloadURL: "https://storage.example.com/file.txt?token=valid",

			expectFileID: uuid.FromStringOrNil("e1eb02c2-1715-11ef-b15f-f3c445db0e34"),
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
			mockSvc.EXPECT().StorageFileDownloadRedirect(req.Context(), tt.agent, tt.expectFileID).Return(tt.responseDownloadURL, nil)

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

func Test_GetStorageFilesIdFile_NoAgent(t *testing.T) {

	tests := []struct {
		name     string
		reqQuery string
	}{
		{
			name:     "no agent in context",
			reqQuery: "/storage_files/e1eb02c2-1715-11ef-b15f-f3c445db0e34/file",
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
			assertErrorResponse(t, w, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED", commonoutline.ServiceNameAPIManager)
		})
	}
}

func Test_GetStorageFilesIdFile_InvalidUUID(t *testing.T) {

	tests := []struct {
		name     string
		agent *auth.AuthIdentity
		reqQuery string
	}{
		{
			name: "invalid UUID",
			agent: auth.NewAgentIdentity(&amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),
			reqQuery: "/storage_files/invalid-uuid/file",
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

func Test_GetStorageFilesIdFile_ServiceError(t *testing.T) {

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
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			}),

			reqQuery: "/storage_files/e1eb02c2-1715-11ef-b15f-f3c445db0e34/file",

			expectFileID: uuid.FromStringOrNil("e1eb02c2-1715-11ef-b15f-f3c445db0e34"),
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
			mockSvc.EXPECT().StorageFileDownloadRedirect(gomock.Any(), tt.agent, tt.expectFileID).Return("", fmt.Errorf("file not found"))

			r.ServeHTTP(w, req)
			// The legacy "file not found" message routes through the
			// translator's substring matcher to NOT_FOUND / 404.
			assertErrorResponse(t, w, cerrors.StatusNotFound, "RESOURCE_NOT_FOUND", commonoutline.ServiceNameAPIManager)
		})
	}
}

// Test_storageFilesPost_MissingAuthIdentity verifies PostStorageFiles
// emits the canonical UNAUTHENTICATED / AUTHENTICATION_REQUIRED envelope
// when auth_identity is missing from the gin context.
func Test_storageFilesPost_MissingAuthIdentity(t *testing.T) {
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

	// Build a multipart body so the request shape matches a real upload,
	// even though auth check should reject before parsing.
	body := &bytes.Buffer{}
	mw := multipart.NewWriter(body)
	fw, _ := mw.CreateFormFile("file", "sample.txt")
	_, _ = fw.Write([]byte("hello"))
	_ = mw.WriteField("type", "rag")
	_ = mw.Close()

	req, _ := http.NewRequest(http.MethodPost, "/storage_files", body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusUnauthenticated, "AUTHENTICATION_REQUIRED", commonoutline.ServiceNameAPIManager)
}

// Test_storageFilesIDDelete_InvalidID verifies that a malformed UUID in
// the path triggers INVALID_ARGUMENT / INVALID_ID before the
// servicehandler is consulted.
func Test_storageFilesIDDelete_InvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	agent := auth.NewAgentIdentity(&amagent.Agent{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
		},
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := servicehandler.NewMockServiceHandler(mc)
	h := &server{serviceHandler: mockSvc}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)
	r.Use(middleware.RequestID())
	r.Use(func(c *gin.Context) {
		c.Set("auth_identity", agent)
	})
	openapi_server.RegisterHandlers(r, h)

	// "not-a-uuid" passes the path-shape check but uuid.FromStringOrNil
	// returns uuid.Nil, so the handler rejects with INVALID_ID.
	req, _ := http.NewRequest(http.MethodDelete, "/storage_files/not-a-uuid", nil)
	r.ServeHTTP(w, req)

	assertErrorResponse(t, w, cerrors.StatusInvalidArgument, "INVALID_ID", commonoutline.ServiceNameAPIManager)
}
