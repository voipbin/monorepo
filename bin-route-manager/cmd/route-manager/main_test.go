package main

import (
	"testing"
	"time"
)

func Test_newPromServer(t *testing.T) {
	srv := newPromServer("127.0.0.1:2112")

	if srv.Addr != "127.0.0.1:2112" {
		t.Errorf("Wrong addr. expect: 127.0.0.1:2112, got: %s", srv.Addr)
	}
	if srv.ReadHeaderTimeout != 10*time.Second {
		t.Errorf("Wrong read header timeout. expect: 10s, got: %v", srv.ReadHeaderTimeout)
	}
	if srv.Handler != nil {
		t.Errorf("Wrong handler. expect: nil (default mux), got: %v", srv.Handler)
	}
}
