package filehandler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

func Test_getFilename(t *testing.T) {

	type test struct {
		name string

		target string

		expectRes string
	}

	tests := []test{
		{
			"recording file",

			"recording/call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",

			"call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := getFilename(tt.target)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_filenameHash(t *testing.T) {
	type test struct {
		name string

		filenames []string

		expectRes string
	}

	tests := []test{
		{
			"recording file",

			[]string{
				"recording/call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_in.wav",
				"recording/call_e825e4c9-e5dc-4d21-8635-4b4a3fed5c98_2023-01-05T08:22:51Z_out.wav",
			},

			"tmp/0c44d476cdf0b43d377c82044155c14b4aba49bb.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := createZipFilepathHash(tt.filenames)
			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

// Test_NewFileHandler_credentials covers the boot-time credential handling. A missing
// GOOGLE_APPLICATION_CREDENTIALS must degrade gracefully instead of killing the service:
// NewFileHandler returns a usable FileHandler with no private key, so every capability
// that does not need a signed URL keeps working and only the signing paths report
// SIGNING_NOT_CONFIGURED. An explicitly configured but unusable path stays fatal --
// absence of the variable is an intentional keyless deployment, a broken path is
// misconfiguration and must be loud.
func Test_NewFileHandler_credentials(t *testing.T) {

	tests := []struct {
		name string

		credPathUnset   bool
		credPathMissing bool

		expectErr        bool
		expectPrivateKey bool
	}{
		{
			name: "credential path is not set",

			credPathUnset: true,

			expectErr:        false,
			expectPrivateKey: false,
		},
		{
			name: "credential path points to a missing file",

			credPathMissing: true,

			expectErr:        true,
			expectPrivateKey: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origCredPath, hadCredPath := os.LookupEnv("GOOGLE_APPLICATION_CREDENTIALS")
			defer func() {
				if hadCredPath {
					_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", origCredPath)
					return
				}
				_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
			}()

			if tt.credPathUnset {
				_ = os.Unsetenv("GOOGLE_APPLICATION_CREDENTIALS")
			}
			if tt.credPathMissing {
				_ = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "does-not-exist.json"))
			}

			// Keep the test hermetic: without this the storage client would run the full
			// application-default-credentials lookup, so the result would depend on
			// whatever ambient credentials (or metadata server) the machine happens to
			// have. The emulator host puts the client in no-auth mode. The credential
			// fallback that covers the same situation in production is exercised by
			// Test_newStorageClient.
			t.Setenv("STORAGE_EMULATOR_HOST", "http://127.0.0.1:1")

			res, err := NewFileHandler(nil, nil, nil, "test-project", "test-bucket-media", "test-bucket-tmp")
			if (err != nil) != tt.expectErr {
				t.Fatalf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr {
				if res != nil {
					t.Errorf("Wrong match. expect: nil FileHandler, got: %v", res)
				}
				return
			}

			if res == nil {
				t.Fatalf("Wrong match. expect: non-nil FileHandler, got: nil")
			}

			h, ok := res.(*fileHandler)
			if !ok {
				t.Fatalf("Wrong match. expect: *fileHandler, got: %T", res)
			}
			if (len(h.privateKey) > 0) != tt.expectPrivateKey {
				t.Errorf("Wrong match. expect private key: %v, got: %v", tt.expectPrivateKey, h.privateKey)
			}
			if !tt.expectPrivateKey && h.accessID != "" {
				t.Errorf("Wrong match. expect: empty access id, got: %s", h.accessID)
			}
			if h.client == nil {
				t.Errorf("Wrong match. expect: non-nil storage client, got: nil")
			}
		})
	}
}

// Test_newStorageClient verifies the unauthenticated fallback and its gate. Losing the
// credential on a keyless deployment must not abort startup: GOOGLE_APPLICATION_CREDENTIALS
// is also the storage client's application-default-credentials source, so a keyless
// deployment fails the authenticated constructor and would otherwise crash-loop. When a
// credential IS configured, a construction failure is a real fault and must stay fatal
// rather than silently running unauthenticated forever.
func Test_newStorageClient(t *testing.T) {

	tests := []struct {
		name string

		newClient func(context.Context, ...option.ClientOption) (*storage.Client, error)
		keyless   bool

		expectCalls    int
		expectFallback bool
		expectErr      bool
	}{
		{
			name: "credentials available",

			newClient: func(_ context.Context, _ ...option.ClientOption) (*storage.Client, error) {
				return &storage.Client{}, nil
			},
			keyless: true,

			expectCalls:    1,
			expectFallback: false,
			expectErr:      false,
		},
		{
			name: "keyless, fallback succeeds",

			newClient: func() func(context.Context, ...option.ClientOption) (*storage.Client, error) {
				calls := 0
				return func(_ context.Context, _ ...option.ClientOption) (*storage.Client, error) {
					calls++
					if calls == 1 {
						return nil, fmt.Errorf("could not find default credentials")
					}
					return &storage.Client{}, nil
				}
			}(),
			keyless: true,

			expectCalls:    2,
			expectFallback: true,
			expectErr:      false,
		},
		{
			name: "keyless, fallback fails",

			newClient: func(_ context.Context, _ ...option.ClientOption) (*storage.Client, error) {
				return nil, fmt.Errorf("dial error")
			},
			keyless: true,

			expectCalls:    2,
			expectFallback: true,
			expectErr:      true,
		},
		{
			name: "credential configured, no fallback",

			newClient: func(_ context.Context, _ ...option.ClientOption) (*storage.Client, error) {
				return nil, fmt.Errorf("could not find default credentials")
			},
			keyless: false,

			expectCalls:    1,
			expectFallback: false,
			expectErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			gotFallback := false
			newClient := func(ctx context.Context, opts ...option.ClientOption) (*storage.Client, error) {
				calls++
				if len(opts) > 0 {
					gotFallback = true
				}
				return tt.newClient(ctx, opts...)
			}

			res, err := newStorageClient(context.Background(), newClient, tt.keyless)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if !tt.expectErr && res == nil {
				t.Errorf("Wrong match. expect: non-nil client, got: nil")
			}
			if tt.expectErr && res != nil {
				t.Errorf("Wrong match. expect: nil client, got: %v", res)
			}
			if calls != tt.expectCalls {
				t.Errorf("Wrong match. expect: %d calls, got: %d", tt.expectCalls, calls)
			}
			if gotFallback != tt.expectFallback {
				t.Errorf("Wrong match. expect fallback: %v, got: %v", tt.expectFallback, gotFallback)
			}
		})
	}
}
