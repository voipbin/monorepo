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

func TestCreateCredentialConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data":{"id":"conn-abc-123"}}`)) //nolint:errcheck
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	connID, err := c.CreateCredentialConnection(context.Background(), "My Provider")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if connID != "conn-abc-123" {
		t.Fatalf("expected conn-abc-123, got %s", connID)
	}
}

func TestDeleteCredentialConnection_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTelnyxClientWithBase("testkey", srv.URL)
	if err := c.DeleteCredentialConnection(context.Background(), "conn-abc-123"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}
