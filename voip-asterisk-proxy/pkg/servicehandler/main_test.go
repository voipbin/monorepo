package servicehandler

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newTestServiceAccountJSON builds a syntactically valid GCP service-account key JSON with a
// freshly generated RSA private key. It is only used to exercise the option.WithAuthCredentialsJSON
// parsing path in NewServiceHandler; it never connects to GCS.
func newTestServiceAccountJSON(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("could not generate rsa key: %v", err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("could not marshal rsa key: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8})

	sa := map[string]string{
		"type":                        "service_account",
		"project_id":                  "test-project",
		"private_key_id":              "testkeyid",
		"private_key":                 string(pemBytes),
		"client_email":                "test@test-project.iam.gserviceaccount.com",
		"client_id":                   "123456789",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url":        "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com",
	}
	b, err := json.Marshal(sa)
	if err != nil {
		t.Fatalf("could not marshal service account json: %v", err)
	}
	return string(b)
}

// Test_NewServiceHandler_emptyCredential_returnsNonNilHandler asserts that when no credential is
// provided AND the ADC-based client cannot be built, NewServiceHandler still returns a non-nil
// handler (never a nil interface). This is the fail-safe that prevents the RPC dispatcher from
// panicking on a nil-interface method call (the root cause of VOIP-1526).
func Test_NewServiceHandler_emptyCredential_returnsNonNilHandler(t *testing.T) {
	h := NewServiceHandler("", "bucket", "/var/spool/asterisk/recording", "recording")
	if h == nil {
		t.Fatal("NewServiceHandler returned a nil interface; it must always return a non-nil handler so the RPC dispatcher does not panic")
	}
}

// Test_RecordingFileMove_nilClient_returnsErrorNoPanic asserts that RecordingFileMove on a handler
// whose GCS client is nil returns a clear error instead of panicking. A real, readable source file
// is placed in recordingAsteriskDirectory so that WITHOUT the guard, execution would proceed past
// os.Open and nil-dereference at h.client.Bucket(...). The guard short-circuits before that.
//
// NEGATIVE CONTROL: removing the `if h.client == nil` guard in RecordingFileMove makes this test
// panic (nil pointer dereference at h.client.Bucket), because os.Open now succeeds on the real
// file and execution reaches the client. This is what makes the assertion actually protect the
// guard rather than pass on an unrelated os.Open error.
func Test_RecordingFileMove_nilClient_returnsErrorNoPanic(t *testing.T) {
	dir := t.TempDir()
	filename := "call_x_in.wav"
	if err := os.WriteFile(filepath.Join(dir, filename), []byte("fake wav"), 0o600); err != nil {
		t.Fatalf("could not write test source file: %v", err)
	}

	h := &serviceHandler{
		client:                     nil,
		recordingBucketName:        "bucket",
		recordingAsteriskDirectory: dir,
		recordingBucketDirectory:   "recording",
	}

	err := h.RecordingFileMove(context.Background(), []string{filename})
	if err == nil {
		t.Fatal("RecordingFileMove with a nil client must return an error, got nil")
	}
	// Assert it is the GUARD error, not an incidental os.Open failure, so the test genuinely
	// exercises the nil-client guard.
	if !strings.Contains(err.Error(), "GCS client is not configured") {
		t.Fatalf("expected the nil-client guard error, got: %v", err)
	}
}

// Test_NewServiceHandler_validCredentialJSON_buildsClient asserts that a well-formed
// service-account JSON drives the option.WithAuthCredentialsJSON path and yields a usable, non-nil
// GCS client (object construction only; no network call is made).
func Test_NewServiceHandler_validCredentialJSON_buildsClient(t *testing.T) {
	saJSON := newTestServiceAccountJSON(t)

	h := NewServiceHandler(saJSON, "bucket", "/var/spool/asterisk/recording", "recording")
	if h == nil {
		t.Fatal("NewServiceHandler returned a nil interface")
	}
	sh, ok := h.(*serviceHandler)
	if !ok {
		t.Fatalf("unexpected handler type: %T", h)
	}
	if sh.client == nil {
		t.Fatal("expected a non-nil GCS client when a valid credential JSON is provided")
	}
}
