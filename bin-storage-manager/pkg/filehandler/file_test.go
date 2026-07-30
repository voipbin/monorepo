package filehandler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	reflect "reflect"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"cloud.google.com/go/storage"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/models/file"
	"monorepo/bin-storage-manager/pkg/accounthandler"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseFile *file.File
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("5f67906c-1531-11ef-acd7-cf9b57d65bcc"),

			responseFile: &file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5f67906c-1531-11ef-acd7-cf9b57d65bcc"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &fileHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().FileGet(ctx, tt.id).Return(tt.responseFile, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFile) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFile, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[file.Field]any

		responseFiles []*file.File
	}{
		{
			name: "normal",

			token: "2024-05-16T03:22:17.995000Z",
			size:  10,
			filters: map[file.Field]any{
				file.FieldCustomerID: uuid.FromStringOrNil("ba5d2ed2-1531-11ef-960b-cfcd7e5676b9"),
			},

			responseFiles: []*file.File{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e05c0c20-1531-11ef-8c1f-e79b24c34783"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &fileHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().FileList(ctx, tt.token, tt.size, tt.filters).Return(tt.responseFiles, nil)

			res, err := h.List(ctx, tt.token, tt.size, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseFiles) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFiles, res)
			}
		})
	}
}

// newFakeGCSServer returns an httptest server that speaks just enough of the GCS
// JSON API for fileHandler.Create: attrs lookup on the source object, a
// not-found on the destination object, the rewrite (copy) call, and the source
// delete. It is wired in through STORAGE_EMULATOR_HOST.
//
// The request shapes matched here (the /rewriteTo/ path, size-as-string, mediaLink)
// are the GCS JSON API wire format used by cloud.google.com/go/storage v1.61.3. If a
// storage library upgrade changes them, this helper is where the failure will land.
func newFakeGCSServer(t *testing.T, srcBucket string, srcFilepath string, dstBucket string, dstFilepath string, mediaLink string, size int64) *httptest.Server {
	t.Helper()

	objectJSON := func(bucket string, name string) string {
		return fmt.Sprintf(
			`{"kind":"storage#object","bucket":%q,"name":%q,"size":"%d","mediaLink":%q,"generation":"1","metageneration":"1"}`,
			bucket, name, size, mediaLink,
		)
	}

	// object paths are matched with their bucket so a move into the wrong bucket fails
	srcObjectPath := fmt.Sprintf("/b/%s/o/%s", srcBucket, srcFilepath)
	dstObjectPath := fmt.Sprintf("/b/%s/o/%s", dstBucket, dstFilepath)
	rewritePath := fmt.Sprintf("%s/rewriteTo/b/%s/o/%s", srcObjectPath, dstBucket, dstFilepath)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, rewritePath):
			_, _ = fmt.Fprintf(
				w,
				`{"kind":"storage#rewriteResponse","totalBytesRewritten":"%d","objectSize":"%d","done":true,"resource":%s}`,
				size, size, objectJSON(dstBucket, dstFilepath),
			)

		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)

		// the destination object must not exist yet, otherwise the move is refused
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, dstObjectPath):
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"Not Found"}}`)

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, srcObjectPath):
			_, _ = fmt.Fprint(w, objectJSON(srcBucket, srcFilepath))

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = fmt.Fprint(w, `{"error":{"code":404,"message":"Not Found"}}`)
		}
	}))
	t.Cleanup(srv.Close)

	return srv
}

// newTestSigningKey generates an RSA key and returns a service account access id plus
// the PEM-encoded PKCS1 private key, in the same shape google.JWTConfigFromJSON hands
// to the handler. It lets the signing-configured path be exercised offline.
func newTestSigningKey(t *testing.T) (string, []byte) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return "test@test-project.iam.gserviceaccount.com", res
}

// Test_Create_downloadURI covers both download-URI outcomes of Create.
//
// With signing configured the record carries a signed URI and an expiration. Without
// it, Create must still succeed: the GCS object has already been moved by the time the
// URI is generated, so returning the error would both break the primary write path and
// orphan the moved object. The record is persisted with an empty URIDownload and a nil
// TMDownloadExpire instead, and DownloadURIRefresh repopulates it once a usable signing
// key exists.
func Test_Create_downloadURI(t *testing.T) {

	tests := []struct {
		name string

		signingConfigured bool

		customerID    uuid.UUID
		ownerID       uuid.UUID
		referenceType file.ReferenceType
		referenceID   uuid.UUID
		fileType      file.Type
		fileName      string
		detail        string
		filename      string
		srcBucket     string
		srcFilepath   string

		bucketMedia string
		filesize    int64
		mediaLink   string

		responseUUID     uuid.UUID
		responseAccount  *account.Account
		responseTMExpire *time.Time
		responseFile     *file.File

		expectDownloadURI bool
	}{
		{
			name: "no signing credential configured",

			signingConfigured: false,

			customerID:    uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b01"),
			ownerID:       uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b02"),
			referenceType: file.ReferenceTypeNormal,
			referenceID:   uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b03"),
			fileType:      file.TypeTalk,
			fileName:      "test file",
			detail:        "test detail",
			filename:      "source.wav",
			srcBucket:     "test-bucket-tmp",
			srcFilepath:   "tmp/source.wav",

			bucketMedia: "test-bucket-media",
			filesize:    11,
			mediaLink:   "https://storage.test/media",

			responseUUID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b04"),
			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b05"),
			},
			responseFile: &file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b04"),
				},
				AccountID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b05"),
				Filesize:  11,
			},

			expectDownloadURI: false,
		},
		{
			name: "signing credential configured",

			signingConfigured: true,

			customerID:    uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b11"),
			ownerID:       uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b12"),
			referenceType: file.ReferenceTypeRecording,
			referenceID:   uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b13"),
			fileType:      file.TypeRecording,
			fileName:      "test file",
			detail:        "test detail",
			filename:      "source.wav",
			srcBucket:     "test-bucket-tmp",
			srcFilepath:   "tmp/source.wav",

			bucketMedia: "test-bucket-media",
			filesize:    11,
			mediaLink:   "https://storage.test/media",

			responseUUID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b14"),
			responseAccount: &account.Account{
				ID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b15"),
			},
			responseTMExpire: &testTMExpire,
			responseFile: &file.File{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b14"),
				},
				AccountID: uuid.FromStringOrNil("dd7f2a0c-6cf0-11f0-8a6a-2f0f2a4a1b15"),
				Filesize:  11,
			},

			expectDownloadURI: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)

			dstFilepath := fmt.Sprintf("%s/%s", bucketDirectoryBin, tt.responseUUID)
			srv := newFakeGCSServer(t, tt.srcBucket, tt.srcFilepath, tt.bucketMedia, dstFilepath, tt.mediaLink, tt.filesize)
			t.Setenv("STORAGE_EMULATOR_HOST", srv.URL)

			ctx := context.Background()
			client, errClient := storage.NewClient(ctx)
			if errClient != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", errClient)
			}
			defer func() {
				_ = client.Close()
			}()

			accessID := ""
			var privateKey []byte
			if tt.signingConfigured {
				accessID, privateKey = newTestSigningKey(t)
			}

			h := &fileHandler{
				utilHandler:    mockUtil,
				notifyHandler:  mockNotify,
				db:             mockDB,
				accountHandler: mockAccount,

				client:      client,
				bucketMedia: tt.bucketMedia,
				bucketTmp:   tt.srcBucket,

				accessID:   accessID,
				privateKey: privateKey,
			}

			mockAccount.EXPECT().ValidateFileInfoByCustomerID(ctx, tt.customerID, int64(1), tt.filesize).Return(tt.responseAccount, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			if tt.expectDownloadURI {
				mockUtil.EXPECT().TimeNowAdd(downloadURLExpiration).Return(tt.responseTMExpire)
			}

			var created *file.File
			mockDB.EXPECT().FileCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, f *file.File) error {
				created = f
				return nil
			})
			mockDB.EXPECT().FileGet(ctx, tt.responseUUID).Return(tt.responseFile, nil)
			mockNotify.EXPECT().PublishEvent(ctx, file.EventTypeFileCreated, tt.responseFile)
			mockAccount.EXPECT().IncreaseFileInfo(ctx, tt.responseFile.AccountID, int64(1), tt.responseFile.Filesize).Return(tt.responseAccount, nil)

			res, err := h.Create(ctx, tt.customerID, tt.ownerID, tt.referenceType, tt.referenceID, tt.fileType, tt.fileName, tt.detail, tt.filename, tt.srcBucket, tt.srcFilepath)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, tt.responseFile) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseFile, res)
			}

			if created == nil {
				t.Fatalf("Wrong match. expect: created file record, got: nil")
			}
			if tt.expectDownloadURI {
				if created.URIDownload == "" {
					t.Errorf("Wrong match. expect: non-empty uri_download, got: empty")
				}
				if created.TMDownloadExpire != tt.responseTMExpire {
					t.Errorf("Wrong match. expect: %v, got: %v", tt.responseTMExpire, created.TMDownloadExpire)
				}
			} else {
				if created.URIDownload != "" {
					t.Errorf("Wrong match. expect: empty uri_download, got: %s", created.URIDownload)
				}
				if created.TMDownloadExpire != nil {
					t.Errorf("Wrong match. expect: nil tm_download_expire, got: %v", created.TMDownloadExpire)
				}
			}
			if created.URIBucket != tt.mediaLink {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.mediaLink, created.URIBucket)
			}
			if created.Filepath != dstFilepath {
				t.Errorf("Wrong match. expect: %s, got: %s", dstFilepath, created.Filepath)
			}
			if created.Filesize != tt.filesize {
				t.Errorf("Wrong match. expect: %d, got: %d", tt.filesize, created.Filesize)
			}
		})
	}
}

// testTMExpire is the fixed download-URI expiration the mocked utilhandler returns.
var testTMExpire = time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC)
