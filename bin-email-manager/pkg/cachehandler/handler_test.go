package cachehandler

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-email-manager/models/email"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
)

// newUnreachableHandler returns a handler whose cache client points at a loopback port
// that nothing listens on, with retries disabled so failures are immediate.
func newUnreachableHandler(t *testing.T) *handler {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return &handler{
		Addr: addr,
		Cache: redis.NewClient(&redis.Options{
			Addr:        addr,
			MaxRetries:  -1,
			DialTimeout: 200 * time.Millisecond,
		}),
	}
}

func Test_NewHandler(t *testing.T) {
	res := NewHandler("127.0.0.1:6379", "synthetic-password", 3)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.Addr != "127.0.0.1:6379" || h.Password != "synthetic-password" || h.DB != 3 {
		t.Errorf("Wrong settings. addr: %s, db: %d", h.Addr, h.DB)
	}
	if h.Cache == nil || h.Locker == nil || h.mapMutex == nil {
		t.Errorf("Expected the cache, the locker and the mutex map to be initialized")
	}
}

func Test_unreachableCache(t *testing.T) {
	h := newUnreachableHandler(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	id := uuid.FromStringOrNil("5e8f0f0a-8888-4888-8888-000000000001")

	if err := h.Connect(); err == nil {
		t.Errorf("Connect: expected an error")
	}

	if err := h.EmailSet(ctx, &email.Email{}); err == nil {
		t.Errorf("EmailSet: expected an error")
	}

	res, err := h.EmailGet(ctx, id)
	if err == nil {
		t.Errorf("EmailGet: expected an error")
	}
	if res != nil {
		t.Errorf("EmailGet: expected a nil result. got: %v", res)
	}
}

// fakeRedisHook answers commands in the process, so no connection is opened.
type fakeRedisHook struct {
	mu   sync.Mutex
	cmds [][]any

	// getValue is answered to GET; an empty value answers with a cache miss
	getValue string
}

func (f *fakeRedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (f *fakeRedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		f.mu.Lock()
		f.cmds = append(f.cmds, cmd.Args())
		f.mu.Unlock()

		switch c := cmd.(type) {
		case *redis.StatusCmd:
			c.SetVal("OK")
		case *redis.StringCmd:
			if f.getValue == "" {
				c.SetErr(redis.Nil)
				return redis.Nil
			}
			c.SetVal(f.getValue)
		}
		return nil
	}
}

func (f *fakeRedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}

func (f *fakeRedisHook) commands() [][]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([][]any{}, f.cmds...)
}

// newHookedHandler returns a handler whose client is answered by the hook.
// The address is a refused loopback address as a second guard against real connections.
func newHookedHandler(t *testing.T, getValue string) (*handler, *fakeRedisHook) {
	t.Helper()

	h := newUnreachableHandler(t)
	hook := &fakeRedisHook{getValue: getValue}
	h.Cache.AddHook(hook)

	return h, hook
}

func Test_Connect_success(t *testing.T) {
	h, hook := newHookedHandler(t, "")

	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	cmds := hook.commands()
	if len(cmds) != 1 || cmds[0][0] != "ping" {
		t.Errorf("Wrong commands. expect a single ping, got: %v", cmds)
	}
}

func Test_EmailSet_success(t *testing.T) {
	h, hook := newHookedHandler(t, "")
	id := uuid.FromStringOrNil("5e8f0f0a-bbbb-4bbb-8bbb-000000000001")

	e := &email.Email{Identity: commonidentity.Identity{ID: id}, Subject: "synthetic subject"}
	if err := h.EmailSet(context.Background(), e); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	cmds := hook.commands()
	if len(cmds) != 1 {
		t.Fatalf("Wrong commands. expect a single set, got: %v", cmds)
	}
	// set <key> <value> ex <seconds>
	if len(cmds[0]) != 5 || cmds[0][0] != "set" || cmds[0][1] != "email_email:"+id.String() || cmds[0][3] != "ex" || cmds[0][4] != int64(24*60*60) {
		t.Errorf("Wrong set command. got: %v", cmds[0])
	}

	var stored email.Email
	raw, ok := cmds[0][2].([]byte)
	if !ok {
		t.Fatalf("Wrong value type. got: %T", cmds[0][2])
	}
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("Could not decode the stored value. err: %v", err)
	}
	if stored.ID != id || stored.Subject != "synthetic subject" {
		t.Errorf("Wrong stored value. got: %s", string(raw))
	}
}

func Test_EmailSet_marshalFails(t *testing.T) {
	h, hook := newHookedHandler(t, "")

	// year 10000 cannot be encoded as JSON
	tm := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	err := h.EmailSet(context.Background(), &email.Email{TMCreate: &tm})
	if err == nil || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("Wrong match. expect a marshal error, got: %v", err)
	}
	if cmds := hook.commands(); len(cmds) != 0 {
		t.Errorf("Expected no command after a marshal failure. got: %v", cmds)
	}
}

func Test_EmailGet(t *testing.T) {
	id := uuid.FromStringOrNil("5e8f0f0a-cccc-4ccc-8ccc-000000000001")

	tests := []struct {
		name string

		cached string

		expectErr     bool
		expectSubject string
	}{
		{
			name:          "hit",
			cached:        `{"id":"5e8f0f0a-cccc-4ccc-8ccc-000000000001","subject":"synthetic subject"}`,
			expectSubject: "synthetic subject",
		},
		{
			name:      "miss",
			cached:    "",
			expectErr: true,
		},
		{
			name:      "undecodable_value",
			cached:    "{not json",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, hook := newHookedHandler(t, tt.cached)

			res, err := h.EmailGet(context.Background(), id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected an error")
				}
				if tt.cached == "" && !errors.Is(err, redis.Nil) {
					t.Errorf("Expected a cache miss. got: %v", err)
				}
				if res != nil {
					t.Errorf("Expected a nil result. got: %v", res)
				}
			} else {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res.ID != id || res.Subject != tt.expectSubject {
					t.Errorf("Wrong result. got: %v", res)
				}
			}

			cmds := hook.commands()
			if len(cmds) != 1 || cmds[0][0] != "get" || cmds[0][1] != "email_email:"+id.String() {
				t.Errorf("Wrong commands. expect a get of the email key, got: %v", cmds)
			}
		})
	}
}
