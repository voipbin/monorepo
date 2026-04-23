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

func TestCreateIPConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/ip_connections" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"conn-abc-123"}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	connID, err := c.CreateIPConnection(context.Background(), "My Provider", "profile-abc-123")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if connID != "conn-abc-123" {
		t.Fatalf("expected conn-abc-123, got %s", connID)
	}
}

func TestDeleteIPConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.DeleteIPConnection(context.Background(), "conn-abc-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestRegisterIP_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/ips" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"data":{"id":"ip-abc-123"}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	ipID, err := c.RegisterIP(context.Background(), "conn-abc-123", "1.2.3.4", 5060)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if ipID != "ip-abc-123" {
		t.Fatalf("expected ip-abc-123, got %s", ipID)
	}
}

func TestDeleteIP_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.DeleteIP(context.Background(), "ip-abc-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
