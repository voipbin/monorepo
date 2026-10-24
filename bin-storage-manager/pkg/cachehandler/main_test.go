package cachehandler

import (
	"errors"
	"testing"
)

func Test_NewHandler(t *testing.T) {
	res, ok := NewHandler("127.0.0.1:0", "pw", 3).(*handler)
	if !ok {
		t.Fatalf("Wrong match. expect: *handler")
	}
	t.Cleanup(func() { _ = res.Cache.Close() })

	if res.Addr != "127.0.0.1:0" || res.Password != "pw" || res.DB != 3 {
		t.Errorf("Wrong match. got: %s/%s/%d", res.Addr, res.Password, res.DB)
	}
	if res.Cache == nil || res.Locker == nil || res.mapMutex == nil {
		t.Errorf("Wrong match. expect: initialized cache, locker and mutex map")
	}

	opts := res.Cache.Options()
	if opts.Addr != "127.0.0.1:0" || opts.Password != "pw" || opts.DB != 3 {
		t.Errorf("Wrong match. got options: %s/%s/%d", opts.Addr, opts.Password, opts.DB)
	}
}

func Test_Connect(t *testing.T) {
	t.Run("ping ok", func(t *testing.T) {
		h, _ := newTestHandler(t)

		if err := h.Connect(); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("ping failed", func(t *testing.T) {
		h, f := newTestHandler(t)
		errPing := errors.New("simulated ping failure")
		f.failErr = errPing

		if err := h.Connect(); !errors.Is(err, errPing) {
			t.Errorf("Wrong match. expect: %v, got: %v", errPing, err)
		}
	})
}
