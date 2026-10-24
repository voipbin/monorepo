package listenhandler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
)

var errTest = errors.New("simulated failure")

func newTestListenHandler(mc *gomock.Controller) (*listenHandler, *storagehandler.MockStorageHandler, *accounthandler.MockAccountHandler) {
	mockStorage := storagehandler.NewMockStorageHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	return &listenHandler{
		sockHandler:    sockhandler.NewMockSockHandler(mc),
		storageHandler: mockStorage,
		accountHandler: mockAccount,
	}, mockStorage, mockAccount
}

// notFoundSamples returns how many requests the router counted under the `notfound` type.
func notFoundSamples(t *testing.T) uint64 {
	t.Helper()

	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	var res uint64
	for _, f := range families {
		if f.GetName() != "storage_manager_receive_request_process_time" {
			continue
		}
		for _, m := range f.GetMetric() {
			for _, l := range m.GetLabel() {
				if l.GetName() == "type" && l.GetValue() == "notfound" {
					res += m.GetHistogram().GetSampleCount()
				}
			}
		}
	}

	return res
}

func Test_processRequest_routing(t *testing.T) {
	id := "6a1b2c3d-1111-2222-3333-444455556666"
	uid := uuid.FromStringOrNil(id)

	// every route reaches exactly the handler method that serves it.
	tests := []struct {
		name    string
		request *sock.Request
		expect  func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler)
	}{
		{
			name:    "accounts post",
			request: &sock.Request{URI: "/v1/accounts", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + id + `"}`)},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				a.EXPECT().Create(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "accounts get",
			request: &sock.Request{URI: "/v1/accounts?page_size=3", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				a.EXPECT().List(gomock.Any(), "", uint64(3), gomock.Any()).Return(nil, nil)
			},
		},
		{
			name:    "accounts id get",
			request: &sock.Request{URI: "/v1/accounts/" + id, Method: sock.RequestMethodGet},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				a.EXPECT().Get(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "accounts id delete",
			request: &sock.Request{URI: "/v1/accounts/" + id, Method: sock.RequestMethodDelete},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				a.EXPECT().Delete(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "compressfiles post",
			request: &sock.Request{URI: "/v1/compressfiles", Method: sock.RequestMethodPost, Data: []byte(`{"file_ids":["` + id + `"]}`)},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().CompressfileCreate(gomock.Any(), gomock.Nil(), []uuid.UUID{uid}).Return(nil, nil)
			},
		},
		{
			name:    "files post",
			request: &sock.Request{URI: "/v1/files", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + id + `","filename":"a.wav"}`)},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().FileCreate(gomock.Any(), uid, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "a.wav", gomock.Any(), gomock.Any()).Return(nil, nil)
			},
		},
		{
			name:    "files get",
			request: &sock.Request{URI: "/v1/files?page_size=4&page_token=tok", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().FileList(gomock.Any(), "tok", uint64(4), gomock.Any()).Return(nil, nil)
			},
		},
		{
			name:    "files id download uri refresh",
			request: &sock.Request{URI: "/v1/files/" + id + "/download_uri_refresh", Method: sock.RequestMethodPost},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().FileDownloadURIRefresh(gomock.Any(), uid).Return("", nil)
			},
		},
		{
			name:    "files id get",
			request: &sock.Request{URI: "/v1/files/" + id, Method: sock.RequestMethodGet},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().FileGet(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "files id delete",
			request: &sock.Request{URI: "/v1/files/" + id, Method: sock.RequestMethodDelete},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().FileDelete(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "recordings id peaks get",
			request: &sock.Request{URI: "/v1/recordings/" + id + "/peaks", Method: sock.RequestMethodGet},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().RecordingPeaks(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "recordings id get",
			request: &sock.Request{URI: "/v1/recordings/" + id, Method: sock.RequestMethodGet},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().RecordingGet(gomock.Any(), uid).Return(nil, nil)
			},
		},
		{
			name:    "recordings id delete",
			request: &sock.Request{URI: "/v1/recordings/" + id, Method: sock.RequestMethodDelete},
			expect: func(s *storagehandler.MockStorageHandler, a *accounthandler.MockAccountHandler) {
				s.EXPECT().RecordingDelete(gomock.Any(), uid).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockStorage, mockAccount := newTestListenHandler(mc)
			tt.expect(mockStorage, mockAccount)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusOK {
				t.Errorf("Wrong match. expect: 200, got: %v", res)
			}
		})
	}
}

func Test_processRequest_notFound(t *testing.T) {
	id := "6a1b2c3d-1111-2222-3333-444455556666"

	tests := []struct {
		name    string
		request *sock.Request
	}{
		{"unknown uri", &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet}},
		{"known uri with a method that is not served", &sock.Request{URI: "/v1/accounts", Method: sock.RequestMethodDelete}},
		{"account sub resource that does not exist", &sock.Request{URI: "/v1/accounts/" + id + "/extra", Method: sock.RequestMethodGet}},
		{"recording post", &sock.Request{URI: "/v1/recordings/" + id, Method: sock.RequestMethodPost}},
		{"download refresh with get", &sock.Request{URI: "/v1/files/" + id + "/download_uri_refresh", Method: sock.RequestMethodGet}},
		{"not a uuid", &sock.Request{URI: "/v1/files/not-a-uuid", Method: sock.RequestMethodGet}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no handler expectation: a request that matches no route reaches nobody.
			h, _, _ := newTestListenHandler(mc)

			before := notFoundSamples(t)
			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusNotFound {
				t.Errorf("Wrong match. expect: 404, got: %v", res)
			}
			if got := notFoundSamples(t) - before; got != 1 {
				t.Errorf("Wrong match. expect the notfound metric to grow by 1, got: %d", got)
			}
		})
	}
}

func Test_processRequest_errorMapping(t *testing.T) {
	id := "6a1b2c3d-1111-2222-3333-444455556666"
	uid := uuid.FromStringOrNil(id)
	request := &sock.Request{URI: "/v1/files/" + id, Method: sock.RequestMethodGet}

	typedNotFound := cerrors.NotFound(commonoutline.ServiceNameStorageManager, "FILE_NOT_FOUND", "The file was not found.")

	tests := []struct {
		name string

		handlerErr error

		expectCode int
		expectErr  error
		expectType string
	}{
		{
			name:       "a typed error becomes a response with its status",
			handlerErr: typedNotFound,
			expectCode: http.StatusNotFound,
			expectType: cerrors.DataTypeVoipbinError,
		},
		{
			name:       "a wrapped typed error is still recognised",
			handlerErr: errors.Join(errTest, typedNotFound),
			expectCode: http.StatusNotFound,
			expectType: cerrors.DataTypeVoipbinError,
		},
		{
			name:       "a legacy not found becomes a 404 response",
			handlerErr: dbhandler.ErrNotFound,
			expectCode: http.StatusNotFound,
		},
		{
			name:       "any other error is passed on to the transport",
			handlerErr: errTest,
			expectErr:  errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockStorage, _ := newTestListenHandler(mc)
			mockStorage.EXPECT().FileGet(gomock.Any(), uid).Return(nil, tt.handlerErr)

			res, err := h.processRequest(request)
			if tt.expectErr != nil {
				if !errors.Is(err, tt.expectErr) || res != nil {
					t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", tt.expectErr, res, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectCode || res.DataType != tt.expectType {
				t.Errorf("Wrong match. expect: %d/%q, got: %+v", tt.expectCode, tt.expectType, res)
			}
		})
	}
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockStorage := storagehandler.NewMockStorageHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)

	res, ok := NewListenHandler(mockSock, mockStorage, mockAccount).(*listenHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *listenHandler")
	}
	if res.sockHandler != mockSock || res.storageHandler != mockStorage || res.accountHandler != mockAccount || res.utilHandler == nil {
		t.Errorf("Wrong match. the dependencies were not stored as given")
	}
}

func Test_Run(t *testing.T) {
	t.Run("queue create failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc) // ConsumeRPC must not start
		h := &listenHandler{sockHandler: mockSock}
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errTest)

		if err := h.Run("test-queue", "delay"); !errors.Is(err, errTest) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTest, err)
		}
	})

	t.Run("consume starts on the created queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		done := make(chan struct{})
		gomock.InOrder(
			mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil),
			mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", "storage-manager", false, false, false, 10, gomock.Any()).
				DoAndReturn(func(_ context.Context, _, _ string, _, _, _ bool, _ int, _ sock.CbMsgRPC) error {
					close(done)
					return nil
				}),
		)

		if err := h.Run("test-queue", "delay"); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. the consumer was not started")
		}
	})
}
