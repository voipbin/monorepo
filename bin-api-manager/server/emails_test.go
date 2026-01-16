package server

import (
	"bytes"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/gens/openapi_server"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	ememail "monorepo/bin-email-manager/models/email"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PostEmails(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string
		reqBody  []byte

		responseEmail *ememail.WebhookMessage

		expectDestinations []commonaddress.Address
		expectSubject      string
		expectContent      string
		expectAttachments  []ememail.Attachment
		expectRes          string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/emails",
			reqBody:  []byte(`{"destinations":[{"type":"email","target":"test@voipbin.net","target_name":"test name"}],"subject":"test subject","content":"test content","attachments":[{"reference_type":"recording","reference_id":"c322a4da-00ee-11f0-aecf-374e31931c10"}]}`),

			responseEmail: &ememail.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c2ca66d0-00ee-11f0-a1a7-d7a5ce06bea8"),
				},
			},

			expectDestinations: []commonaddress.Address{
				{
					Type:       commonaddress.TypeEmail,
					Target:     "test@voipbin.net",
					TargetName: "test name",
				},
			},
			expectSubject: "test subject",
			expectContent: "test content",
			expectAttachments: []ememail.Attachment{
				{
					ReferenceType: ememail.AttachmentReferenceTypeRecording,
					ReferenceID:   uuid.FromStringOrNil("c322a4da-00ee-11f0-aecf-374e31931c10"),
				},
			},
			expectRes: `{"id":"c2ca66d0-00ee-11f0-a1a7-d7a5ce06bea8","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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

			mockSvc.EXPECT().EmailSend(req.Context(), &tt.agent, tt.expectDestinations, tt.expectSubject, tt.expectContent, tt.expectAttachments).Return(tt.responseEmail, nil)

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

func Test_GetEmails(t *testing.T) {

	type test struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseEmails []*ememail.WebhookMessage

		expectPageSize  uint64
		expectPageToken string
		expectRes       string
	}

	tests := []test{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/emails?page_size=20&page_token=2020-09-20%2003:23:20.995000",

			responseEmails: []*ememail.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("7be21f1e-00ef-11f0-8a0c-f7709910a0da"),
					},
				},
			},

			expectPageSize:  20,
			expectPageToken: "2020-09-20 03:23:20.995000",
			expectRes:       `{"result":[{"id":"7be21f1e-00ef-11f0-8a0c-f7709910a0da","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":"","tm_update":"","tm_delete":""}],"next_page_token":""}`,
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
			mockSvc.EXPECT().EmailList(req.Context(), &tt.agent, tt.expectPageSize, tt.expectPageToken).Return(tt.responseEmails, nil)

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

func Test_GetEmailsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseEmail *ememail.WebhookMessage

		expectEmailID uuid.UUID
		expectRes     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/emails/b174da36-00ef-11f0-a47f-0fd0872cf536",

			responseEmail: &ememail.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b174da36-00ef-11f0-a47f-0fd0872cf536"),
				},
			},

			expectEmailID: uuid.FromStringOrNil("b174da36-00ef-11f0-a47f-0fd0872cf536"),
			expectRes:     `{"id":"b174da36-00ef-11f0-a47f-0fd0872cf536","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().EmailGet(req.Context(), &tt.agent, tt.expectEmailID).Return(tt.responseEmail, nil)

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

func Test_DeleteemailsId(t *testing.T) {

	tests := []struct {
		name  string
		agent amagent.Agent

		reqQuery string

		responseEmail *ememail.WebhookMessage

		expectEmailID uuid.UUID
		expectRes     string
	}{
		{
			name: "normal",
			agent: amagent.Agent{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a2ec0ba-8004-11ec-aea5-439829c92a7c"),
				},
			},

			reqQuery: "/emails/25ddd076-00f0-11f0-a451-fbb20c8dd036",

			responseEmail: &ememail.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("25ddd076-00f0-11f0-a451-fbb20c8dd036"),
				},
			},

			expectEmailID: uuid.FromStringOrNil("25ddd076-00f0-11f0-a451-fbb20c8dd036"),
			expectRes:     `{"id":"25ddd076-00f0-11f0-a451-fbb20c8dd036","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"status":"","subject":"","content":"","attachments":null,"tm_create":"","tm_update":"","tm_delete":""}`,
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
			mockSvc.EXPECT().EmailDelete(req.Context(), &tt.agent, tt.expectEmailID).Return(tt.responseEmail, nil)

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
