package telnyxclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidateKey_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/whoami" || r.Header.Get("Authorization") != "Bearer testkey" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.ValidateKey(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidateKey_InvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("badkey", srv.URL)
	if !errors.Is(c.ValidateKey(context.Background()), ErrInvalidKey) {
		t.Fatalf("expected ErrInvalidKey")
	}
}

func TestCreateOutboundVoiceProfile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/outbound_voice_profiles" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"profile-abc-123"}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	profileID, err := c.CreateOutboundVoiceProfile(context.Background(), "My Provider")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if profileID != "profile-abc-123" {
		t.Fatalf("expected profile-abc-123, got %s", profileID)
	}
}

func TestDeleteOutboundVoiceProfile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.DeleteOutboundVoiceProfile(context.Background(), "profile-abc-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCreateFQDNConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/fqdn_connections" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"conn-abc-123"}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	connID, err := c.CreateFQDNConnection(context.Background(), "My Provider", "profile-abc-123", "myprovideruser", "mypassword123")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if connID != "conn-abc-123" {
		t.Fatalf("expected conn-abc-123, got %s", connID)
	}
}

func TestDeleteFQDNConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.DeleteFQDNConnection(context.Background(), "conn-abc-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRegisterFQDN_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/fqdns" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"fqdn-abc-123"}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	fqdnID, err := c.RegisterFQDN(context.Background(), "conn-abc-123", "pstn.voipbin.net", 5060)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if fqdnID != "fqdn-abc-123" {
		t.Fatalf("expected fqdn-abc-123, got %s", fqdnID)
	}
}

func TestDeleteFQDN_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.DeleteFQDN(context.Background(), "fqdn-abc-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
