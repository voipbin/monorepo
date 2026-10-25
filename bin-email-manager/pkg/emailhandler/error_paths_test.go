package emailhandler

import (
	"context"
	stderrors "errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bmbilling "monorepo/bin-billing-manager/models/billing"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cucustomer "monorepo/bin-customer-manager/models/customer"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	"github.com/sendgrid/sendgrid-go"
	"go.uber.org/mock/gomock"
)

type testMocks struct {
	db     *dbhandler.MockDBHandler
	req    *requesthandler.MockRequestHandler
	notify *notifyhandler.MockNotifyHandler
	util   *utilhandler.MockUtilHandler
}

func newErrorPathHandler(mc *gomock.Controller) (*emailHandler, *testMocks) {
	m := &testMocks{
		db:     dbhandler.NewMockDBHandler(mc),
		req:    requesthandler.NewMockRequestHandler(mc),
		notify: notifyhandler.NewMockNotifyHandler(mc),
		util:   utilhandler.NewMockUtilHandler(mc),
	}

	h := &emailHandler{
		db:            m.db,
		reqHandler:    m.req,
		notifyHandler: m.notify,
		utilHandler:   m.util,
	}

	return h, m
}

func Test_Create_errorPaths(t *testing.T) {
	customerID := uuid.FromStringOrNil("5e8f0f0a-2222-4222-8222-000000000001")
	emailID := uuid.FromStringOrNil("5e8f0f0a-2222-4222-8222-000000000002")
	destinations := func() []commonaddress.Address {
		return []commonaddress.Address{
			{Type: commonaddress.TypeEmail, Target: "alice@example.com", TargetName: "alice"},
		}
	}

	tests := []struct {
		name string

		balanceErr error
		createErr  error
		getErr     error

		expectErr string
	}{
		{
			name:       "balance_check_fails",
			balanceErr: stderrors.New("billing unavailable"),
			expectErr:  "could not validate the customer's balance: billing unavailable",
		},
		{
			name:      "db_create_fails",
			createErr: stderrors.New("insert failed"),
			expectErr: "could not create email: could not create email.: insert failed",
		},
		{
			name:      "get_after_create_fails",
			getErr:    stderrors.New("select failed"),
			expectErr: "could not create email: could not get created email.: could not get email: select failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrorPathHandler(mc)
			ctx := context.Background()

			m.req.EXPECT().CustomerV1CustomerGet(ctx, customerID).Return(&cucustomer.Customer{
				ID:                         customerID,
				IdentityVerificationStatus: cucustomer.IdentityVerificationStatusVerified,
			}, nil)
			m.req.EXPECT().BillingV1AccountIsValidBalanceByCustomerID(ctx, customerID, bmbilling.ReferenceTypeEmail, "", 1).Return(true, tt.balanceErr)

			if tt.balanceErr == nil {
				m.util.EXPECT().UUIDCreate().Return(emailID)
				m.db.EXPECT().EmailCreate(ctx, gomock.Any()).Return(tt.createErr)
				if tt.createErr == nil {
					m.db.EXPECT().EmailGet(ctx, emailID).Return(nil, tt.getErr)
				}
			}

			res, err := h.Create(ctx, customerID, uuid.Nil, destinations(), "subject", "content", nil)
			if err == nil || err.Error() != tt.expectErr {
				t.Errorf("Wrong match. expect error: %s, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Expected a nil result. got: %v", res)
			}
		})
	}
}

func Test_Delete_getAfterDeleteFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrorPathHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("5e8f0f0a-3333-4333-8333-000000000001")

	// no PublishWebhookEvent expectation: a failed read must not publish an event
	m.db.EXPECT().EmailDelete(ctx, id).Return(nil)
	m.db.EXPECT().EmailGet(ctx, id).Return(nil, stderrors.New("select failed"))

	res, err := h.Delete(ctx, id)
	if err == nil || err.Error() != "could not get deleted email: select failed" {
		t.Errorf("Wrong match. got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result. got: %v", res)
	}
}

func Test_Get_notFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrorPathHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("5e8f0f0a-4444-4444-8444-000000000001")

	m.db.EXPECT().EmailGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, id)
	if res != nil {
		t.Errorf("Expected a nil result. got: %v", res)
	}

	var verr *cerrors.VoipbinError
	if !stderrors.As(err, &verr) {
		t.Fatalf("Expected a VoipbinError. got: %v", err)
	}
	if verr.Status != cerrors.StatusNotFound || verr.Reason != "EMAIL_NOT_FOUND" {
		t.Errorf("Wrong error. status: %s, reason: %s", verr.Status, verr.Reason)
	}
	if !stderrors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Expected the cause to be kept in the chain. got: %v", err)
	}
}

func Test_UpdateStatus_getAfterUpdateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newErrorPathHandler(mc)
	ctx := context.Background()
	id := uuid.FromStringOrNil("5e8f0f0a-5555-4555-8555-000000000001")

	// no PublishWebhookEvent expectation: a failed read must not publish an event
	m.db.EXPECT().EmailUpdateStatus(ctx, id, email.StatusDelivered).Return(nil)
	m.db.EXPECT().EmailGet(ctx, id).Return(nil, stderrors.New("select failed"))

	res, err := h.UpdateStatus(ctx, id, email.StatusDelivered)
	if err == nil || err.Error() != "could not get updated email.: could not get email: select failed" {
		t.Errorf("Wrong match. got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil result. got: %v", res)
	}
}

func Test_Hook_mailgunURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// strict mocks without expectations: the mailgun hook must not touch the db or publish
	h, _ := newErrorPathHandler(mc)

	if err := h.Hook(context.Background(), "hook.example.com/v1.0/emails/mailgun", []byte(`{}`)); err != nil {
		t.Errorf("Expected nil for a mailgun uri. got: %v", err)
	}
}

func Test_download_errorPaths(t *testing.T) {
	// a loopback port that is closed again, so nothing listens on it
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	closedURL := "http://" + l.Addr().String()
	_ = l.Close()

	truncated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// declare more bytes than are sent, then drop the connection
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("short"))
		if hj, ok := w.(http.Hijacker); ok {
			conn, _, errHijack := hj.Hijack()
			if errHijack == nil {
				_ = conn.Close()
			}
		}
	}))
	defer truncated.Close()

	tests := []struct {
		name string
		uri  string

		expectErr string
	}{
		{
			name:      "invalid_request_url",
			uri:       "http://127.0.0.1:0/\x7f",
			expectErr: "failed to create request",
		},
		{
			name:      "transport_error",
			uri:       closedURL,
			expectErr: "failed to download file",
		},
		{
			name:      "truncated_body",
			uri:       truncated.URL,
			expectErr: "failed to read response body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := download(context.Background(), tt.uri)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
				t.Errorf("Wrong match. expect error containing: %s, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Expected a nil result. got: %d bytes", len(res))
			}
		})
	}
}

func Test_download_sizeBoundary(t *testing.T) {
	tests := []struct {
		name string
		size int

		expectErr bool
	}{
		{name: "exactly_the_maximum", size: defaultMaxDownloadSize, expectErr: false},
		{name: "one_byte_over", size: defaultMaxDownloadSize + 1, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(make([]byte, tt.size))
			}))
			defer server.Close()

			res, err := download(context.Background(), server.URL)
			if tt.expectErr {
				if err == nil || !strings.Contains(err.Error(), "file size exceeds maximum download size") {
					t.Errorf("Wrong match. expect the size error, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if len(res) != tt.size {
				t.Errorf("Wrong size. expect: %d, got: %d", tt.size, len(res))
			}
		})
	}
}

// headerlessTransport answers every request with 202 and no headers.
type headerlessTransport struct {
	requests int
	host     string
}

func (rt *headerlessTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.requests++
	rt.host = r.URL.Hostname()

	return &http.Response{
		StatusCode: http.StatusAccepted,
		Body:       http.NoBody,
		Header:     nil,
		Request:    r,
	}, nil
}

func Test_engineSendgrid_Send_nilHeaders(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// The sendgrid client sends through its package level default client, so only the
	// transport of a copy is replaced and the original is restored afterwards.
	// Nothing leaves the process.
	rt := &headerlessTransport{}
	saved := sendgrid.DefaultClient
	custom := *saved
	custom.HTTPClient = &http.Client{Transport: rt}
	sendgrid.DefaultClient = &custom
	t.Cleanup(func() { sendgrid.DefaultClient = saved })

	h := &engineSendgrid{
		reqHandler: requesthandler.NewMockRequestHandler(mc),
		client:     sendgrid.NewSendClient("test-key"),
	}
	h.client.BaseURL = "http://127.0.0.1:1/v3/mail/send"

	res, err := h.Send(context.Background(), testSendgridEmail(nil))
	if err == nil || !strings.Contains(err.Error(), "response headers are nil") {
		t.Errorf("Wrong match. expect error containing: response headers are nil, got: %v", err)
	}
	if res != "" {
		t.Errorf("Expected an empty message id. got: %s", res)
	}
	if rt.requests != 1 || rt.host != "127.0.0.1" {
		t.Errorf("Wrong transport use. requests: %d, host: %s", rt.requests, rt.host)
	}
}
