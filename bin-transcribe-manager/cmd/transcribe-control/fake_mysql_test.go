package main

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
)

// startFakeMySQL starts a loopback server that speaks just enough of the MySQL wire protocol for
// a client to connect and ping. Every other command, including the session setup and query ones,
// is answered with an error. It returns the address to connect to. The listener and all
// connections are closed when the test is over.
func startFakeMySQL(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}

	var mu sync.Mutex
	var conns []net.Conn
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, errAccept := ln.Accept()
			if errAccept != nil {
				return
			}

			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() { _ = conn.Close() }()
				serveFakeMySQL(conn)
			}()
		}
	}()

	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		for _, c := range conns {
			_ = c.Close()
		}
		mu.Unlock()
		wg.Wait()
	})

	return ln.Addr().String()
}

func serveFakeMySQL(conn net.Conn) {
	// server greeting: protocol 10, native password plugin, no TLS
	greeting := []byte{10}
	greeting = append(greeting, []byte("8.0.0-fake\x00")...)
	greeting = append(greeting, 1, 0, 0, 0)                             // connection id
	greeting = append(greeting, []byte("abcdefgh\x00")...)              // scramble part 1 and filler
	greeting = append(greeting, 0x09, 0xa2)                             // capabilities, lower
	greeting = append(greeting, 0x21)                                   // character set
	greeting = append(greeting, 0x02, 0x00)                             // status
	greeting = append(greeting, 0x08, 0x00)                             // capabilities, upper
	greeting = append(greeting, 21)                                     // scramble length
	greeting = append(greeting, make([]byte, 10)...)                    // reserved
	greeting = append(greeting, []byte("ijklmnopqrst\x00")...)          // scramble part 2
	greeting = append(greeting, []byte("mysql_native_password\x00")...) // plugin
	if !writeFakeMySQLPacket(conn, 0, greeting) {
		return
	}

	// the client's login answer
	if _, ok := readFakeMySQLPacket(conn); !ok {
		return
	}
	if !writeFakeMySQLPacket(conn, 2, []byte{0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}) {
		return
	}

	for {
		payload, ok := readFakeMySQLPacket(conn)
		if !ok || len(payload) == 0 {
			return
		}

		switch payload[0] {
		case 0x01: // quit
			return
		case 0x0e: // ping
			if !writeFakeMySQLPacket(conn, 1, []byte{0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00}) {
				return
			}
		default:
			errPacket := []byte{0xff, 0x51, 0x04, '#', 'H', 'Y', '0', '0', '0'}
			errPacket = append(errPacket, []byte("not supported by the test server")...)
			if !writeFakeMySQLPacket(conn, 1, errPacket) {
				return
			}
		}
	}
}

func readFakeMySQLPacket(conn net.Conn) ([]byte, bool) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, false
	}

	payload := make([]byte, int(header[0])|int(header[1])<<8|int(header[2])<<16)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, false
	}

	return payload, true
}

func writeFakeMySQLPacket(conn net.Conn, seq byte, payload []byte) bool {
	header := make([]byte, 4)
	binary.LittleEndian.PutUint32(header, uint32(len(payload)))
	header[3] = seq

	_, err := conn.Write(append(header, payload...))
	return err == nil
}
