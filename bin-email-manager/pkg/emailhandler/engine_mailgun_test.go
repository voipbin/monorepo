package emailhandler

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-email-manager/models/email"
	smbucketfile "monorepo/bin-storage-manager/models/bucketfile"

	"github.com/gofrs/uuid"
	"github.com/mailgun/mailgun-go/v4"
	"go.uber.org/mock/gomock"
)

// loopbackOnlyTransport fails any request that does not target the loopback address.
type loopbackOnlyTransport struct{}

func (loopbackOnlyTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if r.URL.Hostname() != "127.0.0.1" {
		return nil, fmt.Errorf("blocked non-loopback request. host: %s", r.URL.Hostname())
	}
	return http.DefaultTransport.RoundTrip(r)
}

// mailgunCapture records what a fake mailgun server received.
type mailgunCapture struct {
	mu     sync.Mutex
	calls  int
	path   string
	values map[string][]string
	files  []string
}

func (c *mailgunCapture) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// newMailgunServer starts a fake mailgun send endpoint and a /download endpoint.
func newMailgunServer(t *testing.T, capture *mailgunCapture, sendStatus int, downloadStatus int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/download" {
			w.WriteHeader(downloadStatus)
			_, _ = w.Write([]byte("synthetic-recording-bytes"))
			return
		}

		if errParse := r.ParseMultipartForm(1 << 20); errParse != nil {
			t.Errorf("could not parse the form. err: %v", errParse)
		}

		capture.mu.Lock()
		capture.calls++
		capture.path = r.URL.Path
		capture.values = r.MultipartForm.Value
		for _, fhs := range r.MultipartForm.File {
			for _, fh := range fhs {
				f, _ := fh.Open()
				data, _ := io.ReadAll(f)
				_ = f.Close()
				capture.files = append(capture.files, fh.Filename+"="+string(data))
			}
		}
		capture.mu.Unlock()

		w.WriteHeader(sendStatus)
		_, _ = w.Write([]byte(`{"id":"<synthetic-id@example.com>","message":"Queued. Thank you."}`))
	}))
}

func newTestEngineMailgun(t *testing.T, reqHandler requesthandler.RequestHandler, serverURL string) *engineMailgun {
	t.Helper()

	client := mailgun.NewMailgun(defaultMailgunDomain, "synthetic-key")
	client.SetAPIBase(serverURL + "/v3")
	client.SetClient(&http.Client{Transport: loopbackOnlyTransport{}})

	return &engineMailgun{
		reqHandler: reqHandler,
		client:     client,
	}
}

func testMailgunEmail(attachments []email.Attachment) *email.Email {
	return &email.Email{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("5e8f0f0a-3333-4333-8333-000000000001")},
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

func Test_engineMailgun_Send_success(t *testing.T) {
	tests := []struct {
		name string

		sourceName string

		expectFrom string
	}{
		{
			name:       "source_with_name",
			sourceName: "test service",
			expectFrom: "test service <service@example.com>",
		},
		{
			name:       "source_without_name",
			sourceName: "",
			expectFrom: "service@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			capture := &mailgunCapture{}
			server := newMailgunServer(t, capture, http.StatusOK, http.StatusOK)
			defer server.Close()

			h := newTestEngineMailgun(t, requesthandler.NewMockRequestHandler(mc), server.URL)
			m := testMailgunEmail(nil)
			m.Source.TargetName = tt.sourceName

			res, err := h.Send(context.Background(), m)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res != "<synthetic-id@example.com>" {
				t.Errorf("Wrong message id. got: %s", res)
			}

			capture.mu.Lock()
			defer capture.mu.Unlock()

			if capture.path != "/v3/"+defaultMailgunDomain+"/messages" {
				t.Errorf("Wrong path. got: %s", capture.path)
			}
			if got := capture.values["from"]; len(got) != 1 || got[0] != tt.expectFrom {
				t.Errorf("Wrong from. expect: %s, got: %v", tt.expectFrom, got)
			}
			if got := capture.values["subject"]; len(got) != 1 || got[0] != "test subject" {
				t.Errorf("Wrong subject. got: %v", got)
			}
			if got := capture.values["text"]; len(got) != 1 || got[0] != "test content" {
				t.Errorf("Wrong text. got: %v", got)
			}
			if got := capture.values["to"]; len(got) != 2 || got[0] != "alice <alice@example.com>" || got[1] != "bob@example.com" {
				t.Errorf("Wrong recipients. got: %v", got)
			}
			if got := capture.values["h:X-Voipbin-Message-Id"]; len(got) != 1 || got[0] != m.ID.String() {
				t.Errorf("Wrong message id header. got: %v", got)
			}
			if len(capture.files) != 0 {
				t.Errorf("Expected no attachments. got: %v", capture.files)
			}
		})
	}
}

func Test_engineMailgun_Send_attachment(t *testing.T) {
	recordingID := uuid.FromStringOrNil("5e8f0f0a-4444-4444-8444-000000000001")
	recordingAttachment := []email.Attachment{{ReferenceType: email.AttachmentReferenceTypeRecording, ReferenceID: recordingID}}

	tests := []struct {
		name string

		attachments  []email.Attachment
		lookupErr    bool
		downloadCode int

		expectErr   string
		expectFiles []string
	}{
		{
			name:         "recording_attached",
			attachments:  recordingAttachment,
			downloadCode: http.StatusOK,
			expectFiles:  []string{recordingID.String() + ".zip=synthetic-recording-bytes"},
		},
		{
			name:        "lookup_error_fails_send",
			attachments: recordingAttachment,
			lookupErr:   true,
			expectErr:   "could not get attachment",
		},
		{
			name:         "download_error_fails_send",
			attachments:  recordingAttachment,
			downloadCode: http.StatusInternalServerError,
			expectErr:    "could not download attachment",
		},
		{
			name:        "unknown_reference_type_fails_send",
			attachments: []email.Attachment{{ReferenceType: email.AttachmentReferenceType("unknown"), ReferenceID: recordingID}},
			expectErr:   "unknown attachment reference type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			capture := &mailgunCapture{}
			server := newMailgunServer(t, capture, http.StatusOK, tt.downloadCode)
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

			h := newTestEngineMailgun(t, mockReq, server.URL)

			res, err := h.Send(context.Background(), testMailgunEmail(tt.attachments))

			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
				}
				if res != "" {
					t.Errorf("Expected an empty message id. got: %s", res)
				}
				if capture.count() != 0 {
					t.Errorf("Expected no send call. got: %d", capture.count())
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			capture.mu.Lock()
			defer capture.mu.Unlock()
			if len(capture.files) != len(tt.expectFiles) || capture.files[0] != tt.expectFiles[0] {
				t.Errorf("Wrong attachments. expect: %v, got: %v", tt.expectFiles, capture.files)
			}
		})
	}
}

func Test_engineMailgun_Send_recipientLimits(t *testing.T) {
	tests := []struct {
		name string

		recipients int

		expectErr string
	}{
		{
			name:       "too_many_recipients",
			recipients: mailgun.MaxNumberOfRecipients + 1,
			expectErr:  "could not add recipient",
		},
		{
			name:       "no_recipients",
			recipients: 0,
			expectErr:  "could not send mailgun email",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			capture := &mailgunCapture{}
			server := newMailgunServer(t, capture, http.StatusOK, http.StatusOK)
			defer server.Close()

			h := newTestEngineMailgun(t, requesthandler.NewMockRequestHandler(mc), server.URL)

			m := testMailgunEmail(nil)
			m.Destinations = make([]commonaddress.Address, tt.recipients)
			for i := range m.Destinations {
				m.Destinations[i] = commonaddress.Address{Type: commonaddress.TypeEmail, Target: fmt.Sprintf("user%d@example.com", i)}
			}

			res, err := h.Send(context.Background(), m)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
			if res != "" {
				t.Errorf("Expected an empty message id. got: %s", res)
			}
			if capture.count() != 0 {
				t.Errorf("Expected no server call. got: %d", capture.count())
			}
		})
	}
}

func Test_engineMailgun_Send_serverError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	capture := &mailgunCapture{}
	server := newMailgunServer(t, capture, http.StatusInternalServerError, http.StatusOK)
	defer server.Close()

	h := newTestEngineMailgun(t, requesthandler.NewMockRequestHandler(mc), server.URL)

	res, err := h.Send(context.Background(), testMailgunEmail(nil))
	if err == nil || !strings.Contains(err.Error(), "could not send mailgun email") {
		t.Errorf("Wrong match. expect error containing: could not send mailgun email, got: %v", err)
	}
	if res != "" {
		t.Errorf("Expected an empty message id. got: %s", res)
	}
	if capture.count() != 1 {
		t.Errorf("Expected exactly one send call. got: %d", capture.count())
	}
}

func Test_engineMailgun_Send_transportError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// a closed loopback port: the request is attempted but nothing answers
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	closedURL := "http://" + l.Addr().String()
	_ = l.Close()

	h := newTestEngineMailgun(t, requesthandler.NewMockRequestHandler(mc), closedURL)

	res, errSend := h.Send(context.Background(), testMailgunEmail(nil))
	if errSend == nil || !strings.Contains(errSend.Error(), "could not send mailgun email") {
		t.Errorf("Wrong match. expect error containing: could not send mailgun email, got: %v", errSend)
	}
	if res != "" {
		t.Errorf("Expected an empty message id. got: %s", res)
	}
}
