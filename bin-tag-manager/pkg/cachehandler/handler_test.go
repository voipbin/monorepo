package cachehandler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-tag-manager/models/tag"
)

// fakeRedis is a minimal in-process RESP server. It understands PING, AUTH, SELECT, GET and SET
// (with EX) and records every received command so the tests can assert on the wire traffic.
type fakeRedis struct {
	ln net.Listener

	mu       sync.Mutex
	data     map[string]string
	commands [][]string
	failSet  bool
	conns    []net.Conn
	wg       sync.WaitGroup
}

func newFakeRedis(t *testing.T) *fakeRedis {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen: %v", err)
	}

	f := &fakeRedis{ln: ln, data: map[string]string{}}
	f.wg.Add(1)
	go f.serve()

	t.Cleanup(func() {
		_ = ln.Close()
		f.mu.Lock()
		for _, c := range f.conns {
			_ = c.Close()
		}
		f.mu.Unlock()
		f.wg.Wait()
	})
	return f
}

func (f *fakeRedis) addr() string { return f.ln.Addr().String() }

func (f *fakeRedis) serve() {
	defer f.wg.Done()
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		f.mu.Lock()
		f.conns = append(f.conns, conn)
		f.mu.Unlock()
		f.wg.Add(1)
		go func() {
			defer f.wg.Done()
			defer func() { _ = conn.Close() }()
			f.handle(conn)
		}()
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimRight(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected line: %q", line)
	}
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimRight(hdr, "\r\n")[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:size]))
	}
	return args, nil
}

func (f *fakeRedis) handle(conn net.Conn) {
	r := bufio.NewReader(conn)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}

		f.mu.Lock()
		f.commands = append(f.commands, args)
		var reply string
		switch strings.ToUpper(args[0]) {
		case "PING":
			reply = "+PONG\r\n"
		case "AUTH", "SELECT":
			reply = "+OK\r\n"
		case "SET":
			if f.failSet {
				reply = "-ERR simulated failure\r\n"
			} else {
				f.data[args[1]] = args[2]
				reply = "+OK\r\n"
			}
		case "GET":
			if v, ok := f.data[args[1]]; ok {
				reply = fmt.Sprintf("$%d\r\n%s\r\n", len(v), v)
			} else {
				reply = "$-1\r\n"
			}
		default:
			reply = "-ERR unknown command\r\n"
		}
		f.mu.Unlock()

		if _, err := conn.Write([]byte(reply)); err != nil {
			return
		}
	}
}

func (f *fakeRedis) snapshot() [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]string(nil), f.commands...)
}

func (f *fakeRedis) put(key, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = value
}

func newTestTag() *tag.Tag {
	created := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	return &tag.Tag{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
			CustomerID: uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa"),
		},
		Name:     "test tag",
		Detail:   "test detail",
		TMCreate: &created,
	}
}

func Test_Connect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		srv := newFakeRedis(t)
		h := NewHandler(srv.addr(), "", 0)

		if err := h.Connect(); err != nil {
			t.Fatalf("Connect returned an error: %v", err)
		}

		cmds := srv.snapshot()
		if len(cmds) != 1 || strings.ToUpper(cmds[0][0]) != "PING" {
			t.Errorf("Wrong commands. expect: a single PING, got: %v", cmds)
		}
	})

	t.Run("password and database are sent before the ping", func(t *testing.T) {
		srv := newFakeRedis(t)
		h := NewHandler(srv.addr(), "secret", 4)

		if err := h.Connect(); err != nil {
			t.Fatalf("Connect returned an error: %v", err)
		}

		var names []string
		for _, c := range srv.snapshot() {
			names = append(names, strings.ToUpper(c[0]))
		}
		expect := []string{"AUTH", "SELECT", "PING"}
		if !reflect.DeepEqual(names, expect) {
			t.Errorf("Wrong command order. expect: %v, got: %v", expect, names)
		}
	})

	t.Run("server is not reachable", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("could not listen: %v", err)
		}
		addr := ln.Addr().String()
		_ = ln.Close()

		h := NewHandler(addr, "", 0)
		if err := h.Connect(); err == nil {
			t.Errorf("expected an error for the closed address")
		}
	})
}

func Test_TagSet(t *testing.T) {
	t.Run("stores the json under tag:<id> with a 24 hour expiry", func(t *testing.T) {
		srv := newFakeRedis(t)
		h := NewHandler(srv.addr(), "", 0)
		in := newTestTag()

		if err := h.TagSet(context.Background(), in); err != nil {
			t.Fatalf("TagSet returned an error: %v", err)
		}

		var set []string
		for _, c := range srv.snapshot() {
			if strings.ToUpper(c[0]) == "SET" {
				set = c
			}
		}
		if set == nil {
			t.Fatalf("no SET command was received")
		}

		if set[1] != "tag:11111111-2222-3333-4444-555555555555" {
			t.Errorf("Wrong key. got: %s", set[1])
		}
		if len(set) != 5 || strings.ToUpper(set[3]) != "EX" || set[4] != "86400" {
			t.Errorf("Wrong expiry arguments. expect: EX 86400, got: %v", set[3:])
		}

		var stored tag.Tag
		if err := json.Unmarshal([]byte(set[2]), &stored); err != nil {
			t.Fatalf("stored value is not valid json: %v", err)
		}
		if !reflect.DeepEqual(&stored, in) {
			t.Errorf("Wrong stored tag. expect: %v, got: %v", in, &stored)
		}
	})

	t.Run("server error is returned", func(t *testing.T) {
		srv := newFakeRedis(t)
		srv.mu.Lock()
		srv.failSet = true
		srv.mu.Unlock()
		h := NewHandler(srv.addr(), "", 0)

		err := h.TagSet(context.Background(), newTestTag())
		if err == nil || !strings.Contains(err.Error(), "simulated failure") {
			t.Errorf("Wrong error. got: %v", err)
		}
	})
}

func Test_TagGet(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		srv := newFakeRedis(t)
		h := NewHandler(srv.addr(), "", 0)
		in := newTestTag()

		if err := h.TagSet(context.Background(), in); err != nil {
			t.Fatalf("TagSet returned an error: %v", err)
		}

		res, err := h.TagGet(context.Background(), in.ID)
		if err != nil {
			t.Fatalf("TagGet returned an error: %v", err)
		}
		if !reflect.DeepEqual(res, in) {
			t.Errorf("Wrong tag. expect: %v, got: %v", in, res)
		}
	})

	t.Run("cache miss returns redis.Nil", func(t *testing.T) {
		srv := newFakeRedis(t)
		h := NewHandler(srv.addr(), "", 0)

		res, err := h.TagGet(context.Background(), uuid.FromStringOrNil("00000000-0000-0000-0000-0000000000aa"))
		if !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong error. expect: redis.Nil, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected a nil result on a miss, got: %v", res)
		}
	})

	t.Run("corrupt json returns a syntax error", func(t *testing.T) {
		srv := newFakeRedis(t)
		srv.put("tag:11111111-2222-3333-4444-555555555555", "{not json")
		h := NewHandler(srv.addr(), "", 0)

		res, err := h.TagGet(context.Background(), uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"))
		var syntaxErr *json.SyntaxError
		if !errors.As(err, &syntaxErr) {
			t.Errorf("Wrong error. expect: *json.SyntaxError, got: %v", err)
		}
		if res != nil {
			t.Errorf("expected a nil result on corrupt data, got: %v", res)
		}
	})
}
