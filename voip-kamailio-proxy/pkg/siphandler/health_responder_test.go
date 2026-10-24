package siphandler

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// startResponder binds the fixed SIP port on loopback and answers a single
// request with the given first line. It skips the test when the port is busy.
// The received request is delivered on the returned channel; stop closes the
// socket and joins the goroutine.
func startResponder(t *testing.T, replyFirstLine string) (reqCh <-chan string, stop func()) {
	t.Helper()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5060})
	if err != nil {
		t.Skipf("cannot bind loopback UDP 5060: %v", err)
	}

	ch := make(chan string, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		buf := make([]byte, 4096)
		n, from, errRead := conn.ReadFromUDP(buf)
		if errRead != nil {
			return
		}
		ch <- string(buf[:n])
		_, _ = conn.WriteToUDP([]byte(replyFirstLine+"\r\nContent-Length: 0\r\n\r\n"), from)
	}()

	return ch, func() {
		_ = conn.Close()
		<-done
	}
}

func Test_SendOptionsCheck_response(t *testing.T) {
	tests := []struct {
		name         string
		replyLine    string
		expectCode   string
		expectHealth bool
	}{
		{"200 OK", "SIP/2.0 200 OK", "200", true},
		{"404 is still healthy", "SIP/2.0 404 Not Found", "404", true},
		{"status line without reason phrase", "SIP/2.0 200", "200", true},
		{"malformed first line", "GARBAGE", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqCh, stop := startResponder(t, tt.replyLine)
			defer stop()

			result, err := SendOptionsCheck(context.Background(), "127.0.0.1", 2*time.Second)
			if err != nil {
				t.Fatalf("Expected nil error, got: %v", err)
			}
			if result.Healthy != tt.expectHealth {
				t.Errorf("Expected Healthy=%v, got %v", tt.expectHealth, result.Healthy)
			}
			if result.ResponseCode != tt.expectCode {
				t.Errorf("Expected code %q, got %q", tt.expectCode, result.ResponseCode)
			}

			select {
			case req := <-reqCh:
				if !strings.HasPrefix(req, "OPTIONS sip:healthcheck@127.0.0.1 SIP/2.0\r\n") {
					t.Errorf("Unexpected request line: %q", strings.SplitN(req, "\r\n", 2)[0])
				}
				for _, h := range []string{"Via: SIP/2.0/UDP ", "CSeq: 1 OPTIONS\r\n", "Max-Forwards: 70\r\n"} {
					if !strings.Contains(req, h) {
						t.Errorf("Request is missing %q", h)
					}
				}
			case <-time.After(2 * time.Second):
				t.Errorf("Responder did not receive a request")
			}
		})
	}
}
