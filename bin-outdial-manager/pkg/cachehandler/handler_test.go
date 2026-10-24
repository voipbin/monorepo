package cachehandler

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/models/outdialtargetcall"
)

// fakeRedis is a minimal in-memory RESP server that understands PING, GET and SET.
type fakeRedis struct {
	mu      sync.Mutex
	store   map[string]string
	setArgs map[string][]string // key -> arguments after the value of the last SET
	failAll bool                // answer every command with an error reply
	conns   []net.Conn
}

func newFakeRedis() *fakeRedis {
	return &fakeRedis{
		store:   map[string]string{},
		setArgs: map[string][]string{},
	}
}

// client returns a redis client whose connections are served by the fake.
func (f *fakeRedis) client(t *testing.T) *redis.Client {
	t.Helper()

	c := redis.NewClient(&redis.Options{
		Addr:        "203.0.113.1:6379",
		MaxRetries:  -1,
		DialTimeout: time.Second,
		Dialer: func(ctx context.Context, network, addr string) (net.Conn, error) {
			cli, srv := net.Pipe()
			f.mu.Lock()
			f.conns = append(f.conns, cli, srv)
			f.mu.Unlock()
			go f.serve(srv)
			return cli, nil
		},
	})

	t.Cleanup(func() {
		_ = c.Close()
		f.mu.Lock()
		defer f.mu.Unlock()
		for _, cn := range f.conns {
			_ = cn.Close()
		}
	})

	return c
}

func (f *fakeRedis) serve(conn net.Conn) {
	r := bufio.NewReader(conn)
	for {
		args, err := readCommand(r)
		if err != nil {
			return
		}

		if _, err := conn.Write([]byte(f.handle(args))); err != nil {
			return
		}
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

	res := make([]string, 0, n)
	for i := 0; i < n; i++ {
		hdr, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		size, err := strconv.Atoi(strings.TrimSpace(hdr[1:]))
		if err != nil {
			return nil, err
		}
		buf := make([]byte, size+2)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, err
		}
		res = append(res, string(buf[:size]))
	}

	return res, nil
}

func (f *fakeRedis) handle(args []string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failAll {
		return "-ERR forced failure\r\n"
	}

	switch strings.ToUpper(args[0]) {
	case "PING":
		return "+PONG\r\n"
	case "GET":
		v, ok := f.store[args[1]]
		if !ok {
			return "$-1\r\n"
		}
		return fmt.Sprintf("$%d\r\n%s\r\n", len(v), v)
	case "SET":
		f.store[args[1]] = args[2]
		f.setArgs[args[1]] = args[3:]
		return "+OK\r\n"
	}

	return "-ERR unknown command\r\n"
}

func (f *fakeRedis) get(key string) (string, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.store[key]
	return v, ok
}

func (f *fakeRedis) args(key string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.setArgs[key]
}

func (f *fakeRedis) size() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.store)
}

func (f *fakeRedis) put(key, value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[key] = value
}

func (f *fakeRedis) setFailAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failAll = true
}

func newTestHandler(t *testing.T, f *fakeRedis) *handler {
	t.Helper()
	return &handler{Cache: f.client(t)}
}

func Test_Outdial_SetGet(t *testing.T) {
	fk := newFakeRedis()
	h := newTestHandler(t, fk)
	ctx := context.Background()

	id := uuid.FromStringOrNil("6a8f0cd4-b9c5-11ef-8d0a-0f1e2d3c4b5a")
	in := &outdial.Outdial{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: uuid.FromStringOrNil("7b9a1de5-b9c5-11ef-9e1b-1a2b3c4d5e6f"),
		},
		Name: "test outdial",
	}

	if err := h.OutdialSet(ctx, in); err != nil {
		t.Fatalf("OutdialSet returned an error: %v", err)
	}

	key := "outdial:" + id.String()
	raw, ok := fk.get(key)
	if !ok {
		t.Fatalf("expected the key %q to be stored", key)
	}
	var stored outdial.Outdial
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("stored value is not valid json: %v", err)
	}
	if stored.ID != id || stored.Name != "test outdial" || stored.CustomerID != in.CustomerID {
		t.Errorf("stored value mismatch. got: %+v", stored)
	}
	if got := fk.args(key); len(got) != 2 || !strings.EqualFold(got[0], "ex") || got[1] != "86400" {
		t.Errorf("expected ttl arguments [ex 86400], got: %v", got)
	}

	res, err := h.OutdialGet(ctx, id)
	if err != nil {
		t.Fatalf("OutdialGet returned an error: %v", err)
	}
	if res.ID != id || res.Name != "test outdial" {
		t.Errorf("OutdialGet mismatch. got: %+v", res)
	}
}

func Test_OutdialTarget_SetGet(t *testing.T) {
	fk := newFakeRedis()
	h := newTestHandler(t, fk)
	ctx := context.Background()

	id := uuid.FromStringOrNil("8c0b2ef6-b9c5-11ef-8f2c-2b3c4d5e6f70")
	in := &outdialtarget.OutdialTarget{
		ID:        id,
		OutdialID: uuid.FromStringOrNil("9d1c3f07-b9c5-11ef-9a3d-3c4d5e6f7081"),
		Name:      "test target",
		TryCount2: 3,
	}

	if err := h.OutdialTargetSet(ctx, in); err != nil {
		t.Fatalf("OutdialTargetSet returned an error: %v", err)
	}

	key := "outdialtarget:" + id.String()
	raw, ok := fk.get(key)
	if !ok {
		t.Fatalf("expected the key %q to be stored", key)
	}
	var stored outdialtarget.OutdialTarget
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("stored value is not valid json: %v", err)
	}
	if stored.ID != id || stored.OutdialID != in.OutdialID || stored.TryCount2 != 3 {
		t.Errorf("stored value mismatch. got: %+v", stored)
	}
	if got := fk.args(key); len(got) != 2 || !strings.EqualFold(got[0], "ex") || got[1] != "86400" {
		t.Errorf("expected ttl arguments [ex 86400], got: %v", got)
	}

	res, err := h.OutdialTargetGet(ctx, id)
	if err != nil {
		t.Fatalf("OutdialTargetGet returned an error: %v", err)
	}
	if res.ID != id || res.Name != "test target" || res.TryCount2 != 3 {
		t.Errorf("OutdialTargetGet mismatch. got: %+v", res)
	}
}

func Test_OutdialTargetCall_SetGet(t *testing.T) {
	// ID, ActiveflowID and ReferenceID differ so that a key built from the
	// wrong field fails.
	call := &outdialtargetcall.OutdialTargetCall{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("ae2d4018-b9c5-11ef-8b4e-4d5e6f708192"),
		},
		ActiveflowID: uuid.FromStringOrNil("bf3e5129-b9c5-11ef-9c5f-5e6f708192a3"),
		ReferenceID:  uuid.FromStringOrNil("c04f623a-b9c5-11ef-8d60-6f708192a3b4"),
		TryCount:     2,
	}

	tests := []struct {
		name string

		key string
		set func(h *handler, ctx context.Context) error
		get func(h *handler, ctx context.Context) (*outdialtargetcall.OutdialTargetCall, error)
	}{
		{
			name: "by id",
			key:  "outdialtargetcall:" + call.ID.String(),
			set: func(h *handler, ctx context.Context) error {
				return h.OutdialTargetCallSet(ctx, call)
			},
			get: func(h *handler, ctx context.Context) (*outdialtargetcall.OutdialTargetCall, error) {
				return h.OutdialTargetCallGet(ctx, call.ID)
			},
		},
		{
			name: "by activeflow id",
			key:  "outdialtargetcall_activeflowid:" + call.ActiveflowID.String(),
			set: func(h *handler, ctx context.Context) error {
				return h.OutdialTargetCallSetByActiveflowID(ctx, call)
			},
			get: func(h *handler, ctx context.Context) (*outdialtargetcall.OutdialTargetCall, error) {
				return h.OutdialTargetCallGetByActiveflowID(ctx, call.ActiveflowID)
			},
		},
		{
			name: "by reference id",
			key:  "outdialtargetcall_referenceid:" + call.ReferenceID.String(),
			set: func(h *handler, ctx context.Context) error {
				return h.OutdialTargetCallSetByReferenceID(ctx, call)
			},
			get: func(h *handler, ctx context.Context) (*outdialtargetcall.OutdialTargetCall, error) {
				return h.OutdialTargetCallGetByReferenceID(ctx, call.ReferenceID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fk := newFakeRedis()
			h := newTestHandler(t, fk)
			ctx := context.Background()

			if err := tt.set(h, ctx); err != nil {
				t.Fatalf("set returned an error: %v", err)
			}

			raw, ok := fk.get(tt.key)
			if !ok {
				t.Fatalf("expected the key %q to be stored", tt.key)
			}
			if fk.size() != 1 {
				t.Errorf("expected exactly one key, got: %d", fk.size())
			}
			var stored outdialtargetcall.OutdialTargetCall
			if err := json.Unmarshal([]byte(raw), &stored); err != nil {
				t.Fatalf("stored value is not valid json: %v", err)
			}
			if stored.ID != call.ID || stored.ActiveflowID != call.ActiveflowID || stored.ReferenceID != call.ReferenceID || stored.TryCount != 2 {
				t.Errorf("stored value mismatch. got: %+v", stored)
			}
			if got := fk.args(tt.key); len(got) != 2 || !strings.EqualFold(got[0], "ex") || got[1] != "86400" {
				t.Errorf("expected ttl arguments [ex 86400], got: %v", got)
			}

			res, err := tt.get(h, ctx)
			if err != nil {
				t.Fatalf("get returned an error: %v", err)
			}
			if res.ID != call.ID || res.TryCount != 2 {
				t.Errorf("get mismatch. got: %+v", res)
			}
		})
	}
}

func Test_Get_missingKey(t *testing.T) {
	id := uuid.FromStringOrNil("d15a734b-b9c5-11ef-9e71-708192a3b4c5")

	tests := []struct {
		name string
		fn   func(h *handler, ctx context.Context) error
	}{
		{"outdial", func(h *handler, ctx context.Context) error { _, err := h.OutdialGet(ctx, id); return err }},
		{"outdialtarget", func(h *handler, ctx context.Context) error { _, err := h.OutdialTargetGet(ctx, id); return err }},
		{"outdialtargetcall", func(h *handler, ctx context.Context) error { _, err := h.OutdialTargetCallGet(ctx, id); return err }},
		{"by activeflow id", func(h *handler, ctx context.Context) error {
			_, err := h.OutdialTargetCallGetByActiveflowID(ctx, id)
			return err
		}},
		{"by reference id", func(h *handler, ctx context.Context) error {
			_, err := h.OutdialTargetCallGetByReferenceID(ctx, id)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t, newFakeRedis())

			if err := tt.fn(h, context.Background()); !errors.Is(err, redis.Nil) {
				t.Errorf("expected redis.Nil, got: %v", err)
			}
		})
	}
}

func Test_getSerialize_corruptedValue(t *testing.T) {
	fk := newFakeRedis()
	h := newTestHandler(t, fk)
	ctx := context.Background()

	id := uuid.FromStringOrNil("e26b845c-b9c5-11ef-8f82-8192a3b4c5d6")
	fk.put("outdial:"+id.String(), "not a json document")

	res, err := h.OutdialGet(ctx, id)
	if err == nil {
		t.Fatalf("expected an unmarshal error, got nil and result: %+v", res)
	}
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Errorf("expected a json syntax error, got: %v", err)
	}
	if res != nil {
		t.Errorf("expected a nil result on error, got: %+v", res)
	}
}

func Test_Set_cacheError(t *testing.T) {
	id := uuid.FromStringOrNil("f37c956d-b9c5-11ef-9093-92a3b4c5d6e7")

	tests := []struct {
		name string
		fn   func(h *handler, ctx context.Context) error
	}{
		{"outdial", func(h *handler, ctx context.Context) error {
			return h.OutdialSet(ctx, &outdial.Outdial{Identity: commonidentity.Identity{ID: id}})
		}},
		{"outdialtarget", func(h *handler, ctx context.Context) error {
			return h.OutdialTargetSet(ctx, &outdialtarget.OutdialTarget{ID: id})
		}},
		{"outdialtargetcall", func(h *handler, ctx context.Context) error {
			return h.OutdialTargetCallSet(ctx, &outdialtargetcall.OutdialTargetCall{Identity: commonidentity.Identity{ID: id}})
		}},
		{"by activeflow id", func(h *handler, ctx context.Context) error {
			return h.OutdialTargetCallSetByActiveflowID(ctx, &outdialtargetcall.OutdialTargetCall{ActiveflowID: id})
		}},
		{"by reference id", func(h *handler, ctx context.Context) error {
			return h.OutdialTargetCallSetByReferenceID(ctx, &outdialtargetcall.OutdialTargetCall{ReferenceID: id})
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fk := newFakeRedis()
			fk.setFailAll()
			h := newTestHandler(t, fk)

			err := tt.fn(h, context.Background())
			if err == nil || !strings.Contains(err.Error(), "forced failure") {
				t.Errorf("expected the cache error to be returned, got: %v", err)
			}
			if fk.size() != 0 {
				t.Errorf("expected nothing stored, got: %d keys", fk.size())
			}
		})
	}
}

func Test_Get_cacheError(t *testing.T) {
	fk := newFakeRedis()
	fk.setFailAll()
	h := newTestHandler(t, fk)

	_, err := h.OutdialGet(context.Background(), uuid.FromStringOrNil("048da67e-b9c6-11ef-91a4-a3b4c5d6e7f8"))
	if err == nil || errors.Is(err, redis.Nil) || !strings.Contains(err.Error(), "forced failure") {
		t.Errorf("expected the server error to be returned, got: %v", err)
	}
}

func Test_setSerialize_marshalError(t *testing.T) {
	fk := newFakeRedis()
	h := newTestHandler(t, fk)

	// a channel cannot be marshaled to json
	if err := h.setSerialize(context.Background(), "key", make(chan int)); err == nil {
		t.Errorf("expected a marshal error, got nil")
	}
	if fk.size() != 0 {
		t.Errorf("expected nothing stored, got: %d keys", fk.size())
	}
}

func Test_Connect(t *testing.T) {
	t.Run("ping ok", func(t *testing.T) {
		h := newTestHandler(t, newFakeRedis())

		if err := h.Connect(); err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
	})

	t.Run("ping error", func(t *testing.T) {
		fk := newFakeRedis()
		fk.setFailAll()
		h := newTestHandler(t, fk)

		if err := h.Connect(); err == nil {
			t.Errorf("expected an error, got nil")
		}
	})
}

func Test_NewHandler(t *testing.T) {
	// redis.NewClient does not dial, so no server is needed.
	res := NewHandler("203.0.113.9:6380", "secret-for-test", 3)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("expected *handler, got: %T", res)
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	opts := h.Cache.Options()
	if opts.Addr != "203.0.113.9:6380" || opts.Password != "secret-for-test" || opts.DB != 3 {
		t.Errorf("client options mismatch. addr: %s, db: %d", opts.Addr, opts.DB)
	}
	if h.Addr != "203.0.113.9:6380" || h.Password != "secret-for-test" || h.DB != 3 {
		t.Errorf("handler fields mismatch. addr: %s, db: %d", h.Addr, h.DB)
	}
}
