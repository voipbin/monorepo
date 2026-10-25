package main

import (
	"bufio"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeAMI is a loopback Asterisk Manager Interface stand-in that speaks just
// enough of the protocol for the amigo client: a banner line, a login answer
// and a blank line terminated answer per action.
type fakeAMI struct {
	ln net.Listener

	// actions receives every frame (including Login) the server reads.
	actions chan map[string]string

	mu    sync.Mutex
	conns []net.Conn
}

func newFakeAMI(t *testing.T) *fakeAMI {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}

	f := &fakeAMI{
		ln:      ln,
		actions: make(chan map[string]string, 32),
	}
	go f.serve()

	t.Cleanup(f.close)

	return f
}

func (f *fakeAMI) hostPort(t *testing.T) (string, string) {
	t.Helper()

	host, port, err := net.SplitHostPort(f.ln.Addr().String())
	if err != nil {
		t.Fatalf("could not split the address. err: %v", err)
	}

	return host, port
}

func (f *fakeAMI) close() {
	_ = f.ln.Close()

	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.conns {
		_ = c.Close()
	}
}

func (f *fakeAMI) serve() {
	for {
		c, err := f.ln.Accept()
		if err != nil {
			return
		}

		f.mu.Lock()
		f.conns = append(f.conns, c)
		f.mu.Unlock()

		go f.handle(c)
	}
}

func (f *fakeAMI) handle(c net.Conn) {
	defer func() {
		_ = c.Close()
	}()

	if _, err := c.Write([]byte("Asterisk Call Manager/1.0\r\n")); err != nil {
		return
	}

	r := bufio.NewReader(c)
	for {
		msg, err := readFrame(r)
		if err != nil {
			return
		}

		select {
		case f.actions <- msg:
		default:
		}

		var answer string
		if strings.EqualFold(msg["Action"], "Login") {
			answer = "Response: Success\r\nMessage: Authentication accepted\r\n\r\n"
		} else {
			answer = "Response: Success\r\nActionID: " + msg["ActionID"] + "\r\nMessage: handled " + msg["Action"] + "\r\n\r\n"
		}
		if _, err := c.Write([]byte(answer)); err != nil {
			return
		}
	}
}

// readFrame reads one blank line terminated "Key: Value" frame.
func readFrame(r *bufio.Reader) (map[string]string, error) {
	m := map[string]string{}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if len(m) == 0 {
				continue
			}
			return m, nil
		}

		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		m[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
}

// waitConnected polls until the amigo client has logged in or the deadline passes.
func waitConnected(t *testing.T, connected func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for !connected() {
		if time.Now().After(deadline) {
			t.Fatalf("the AMI client did not connect in time")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitAction returns the next frame the fake server saw, or fails on timeout.
func (f *fakeAMI) waitAction(t *testing.T) map[string]string {
	t.Helper()

	select {
	case m := <-f.actions:
		return m
	case <-time.After(5 * time.Second):
		t.Fatalf("the fake AMI server did not receive a frame in time")
		return nil
	}
}
