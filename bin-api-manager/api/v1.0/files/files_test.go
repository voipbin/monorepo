package files

import (
	"bytes"
	"mime/multipart"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/api/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	smfile "monorepo/bin-storage-manager/models/file"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func setupServer(app *gin.Engine) {
	v1 := app.RouterGroup.Group("/v1.0")
	ApplyRoutes(v1)
}

// func Test_filesPOST(t *testing.T) {

// 	tests := []struct {
// 		name  string
// 		agent amagent.Agent

// 		reqQuery string
// 		filename string
// 		resFile  *smfile.WebhookMessage
// 	}{
// 		{
// 			"normal",
// 			amagent.Agent{
// 				Identity: commonidentity.Identity{
// 					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
// 				},
// 			},

// 			"/v1.0/files",
// 			"testfile.txt",
// 			&smfile.WebhookMessage{
// 				ID: uuid.FromStringOrNil("39ae35ca-1710-11ef-bae6-afeb7c57c901"),
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// create mock
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSvc := servicehandler.NewMockServiceHandler(mc)

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("agent", tt.agent)
// 			})
// 			setupServer(r)

// 			body := new(bytes.Buffer)
// 			writer := multipart.NewWriter(body)
// 			part, err := writer.CreateFormFile("file", tt.filename)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			// 10 MB
// 			testFileData := bytes.Repeat([]byte("a"), int(10<<20))
// 			_, err = part.Write(testFileData)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 			writer.Close()

// 			req, _ := http.NewRequest("POST", tt.reqQuery, body)
// 			req.Header.Add("Content-Type", writer.FormDataContentType())

// 			mockSvc.EXPECT().StorageFileCreate(req.Context(), &tt.agent, gomock.Any(), "", "", tt.filename).Return(tt.resFile, nil)

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

func Test_filesPOST_err(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		filename string
		filesize int
		resFile  *smfile.WebhookMessage
	}{
		{
			"file size over max size",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/files",
			"testfile.txt",
			int(constMaxFileSize) + 1,
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("39ae35ca-1710-11ef-bae6-afeb7c57c901"),
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
			writer.Close()

			req, _ := http.NewRequest("POST", tt.reqQuery, body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			r.ServeHTTP(w, req)
			if w.Code == http.StatusOK {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

// func Test_filesGET(t *testing.T) {

// 	type test struct {
// 		name  string
// 		agent amagent.Agent

// 		reqQuery string

// 		expectExt []*smfile.WebhookMessage
// 	}

// 	tests := []test{
// 		{
// 			"normal",
// 			amagent.Agent{
// 				Identity: commonidentity.Identity{
// 					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
// 				},
// 			},

// 			"/v1.0/files?token_size=100",
// 			[]*smfile.WebhookMessage{
// 				{
// 					ID: uuid.FromStringOrNil("2fbb29c0-6fb0-11eb-b2ef-4303769ecba5"),
// 				},
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// create mock
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockSvc := servicehandler.NewMockServiceHandler(mc)

// 			w := httptest.NewRecorder()
// 			_, r := gin.CreateTestContext(w)

// 			r.Use(func(c *gin.Context) {
// 				c.Set(common.OBJServiceHandler, mockSvc)
// 				c.Set("agent", tt.agent)
// 			})
// 			setupServer(r)

// 			req, _ := http.NewRequest("GET", tt.reqQuery, nil)
// 			mockSvc.EXPECT().StorageFileGetsByOnwerID(req.Context(), &tt.agent, uint64(100), "").Return(tt.expectExt, nil)

// 			r.ServeHTTP(w, req)
// 			if w.Code != http.StatusOK {
// 				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
// 			}
// 		})
// 	}
// }

func Test_filesIDGET(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseFile *smfile.WebhookMessage

		expectFileID uuid.UUID
		expectRes    string
	}

	tests := []test{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/files/e1eb02c2-1715-11ef-b15f-f3c445db0e34",
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("e1eb02c2-1715-11ef-b15f-f3c445db0e34"),
			},

			uuid.FromStringOrNil("e1eb02c2-1715-11ef-b15f-f3c445db0e34"),
			`{"id":"e1eb02c2-1715-11ef-b15f-f3c445db0e34","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().StorageFileGet(req.Context(), &tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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

func Test_filesIDDELETE(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery     string
		responseFile *smfile.WebhookMessage

		expectFileID uuid.UUID
		expectRes    string
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			"/v1.0/files/22bad83e-1718-11ef-8e93-63a03937356b",
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("22bad83e-1718-11ef-8e93-63a03937356b"),
			},

			uuid.FromStringOrNil("22bad83e-1718-11ef-8e93-63a03937356b"),
			`{"id":"22bad83e-1718-11ef-8e93-63a03937356b","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().StorageFileDelete(req.Context(), &tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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
