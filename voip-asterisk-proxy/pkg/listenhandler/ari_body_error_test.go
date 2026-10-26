package listenhandler

import (
	"bufio"
	"errors"
	"io"
	"net"
	"testing"

	"monorepo/bin-common-handler/models/sock"
)

// Test_ariSendRequestToAsterisk_truncatedBody checks that a response whose body
// is shorter than its announced Content-Length is reported as an error and not
// returned as a partial success.
func Test_ariSendRequestToAsterisk_truncatedBody(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	t.Cleanup(func() {
		_ = ln.Close()
	})

	go func() {
		c, errAccept := ln.Accept()
		if errAccept != nil {
			return
		}
		defer func() {
			_ = c.Close()
		}()

		// read the request headers up to the blank line.
		r := bufio.NewReader(c)
		for {
			line, errRead := r.ReadString('\n')
			if errRead != nil {
				return
			}
			if line == "\r\n" {
				break
			}
		}

		_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 100\r\n\r\n{\"id\":"))
	}()

	h := &listenHandler{
		ariAddr:    ln.Addr().String(),
		ariAccount: "fake-user:fake-secret",
	}

	statusCode, data, err := h.ariSendRequestToAsterisk(&sock.Request{
		URI:    "/ari/channels",
		Method: sock.RequestMethodGet,
	})
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("Wrong error. expect: %v, got: %v", io.ErrUnexpectedEOF, err)
	}
	if statusCode != 0 {
		t.Errorf("Wrong status code. expect: 0, got: %d", statusCode)
	}
	if data != nil {
		t.Errorf("Wrong data. expect: nil, got: %d bytes", len(data))
	}
}
