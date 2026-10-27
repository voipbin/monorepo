package cachehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
)

// newTestHandler returns a handler backed by an isolated in-memory redis server.
// The client does not retry, so calls against a closed server fail immediately.
func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h := &handler{
		Addr:  mr.Addr(),
		Cache: redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1}),
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func testCtx(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)

	return ctx
}

func ptrBool(v bool) *bool {
	return &v
}

func ptrTime(v time.Time) *time.Time {
	return &v
}

// Each builder fills the entity fields that matter for the cache round trip with distinct non-zero
// values (the cosmetic ThemeConfig fields are only partly filled). The fixtures are rebuilt for
// the expected side so that a stored value is never compared with itself.
func testWidget() *widget.Widget {
	return &widget.Widget{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000002"),
		},
		Name:               "synthetic widget",
		Status:             widget.StatusInactive,
		DirectID:           uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000003"),
		Hash:               "synthetic-hash",
		SessionFlowID:      uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000004"),
		MessageFlowID:      uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000005"),
		SessionIdleTimeout: 901,
		ThemeConfig: &widget.ThemeConfig{
			PrimaryColor:               "#112233",
			HeaderTitle:                "synthetic title",
			Position:                   widget.WidgetPositionBottomLeft,
			ConnectingIndicatorEnabled: ptrBool(false),
			TypingIndicatorEnabled:     ptrBool(true),
		},
		TMCreate: ptrTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)),
		TMUpdate: ptrTime(time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)),
		TMDelete: ptrTime(time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)),
	}
}

func testSession() *session.Session {
	return &session.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000002"),
		},
		WidgetID: uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000003"),
		Status:   session.StatusEnded,
		PageURL:  "https://example.invalid/page",
		Referrer: "https://example.invalid/ref",
		Peer: commonaddress.Address{
			Type:       commonaddress.TypeAgent,
			Target:     "peer-target",
			TargetName: "peer-target-name",
		},
		Local: commonaddress.Address{
			Type:   commonaddress.TypeAI,
			Target: "local-target",
			Name:   "local-name",
		},
		ActiveflowID:   uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000004"),
		TMLastActivity: ptrTime(time.Date(2026, 4, 5, 6, 7, 8, 0, time.UTC)),
		TMCreate:       ptrTime(time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC)),
		TMUpdate:       ptrTime(time.Date(2026, 6, 7, 8, 9, 10, 0, time.UTC)),
		TMEnd:          ptrTime(time.Date(2026, 7, 8, 9, 10, 11, 0, time.UTC)),
	}
}

func testMessage() *message.Message {
	return &message.Message{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000002"),
		},
		WidgetID:     uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000003"),
		SessionID:    uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000004"),
		Direction:    message.DirectionOutbound,
		Status:       message.StatusDelivered,
		SenderID:     uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000005"),
		ActiveflowID: uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000006"),
		Text:         "synthetic text",
		TMCreate:     ptrTime(time.Date(2026, 8, 9, 10, 11, 12, 0, time.UTC)),
		TMDelete:     ptrTime(time.Date(2026, 9, 10, 11, 12, 13, 0, time.UTC)),
	}
}

func Test_NewHandler_Connect(t *testing.T) {
	mr := miniredis.RunT(t)

	res := NewHandler(mr.Addr(), "", 3)
	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler, got: %T", res)
	}
	if h.Addr != mr.Addr() || h.Password != "" || h.DB != 3 {
		t.Errorf("Wrong match.\nexpect: %s, empty password, 3\ngot: %s, %q, %d", mr.Addr(), h.Addr, h.Password, h.DB)
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	if err := h.Connect(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}

	// the database index must reach the client: a write lands in database 3, not database 0
	key := "webchat:widget:" + testWidget().ID.String()
	if err := h.WidgetSet(testCtx(t), testWidget()); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if !mr.DB(3).Exists(key) || mr.Exists(key) {
		t.Errorf("Wrong match. expect: key in database 3 only, got in 3: %v, in 0: %v", mr.DB(3).Exists(key), mr.Exists(key))
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
}

func Test_Set_Get(t *testing.T) {
	type testCase struct {
		name      string
		expectKey string

		set func(ctx context.Context, h *handler) error
		get func(ctx context.Context, h *handler) (any, error)

		expectRes any
	}

	tests := []testCase{
		{
			name:      "widget",
			expectKey: "webchat:widget:a0000000-0000-4000-8000-000000000001",
			set: func(ctx context.Context, h *handler) error {
				return h.WidgetSet(ctx, testWidget())
			},
			get: func(ctx context.Context, h *handler) (any, error) {
				return h.WidgetGet(ctx, uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001"))
			},
			expectRes: testWidget(),
		},
		{
			name:      "session",
			expectKey: "webchat:session:b0000000-0000-4000-8000-000000000001",
			set: func(ctx context.Context, h *handler) error {
				return h.SessionSet(ctx, testSession())
			},
			get: func(ctx context.Context, h *handler) (any, error) {
				return h.SessionGet(ctx, uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001"))
			},
			expectRes: func() *session.Session {
				// the soft-delete marker is not part of the serialized form
				s := testSession()
				s.TMDelete = nil
				return s
			}(),
		},
		{
			name:      "message",
			expectKey: "webchat:message:c0000000-0000-4000-8000-000000000001",
			set: func(ctx context.Context, h *handler) error {
				return h.MessageSet(ctx, testMessage())
			},
			get: func(ctx context.Context, h *handler) (any, error) {
				return h.MessageGet(ctx, uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001"))
			},
			expectRes: testMessage(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestHandler(t)
			ctx := testCtx(t)

			if err := tt.set(ctx, h); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if !mr.Exists(tt.expectKey) {
				t.Fatalf("Wrong match. expect: key %s, got: keys %v", tt.expectKey, mr.Keys())
			}
			if ttl := mr.TTL(tt.expectKey); ttl != 24*time.Hour {
				t.Errorf("Wrong match. expect: %v, got: %v", 24*time.Hour, ttl)
			}

			res, err := tt.get(ctx, h)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Set_overwrite(t *testing.T) {
	tests := []struct {
		name string

		setOld func(ctx context.Context, h *handler) error
		setNew func(ctx context.Context, h *handler) error
		get    func(ctx context.Context, h *handler) (any, error)

		expectRes any
	}{
		{
			name: "widget",
			setOld: func(ctx context.Context, h *handler) error {
				return h.WidgetSet(ctx, testWidget())
			},
			setNew: func(ctx context.Context, h *handler) error {
				w := testWidget()
				w.Name = "renamed widget"
				return h.WidgetSet(ctx, w)
			},
			get: func(ctx context.Context, h *handler) (any, error) {
				return h.WidgetGet(ctx, uuid.FromStringOrNil("a0000000-0000-4000-8000-000000000001"))
			},
			expectRes: func() *widget.Widget {
				w := testWidget()
				w.Name = "renamed widget"
				return w
			}(),
		},
		{
			name: "session",
			setOld: func(ctx context.Context, h *handler) error {
				return h.SessionSet(ctx, testSession())
			},
			setNew: func(ctx context.Context, h *handler) error {
				s := testSession()
				s.Status = session.StatusActive
				return h.SessionSet(ctx, s)
			},
			get: func(ctx context.Context, h *handler) (any, error) {
				return h.SessionGet(ctx, uuid.FromStringOrNil("b0000000-0000-4000-8000-000000000001"))
			},
			expectRes: func() *session.Session {
				s := testSession()
				s.Status = session.StatusActive
				s.TMDelete = nil
				return s
			}(),
		},
		{
			name: "message",
			setOld: func(ctx context.Context, h *handler) error {
				return h.MessageSet(ctx, testMessage())
			},
			setNew: func(ctx context.Context, h *handler) error {
				m := testMessage()
				m.Text = "edited text"
				return h.MessageSet(ctx, m)
			},
			get: func(ctx context.Context, h *handler) (any, error) {
				return h.MessageGet(ctx, uuid.FromStringOrNil("c0000000-0000-4000-8000-000000000001"))
			},
			expectRes: func() *message.Message {
				m := testMessage()
				m.Text = "edited text"
				return m
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t)
			ctx := testCtx(t)

			if err := tt.setOld(ctx, h); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if err := tt.setNew(ctx, h); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := tt.get(ctx, h)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get_miss(t *testing.T) {
	missingID := uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000001")

	tests := []struct {
		name string
		get  func(ctx context.Context, h *handler) (any, error)
	}{
		{"widget", func(ctx context.Context, h *handler) (any, error) { return h.WidgetGet(ctx, missingID) }},
		{"session", func(ctx context.Context, h *handler) (any, error) { return h.SessionGet(ctx, missingID) }},
		{"message", func(ctx context.Context, h *handler) (any, error) { return h.MessageGet(ctx, missingID) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler(t)

			res, err := tt.get(testCtx(t), h)
			if !errors.Is(err, redis.Nil) {
				t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
			}
			if !isNilResult(res) {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Get_malformed(t *testing.T) {
	id := uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000002")

	tests := []struct {
		name string
		key  string
		get  func(ctx context.Context, h *handler) (any, error)
	}{
		{"widget", "webchat:widget:" + id.String(), func(ctx context.Context, h *handler) (any, error) { return h.WidgetGet(ctx, id) }},
		{"session", "webchat:session:" + id.String(), func(ctx context.Context, h *handler) (any, error) { return h.SessionGet(ctx, id) }},
		{"message", "webchat:message:" + id.String(), func(ctx context.Context, h *handler) (any, error) { return h.MessageGet(ctx, id) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestHandler(t)
			if err := mr.Set(tt.key, "{not json"); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			res, err := tt.get(testCtx(t), h)
			if err == nil || errors.Is(err, redis.Nil) {
				t.Errorf("Wrong match. expect: json error, got: %v", err)
			}
			if !isNilResult(res) {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Set_Get_closedServer(t *testing.T) {
	id := uuid.FromStringOrNil("d0000000-0000-4000-8000-000000000003")

	tests := []struct {
		name string
		set  func(ctx context.Context, h *handler) error
		get  func(ctx context.Context, h *handler) (any, error)
	}{
		{
			"widget",
			func(ctx context.Context, h *handler) error { return h.WidgetSet(ctx, testWidget()) },
			func(ctx context.Context, h *handler) (any, error) { return h.WidgetGet(ctx, id) },
		},
		{
			"session",
			func(ctx context.Context, h *handler) error { return h.SessionSet(ctx, testSession()) },
			func(ctx context.Context, h *handler) (any, error) { return h.SessionGet(ctx, id) },
		},
		{
			"message",
			func(ctx context.Context, h *handler) error { return h.MessageSet(ctx, testMessage()) },
			func(ctx context.Context, h *handler) (any, error) { return h.MessageGet(ctx, id) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newTestHandler(t)
			mr.Close()
			ctx := testCtx(t)

			if err := tt.set(ctx, h); err == nil {
				t.Errorf("Wrong match. expect: set err, got: ok")
			}

			res, err := tt.get(ctx, h)
			if err == nil || errors.Is(err, redis.Nil) {
				t.Errorf("Wrong match. expect: connection err, got: %v", err)
			}
			if !isNilResult(res) {
				t.Errorf("Wrong match. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_setSerialize_marshalError(t *testing.T) {
	h, mr := newTestHandler(t)

	// a channel cannot be serialized
	if err := h.setSerialize(testCtx(t), "webchat:test:unserializable", make(chan int)); err == nil {
		t.Errorf("Wrong match. expect: err, got: ok")
	}
	if len(mr.Keys()) != 0 {
		t.Errorf("Wrong match. expect: no key, got: %v", mr.Keys())
	}
}

// isNilResult reports whether the typed pointer returned through any is nil.
func isNilResult(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}
