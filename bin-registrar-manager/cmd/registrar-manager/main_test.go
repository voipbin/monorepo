package main

import (
	"net"
	"net/http"
	"sync"
	"testing"
	"time"
)

var (
	promOnce sync.Once
	promAddr string
)

func Test_initProm(t *testing.T) {
	// initProm registers on http.DefaultServeMux, which can happen only once per
	// process, so repeated runs (-count=N) reuse the first server.
	var err error
	promOnce.Do(func() {
		// reserve a free local port, then hand it to initProm
		l, errListen := net.Listen("tcp", "127.0.0.1:0")
		if errListen != nil {
			err = errListen
			return
		}
		promAddr = l.Addr().String()
		_ = l.Close()

		initProm("/metrics", promAddr)
	})
	if err != nil || promAddr == "" {
		t.Fatalf("could not start the metrics server. err: %v", err)
	}
	addr := promAddr

	var resp *http.Response
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err = http.Get("http://" + addr + "/metrics") //nolint:noctx // test helper
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("metrics endpoint did not come up. err: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("wrong status. expect: %d, got: %d", http.StatusOK, resp.StatusCode)
	}
}

func Test_promReadHeaderTimeout(t *testing.T) {
	if promReadHeaderTimeout <= 0 {
		t.Errorf("the metrics server must bound the header read. got: %v", promReadHeaderTimeout)
	}
}
