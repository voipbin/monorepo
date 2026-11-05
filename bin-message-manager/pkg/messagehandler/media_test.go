package messagehandler

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	smfile "monorepo/bin-storage-manager/models/file"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

var (
	testCustomerID = uuid.FromStringOrNil("feef3a64-4fab-46af-a61b-6a7ce31b84a9")
	testOtherID    = uuid.FromStringOrNil("11a0a0a0-0000-4000-8000-000000000001")
	testMediaID1   = uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000001")
	testMediaID2   = uuid.FromStringOrNil("a1000000-0000-4000-8000-000000000002")
)

func testMediaFile(id uuid.UUID, customerID uuid.UUID, filename string, size int64) *smfile.File {
	return &smfile.File{
		Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
		Type:     smfile.TypeMedia,
		Filename: filename,
		Filesize: size,
	}
}

func assertVoipbinError(t *testing.T, err error, status cerrors.Status, reason string) {
	t.Helper()

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect a typed error %s/%s, got: %v", status, reason, err)
	}
	if ve.Status != status || ve.Reason != reason {
		t.Errorf("Wrong match. expect: %s/%s, got: %s/%s", status, reason, ve.Status, ve.Reason)
	}
}

func Test_validateMMSText(t *testing.T) {
	if err := validateMMSText(""); err != nil {
		t.Errorf("Wrong match. an empty text is allowed for a media only message, got: %v", err)
	}
	if err := validateMMSText(strings.Repeat("가", mmsTextMaxLength)); err != nil {
		t.Errorf("Wrong match. the limit counts characters, not bytes, got: %v", err)
	}
	assertVoipbinError(t, validateMMSText(strings.Repeat("a", mmsTextMaxLength+1)), cerrors.StatusInvalidArgument, "TEXT_TOO_LONG")
}

func Test_validateMediaIDs(t *testing.T) {
	tooMany := []uuid.UUID{}
	for i := 0; i < mmsMediaMaxCount+1; i++ {
		tooMany = append(tooMany, uuid.Must(uuid.NewV4()))
	}

	tests := []struct {
		name string

		ids []uuid.UUID

		expectReason string
	}{
		{"empty list", []uuid.UUID{}, "MEDIA_REQUIRED"},
		{"nil list", nil, "MEDIA_REQUIRED"},
		{"too many", tooMany, "MEDIA_TOO_MANY"},
		{"nil id", []uuid.UUID{testMediaID1, uuid.Nil}, "INVALID_MEDIA_ID"},
		{"duplicate id", []uuid.UUID{testMediaID1, testMediaID1}, "MEDIA_DUPLICATE"},
		{"max count is allowed", tooMany[:mmsMediaMaxCount], ""},
		{"one id", []uuid.UUID{testMediaID1}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMediaIDs(tt.ids)
			if tt.expectReason == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			assertVoipbinError(t, err, cerrors.StatusInvalidArgument, tt.expectReason)
		})
	}
}

func Test_validateMedias(t *testing.T) {
	deleted := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	notFound := cerrors.NotFound(commonoutline.ServiceNameStorageManager, "FILE_NOT_FOUND", "not found")

	tests := []struct {
		name string

		ids       []uuid.UUID
		responses map[uuid.UUID]*smfile.File
		errs      map[uuid.UUID]error

		expectStatus cerrors.Status
		expectReason string
	}{
		{
			name: "ok",
			ids:  []uuid.UUID{testMediaID1, testMediaID2},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.PNG", 800*1024),
				testMediaID2: testMediaFile(testMediaID2, testCustomerID, "b.pdf", 50*1024),
			},
		},
		{
			name:         "not found is typed not found by the storage",
			ids:          []uuid.UUID{testMediaID1},
			errs:         map[uuid.UUID]error{testMediaID1: notFound},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_NOT_FOUND",
		},
		{
			name: "deleted file",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: func() *smfile.File {
					f := testMediaFile(testMediaID1, testCustomerID, "a.png", 10)
					f.TMDelete = &deleted
					return f
				}(),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_NOT_FOUND",
		},
		{
			name: "file of another customer gives the same answer as a missing file",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testOtherID, "a.png", 10),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_NOT_FOUND",
		},
		{
			name: "file not uploaded as media",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: func() *smfile.File {
					f := testMediaFile(testMediaID1, testCustomerID, "a.png", 10)
					f.Type = smfile.TypeRAG
					return f
				}(),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_FILE_TYPE_INVALID",
		},
		{
			name: "unsupported extension",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.exe", 10),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TYPE_UNSUPPORTED",
		},
		{
			name: "no extension",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "noextension", 10),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TYPE_UNSUPPORTED",
		},
		{
			name: "image over its limit",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.jpg", mmsMediaMaxSizeImage+1),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TOO_LARGE",
		},
		{
			name: "image at its limit is accepted",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.jpg", mmsMediaMaxSizeImage),
			},
		},
		{
			name: "audio over its smaller limit",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.mp3", mmsMediaMaxSizeOther+1),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TOO_LARGE",
		},
		{
			name: "total exactly at the limit is accepted",
			ids:  []uuid.UUID{testMediaID1, testMediaID2},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.png", mmsMediaMaxTotalSize-100),
				testMediaID2: testMediaFile(testMediaID2, testCustomerID, "b.mp3", 100),
			},
		},
		{
			name: "total one byte over the limit",
			ids:  []uuid.UUID{testMediaID1, testMediaID2},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.png", mmsMediaMaxTotalSize-100),
				testMediaID2: testMediaFile(testMediaID2, testCustomerID, "b.mp3", 101),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TOO_LARGE",
		},
		{
			name: "an empty file is not judged here, the provider decides",
			ids:  []uuid.UUID{testMediaID1},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.png", 0),
			},
		},
		{
			name: "total over the limit",
			ids:  []uuid.UUID{testMediaID1, testMediaID2},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.png", 500*1024),
				testMediaID2: testMediaFile(testMediaID2, testCustomerID, "b.png", 500*1024),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TOO_LARGE",
		},
		{
			name: "the error of the lowest index wins",
			ids:  []uuid.UUID{testMediaID1, testMediaID2},
			responses: map[uuid.UUID]*smfile.File{
				testMediaID1: testMediaFile(testMediaID1, testCustomerID, "a.exe", 10),
				testMediaID2: testMediaFile(testMediaID2, testOtherID, "b.png", 10),
			},
			expectStatus: cerrors.StatusInvalidArgument,
			expectReason: "MEDIA_TYPE_UNSUPPORTED",
		},
		{
			name:         "storage outage is returned unchanged",
			ids:          []uuid.UUID{testMediaID1},
			errs:         map[uuid.UUID]error{testMediaID1: cerrors.Unavailable(commonoutline.ServiceNameStorageManager, "DOWN", "down")},
			expectStatus: cerrors.StatusUnavailable,
			expectReason: "DOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &messageHandler{reqHandler: mockReq}
			ctx := context.Background()

			for _, id := range tt.ids {
				mockReq.EXPECT().StorageV1FileGet(gomock.Any(), id).Return(tt.responses[id], tt.errs[id])
			}

			err := h.validateMedias(ctx, testCustomerID, tt.ids)
			if tt.expectReason == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			assertVoipbinError(t, err, tt.expectStatus, tt.expectReason)
		})
	}
}

func Test_validateMedias_listErrorsBeforeStorage(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// no expectation on the request handler: a bad list must not reach the storage.
	h := &messageHandler{reqHandler: requesthandler.NewMockRequestHandler(mc)}

	assertVoipbinError(t, h.validateMedias(context.Background(), testCustomerID, []uuid.UUID{testMediaID1, testMediaID1}), cerrors.StatusInvalidArgument, "MEDIA_DUPLICATE")
}

func Test_validateMedias_deadline(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &messageHandler{reqHandler: mockReq}

	ctx, cancel := context.WithCancel(context.Background())
	mockReq.EXPECT().StorageV1FileGet(gomock.Any(), testMediaID1).DoAndReturn(func(_ context.Context, _ uuid.UUID) (*smfile.File, error) {
		cancel()
		return nil, context.Canceled
	})

	assertVoipbinError(t, h.validateMedias(ctx, testCustomerID, []uuid.UUID{testMediaID1}), cerrors.StatusUnavailable, "MEDIA_CHECK_TIMEOUT")
}

func Test_refreshMediaURLs(t *testing.T) {
	t.Run("urls come back in the order of the ids", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &messageHandler{reqHandler: mockReq}

		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return("https://example.invalid/1", nil)
		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID2).Return("https://example.invalid/2", nil)

		res, err := h.refreshMediaURLs(context.Background(), []uuid.UUID{testMediaID1, testMediaID2})
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if len(res) != 2 || res[0] != "https://example.invalid/1" || res[1] != "https://example.invalid/2" {
			t.Errorf("Wrong match. got: %v", res)
		}
	})

	t.Run("a refresh error is returned unchanged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &messageHandler{reqHandler: mockReq}

		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return("", cerrors.Unavailable(commonoutline.ServiceNameStorageManager, "SIGNING_NOT_CONFIGURED", "no key"))

		_, err := h.refreshMediaURLs(context.Background(), []uuid.UUID{testMediaID1})
		assertVoipbinError(t, err, cerrors.StatusUnavailable, "SIGNING_NOT_CONFIGURED")
	})

	t.Run("an empty url is an error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &messageHandler{reqHandler: mockReq}

		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).Return("", nil)

		_, err := h.refreshMediaURLs(context.Background(), []uuid.UUID{testMediaID1})
		assertVoipbinError(t, err, cerrors.StatusUnavailable, "MEDIA_URL_UNAVAILABLE")
	})

	t.Run("deadline", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &messageHandler{reqHandler: mockReq}

		ctx, cancel := context.WithCancel(context.Background())
		mockReq.EXPECT().StorageV1FileDownloadURIRefresh(gomock.Any(), testMediaID1).DoAndReturn(func(_ context.Context, _ uuid.UUID) (string, error) {
			cancel()
			return "", context.Canceled
		})

		_, err := h.refreshMediaURLs(ctx, []uuid.UUID{testMediaID1})
		assertVoipbinError(t, err, cerrors.StatusUnavailable, "MEDIA_CHECK_TIMEOUT")
	})
}

// Test_validateMedias_concurrency checks that at most 10 storage calls are in flight, and more than one.
func Test_validateMedias_concurrency(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &messageHandler{reqHandler: mockReq}

	var inFlight, peak int32
	ids := []uuid.UUID{}
	for i := 0; i < 10; i++ {
		ids = append(ids, uuid.Must(uuid.NewV4()))
	}
	mockReq.EXPECT().StorageV1FileGet(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, id uuid.UUID) (*smfile.File, error) {
		cur := atomic.AddInt32(&inFlight, 1)
		for {
			p := atomic.LoadInt32(&peak)
			if cur <= p || atomic.CompareAndSwapInt32(&peak, p, cur) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		atomic.AddInt32(&inFlight, -1)

		return testMediaFile(id, testCustomerID, "a.png", 10), nil
	}).Times(10)

	if err := h.validateMedias(context.Background(), testCustomerID, ids); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if peak < 2 || peak > 10 {
		t.Errorf("Wrong concurrency. expect between 2 and 10 in flight, got: %d", peak)
	}
}
