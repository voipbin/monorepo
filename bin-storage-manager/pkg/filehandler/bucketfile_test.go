package filehandler

import (
	stderrors "errors"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
)

// Test_bucketfileGenerateDownloadURI_noPrivateKey verifies that the signing path
// reports a structured, typed error when no signing credential was configured, so
// API callers see a 503/SIGNING_NOT_CONFIGURED instead of an opaque 500.
func Test_bucketfileGenerateDownloadURI_noPrivateKey(t *testing.T) {

	tests := []struct {
		name string

		accessID   string
		privateKey []byte

		bucketName string
		filepath   string
		filename   string

		expectStatus cerrors.Status
		expectReason string
		expectDomain string
	}{
		{
			name: "no signing credential configured",

			accessID:   "",
			privateKey: nil,

			bucketName: "test-bucket-media",
			filepath:   "bin/8f8e5a5a-6cf3-11f0-9a1c-9f2c2b0a3d01",
			filename:   "test.wav",

			expectStatus: cerrors.StatusUnavailable,
			expectReason: errSigningNotConfiguredReason,
			expectDomain: string(commonoutline.ServiceNameStorageManager),
		},
		{
			name: "access id present but no private key",

			accessID:   "test@test-project.iam.gserviceaccount.com",
			privateKey: nil,

			bucketName: "test-bucket-media",
			filepath:   "bin/8f8e5a5a-6cf3-11f0-9a1c-9f2c2b0a3d02",
			filename:   "",

			expectStatus: cerrors.StatusUnavailable,
			expectReason: errSigningNotConfiguredReason,
			expectDomain: string(commonoutline.ServiceNameStorageManager),
		},
		{
			// google.JWTConfigFromJSON does not reject an empty private_key, so a
			// malformed secret produces a non-nil zero-length key. It must still be
			// treated as "not configured" rather than reaching the signer.
			name: "private key present but empty",

			accessID:   "test@test-project.iam.gserviceaccount.com",
			privateKey: []byte{},

			bucketName: "test-bucket-media",
			filepath:   "bin/8f8e5a5a-6cf3-11f0-9a1c-9f2c2b0a3d03",
			filename:   "test.wav",

			expectStatus: cerrors.StatusUnavailable,
			expectReason: errSigningNotConfiguredReason,
			expectDomain: string(commonoutline.ServiceNameStorageManager),
		},
		{
			// same for an empty client_email: signing cannot work without it
			name: "private key present but no access id",

			accessID:   "",
			privateKey: []byte("-----BEGIN RSA PRIVATE KEY-----\nnot-a-real-key\n-----END RSA PRIVATE KEY-----\n"),

			bucketName: "test-bucket-media",
			filepath:   "bin/8f8e5a5a-6cf3-11f0-9a1c-9f2c2b0a3d04",
			filename:   "test.wav",

			expectStatus: cerrors.StatusUnavailable,
			expectReason: errSigningNotConfiguredReason,
			expectDomain: string(commonoutline.ServiceNameStorageManager),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &fileHandler{
				accessID:   tt.accessID,
				privateKey: tt.privateKey,
			}

			res, err := h.bucketfileGenerateDownloadURI(tt.bucketName, tt.filepath, time.Now().UTC().Add(time.Hour), tt.filename)
			if err == nil {
				t.Fatalf("Wrong match. expect: non-nil error, got: nil error")
			}
			if res != "" {
				t.Errorf("Wrong match. expect: empty uri, got: %s", res)
			}

			var ve *cerrors.VoipbinError
			if !stderrors.As(err, &ve) {
				t.Fatalf("Wrong match. expect: *cerrors.VoipbinError, got: %T", err)
			}
			if ve.Status != tt.expectStatus {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectStatus, ve.Status)
			}
			if ve.Reason != tt.expectReason {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectReason, ve.Reason)
			}
			if ve.Domain != tt.expectDomain {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectDomain, ve.Domain)
			}
		})
	}
}
