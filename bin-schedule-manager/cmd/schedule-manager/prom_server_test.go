package main

import "testing"

func Test_newPromServer(t *testing.T) {
	srv := newPromServer(":2112")

	if srv.Addr != ":2112" {
		t.Errorf("Addr = %q, expected %q", srv.Addr, ":2112")
	}
	if srv.ReadHeaderTimeout != promReadHeaderTimeout || srv.ReadHeaderTimeout <= 0 {
		t.Errorf("ReadHeaderTimeout = %v, expected %v", srv.ReadHeaderTimeout, promReadHeaderTimeout)
	}
	if srv.Handler != nil {
		t.Errorf("Handler must stay nil (DefaultServeMux, where promhttp is registered)")
	}
}
