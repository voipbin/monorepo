package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/models/recordingpeak"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/storagehandler"
)

const testID = "5b2c3d4e-1111-2222-3333-444455556666"

type (
	storageMock = storagehandler.MockStorageHandler
	accountMock = accounthandler.MockAccountHandler
)

func Test_handlers_invalidBody(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{"accounts post", &sock.Request{URI: "/v1/accounts", Method: sock.RequestMethodPost, Data: []byte(`{not json`)}},
		{"accounts get", &sock.Request{URI: "/v1/accounts?page_size=3", Method: sock.RequestMethodGet, Data: []byte(`{not json`)}},
		{"compressfiles post", &sock.Request{URI: "/v1/compressfiles", Method: sock.RequestMethodPost, Data: []byte(`{not json`)}},
		{"files post", &sock.Request{URI: "/v1/files", Method: sock.RequestMethodPost, Data: []byte(`{not json`)}},
		{"files get", &sock.Request{URI: "/v1/files?page_size=3", Method: sock.RequestMethodGet, Data: []byte(`{not json`)}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectations: a body that cannot be parsed must not reach any business handler.
			h, _, _ := newTestListenHandler(mc)

			res, err := h.processRequest(tt.request)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil response, got: %v", res)
			}
		})
	}
}

func Test_handlers_invalidFilter(t *testing.T) {
	// "deleted" must be a boolean; any other type cannot be converted and is a client error.
	tests := []struct {
		name string
		uri  string
	}{
		{"accounts get", "/v1/accounts?page_size=3"},
		{"files get", "/v1/files?page_size=3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _ := newTestListenHandler(mc) // the list calls must not be reached

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: sock.RequestMethodGet, Data: []byte(`{"deleted":123}`)})
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}

func Test_handlers_filtersAndPagingReachTheHandler(t *testing.T) {
	ctx := gomock.Any()

	t.Run("accounts", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockAccount := newTestListenHandler(mc)
		mockAccount.EXPECT().List(ctx, "tok-a", uint64(7), map[account.Field]any{
			account.FieldDeleted:    false,
			account.FieldCustomerID: uuid.FromStringOrNil(testID),
		}).Return([]*account.Account{}, nil)

		res, err := h.processRequest(&sock.Request{
			URI:    "/v1/accounts?page_size=7&page_token=tok-a",
			Method: sock.RequestMethodGet,
			Data:   []byte(`{"deleted":false,"customer_id":"` + testID + `"}`),
		})
		if err != nil || res == nil || res.StatusCode != http.StatusOK || string(res.Data) != `[]` {
			t.Errorf("Wrong match. got: %v/%v", res, err)
		}
	})

	t.Run("files", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockStorage, _ := newTestListenHandler(mc)
		mockStorage.EXPECT().FileList(ctx, "tok-f", uint64(9), map[file.Field]any{
			file.FieldDeleted:     false,
			file.FieldReferenceID: uuid.FromStringOrNil(testID),
		}).Return([]*file.File{}, nil)

		res, err := h.processRequest(&sock.Request{
			URI:    "/v1/files?page_size=9&page_token=tok-f",
			Method: sock.RequestMethodGet,
			Data:   []byte(`{"deleted":false,"reference_id":"` + testID + `"}`),
		})
		if err != nil || res == nil || res.StatusCode != http.StatusOK || string(res.Data) != `[]` {
			t.Errorf("Wrong match. got: %v/%v", res, err)
		}
	})
}

func Test_handlers_businessErrorsPropagate(t *testing.T) {
	uid := uuid.FromStringOrNil(testID)

	tests := []struct {
		name    string
		request *sock.Request
		expect  func(s *storageMock, a *accountMock)
	}{
		{
			name:    "accounts post",
			request: &sock.Request{URI: "/v1/accounts", Method: sock.RequestMethodPost, Data: []byte(`{"customer_id":"` + testID + `"}`)},
			expect:  func(s *storageMock, a *accountMock) { a.EXPECT().Create(gomock.Any(), uid).Return(nil, errTest) },
		},
		{
			name:    "accounts get",
			request: &sock.Request{URI: "/v1/accounts?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expect: func(s *storageMock, a *accountMock) {
				a.EXPECT().List(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, errTest)
			},
		},
		{
			name:    "accounts id get",
			request: &sock.Request{URI: "/v1/accounts/" + testID, Method: sock.RequestMethodGet},
			expect:  func(s *storageMock, a *accountMock) { a.EXPECT().Get(gomock.Any(), uid).Return(nil, errTest) },
		},
		{
			name:    "accounts id delete",
			request: &sock.Request{URI: "/v1/accounts/" + testID, Method: sock.RequestMethodDelete},
			expect:  func(s *storageMock, a *accountMock) { a.EXPECT().Delete(gomock.Any(), uid).Return(nil, errTest) },
		},
		{
			name:    "compressfiles post",
			request: &sock.Request{URI: "/v1/compressfiles", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			expect: func(s *storageMock, a *accountMock) {
				s.EXPECT().CompressfileCreate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errTest)
			},
		},
		{
			name:    "files post",
			request: &sock.Request{URI: "/v1/files", Method: sock.RequestMethodPost, Data: []byte(`{}`)},
			expect: func(s *storageMock, a *accountMock) {
				s.EXPECT().FileCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errTest)
			},
		},
		{
			name:    "files get",
			request: &sock.Request{URI: "/v1/files?page_size=1", Method: sock.RequestMethodGet, Data: []byte(`{}`)},
			expect: func(s *storageMock, a *accountMock) {
				s.EXPECT().FileList(gomock.Any(), "", uint64(1), gomock.Any()).Return(nil, errTest)
			},
		},
		{
			name:    "files id get",
			request: &sock.Request{URI: "/v1/files/" + testID, Method: sock.RequestMethodGet},
			expect:  func(s *storageMock, a *accountMock) { s.EXPECT().FileGet(gomock.Any(), uid).Return(nil, errTest) },
		},
		{
			name:    "files id delete",
			request: &sock.Request{URI: "/v1/files/" + testID, Method: sock.RequestMethodDelete},
			expect:  func(s *storageMock, a *accountMock) { s.EXPECT().FileDelete(gomock.Any(), uid).Return(nil, errTest) },
		},
		{
			name:    "files id download uri refresh",
			request: &sock.Request{URI: "/v1/files/" + testID + "/download_uri_refresh", Method: sock.RequestMethodPost},
			expect: func(s *storageMock, a *accountMock) {
				s.EXPECT().FileDownloadURIRefresh(gomock.Any(), uid).Return("", errTest)
			},
		},
		{
			name:    "recordings id get",
			request: &sock.Request{URI: "/v1/recordings/" + testID, Method: sock.RequestMethodGet},
			expect:  func(s *storageMock, a *accountMock) { s.EXPECT().RecordingGet(gomock.Any(), uid).Return(nil, errTest) },
		},
		{
			name:    "recordings id peaks get",
			request: &sock.Request{URI: "/v1/recordings/" + testID + "/peaks", Method: sock.RequestMethodGet},
			expect: func(s *storageMock, a *accountMock) {
				s.EXPECT().RecordingPeaks(gomock.Any(), uid).Return(nil, errTest)
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
			if !errors.Is(err, errTest) || res != nil {
				t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTest, res, err)
			}
		})
	}
}

func Test_v1FilesIDDownloadURIRefresh(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockStorage, _ := newTestListenHandler(mc)
	mockStorage.EXPECT().FileDownloadURIRefresh(gomock.Any(), uuid.FromStringOrNil(testID)).Return("https://example.invalid/signed?a=1&b=2", nil)

	res, err := h.processRequest(&sock.Request{URI: "/v1/files/" + testID + "/download_uri_refresh", Method: sock.RequestMethodPost})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusOK || res.DataType != "application/json" {
		t.Errorf("Wrong match. got: %d/%s", res.StatusCode, res.DataType)
	}

	var got string
	if err := json.Unmarshal(res.Data, &got); err != nil || got != "https://example.invalid/signed?a=1&b=2" {
		t.Errorf("Wrong match. expect the uri as a json string, got: %s (%v)", res.Data, err)
	}
}

func Test_v1RecordingsIDPeaksGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockStorage, _ := newTestListenHandler(mc)
	mockStorage.EXPECT().RecordingPeaks(gomock.Any(), uuid.FromStringOrNil(testID)).Return(map[string]recordingpeak.RecordingFilePeak{
		"in.wav": {Peaks: []float64{0.5, 1}, Duration: 2.5},
	}, nil)

	res, err := h.processRequest(&sock.Request{URI: "/v1/recordings/" + testID + "/peaks", Method: sock.RequestMethodGet})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.StatusCode != http.StatusOK || res.DataType != "application/json" {
		t.Errorf("Wrong match. got: %d/%s", res.StatusCode, res.DataType)
	}

	var got map[string]recordingpeak.RecordingFilePeak
	if err := json.Unmarshal(res.Data, &got); err != nil {
		t.Fatalf("Wrong match. expect: json, got: %s (%v)", res.Data, err)
	}
	if p := got["in.wav"]; len(p.Peaks) != 2 || p.Peaks[0] != 0.5 || p.Peaks[1] != 1 || p.Duration != 2.5 {
		t.Errorf("Wrong match. got: %+v", got)
	}
}

func Test_recordings_shortURI(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		run  func(h *listenHandler) (*sock.Response, error)
	}{
		{"get", func(h *listenHandler) (*sock.Response, error) {
			return h.v1RecordingsIDGet(ctx, &sock.Request{URI: "/v1/recordings"})
		}},
		{"peaks", func(h *listenHandler) (*sock.Response, error) {
			return h.v1RecordingsIDPeaksGet(ctx, &sock.Request{URI: "/v1/recordings/" + testID})
		}},
		{"delete", func(h *listenHandler) (*sock.Response, error) {
			return h.v1RecordingsIDDelete(ctx, &sock.Request{URI: "/v1/recordings"})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _ := newTestListenHandler(mc) // the storage handler must not be reached

			res, err := tt.run(h)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res == nil || res.StatusCode != http.StatusBadRequest {
				t.Errorf("Wrong match. expect: 400, got: %v", res)
			}
		})
	}
}
