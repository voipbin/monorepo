package transcripthandler

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-transcribe-manager/pkg/dbhandler"

	gomock "go.uber.org/mock/gomock"
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

func Test_NewTranscriptHandler_withCredentials(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", writeFakeServiceAccount(t))

	h := NewTranscriptHandler(requesthandler.NewMockRequestHandler(mc), dbhandler.NewMockDBHandler(mc), notifyhandler.NewMockNotifyHandler(mc))
	if h == nil {
		t.Fatalf("NewTranscriptHandler() returned nil with usable credentials")
	}

	th, ok := h.(*transcriptHandler)
	if !ok {
		t.Fatalf("NewTranscriptHandler() returned an unexpected type: %T", h)
	}
	if th.clientSpeech == nil || th.reqHandler == nil || th.db == nil || th.notifyHandler == nil || th.utilHandler == nil {
		t.Errorf("NewTranscriptHandler() left a dependency unset: %+v", th)
	}
}

func Test_NewTranscriptHandler_missingCredentials(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filepath.Join(t.TempDir(), "missing.json"))

	h := NewTranscriptHandler(requesthandler.NewMockRequestHandler(mc), dbhandler.NewMockDBHandler(mc), notifyhandler.NewMockNotifyHandler(mc))
	if h != nil {
		t.Errorf("NewTranscriptHandler() expected nil when the speech client cannot be created. got: %v", h)
	}
}
