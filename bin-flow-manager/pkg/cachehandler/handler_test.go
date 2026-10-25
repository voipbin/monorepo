package cachehandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/variable"
)

// newTestHandler returns a handler backed by an in-memory redis server.
func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	cache := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1})
	t.Cleanup(func() { _ = cache.Close() })

	h := &handler{
		Addr:     mr.Addr(),
		Cache:    cache,
		Locker:   redsync.New(goredis.NewPool(cache)),
		mapMutex: make(map[string]*redsync.Mutex),
	}

	return h, mr
}

func Test_NewHandler_Connect(t *testing.T) {
	mr := miniredis.RunT(t)

	h := NewHandler(mr.Addr(), "", 0)
	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the server goes away, so the connection check must fail.
	mr.Close()
	if err := h.Connect(); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
}

func Test_FlowSetGetDel(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	f := &flow.Flow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("5f0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01"),
			CustomerID: uuid.FromStringOrNil("5f0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e02"),
		},
		Name:    "test flow",
		Actions: []action.Action{{ID: uuid.FromStringOrNil("5f0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e03"), Type: action.TypeAnswer}},
	}

	// not cached yet
	if _, err := h.FlowGet(ctx, f.ID); !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}

	if err := h.FlowSet(ctx, f); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := fmt.Sprintf("flow:%s", f.ID)
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect ttl: %v, got: %v", 24*time.Hour, ttl)
	}

	res, err := h.FlowGet(ctx, f.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != f.ID || res.CustomerID != f.CustomerID || res.Name != f.Name || len(res.Actions) != 1 || res.Actions[0].Type != action.TypeAnswer {
		t.Errorf("Wrong match. expect: %v, got: %v", f, res)
	}

	if err := h.FlowDel(ctx, f.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists(key) {
		t.Errorf("Wrong match. the key must be deleted. key: %s", key)
	}
	if _, err := h.FlowGet(ctx, f.ID); !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}
}

func Test_ActiveflowSetGet(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("6a0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01"),
		},
		ReferenceType: activeflow.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("6a0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e02"),
	}

	if _, err := h.ActiveflowGet(ctx, af.ID); !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}

	if err := h.ActiveflowSet(ctx, af); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.ActiveflowGet(ctx, af.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != af.ID || res.ReferenceType != af.ReferenceType || res.ReferenceID != af.ReferenceID {
		t.Errorf("Wrong match. expect: %v, got: %v", af, res)
	}
}

func Test_VariableSetGet(t *testing.T) {
	h, _ := newTestHandler(t)
	ctx := context.Background()

	v := &variable.Variable{
		ID:        uuid.FromStringOrNil("7a0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01"),
		Variables: map[string]string{"key1": "value1", "key2": "value2"},
	}

	if _, err := h.VariableGet(ctx, v.ID); !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}

	if err := h.VariableSet(ctx, v); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	res, err := h.VariableGet(ctx, v.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != v.ID || len(res.Variables) != 2 || res.Variables["key1"] != "value1" || res.Variables["key2"] != "value2" {
		t.Errorf("Wrong match. expect: %v, got: %v", v, res)
	}
}

func Test_Get_corruptedData(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("8a0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01")
	for _, prefix := range []string{"flow", "activeflow", "variable"} {
		if err := mr.Set(fmt.Sprintf("%s:%s", prefix, id), "{not-json"); err != nil {
			t.Fatalf("Could not prepare the data. err: %v", err)
		}
	}

	if res, err := h.FlowGet(ctx, id); err == nil || res != nil || errors.Is(err, redis.Nil) {
		t.Errorf("FlowGet: Wrong match. expect: decode error, got: %v, %v", res, err)
	}
	if res, err := h.ActiveflowGet(ctx, id); err == nil || res != nil || errors.Is(err, redis.Nil) {
		t.Errorf("ActiveflowGet: Wrong match. expect: decode error, got: %v, %v", res, err)
	}
	if res, err := h.VariableGet(ctx, id); err == nil || res != nil || errors.Is(err, redis.Nil) {
		t.Errorf("VariableGet: Wrong match. expect: decode error, got: %v, %v", res, err)
	}
}

func Test_Set_Del_serverDown(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("9a0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01")
	mr.Close()

	if err := h.FlowSet(ctx, &flow.Flow{Identity: commonidentity.Identity{ID: id}}); err == nil {
		t.Errorf("FlowSet: Wrong match. expect: error, got: nil")
	}
	if err := h.ActiveflowSet(ctx, &activeflow.Activeflow{Identity: commonidentity.Identity{ID: id}}); err == nil {
		t.Errorf("ActiveflowSet: Wrong match. expect: error, got: nil")
	}
	if err := h.VariableSet(ctx, &variable.Variable{ID: id}); err == nil {
		t.Errorf("VariableSet: Wrong match. expect: error, got: nil")
	}
	if err := h.FlowDel(ctx, id); err == nil {
		t.Errorf("FlowDel: Wrong match. expect: error, got: nil")
	}
}

func Test_ActiveflowGetWithLock_ReleaseLock(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	af := &activeflow.Activeflow{
		Identity:      commonidentity.Identity{ID: uuid.FromStringOrNil("aa0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01")},
		ReferenceType: activeflow.ReferenceTypeCall,
	}
	if err := h.ActiveflowSet(ctx, af); err != nil {
		t.Fatalf("Could not prepare the data. err: %v", err)
	}
	lockKey := fmt.Sprintf("activeflow:lock:%s", af.ID)

	res, err := h.ActiveflowGetWithLock(ctx, af.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID != af.ID {
		t.Errorf("Wrong match. expect: %s, got: %s", af.ID, res.ID)
	}
	if !mr.Exists(lockKey) {
		t.Errorf("Wrong match. the lock key must exist while locked. key: %s", lockKey)
	}

	if err := h.ActiveflowReleaseLock(ctx, af.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists(lockKey) {
		t.Errorf("Wrong match. the lock key must be removed after release. key: %s", lockKey)
	}

	// the second release has no mutex to release.
	err = h.ActiveflowReleaseLock(ctx, af.ID)
	if err == nil || err.Error() != "no mutex" {
		t.Errorf("Wrong match. expect: no mutex, got: %v", err)
	}

	// the lock can be taken again after the release.
	if _, err := h.ActiveflowGetWithLock(ctx, af.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if err := h.ActiveflowReleaseLock(ctx, af.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ActiveflowGetWithLock_contention(t *testing.T) {
	if testing.Short() {
		t.Skip("the lock retry takes several seconds")
	}

	h, _ := newTestHandler(t)
	ctx := context.Background()

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("ab0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01")},
	}
	if err := h.ActiveflowSet(ctx, af); err != nil {
		t.Fatalf("Could not prepare the data. err: %v", err)
	}

	if _, err := h.ActiveflowGetWithLock(ctx, af.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the lock is held, so another caller must not get the activeflow.
	res, err := h.ActiveflowGetWithLock(ctx, af.ID)
	if err == nil {
		t.Errorf("Wrong match. expect: lock error, got: nil")
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	if err := h.ActiveflowReleaseLock(ctx, af.ID); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ActiveflowGetWithLock_notCached(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	id := uuid.FromStringOrNil("ac0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01")
	lockKey := fmt.Sprintf("activeflow:lock:%s", id)

	res, err := h.ActiveflowGetWithLock(ctx, id)
	if !errors.Is(err, redis.Nil) {
		t.Errorf("Wrong match. expect: %v, got: %v", redis.Nil, err)
	}
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	// the lock must not be left behind when the data could not be read.
	if mr.Exists(lockKey) {
		t.Errorf("Wrong match. the lock must be released on failure. key: %s", lockKey)
	}
	if _, ok := h.mapMutex[id.String()]; ok {
		t.Errorf("Wrong match. the mutex must not be kept on failure.")
	}

	// so the activeflow is available at once when the data is cached.
	if err := h.ActiveflowSet(ctx, &activeflow.Activeflow{Identity: commonidentity.Identity{ID: id}}); err != nil {
		t.Fatalf("Could not prepare the data. err: %v", err)
	}
	if _, err := h.ActiveflowGetWithLock(ctx, id); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

func Test_ActiveflowReleaseLock_lockExpired(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("ad0c1d3a-6a58-4b0e-8c1f-0a1b2c3d4e01")},
	}
	if err := h.ActiveflowSet(ctx, af); err != nil {
		t.Fatalf("Could not prepare the data. err: %v", err)
	}
	if _, err := h.ActiveflowGetWithLock(ctx, af.ID); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	// the lock disappears on the server side, e.g. by expiration.
	mr.Del(fmt.Sprintf("activeflow:lock:%s", af.ID))

	err := h.ActiveflowReleaseLock(ctx, af.ID)
	if err == nil {
		t.Fatalf("Wrong match. expect: unlock error, got: nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "lock") {
		t.Errorf("Wrong match. expect: lock related error, got: %v", err)
	}

	// the mutex entry is dropped even when the unlock failed.
	err = h.ActiveflowReleaseLock(ctx, af.ID)
	if err == nil || err.Error() != "no mutex" {
		t.Errorf("Wrong match. expect: no mutex, got: %v", err)
	}
}
