package listenhandler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/models/email"
	"monorepo/bin-email-manager/pkg/emailhandler"

	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
	"go.uber.org/mock/gomock"
)

// unmarshalableEmail returns an email whose timestamp cannot be encoded as JSON (year 10000).
func unmarshalableEmail() *email.Email {
	tm := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)

	return &email.Email{TMCreate: &tm}
}

func Test_errorResponse_toResponseFails(t *testing.T) {
	// a channel in the details cannot be encoded, so the typed error cannot be converted
	ve := cerrors.NotFound(commonoutline.ServiceNameEmailManager, "EMAIL_NOT_FOUND", "x")
	ve.Details = []map[string]any{{"bad": make(chan int)}}

	res := errorResponse(ve)
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong status. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
	if res.DataType == cerrors.DataTypeVoipbinError || len(res.Data) != 0 {
		t.Errorf("Expected a plain status response. data type: %q, data: %q", res.DataType, string(res.Data))
	}
}

func Test_v1EmailsGet_invalidURI(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// strict mock without expectations: the handler must not be reached
	h := &listenHandler{emailHandler: emailhandler.NewMockEmailHandler(mc)}

	res, err := h.v1EmailsGet(context.Background(), &sock.Request{URI: "/v1/emails%zz?page_size=10", Method: sock.RequestMethodGet})
	if err == nil || !strings.Contains(err.Error(), "invalid URL escape") {
		t.Errorf("Wrong match. expect an invalid URL escape error, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected a nil response. got: %v", res)
	}
}

func Test_marshalFailures(t *testing.T) {
	id := "5e8f0f0a-6666-4666-8666-000000000001"

	tests := []struct {
		name string

		call func(h *listenHandler, m *emailhandler.MockEmailHandler) (*sock.Response, error)
	}{
		{
			name: "list",
			call: func(h *listenHandler, m *emailhandler.MockEmailHandler) (*sock.Response, error) {
				m.EXPECT().List(gomock.Any(), "", uint64(0), gomock.Any()).Return([]*email.Email{unmarshalableEmail()}, nil)
				return h.v1EmailsGet(context.Background(), &sock.Request{URI: "/v1/emails", Method: sock.RequestMethodGet, Data: []byte(`{}`)})
			},
		},
		{
			name: "post",
			call: func(h *listenHandler, m *emailhandler.MockEmailHandler) (*sock.Response, error) {
				m.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "s", "", gomock.Any()).Return(unmarshalableEmail(), nil)
				return h.v1EmailsPost(context.Background(), &sock.Request{URI: "/v1/emails", Method: sock.RequestMethodPost, Data: []byte(`{"subject":"s"}`)})
			},
		},
		{
			name: "get",
			call: func(h *listenHandler, m *emailhandler.MockEmailHandler) (*sock.Response, error) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(unmarshalableEmail(), nil)
				return h.v1EmailsIDGet(context.Background(), &sock.Request{URI: "/v1/emails/" + id, Method: sock.RequestMethodGet})
			},
		},
		{
			name: "delete",
			call: func(h *listenHandler, m *emailhandler.MockEmailHandler) (*sock.Response, error) {
				m.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(unmarshalableEmail(), nil)
				return h.v1EmailsIDDelete(context.Background(), &sock.Request{URI: "/v1/emails/" + id, Method: sock.RequestMethodDelete})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockEmail := emailhandler.NewMockEmailHandler(mc)
			h := &listenHandler{emailHandler: mockEmail}

			res, err := tt.call(h, mockEmail)
			if err == nil || !strings.Contains(err.Error(), "could not marshal the res") {
				t.Errorf("Wrong match. expect a marshal error, got: %v", err)
			}
			if res != nil {
				t.Errorf("Expected a nil response. got: %v", res)
			}
		})
	}
}

func Test_Run_queueCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// strict mock: ConsumeRPC must not be called after a failed queue creation
	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(fmt.Errorf("queue failed"))

	err := h.Run("test-queue", "test-delay")
	if err == nil || err.Error() != "could not declare the queue for listenHandler. err: queue failed" {
		t.Errorf("Wrong match. got: %v", err)
	}
}

func Test_Run_consume(t *testing.T) {
	tests := []struct {
		name string

		consumeErr error

		expectLogged bool
	}{
		{name: "consume_returns_nil", consumeErr: nil, expectLogged: false},
		{name: "consume_returns_error", consumeErr: fmt.Errorf("consume failed"), expectLogged: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			hook := logrustest.NewGlobal()
			oldHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
			logrus.StandardLogger().AddHook(hook)
			t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(oldHooks) })

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{sockHandler: mockSock}

			consumed := make(chan struct{})
			mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
			mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", gomock.Any(), false, false, false, 10, gomock.Any()).
				DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
					defer close(consumed)
					return tt.consumeErr
				})

			if err := h.Run("test-queue", "test-delay"); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			select {
			case <-consumed:
			case <-time.After(2 * time.Second):
				t.Fatalf("ConsumeRPC was not called")
			}

			if !tt.expectLogged {
				return
			}

			// the error is logged by the consuming goroutine after ConsumeRPC returns
			deadline := time.Now().Add(2 * time.Second)
			for time.Now().Before(deadline) {
				for _, entry := range hook.AllEntries() {
					if strings.Contains(entry.Message, "Could not consume the request message correctly. err: consume failed") {
						return
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
			t.Errorf("Expected the consume error to be logged")
		})
	}
}
