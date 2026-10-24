package emailhandler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-email-manager/models/email"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"

	"github.com/gofrs/uuid"
	"github.com/sendgrid/sendgrid-go"
	"go.uber.org/mock/gomock"
)

// sendgridCapture records the requests a fake sendgrid server received.
type sendgridCapture struct {
	mu     sync.Mutex
	bodies [][]byte
}

func (c *sendgridCapture) add(b []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bodies = append(c.bodies, b)
}

func (c *sendgridCapture) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.bodies)
}

func (c *sendgridCapture) last() []byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.bodies) == 0 {
		return nil
	}
	return c.bodies[len(c.bodies)-1]
}

// newTestEngineSendgrid returns an engine whose client talks to a loopback server only.
func newTestEngineSendgrid(t *testing.T, reqHandler requesthandler.RequestHandler, serverURL string) *engineSendgrid {
	t.Helper()

	client := sendgrid.NewSendClient("SG.test-key")
	client.BaseURL = serverURL + "/v3/mail/send"

	u, err := url.Parse(client.BaseURL)
	if err != nil {
		t.Fatalf("could not parse the base url. err: %v", err)
	}
	if u.Hostname() != "127.0.0.1" {
		t.Fatalf("the sendgrid client must only talk to loopback. host: %s", u.Hostname())
	}

	return &engineSendgrid{
		reqHandler: reqHandler,
		client:     client,
	}
}

func testSendgridEmail(attachments []email.Attachment) *email.Email {
	return &email.Email{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("5e8f0f0a-1111-4111-8111-000000000001")},
		Source: &commonaddress.Address{
			Type:       commonaddress.TypeEmail,
			Target:     "service@example.com",
			TargetName: "test service",
		},
		Destinations: []commonaddress.Address{
			{Type: commonaddress.TypeEmail, Target: "alice@example.com", TargetName: "alice"},
			{Type: commonaddress.TypeEmail, Target: "bob@example.com"},
		},
		Subject:     "test subject",
		Content:     "test content",
		Attachments: attachments,
	}
}

func Test_engineSendgrid_Send_success(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	capture := &sendgridCapture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v3/mail/send" {
			t.Errorf("unexpected request. method: %s, path: %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer SG.test-key" {
			t.Errorf("wrong authorization header. got: %s", got)
		}
		b, _ := io.ReadAll(r.Body)
		capture.add(b)
		w.Header().Set("X-Message-Id", "synthetic-message-id")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	h := newTestEngineSendgrid(t, requesthandler.NewMockRequestHandler(mc), server.URL)
	m := testSendgridEmail(nil)

	res, err := h.Send(context.Background(), m)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != "synthetic-message-id" {
		t.Errorf("Wrong message id. expect: synthetic-message-id, got: %s", res)
	}

	var body struct {
		From             map[string]string `json:"from"`
		Subject          string            `json:"subject"`
		Content          []map[string]any  `json:"content"`
		Personalizations []struct {
			To         []map[string]string `json:"to"`
			CustomArgs map[string]string   `json:"custom_args"`
		} `json:"personalizations"`
		Attachments []map[string]any `json:"attachments"`
	}
	if errUnmarshal := json.Unmarshal(capture.last(), &body); errUnmarshal != nil {
		t.Fatalf("could not parse the request body. err: %v", errUnmarshal)
	}

	if body.From["email"] != "service@example.com" || body.From["name"] != "test service" {
		t.Errorf("Wrong from. got: %v", body.From)
	}
	if body.Subject != "test subject" {
		t.Errorf("Wrong subject. got: %s", body.Subject)
	}
	if len(body.Content) != 1 || body.Content[0]["type"] != "text/plain" || body.Content[0]["value"] != "test content" {
		t.Errorf("Wrong content. got: %v", body.Content)
	}
	if len(body.Personalizations) != 1 {
		t.Fatalf("Wrong personalizations count. got: %d", len(body.Personalizations))
	}
	if len(body.Personalizations[0].To) != 2 || body.Personalizations[0].To[0]["email"] != "alice@example.com" || body.Personalizations[0].To[1]["email"] != "bob@example.com" {
		t.Errorf("Wrong recipients. got: %v", body.Personalizations[0].To)
	}
	if body.Personalizations[0].CustomArgs["voipbin_message_id"] != m.ID.String() {
		t.Errorf("Wrong custom arg. got: %v", body.Personalizations[0].CustomArgs)
	}
	if len(body.Attachments) != 0 {
		t.Errorf("Expected no attachments. got: %v", body.Attachments)
	}
}

func Test_engineSendgrid_Send_attachment(t *testing.T) {
	recordingID := uuid.FromStringOrNil("5e8f0f0a-2222-4222-8222-000000000001")

	tests := []struct {
		name string

		attachments []email.Attachment
		// recording lookup result: download path to serve, or lookup error
		lookupErr    bool
		downloadCode int

		expectAttachment bool
	}{
		{
			name:             "recording_attached",
			attachments:      []email.Attachment{{ReferenceType: email.AttachmentReferenceTypeRecording, ReferenceID: recordingID}},
			downloadCode:     http.StatusOK,
			expectAttachment: true,
		},
		{
			name:             "lookup_error_is_skipped",
			attachments:      []email.Attachment{{ReferenceType: email.AttachmentReferenceTypeRecording, ReferenceID: recordingID}},
			lookupErr:        true,
			expectAttachment: false,
		},
		{
			name:             "download_error_is_skipped",
			attachments:      []email.Attachment{{ReferenceType: email.AttachmentReferenceTypeRecording, ReferenceID: recordingID}},
			downloadCode:     http.StatusInternalServerError,
			expectAttachment: false,
		},
		{
			name:             "unknown_reference_type_is_skipped",
			attachments:      []email.Attachment{{ReferenceType: email.AttachmentReferenceType("unknown"), ReferenceID: recordingID}},
			expectAttachment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			capture := &sendgridCapture{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v3/mail/send":
					b, _ := io.ReadAll(r.Body)
					capture.add(b)
					w.Header().Set("X-Message-Id", "synthetic-message-id")
					w.WriteHeader(http.StatusAccepted)
				case "/download":
					w.WriteHeader(tt.downloadCode)
					_, _ = w.Write([]byte("synthetic-recording-bytes"))
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			if tt.attachments[0].ReferenceType == email.AttachmentReferenceTypeRecording {
				if tt.lookupErr {
					mockReq.EXPECT().StorageV1RecordingGet(gomock.Any(), recordingID, 60000).Return(nil, fmt.Errorf("lookup failed"))
				} else {
					mockReq.EXPECT().StorageV1RecordingGet(gomock.Any(), recordingID, 60000).Return(&smbucketfile.BucketFile{
						ReferenceID: recordingID,
						DownloadURI: server.URL + "/download",
					}, nil)
				}
			}

			h := newTestEngineSendgrid(t, mockReq, server.URL)

			res, err := h.Send(context.Background(), testSendgridEmail(tt.attachments))
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != "synthetic-message-id" {
				t.Errorf("Wrong message id. got: %s", res)
			}
			if capture.count() != 1 {
				t.Fatalf("Wrong send count. expect: 1, got: %d", capture.count())
			}

			var body struct {
				Attachments []struct {
					Content     string `json:"content"`
					Type        string `json:"type"`
					Filename    string `json:"filename"`
					Disposition string `json:"disposition"`
				} `json:"attachments"`
			}
			if errUnmarshal := json.Unmarshal(capture.last(), &body); errUnmarshal != nil {
				t.Fatalf("could not parse the request body. err: %v", errUnmarshal)
			}

			if !tt.expectAttachment {
				if len(body.Attachments) != 0 {
					t.Errorf("Expected the attachment to be skipped. got: %v", body.Attachments)
				}
				return
			}

			if len(body.Attachments) != 1 {
				t.Fatalf("Wrong attachment count. expect: 1, got: %d", len(body.Attachments))
			}
			a := body.Attachments[0]
			if a.Filename != recordingID.String()+".zip" || a.Type != "application/zip" || a.Disposition != "attachment" {
				t.Errorf("Wrong attachment meta. got: %+v", a)
			}
			if a.Content != base64.StdEncoding.EncodeToString([]byte("synthetic-recording-bytes")) {
				t.Errorf("Wrong attachment content. got: %s", a.Content)
			}
		})
	}
}

func Test_engineSendgrid_Send_responseErrors(t *testing.T) {
	tests := []struct {
		name string

		status    int
		messageID string

		expectErr string
	}{
		{
			name:      "no_message_id_header",
			status:    http.StatusAccepted,
			expectErr: "could not get message id from response headers",
		},
		{
			name:      "server_error_without_message_id",
			status:    http.StatusInternalServerError,
			expectErr: "could not get message id from response headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer server.Close()

			h := newTestEngineSendgrid(t, requesthandler.NewMockRequestHandler(mc), server.URL)

			res, err := h.Send(context.Background(), testSendgridEmail(nil))
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
			if res != "" {
				t.Errorf("Expected an empty message id. got: %s", res)
			}
		})
	}
}

func Test_engineSendgrid_Send_transportError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// reserve a loopback port, then close it so nothing is listening
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	closedURL := "http://" + l.Addr().String()
	_ = l.Close()

	h := newTestEngineSendgrid(t, requesthandler.NewMockRequestHandler(mc), closedURL)

	res, errSend := h.Send(context.Background(), testSendgridEmail(nil))
	if errSend == nil || !strings.Contains(errSend.Error(), "could not send email.") {
		t.Errorf("Wrong match. expect error containing: could not send email., got: %v", errSend)
	}
	if res != "" {
		t.Errorf("Expected an empty message id. got: %s", res)
	}
}
