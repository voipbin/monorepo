package homerhandler

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Test_homer_body_read_failure checks that a response whose body ends before the declared length is
// reported as a body read failure by each request method.
func Test_homer_body_read_failure(t *testing.T) {
	expectMessages := map[string]string{
		"GetSIPMessages": "could not read response body for SIP call ID call-abc",
		"GetPcap":        "could not read PCAP response body for SIP call ID call-abc",
		"GetRTCPPcap":    "could not read RTCP PCAP response body for SIP call ID call-abc",
	}

	for _, hc := range homerCalls() {
		t.Run(hc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// declare 1000 bytes and send 5, then end the response
				w.Header().Set("Content-Length", "1000")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("short"))
			}))
			defer srv.Close()

			h := NewHomerHandler(srv.URL, "test-token")

			_, err := hc.call(h, context.Background(), "call-abc", testFrom, testTo)
			expect, ok := expectMessages[hc.name]
			if !ok {
				t.Fatalf("Wrong match. no expected message for the method %s", hc.name)
			}
			if err == nil || !strings.Contains(err.Error(), expect) {
				t.Fatalf("Wrong match. expect error containing: %s, got: %v", expect, err)
			}
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Errorf("Wrong match. expect errors.Is: %v, got: %v", io.ErrUnexpectedEOF, err)
			}
		})
	}
}
