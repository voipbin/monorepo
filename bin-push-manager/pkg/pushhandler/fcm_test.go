package pushhandler

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-push-manager/models/notification"
)

const fakeToken = "fake-device-token-AbCdEf-0123456789"

// fakeServiceAccountJSON returns a service account JSON with a freshly generated key. Every value is fake.
func fakeServiceAccountJSON(t *testing.T, projectID string) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	der := x509.MarshalPKCS1PrivateKey(key)
	pemKey := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der})

	m, err := json.Marshal(map[string]string{
		"type":           "service_account",
		"project_id":     projectID,
		"private_key_id": "fake-key-id",
		"private_key":    string(pemKey),
		"client_email":   "fake@example.invalid",
		"client_id":      "1234567890",
		"token_uri":      "https://oauth2.invalid/token",
	})
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	return m
}

type captured struct {
	path        string
	method      string
	contentType string
	body        map[string]any
	raw         string
}

// newFakeFCM starts a fake FCM endpoint that answers with the given status and body.
func newFakeFCM(t *testing.T, status int, response string, got *captured) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.path = r.URL.Path
		got.method = r.Method
		got.contentType = r.Header.Get("Content-Type")
		got.raw = string(raw)
		_ = json.Unmarshal(raw, &got.body)

		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)

	return srv
}

func Test_fcmSender_Send_requestShape(t *testing.T) {
	var got captured
	srv := newFakeFCM(t, 200, `{"name":"projects/p/messages/1"}`, &got)
	s := newFCMSender(srv.Client(), srv.URL, "demo-project")

	data := map[string]string{"call_id": "c1", "join_address": "sip:direct.abc@sip.example.invalid"}
	if err := s.Send(context.Background(), fakeToken, notification.KindCallIncoming, data); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	if got.method != http.MethodPost || got.path != "/v1/projects/demo-project/messages:send" {
		t.Errorf("Wrong match. got: %s %s", got.method, got.path)
	}
	if got.contentType != "application/json" {
		t.Errorf("Wrong match. expect: application/json, got: %s", got.contentType)
	}

	expect := `{"message":{"token":"` + fakeToken + `","data":{"call_id":"c1","join_address":"sip:direct.abc@sip.example.invalid","kind":"call_incoming"},"android":{"priority":"HIGH","ttl":"60s"}}}`
	if got.raw != expect {
		t.Errorf("Wrong match.\nexpect: %s\ngot: %s", expect, got.raw)
	}

	// the caller's map is not changed by adding the kind
	if _, ok := data[notification.DataKeyKind]; ok {
		t.Errorf("Wrong match. the caller's data must not get the kind key")
	}
}

func Test_fcmSender_Send_errors(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		response     string
		expectErr    error // matched with errors.Is when not nil
		expectAnyErr bool
	}{
		{"success", 200, `{}`, nil, false},
		{"unregistered", 404, `{"error":{"code":404,"status":"NOT_FOUND","details":[{"@type":"type.googleapis.com/google.firebase.fcm.v1.FcmError","errorCode":"UNREGISTERED"}]}}`, ErrUnregistered, true},
		{"invalid argument can be a payload error and is not unregistered", 400, `{"error":{"code":400,"status":"INVALID_ARGUMENT","details":[{"errorCode":"INVALID_ARGUMENT"}]}}`, nil, true},
		{"server error", 503, `{"error":{"code":503,"status":"UNAVAILABLE"}}`, nil, true},
		{"unparsable error body", 500, `not json`, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got captured
			srv := newFakeFCM(t, tt.status, tt.response, &got)
			s := newFCMSender(srv.Client(), srv.URL, "demo-project")

			err := s.Send(context.Background(), fakeToken, notification.KindCallCancelled, map[string]string{"cancel_key": "k"})
			if tt.expectAnyErr != (err != nil) {
				t.Fatalf("Wrong match. expect error: %v, got: %v", tt.expectAnyErr, err)
			}
			if tt.expectErr != nil && !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErr == nil && errors.Is(err, ErrUnregistered) {
				t.Errorf("Wrong match. must not be unregistered. got: %v", err)
			}
			if err != nil && strings.Contains(err.Error(), fakeToken) {
				t.Errorf("Wrong match. the token must not be in the error. got: %v", err)
			}
		})
	}
}

func Test_fcmSender_Send_transportError(t *testing.T) {
	var got captured
	srv := newFakeFCM(t, 200, `{}`, &got)
	endpoint := srv.URL
	client := srv.Client()
	srv.Close()

	s := newFCMSender(client, endpoint, "demo-project")
	err := s.Send(context.Background(), fakeToken, notification.KindCallIncoming, nil)
	if err == nil {
		t.Fatalf("Wrong match. expect: error, got: nil")
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Errorf("Wrong match. the token must not be in the error. got: %v", err)
	}
}

func Test_fcmSender_Send_canceledContext(t *testing.T) {
	var got captured
	srv := newFakeFCM(t, 200, `{}`, &got)
	s := newFCMSender(srv.Client(), srv.URL, "demo-project")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Send(ctx, fakeToken, notification.KindCallIncoming, nil); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

// The OAuth token refresh does not take the context of a send, so it has its own timeout. It must not be
// longer than the RPC limit of the notification request (8 seconds), or a call would be told "unknown" first.
func Test_tokenRefreshTimeout(t *testing.T) {
	if tokenRefreshTimeout <= 0 || tokenRefreshTimeout >= 8*time.Second {
		t.Errorf("Wrong match. expect: 0 < timeout < 8s, got: %v", tokenRefreshTimeout)
	}
}

func Test_newFCMSenderFromJSON(t *testing.T) {
	t.Run("valid service account uses its project id", func(t *testing.T) {
		res, err := newFCMSenderFromJSON(fakeServiceAccountJSON(t, "demo-project"))
		if err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		s, ok := res.(*fcmSender)
		if !ok || s.projectID != "demo-project" || s.endpoint != fcmEndpoint {
			t.Errorf("Wrong match. got: %#v", res)
		}
	})

	t.Run("no project id", func(t *testing.T) {
		if _, err := newFCMSenderFromJSON(fakeServiceAccountJSON(t, "")); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("malformed", func(t *testing.T) {
		_, err := newFCMSenderFromJSON([]byte(`{"private_key":"SECRET-VALUE-MUST-NOT-LEAK"`))
		if err == nil {
			t.Fatalf("Wrong match. expect: error, got: nil")
		}
		if strings.Contains(err.Error(), "SECRET-VALUE-MUST-NOT-LEAK") {
			t.Errorf("Wrong match. the content must not be in the error. got: %v", err)
		}
	})
}

func Test_NewSender(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, content []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, content, 0o600); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}
		return p
	}

	tests := []struct {
		name          string
		path          string
		expectNotConf bool
	}{
		{"path not set", "", true},
		{"blank path", "   ", true},
		{"file missing", filepath.Join(dir, "missing.json"), true},
		{"file empty", write("empty.json", nil), true},
		{"file blank", write("blank.json", []byte("  \n")), true},
		{"file not json", write("bad.json", []byte("SECRET-NOT-JSON")), true},
		{"valid file", write("ok.json", fakeServiceAccountJSON(t, "demo-project")), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			logrus.SetOutput(&logs)
			defer logrus.SetOutput(os.Stderr)

			s := NewSender(tt.path)
			_, notConf := s.(*notConfiguredSender)
			if notConf != tt.expectNotConf {
				t.Fatalf("Wrong match. expect not configured: %v, got: %v", tt.expectNotConf, notConf)
			}
			if strings.Contains(logs.String(), "SECRET-NOT-JSON") || strings.Contains(logs.String(), "PRIVATE KEY") {
				t.Errorf("Wrong match. the file content must not be logged. got: %s", logs.String())
			}

			if notConf {
				if err := s.Send(context.Background(), fakeToken, notification.KindCallIncoming, nil); !errors.Is(err, ErrNotConfigured) {
					t.Errorf("Wrong match. expect: %v, got: %v", ErrNotConfigured, err)
				}
			}
		})
	}
}
