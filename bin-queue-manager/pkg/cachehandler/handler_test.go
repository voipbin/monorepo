package cachehandler

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
)

const testErrTimeout = 10 * time.Second

func newTestHandler(t *testing.T) (*handler, *miniredis.Miniredis) {
	t.Helper()

	mr := miniredis.RunT(t)
	h, ok := NewHandler(mr.Addr(), "", 0).(*handler)
	if !ok {
		t.Fatalf("unexpected handler type")
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	return h, mr
}

func Test_NewHandler_Connect(t *testing.T) {
	h, mr := newTestHandler(t)

	if err := h.Connect(); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if h.Addr != mr.Addr() || h.DB != 0 {
		t.Errorf("Wrong match. addr: %s, db: %d", h.Addr, h.DB)
	}

	mr.Close()

	done := make(chan error, 1)
	go func() { done <- h.Connect() }()
	select {
	case err := <-done:
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	case <-time.After(testErrTimeout):
		t.Fatalf("Connect did not return in time")
	}
}

func Test_QueueSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	q := &queue.Queue{Name: "test queue", Detail: "test detail", WaitTimeout: 6000, ServiceTimeout: 60000}
	q.ID = uuid.FromStringOrNil("2b3c4d5e-0001-11f1-9a00-000000000001")

	if err := h.QueueSet(ctx, q); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "queue:queue:2b3c4d5e-0001-11f1-9a00-000000000001"
	if !mr.Exists(key) {
		t.Fatalf("Wrong match. key not stored: %s", key)
	}
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect: 24h, got: %v", ttl)
	}

	res, err := h.QueueGet(ctx, q.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID.String() != "2b3c4d5e-0001-11f1-9a00-000000000001" || res.Name != "test queue" || res.Detail != "test detail" || res.WaitTimeout != 6000 || res.ServiceTimeout != 60000 {
		t.Errorf("Wrong match. got: %+v", res)
	}
}

func Test_QueueGet_error(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("2b3c4d5e-0002-11f1-9a00-000000000002")

	// missing key
	if res, err := h.QueueGet(ctx, id); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
	}

	// corrupted value
	if err := mr.Set("queue:queue:2b3c4d5e-0002-11f1-9a00-000000000002", "{not json"); err != nil {
		t.Fatalf("could not prepare: %v", err)
	}
	if res, err := h.QueueGet(ctx, id); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
	}
}

func Test_QueuecallSetGet(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	qc := &queuecall.Queuecall{Status: queuecall.StatusWaiting}
	qc.ID = uuid.FromStringOrNil("2b3c4d5e-0003-11f1-9a00-000000000003")
	qc.ReferenceID = uuid.FromStringOrNil("2b3c4d5e-0004-11f1-9a00-000000000004")

	if err := h.QueuecallSet(ctx, qc); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	key := "queue:queuecall:2b3c4d5e-0003-11f1-9a00-000000000003"
	if ttl := mr.TTL(key); ttl != 24*time.Hour {
		t.Errorf("Wrong match. expect: 24h, got: %v", ttl)
	}

	res, err := h.QueuecallGet(ctx, qc.ID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID.String() != "2b3c4d5e-0003-11f1-9a00-000000000003" || res.ReferenceID.String() != "2b3c4d5e-0004-11f1-9a00-000000000004" || res.Status != queuecall.StatusWaiting {
		t.Errorf("Wrong match. got: %+v", res)
	}

	keys := mr.Keys()
	if len(keys) != 1 || keys[0] != key {
		t.Errorf("Wrong match. expect: only %s, got: %v", key, keys)
	}
}

func Test_QueuecallGet_error(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	id := uuid.FromStringOrNil("2b3c4d5e-0005-11f1-9a00-000000000005")

	if res, err := h.QueuecallGet(ctx, id); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
	}

	if err := mr.Set("queue:queuecall:2b3c4d5e-0005-11f1-9a00-000000000005", "[1,2"); err != nil {
		t.Fatalf("could not prepare: %v", err)
	}
	if res, err := h.QueuecallGet(ctx, id); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
	}
}

func Test_QueuecallGetByReferenceID(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()
	refID := uuid.FromStringOrNil("2b3c4d5e-0006-11f1-9a00-000000000006")

	if res, err := h.QueuecallGetByReferenceID(ctx, refID); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
	}

	key := "queue:queuecall:reference_id:2b3c4d5e-0006-11f1-9a00-000000000006"
	if err := mr.Set(key, `{"id":"2b3c4d5e-0007-11f1-9a00-000000000007","status":"connecting"}`); err != nil {
		t.Fatalf("could not prepare: %v", err)
	}
	res, err := h.QueuecallGetByReferenceID(ctx, refID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res.ID.String() != "2b3c4d5e-0007-11f1-9a00-000000000007" || res.Status != queuecall.StatusConnecting {
		t.Errorf("Wrong match. got: %+v", res)
	}

	if err := mr.Set(key, "oops"); err != nil {
		t.Fatalf("could not prepare: %v", err)
	}
	if res, err := h.QueuecallGetByReferenceID(ctx, refID); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error and nil, got: %v, %v", res, err)
	}
}

func Test_SetServerDown(t *testing.T) {
	h, mr := newTestHandler(t)
	mr.Close()

	ctx, cancel := context.WithTimeout(context.Background(), testErrTimeout)
	defer cancel()

	q := &queue.Queue{}
	q.ID = uuid.FromStringOrNil("2b3c4d5e-0008-11f1-9a00-000000000008")
	qc := &queuecall.Queuecall{}
	qc.ID = uuid.FromStringOrNil("2b3c4d5e-0009-11f1-9a00-000000000009")

	if err := h.QueueSet(ctx, q); err == nil {
		t.Errorf("QueueSet: expect: error, got: nil")
	}
	if err := h.QueuecallSet(ctx, qc); err == nil {
		t.Errorf("QueuecallSet: expect: error, got: nil")
	}
	if _, err := h.QueueGet(ctx, q.ID); err == nil {
		t.Errorf("QueueGet: expect: error, got: nil")
	}
	if _, err := h.QueuecallGet(ctx, qc.ID); err == nil {
		t.Errorf("QueuecallGet: expect: error, got: nil")
	}
	if _, err := h.ReconcileLockAcquire(ctx, "instance-a", time.Minute); err == nil {
		t.Errorf("ReconcileLockAcquire: expect: error, got: nil")
	}
	if err := h.ReconcileLockRelease(ctx); err == nil {
		t.Errorf("ReconcileLockRelease: expect: error, got: nil")
	}
}

func Test_ReconcileLock(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	ok, err := h.ReconcileLockAcquire(ctx, "instance-a", 35*time.Second)
	if err != nil || !ok {
		t.Fatalf("Wrong match. expect: true, nil, got: %v, %v", ok, err)
	}
	if v, _ := mr.Get("queue:reconcile:match"); v != "instance-a" {
		t.Errorf("Wrong match. expect: instance-a, got: %s", v)
	}
	if ttl := mr.TTL("queue:reconcile:match"); ttl != 35*time.Second {
		t.Errorf("Wrong match. expect: 35s, got: %v", ttl)
	}

	// held by another replica
	ok, err = h.ReconcileLockAcquire(ctx, "instance-b", 35*time.Second)
	if err != nil || ok {
		t.Errorf("Wrong match. expect: false, nil, got: %v, %v", ok, err)
	}
	if v, _ := mr.Get("queue:reconcile:match"); v != "instance-a" {
		t.Errorf("Wrong match. the holder must not change, got: %s", v)
	}

	// expired
	mr.FastForward(36 * time.Second)
	ok, err = h.ReconcileLockAcquire(ctx, "instance-b", 35*time.Second)
	if err != nil || !ok {
		t.Errorf("Wrong match. expect: true, nil, got: %v, %v", ok, err)
	}
	if v, _ := mr.Get("queue:reconcile:match"); v != "instance-b" {
		t.Errorf("Wrong match. expect: instance-b, got: %s", v)
	}

	// release
	if err := h.ReconcileLockRelease(ctx); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
	if mr.Exists("queue:reconcile:match") {
		t.Errorf("Wrong match. the lock must be removed")
	}
}
