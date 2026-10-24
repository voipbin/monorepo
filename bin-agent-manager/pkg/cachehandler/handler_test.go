package cachehandler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-agent-manager/models/agent"
)

// fakeRedis is a minimal in-process RESP server bound to a loopback port chosen
// by the operating system. It understands PING, GET, SET (with EX), and DEL,
// keeps the values in a map, and records every command it receives.
type fakeRedis struct {
	ln net.Listener

	mu       sync.Mutex
	data     map[string]string
	commands [][]string

	// failWith, when set, is returned as an error reply to every command.
	failWith string
}

func newFakeRedis(t *testing.T) *fakeRedis {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen on loopback: %v", err)
	}

	f := &fakeRedis{ln: ln, data: map[string]string{}}
	go f.serve()
	t.Cleanup(func() { _ = ln.Close() })

	return f
}

func (f *fakeRedis) addr() string { return f.ln.Addr().String() }

func (f *fakeRedis) setFail(msg string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failWith = msg
}

func (f *fakeRedis) put(key, val string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.data[key] = val
}

func (f *fakeRedis) get(key string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.data[key]
	return v, ok
}

// commandsNamed returns the recorded commands whose name matches, case-insensitively.
func (f *fakeRedis) commandsNamed(name string) [][]string {
	f.mu.Lock()
	defer f.mu.Unlock()

	res := [][]string{}
	for _, c := range f.commands {
		if strings.EqualFold(c[0], name) {
			res = append(res, c)
		}
	}
	return res
}

func (f *fakeRedis) serve() {
	for {
		conn, err := f.ln.Accept()
		if err != nil {
			return
		}
		go f.handle(conn)
	}
}

func readCommand(r *bufio.Reader) ([]string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(line, "*") {
		return nil, fmt.Errorf("unexpected line: %q", line)
	}
	n, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}

	args := make([]string, 0, n)
	for i := 0; i < n; i++ {
		header, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(header[1:]))
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
	defer func() { _ = conn.Close() }()

	r := bufio.NewReader(conn)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}

		f.mu.Lock()
		f.commands = append(f.commands, args)
		failWith := f.failWith
		f.mu.Unlock()

		if failWith != "" {
			_, _ = fmt.Fprintf(conn, "-ERR %s\r\n", failWith)
			continue
		}

		switch strings.ToUpper(args[0]) {
		case "PING":
			_, _ = conn.Write([]byte("+PONG\r\n"))
		case "SET":
			f.put(args[1], args[2])
			_, _ = conn.Write([]byte("+OK\r\n"))
		case "GET":
			if v, ok := f.get(args[1]); ok {
				_, _ = fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(v), v)
			} else {
				_, _ = conn.Write([]byte("$-1\r\n"))
			}
		case "DEL":
			f.mu.Lock()
			deleted := 0
			for _, k := range args[1:] {
				if _, ok := f.data[k]; ok {
					delete(f.data, k)
					deleted++
				}
			}
			f.mu.Unlock()
			_, _ = fmt.Fprintf(conn, ":%d\r\n", deleted)
		default:
			_, _ = fmt.Fprintf(conn, "-ERR unknown command %q\r\n", args[0])
		}
	}
}

// newTestHandler returns a handler wired to the fake server without retries, so
// a failing command returns at once.
func newTestHandler(t *testing.T, f *fakeRedis) *handler {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: f.addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = client.Close() })

	return &handler{Addr: f.addr(), Cache: client}
}

func Test_Connect(t *testing.T) {
	t.Run("pong is a success and the handler comes from NewHandler", func(t *testing.T) {
		f := newFakeRedis(t)

		h := NewHandler(f.addr(), "", 0)
		if err := h.Connect(); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if len(f.commandsNamed("PING")) != 1 {
			t.Errorf("Wrong match. expect exactly one PING, got: %v", f.commandsNamed("PING"))
		}
	})

	t.Run("an error reply is a failure", func(t *testing.T) {
		f := newFakeRedis(t)
		f.setFail("down for test")

		h := NewHandler(f.addr(), "", 0)
		err := h.Connect()
		if err == nil || !strings.Contains(err.Error(), "down for test") {
			t.Errorf("Wrong match. expect the server error, got: %v", err)
		}
	})
}

func Test_AgentSetGetDel(t *testing.T) {
	ctx := context.Background()
	idA := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	idB := uuid.FromStringOrNil("8e3b890a-2fd7-11ef-b442-133f59be8b36")

	newAgent := func(id uuid.UUID, username string) *agent.Agent {
		a := &agent.Agent{Username: username, Name: "name-" + username}
		a.ID = id
		return a
	}

	t.Run("round trip keeps the value and stores it for 24 hours", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		if err := h.AgentSet(ctx, newAgent(idA, "user-a")); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		sets := f.commandsNamed("SET")
		if len(sets) != 1 {
			t.Fatalf("Wrong match. expect one SET, got: %v", sets)
		}
		if len(sets[0]) != 5 || !strings.EqualFold(sets[0][3], "ex") || sets[0][4] != "86400" {
			t.Errorf("Wrong match. expect the SET to carry EX 86400, got: %v", sets[0])
		}

		got, err := h.AgentGet(ctx, idA)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if got.ID != idA || got.Username != "user-a" || got.Name != "name-user-a" {
			t.Errorf("Wrong match. got: %+v", got)
		}
	})

	t.Run("different ids are stored under different keys", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		if err := h.AgentSet(ctx, newAgent(idA, "user-a")); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if err := h.AgentSet(ctx, newAgent(idB, "user-b")); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		a, err := h.AgentGet(ctx, idA)
		if err != nil || a.Username != "user-a" {
			t.Errorf("Wrong match. expect user-a, got: %+v, err: %v", a, err)
		}
		b, err := h.AgentGet(ctx, idB)
		if err != nil || b.Username != "user-b" {
			t.Errorf("Wrong match. expect user-b, got: %+v, err: %v", b, err)
		}
	})

	t.Run("a missing key is redis.Nil and returns nil", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		got, err := h.AgentGet(ctx, idA)
		if !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect redis.Nil, got: %v", err)
		}
		if got != nil {
			t.Errorf("Wrong match. expect nil, got: %+v", got)
		}
	})

	t.Run("a corrupted value is an error and returns nil", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		if err := h.AgentSet(ctx, newAgent(idA, "user-a")); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		// overwrite the stored value with something that is not JSON
		f.mu.Lock()
		for k := range f.data {
			f.data[k] = "{corrupted"
		}
		f.mu.Unlock()

		got, err := h.AgentGet(ctx, idA)
		if err == nil || errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect a decode error, got: %v", err)
		}
		if got != nil {
			t.Errorf("Wrong match. expect nil, got: %+v", got)
		}
	})

	t.Run("delete removes the value", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		if err := h.AgentSet(ctx, newAgent(idA, "user-a")); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if err := h.AgentDel(ctx, idA); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		if _, err := h.AgentGet(ctx, idA); !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect redis.Nil after the delete, got: %v", err)
		}
		if len(f.commandsNamed("DEL")) != 1 {
			t.Errorf("Wrong match. expect one DEL, got: %v", f.commandsNamed("DEL"))
		}
	})

	t.Run("server errors are returned", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)
		f.setFail("broken for test")

		if err := h.AgentSet(ctx, newAgent(idA, "user-a")); err == nil || !strings.Contains(err.Error(), "broken for test") {
			t.Errorf("Wrong match. AgentSet expect the server error, got: %v", err)
		}
		if got, err := h.AgentGet(ctx, idA); err == nil || got != nil {
			t.Errorf("Wrong match. AgentGet expect an error and nil, got: %+v, %v", got, err)
		}
		if err := h.AgentDel(ctx, idA); err == nil || !strings.Contains(err.Error(), "broken for test") {
			t.Errorf("Wrong match. AgentDel expect the server error, got: %v", err)
		}
	})
}

func Test_PasswordResetToken(t *testing.T) {
	ctx := context.Background()
	agentID := uuid.FromStringOrNil("464a277e-2d8d-11ef-8bc6-d7b95604d6f6")
	const token = "token-a"

	t.Run("set stores the agent id under the prefixed key with the given ttl", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		if err := h.PasswordResetTokenSet(ctx, token, agentID, 15*time.Minute); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		sets := f.commandsNamed("SET")
		if len(sets) != 1 {
			t.Fatalf("Wrong match. expect one SET, got: %v", sets)
		}
		if sets[0][1] != "password_reset:"+token || sets[0][2] != agentID.String() {
			t.Errorf("Wrong match. expect key and value, got: %v", sets[0])
		}
		if len(sets[0]) != 5 || !strings.EqualFold(sets[0][3], "ex") || sets[0][4] != "900" {
			t.Errorf("Wrong match. expect EX 900, got: %v", sets[0])
		}
	})

	t.Run("get returns the stored agent id", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)
		f.put("password_reset:"+token, agentID.String())

		got, err := h.PasswordResetTokenGet(ctx, token)
		if err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if got != agentID {
			t.Errorf("Wrong match. expect: %s, got: %s", agentID, got)
		}
	})

	t.Run("get of an unknown token is redis.Nil", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		got, err := h.PasswordResetTokenGet(ctx, "unknown")
		if !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect redis.Nil, got: %v", err)
		}
		if got != uuid.Nil {
			t.Errorf("Wrong match. expect the nil uuid, got: %s", got)
		}
	})

	t.Run("get of a value that is not a uuid is an error", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)
		f.put("password_reset:"+token, "not-a-uuid")

		got, err := h.PasswordResetTokenGet(ctx, token)
		if err == nil || !strings.HasPrefix(err.Error(), "could not parse agent id from token") {
			t.Errorf("Wrong match. expect a parse error, got: %v", err)
		}
		if got != uuid.Nil {
			t.Errorf("Wrong match. expect the nil uuid, got: %s", got)
		}
	})

	t.Run("delete removes the token", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)

		if err := h.PasswordResetTokenSet(ctx, token, agentID, time.Minute); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}
		if err := h.PasswordResetTokenDelete(ctx, token); err != nil {
			t.Fatalf("Wrong match. expect ok, got: %v", err)
		}

		dels := f.commandsNamed("DEL")
		if len(dels) != 1 || len(dels[0]) != 2 || dels[0][1] != "password_reset:"+token {
			t.Errorf("Wrong match. expect DEL of the prefixed key, got: %v", dels)
		}
		if _, err := h.PasswordResetTokenGet(ctx, token); !errors.Is(err, redis.Nil) {
			t.Errorf("Wrong match. expect redis.Nil after the delete, got: %v", err)
		}
	})

	t.Run("server errors are returned", func(t *testing.T) {
		f := newFakeRedis(t)
		h := newTestHandler(t, f)
		f.setFail("broken for test")

		if err := h.PasswordResetTokenSet(ctx, token, agentID, time.Minute); err == nil || !strings.Contains(err.Error(), "broken for test") {
			t.Errorf("Wrong match. Set expect the server error, got: %v", err)
		}
		if got, err := h.PasswordResetTokenGet(ctx, token); err == nil || got != uuid.Nil {
			t.Errorf("Wrong match. Get expect an error and the nil uuid, got: %s, %v", got, err)
		}
		if err := h.PasswordResetTokenDelete(ctx, token); err == nil || !strings.Contains(err.Error(), "broken for test") {
			t.Errorf("Wrong match. Delete expect the server error, got: %v", err)
		}
	})
}
