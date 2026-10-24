package filehandler

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/dbhandler"

	stderrors "errors"
)

// checkSignedURL verifies the parts of a V4 signed URL that matter to a caller:
// the target object, the signing algorithm, the signer and the validity window.
func checkSignedURL(t *testing.T, raw string, bucket string, object string, accessID string, validity time.Duration) url.Values {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("Wrong match. expect: a url, got: %q (%v)", raw, err)
	}
	// the host is the emulator's while STORAGE_EMULATOR_HOST is set, so only the object path is checked.
	if u.Path != "/"+bucket+"/"+object {
		t.Errorf("Wrong match. expect: path /%s/%s, got: %s", bucket, object, u.Path)
	}

	q := u.Query()
	if q.Get("X-Goog-Algorithm") != "GOOG4-RSA-SHA256" {
		t.Errorf("Wrong match. got algorithm: %q", q.Get("X-Goog-Algorithm"))
	}
	if !strings.HasPrefix(q.Get("X-Goog-Credential"), accessID+"/") {
		t.Errorf("Wrong match. the credential must start with the signer %q, got: %q", accessID, q.Get("X-Goog-Credential"))
	}
	if q.Get("X-Goog-Signature") == "" {
		t.Errorf("Wrong match. expect: a signature")
	}

	// the signing clock starts before this check, so the window is slightly short of the full validity; the margin is generous for slow CI hosts.
	expires, err := strconv.Atoi(q.Get("X-Goog-Expires"))
	if err != nil || expires > int(validity.Seconds()) || expires < int(validity.Seconds())-120 {
		t.Errorf("Wrong match. expect: about %d seconds, got: %q (%v)", int(validity.Seconds()), q.Get("X-Goog-Expires"), err)
	}

	return q
}

func Test_DownloadURIRefresh_paths(t *testing.T) {
	ctx := context.Background()

	id := uuid.FromStringOrNil("2b3c4d5e-0000-0000-0000-000000000001")
	stored := &file.File{
		Identity:   commonidentity.Identity{ID: id},
		BucketName: "media",
		Filepath:   "bin/obj1",
		Filename:   "report one.wav",
	}
	tmExpire := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)

	tests := []struct {
		name string

		signing bool

		dbGetErr    error
		dbUpdateErr error

		expectExpire bool // whether the expiry is asked from the clock
		expectUpdate bool
		expectErr    error
		expectTyped  string // expected reason of a typed error
	}{
		{
			name:         "normal",
			signing:      true,
			expectExpire: true,
			expectUpdate: true,
		},
		{
			name:      "the file does not exist",
			signing:   true,
			dbGetErr:  dbhandler.ErrNotFound,
			expectErr: dbhandler.ErrNotFound,
		},
		{
			name:         "no signing key yields the typed unavailable error and no db update",
			signing:      false,
			expectExpire: true,
			expectTyped:  errSigningNotConfiguredReason,
		},
		{
			name:         "the db update fails",
			signing:      true,
			dbUpdateErr:  errTest,
			expectExpire: true,
			expectUpdate: true,
			expectErr:    errTest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := &fileHandler{db: mockDB, utilHandler: mockUtil}
			accessID := ""
			if tt.signing {
				var key []byte
				accessID, key = newTestSigningKey(t)
				h.accessID, h.privateKey = accessID, key
			}

			if tt.dbGetErr != nil {
				mockDB.EXPECT().FileGet(ctx, id).Return(nil, tt.dbGetErr)
			} else {
				mockDB.EXPECT().FileGet(ctx, id).Return(stored, nil)
			}
			if tt.expectExpire {
				mockUtil.EXPECT().TimeNowAdd(downloadURLExpiration).Return(&tmExpire)
			}

			var updated map[file.Field]any
			if tt.expectUpdate {
				mockDB.EXPECT().FileUpdate(ctx, id, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, fields map[file.Field]any) error {
					updated = fields
					return tt.dbUpdateErr
				})
			}

			res, err := h.DownloadURIRefresh(ctx, id)

			if tt.expectTyped != "" {
				var ve *cerrors.VoipbinError
				if !stderrors.As(err, &ve) || ve.Reason != tt.expectTyped || res != "" {
					t.Errorf("Wrong match. expect: %q/typed %s, got: %q/%v", "", tt.expectTyped, res, err)
				}
				return
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Fatalf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr != nil {
				if res != "" {
					t.Errorf("Wrong match. expect: empty uri, got: %q", res)
				}
				return
			}

			q := checkSignedURL(t, res, "media", "bin/obj1", accessID, downloadURLExpiration)
			if got := q.Get("response-content-disposition"); got != `attachment; filename="report one.wav"` {
				t.Errorf("Wrong match. got disposition: %q", got)
			}

			// the stored fields are exactly the returned uri and the clock's expiry.
			if len(updated) != 2 || updated[file.FieldURIDownload] != res || updated[file.FieldTMDownloadExpire] != &tmExpire {
				t.Errorf("Wrong match. got update fields: %v", updated)
			}
		})
	}
}

func Test_DownloadURIGet_paths(t *testing.T) {
	ctx := context.Background()

	t.Run("normal", func(t *testing.T) {
		fake := newFakeGCS(t, map[string]int64{"tmp/zip/a.zip": 5})
		accessID, key := newTestSigningKey(t)
		h := &fileHandler{client: fake.client(t), bucketTmp: "tmp", accessID: accessID, privateKey: key}

		bucketURI, downloadURI, err := h.DownloadURIGet(ctx, "tmp", "zip/a.zip", 24*time.Hour)
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		if bucketURI != "https://storage.test/download/tmp/zip/a.zip" {
			t.Errorf("Wrong match. got bucket uri: %q", bucketURI)
		}

		q := checkSignedURL(t, downloadURI, "tmp", "zip/a.zip", accessID, 24*time.Hour)
		if q.Has("response-content-disposition") {
			t.Errorf("Wrong match. a zip download carries no forced filename, got: %q", q.Get("response-content-disposition"))
		}
	})

	t.Run("the object attributes cannot be read", func(t *testing.T) {
		fake := newFakeGCS(t, map[string]int64{})
		accessID, key := newTestSigningKey(t)
		h := &fileHandler{client: fake.client(t), bucketTmp: "tmp", accessID: accessID, privateKey: key}

		bucketURI, downloadURI, err := h.DownloadURIGet(ctx, "tmp", "zip/missing.zip", time.Hour)
		if err == nil || bucketURI != "" || downloadURI != "" {
			t.Errorf("Wrong match. expect: empty/error, got: %q/%q/%v", bucketURI, downloadURI, err)
		}
	})

	t.Run("no signing key yields the typed unavailable error", func(t *testing.T) {
		fake := newFakeGCS(t, map[string]int64{"tmp/zip/a.zip": 5})
		h := &fileHandler{client: fake.client(t), bucketTmp: "tmp"}

		_, downloadURI, err := h.DownloadURIGet(ctx, "tmp", "zip/a.zip", time.Hour)

		var ve *cerrors.VoipbinError
		if !stderrors.As(err, &ve) || ve.Reason != errSigningNotConfiguredReason || downloadURI != "" {
			t.Errorf("Wrong match. expect: typed %s, got: %q/%v", errSigningNotConfiguredReason, downloadURI, err)
		}
	})
}
