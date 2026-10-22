package main

import (
	"testing"
	"time"
)

func Test_newPromServer(t *testing.T) {
	srv := newPromServer(":2112")

	if srv.Addr != ":2112" {
		t.Errorf("Wrong match. expect: :2112, got: %s", srv.Addr)
	}
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("Wrong match. expect: %v, got: %v", 10*time.Second, srv.ReadHeaderTimeout)
	}
	if srv.Handler != nil {
		t.Errorf("Wrong match. expect: nil (http.DefaultServeMux), got: %v", srv.Handler)
	}
}
