package service_agents

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

func Test_filesGET(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery      string
		responseFiles []*smfile.WebhookMessage

		expectPageToken string
		expectPageSize  uint64
		expectRes       string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/v1.0/service_agents/files?page_token=2020-09-20%2003:23:20.995000&page_size=10",

			responseFiles: []*smfile.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("89f20424-c063-11ef-850f-ff10a10c813c"),
				},
				{
					ID: uuid.FromStringOrNil("8a56f320-c063-11ef-9e55-37bada852d90"),
				},
			},

			expectPageToken: "2020-09-20 03:23:20.995000",
			expectPageSize:  10,
			expectRes:       `{"result":[{"id":"89f20424-c063-11ef-850f-ff10a10c813c","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""},{"id":"8a56f320-c063-11ef-9e55-37bada852d90","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ServiceAgentFileGets(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseFiles, nil)

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

func Test_filesPOST(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		filename string
		resFile  *smfile.WebhookMessage
	}{
		{
			"normal",
			amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("8df6606e-c064-11ef-a218-e33471dfe402"),
				},
			},

			"/v1.0/service_agents/files",
			"testfile.txt",
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("8e4c6f0e-c064-11ef-bb56-1378d1beb8d3"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// 10 MB
			testFileData := bytes.Repeat([]byte("a"), int(10<<20))
			_, err = part.Write(testFileData)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			writer.Close()

			req, _ := http.NewRequest("POST", tt.reqQuery, body)
			req.Header.Add("Content-Type", writer.FormDataContentType())

			mockSvc.EXPECT().ServiceAgentFileCreate(req.Context(), &tt.agent, gomock.Any(), "", "", tt.filename).Return(tt.resFile, nil)

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Wrong match. expect: %d, got: %d", http.StatusOK, w.Code)
			}
		})
	}
}

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
					ID: uuid.FromStringOrNil("9a396eca-c064-11ef-80a5-83bf037694fc"),
				},
			},

			"/v1.0/service_agents/files",
			"testfile.txt",
			int(constMaxFileSize) + 1,
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("9a5e61e4-c064-11ef-8932-9f421638e787"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
					ID: uuid.FromStringOrNil("b842ed2e-c064-11ef-9fa3-1b01edf05df4"),
				},
			},

			"/v1.0/service_agents/files/b88b4e20-c064-11ef-87eb-97539ef68493",
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("b88b4e20-c064-11ef-87eb-97539ef68493"),
			},

			uuid.FromStringOrNil("b88b4e20-c064-11ef-87eb-97539ef68493"),
			`{"id":"b88b4e20-c064-11ef-87eb-97539ef68493","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ServiceAgentFileGet(req.Context(), &tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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
					ID: uuid.FromStringOrNil("b9025560-c064-11ef-a9ed-730006b7287a"),
				},
			},

			"/v1.0/service_agents/files/b92d7ca4-c064-11ef-92a7-93f60933d0ba",
			&smfile.WebhookMessage{
				ID: uuid.FromStringOrNil("b92d7ca4-c064-11ef-92a7-93f60933d0ba"),
			},

			uuid.FromStringOrNil("b92d7ca4-c064-11ef-92a7-93f60933d0ba"),
			`{"id":"b92d7ca4-c064-11ef-92a7-93f60933d0ba","customer_id":"00000000-0000-0000-0000-000000000000","owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","filename":"","filesize":0,"uri_download":"","tm_download_expire":"","tm_create":"","tm_update":"","tm_delete":""}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			mockSvc.EXPECT().ServiceAgentFileDelete(req.Context(), &tt.agent, tt.expectFileID).Return(tt.responseFile, nil)

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
