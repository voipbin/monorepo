package streaminghandler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

// writeFakeServiceAccount writes a service account file with a throwaway key generated at test
// time, so no key material is stored in the repository. The client libraries read it lazily and
// the tests never make a request, so nothing leaves the machine.
func writeFakeServiceAccount(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("could not generate the key. err: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("could not marshal the key. err: %v", err)
	}
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})

	content, err := json.Marshal(map[string]string{
		"type":           "service_account",
		"project_id":     "test-project",
		"private_key_id": "test-key-id",
		"private_key":    string(pemKey),
		"client_email":   "tester@test-project.iam.gserviceaccount.com",
		"client_id":      "1",
		"token_uri":      "https://oauth2.example.invalid/token",
	})
	if err != nil {
		t.Fatalf("could not marshal the account. err: %v", err)
	}

	path := filepath.Join(t.TempDir(), "account.json")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("could not write the account file. err: %v", err)
	}

	return path
}

func Test_NewStreamingHandler_providers(t *testing.T) {
	tests := []struct {
		name string

		gcpCredentials bool
		accessKey      string
		secretKey      string

		expectDisabled bool
		expectGCP      bool
		expectAWS      bool
	}{
		{"gcp only", true, "", "", false, true, false},
		{"aws only", false, "test_access_key", "test_secret_key", false, false, true},
		{"both", true, "test_access_key", "test_secret_key", false, true, true},
		{"none degrades to the disabled handler", false, "", "", true, false, false},
		{"aws access key without secret is ignored", false, "test_access_key", "", true, false, false},
		{"aws secret without access key is ignored", false, "", "test_secret_key", true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			path := filepath.Join(t.TempDir(), "missing.json")
			if tt.gcpCredentials {
				path = writeFakeServiceAccount(t)
			}
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", path)

			h := NewStreamingHandler(
				requesthandler.NewMockRequestHandler(mc),
				notifyhandler.NewMockNotifyHandler(mc),
				transcripthandler.NewMockTranscriptHandler(mc),
				tt.accessKey,
				tt.secretKey,
			)

			sh, ok := h.(*streamingHandler)
			if tt.expectDisabled {
				if ok {
					t.Fatalf("expected the disabled handler, got a streaming handler")
				}
				return
			}
			if !ok {
				t.Fatalf("expected a streaming handler, got %T", h)
			}
			if sh.mapStreaming == nil {
				t.Errorf("mapStreaming was not initialized")
			}
			if (sh.gcpClient != nil) != tt.expectGCP {
				t.Errorf("gcp client presence mismatch. expect: %v", tt.expectGCP)
			}
			if (sh.awsClient != nil) != tt.expectAWS {
				t.Errorf("aws client presence mismatch. expect: %v", tt.expectAWS)
			}
		})
	}
}
