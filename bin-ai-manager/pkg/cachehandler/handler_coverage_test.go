package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/models/summary"
	"monorepo/bin-ai-manager/models/team"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// newCoverageTestHandler wires a handler to a per-test in-process Redis and
// closes both the server and the client when the test ends.
func newCoverageTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("could not start miniredis. err: %v", err)
	}
	t.Cleanup(mr.Close)

	cache := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = cache.Close() })

	return &handler{Addr: mr.Addr(), Cache: cache}, mr
}

func Test_ObjectCache_RoundTrip(t *testing.T) {
	ctx := context.Background()

	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")
	customerID := uuid.FromStringOrNil("22222222-0000-0000-0000-000000000002")
	refID := uuid.FromStringOrNil("33333333-0000-0000-0000-000000000003")
	pcID := uuid.FromStringOrNil("44444444-0000-0000-0000-000000000004")

	tests := []struct {
		name      string
		set       func(h *handler) error
		get       func(h *handler) (uuid.UUID, string, error)
		keys      []string
		expectID  uuid.UUID
		expectTag string
	}{
		{
			name: "ai",
			set: func(h *handler) error {
				return h.AISet(ctx, &ai.AI{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Name: "ai-name"})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.AIGet(ctx, id)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.Name, nil
			},
			keys:      []string{"ai:ai:" + id.String()},
			expectID:  id,
			expectTag: "ai-name",
		},
		{
			name: "message",
			set: func(h *handler) error {
				return h.MessageSet(ctx, &message.Message{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Content: "hello"})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.MessageGet(ctx, id)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.Content, nil
			},
			keys:      []string{"ai:message:" + id.String()},
			expectID:  id,
			expectTag: "hello",
		},
		{
			name: "summary",
			set: func(h *handler) error {
				return h.SummarySet(ctx, &summary.Summary{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Content: "sum"})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.SummaryGet(ctx, id)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.Content, nil
			},
			keys:      []string{"ai:summary:" + id.String()},
			expectID:  id,
			expectTag: "sum",
		},
		{
			name: "team",
			set: func(h *handler) error {
				return h.TeamSet(ctx, &team.Team{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, Name: "team-name"})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.TeamGet(ctx, id)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.Name, nil
			},
			keys:      []string{"ai:team:" + id.String()},
			expectID:  id,
			expectTag: "team-name",
		},
		{
			name: "aicall by id",
			set: func(h *handler) error {
				return h.AIcallSet(ctx, &aicall.AIcall{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, ReferenceID: refID, PipecatcallID: pcID})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.AIcallGet(ctx, id)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.ReferenceID.String() + "/" + r.PipecatcallID.String(), nil
			},
			keys: []string{
				"ai:aicall:" + id.String(),
				"ai:aicall:reference_id:" + refID.String(),
				"ai:aicall:pipecatcall_id:" + pcID.String(),
			},
			expectID:  id,
			expectTag: refID.String() + "/" + pcID.String(),
		},
		{
			name: "aicall by reference id",
			set: func(h *handler) error {
				return h.AIcallSet(ctx, &aicall.AIcall{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, ReferenceID: refID, PipecatcallID: pcID})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.AIcallGetByReferenceID(ctx, refID)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.ReferenceID.String(), nil
			},
			keys:      []string{"ai:aicall:" + id.String()},
			expectID:  id,
			expectTag: refID.String(),
		},
		{
			name: "aicall by pipecatcall id",
			set: func(h *handler) error {
				return h.AIcallSet(ctx, &aicall.AIcall{Identity: commonidentity.Identity{ID: id, CustomerID: customerID}, ReferenceID: refID, PipecatcallID: pcID})
			},
			get: func(h *handler) (uuid.UUID, string, error) {
				r, err := h.AIcallGetByPipecatcallID(ctx, pcID)
				if err != nil {
					return uuid.Nil, "", err
				}
				return r.ID, r.PipecatcallID.String(), nil
			},
			keys:      []string{"ai:aicall:" + id.String()},
			expectID:  id,
			expectTag: pcID.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newCoverageTestHandler(t)

			if errSet := tt.set(h); errSet != nil {
				t.Fatalf("set failed. err: %v", errSet)
			}

			for _, key := range tt.keys {
				if !mr.Exists(key) {
					t.Errorf("expected key %s to be written", key)
					continue
				}
				if ttl := mr.TTL(key); ttl != 24*time.Hour {
					t.Errorf("ttl mismatch for %s. expected: %v, got: %v", key, 24*time.Hour, ttl)
				}
			}

			gotID, gotTag, errGet := tt.get(h)
			if errGet != nil {
				t.Fatalf("get failed. err: %v", errGet)
			}
			if gotID != tt.expectID || gotTag != tt.expectTag {
				t.Errorf("round trip mismatch. expected: %s/%s, got: %s/%s", tt.expectID, tt.expectTag, gotID, gotTag)
			}
		})
	}
}

func Test_ObjectCache_GetMiss(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string
		get  func(h *handler) error
	}{
		{"ai", func(h *handler) error { _, err := h.AIGet(ctx, id); return err }},
		{"aicall", func(h *handler) error { _, err := h.AIcallGet(ctx, id); return err }},
		{"aicall by reference id", func(h *handler) error { _, err := h.AIcallGetByReferenceID(ctx, id); return err }},
		{"aicall by pipecatcall id", func(h *handler) error { _, err := h.AIcallGetByPipecatcallID(ctx, id); return err }},
		{"message", func(h *handler) error { _, err := h.MessageGet(ctx, id); return err }},
		{"summary", func(h *handler) error { _, err := h.SummaryGet(ctx, id); return err }},
		{"team", func(h *handler) error { _, err := h.TeamGet(ctx, id); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newCoverageTestHandler(t)

			if err := tt.get(h); err != redis.Nil {
				t.Errorf("expected redis.Nil on a miss, got: %v", err)
			}
		})
	}
}

func Test_ObjectCache_CorruptStoredValue(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string
		key  string
		get  func(h *handler) error
	}{
		{"ai", "ai:ai:" + id.String(), func(h *handler) error { _, err := h.AIGet(ctx, id); return err }},
		{"aicall", "ai:aicall:" + id.String(), func(h *handler) error { _, err := h.AIcallGet(ctx, id); return err }},
		{"message", "ai:message:" + id.String(), func(h *handler) error { _, err := h.MessageGet(ctx, id); return err }},
		{"summary", "ai:summary:" + id.String(), func(h *handler) error { _, err := h.SummaryGet(ctx, id); return err }},
		{"team", "ai:team:" + id.String(), func(h *handler) error { _, err := h.TeamGet(ctx, id); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newCoverageTestHandler(t)
			if errSet := mr.Set(tt.key, "{not json"); errSet != nil {
				t.Fatalf("could not store the corrupt value. err: %v", errSet)
			}

			err := tt.get(h)
			if err == nil || err == redis.Nil {
				t.Errorf("expected a decode error, got: %v", err)
			}
		})
	}
}

func Test_ObjectCache_SetServerDown(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("11111111-0000-0000-0000-000000000001")

	tests := []struct {
		name string
		set  func(h *handler) error
	}{
		{"ai", func(h *handler) error { return h.AISet(ctx, &ai.AI{Identity: commonidentity.Identity{ID: id}}) }},
		{"aicall", func(h *handler) error {
			return h.AIcallSet(ctx, &aicall.AIcall{Identity: commonidentity.Identity{ID: id}})
		}},
		{"message", func(h *handler) error {
			return h.MessageSet(ctx, &message.Message{Identity: commonidentity.Identity{ID: id}})
		}},
		{"summary", func(h *handler) error {
			return h.SummarySet(ctx, &summary.Summary{Identity: commonidentity.Identity{ID: id}})
		}},
		{"team", func(h *handler) error { return h.TeamSet(ctx, &team.Team{Identity: commonidentity.Identity{ID: id}}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mr := newCoverageTestHandler(t)
			mr.Close()

			if err := tt.set(h); err == nil {
				t.Errorf("expected an error while the server is down")
			}
		})
	}
}

func Test_Listen_ResolverSet(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)

	transcribeID := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")
	a1 := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaa1")
	a2 := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaa2")

	empty, err := h.ListenAIcallIDsGet(ctx, transcribeID)
	if err != nil || len(empty) != 0 {
		t.Fatalf("expected an empty resolution for an unknown session. res: %v, err: %v", empty, err)
	}

	for _, id := range []uuid.UUID{a1, a2} {
		if errAdd := h.ListenAIcallIDAdd(ctx, transcribeID, id, time.Minute); errAdd != nil {
			t.Fatalf("add failed. err: %v", errAdd)
		}
	}
	// a member that is not a uuid cannot address an AIcall and must be skipped.
	if _, errMember := mr.SAdd(listenTranscribeKey(transcribeID), "not-a-uuid"); errMember != nil {
		t.Fatalf("could not add the malformed member. err: %v", errMember)
	}

	got, err := h.ListenAIcallIDsGet(ctx, transcribeID)
	if err != nil {
		t.Fatalf("get failed. err: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected both listeners and no malformed member, got: %v", got)
	}

	// removing one listener must leave the other in place.
	if errRemove := h.ListenAIcallIDRemove(ctx, transcribeID, a1); errRemove != nil {
		t.Fatalf("remove failed. err: %v", errRemove)
	}
	got, err = h.ListenAIcallIDsGet(ctx, transcribeID)
	if err != nil {
		t.Fatalf("get after remove failed. err: %v", err)
	}
	if len(got) != 1 || got[0] != a2 {
		t.Errorf("expected only the second listener to remain, got: %v", got)
	}
}

func Test_Listen_ResolverSet_ServerDown(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)
	mr.Close()

	id := uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555")

	if _, err := h.ListenAIcallIDsGet(ctx, id); err == nil {
		t.Errorf("expected ListenAIcallIDsGet to fail while the server is down")
	}
	if err := h.ListenAIcallIDRemove(ctx, id, id); err == nil {
		t.Errorf("expected ListenAIcallIDRemove to fail while the server is down")
	}
	if _, err := h.ListenTurnTryLock(ctx, id, time.Minute); err == nil {
		t.Errorf("expected ListenTurnTryLock to fail while the server is down")
	}
	if _, err := h.ListenTurnCountIncr(ctx, id, time.Minute); err == nil {
		t.Errorf("expected ListenTurnCountIncr to fail while the server is down")
	}
	if _, err := h.ListenTurnPipecatcallIDIsMember(ctx, id, id); err == nil {
		t.Errorf("expected ListenTurnPipecatcallIDIsMember to fail while the server is down")
	}
	if _, err := h.ListenStartLockAcquire(ctx, id, "tok", time.Minute); err == nil {
		t.Errorf("expected ListenStartLockAcquire to fail while the server is down")
	}
	if err := h.ListenStartLockRelease(ctx, id, "tok"); err == nil {
		t.Errorf("expected ListenStartLockRelease to fail while the server is down")
	}
	if err := h.ListenStateClear(ctx, id); err == nil {
		t.Errorf("expected ListenStateClear to fail while the server is down")
	}
}

func Test_Listen_TurnTryLock(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	won, err := h.ListenTurnTryLock(ctx, aicallID, 5*time.Second)
	if err != nil || !won {
		t.Fatalf("first caller must win the lock. won: %v, err: %v", won, err)
	}
	if ttl := mr.TTL(listenLockKey(aicallID)); ttl != 5*time.Second {
		t.Errorf("lock ttl mismatch. expected: 5s, got: %v", ttl)
	}

	won, err = h.ListenTurnTryLock(ctx, aicallID, 5*time.Second)
	if err != nil || won {
		t.Errorf("second caller must lose without an error. won: %v, err: %v", won, err)
	}

	// once the debounce window elapses the lock can be taken again.
	mr.FastForward(6 * time.Second)
	won, err = h.ListenTurnTryLock(ctx, aicallID, 5*time.Second)
	if err != nil || !won {
		t.Errorf("lock must be available after its ttl. won: %v, err: %v", won, err)
	}
}

func Test_Listen_TurnPipecatcallIDIsMember(t *testing.T) {
	ctx := context.Background()
	h, _ := newCoverageTestHandler(t)
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")
	registered := uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")
	other := uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-000000000000")

	if err := h.ListenTurnPipecatcallIDAdd(ctx, aicallID, registered, time.Minute); err != nil {
		t.Fatalf("add failed. err: %v", err)
	}

	tests := []struct {
		name   string
		id     uuid.UUID
		expect bool
	}{
		{"registered id", registered, true},
		{"id that was never registered", other, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.ListenTurnPipecatcallIDIsMember(ctx, aicallID, tt.id)
			if err != nil {
				t.Fatalf("unexpected error. err: %v", err)
			}
			if got != tt.expect {
				t.Errorf("membership mismatch. expected: %v, got: %v", tt.expect, got)
			}
		})
	}
}

func Test_Listen_StartLock(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")

	got, err := h.ListenStartLockAcquire(ctx, aicallID, "token-a", time.Minute)
	if err != nil || !got {
		t.Fatalf("first acquire must succeed. got: %v, err: %v", got, err)
	}
	if ttl := mr.TTL(listenStartLockKey(aicallID)); ttl != time.Minute {
		t.Errorf("start lock ttl mismatch. expected: %v, got: %v", time.Minute, ttl)
	}

	got, err = h.ListenStartLockAcquire(ctx, aicallID, "token-b", time.Minute)
	if err != nil || got {
		t.Errorf("second acquire must report not-acquired without an error. got: %v, err: %v", got, err)
	}

	// a release with someone else's token must not take the lock away.
	if errRelease := h.ListenStartLockRelease(ctx, aicallID, "token-b"); errRelease != nil {
		t.Fatalf("release with a foreign token must be a no-op, not an error. err: %v", errRelease)
	}
	if !mr.Exists(listenStartLockKey(aicallID)) {
		t.Fatalf("a foreign token released the lock")
	}

	// the holder's own token releases it.
	if errRelease := h.ListenStartLockRelease(ctx, aicallID, "token-a"); errRelease != nil {
		t.Fatalf("release failed. err: %v", errRelease)
	}
	if mr.Exists(listenStartLockKey(aicallID)) {
		t.Errorf("the lock must be gone after its holder released it")
	}

	got, err = h.ListenStartLockAcquire(ctx, aicallID, "token-b", time.Minute)
	if err != nil || !got {
		t.Errorf("acquire after release must succeed. got: %v, err: %v", got, err)
	}
}

func Test_Listen_StateClear(t *testing.T) {
	ctx := context.Background()
	h, mr := newCoverageTestHandler(t)
	aicallID := uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa")
	pcID := uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff")

	if err := h.ListenPendingPush(ctx, aicallID, "line", time.Minute); err != nil {
		t.Fatalf("pending push failed. err: %v", err)
	}
	if err := h.ListenWindowPush(ctx, aicallID, "line", 5, time.Minute); err != nil {
		t.Fatalf("window push failed. err: %v", err)
	}
	if _, err := h.ListenTurnTryLock(ctx, aicallID, time.Minute); err != nil {
		t.Fatalf("lock failed. err: %v", err)
	}
	if _, err := h.ListenTurnCountIncr(ctx, aicallID, time.Minute); err != nil {
		t.Fatalf("turn incr failed. err: %v", err)
	}
	if err := h.ListenTurnPipecatcallIDAdd(ctx, aicallID, pcID, time.Minute); err != nil {
		t.Fatalf("turn pipecatcall add failed. err: %v", err)
	}

	if err := h.ListenStateClear(ctx, aicallID); err != nil {
		t.Fatalf("clear failed. err: %v", err)
	}

	for _, key := range []string{listenPendingKey(aicallID), listenWindowKey(aicallID), listenLockKey(aicallID), listenTurnsKey(aicallID)} {
		if mr.Exists(key) {
			t.Errorf("expected %s to be cleared", key)
		}
	}
	// the self-expiring turn id set is deliberately left alone.
	if !mr.Exists(listenTurnPipecatcallIDKey(aicallID)) {
		t.Errorf("the turn pipecatcall id set must survive a state clear")
	}
}

func Test_Connect(t *testing.T) {
	h, mr := newCoverageTestHandler(t)

	if err := h.Connect(); err != nil {
		t.Errorf("expected a ping to succeed against a live server, got: %v", err)
	}

	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("expected a ping to fail against a closed server")
	}
}
