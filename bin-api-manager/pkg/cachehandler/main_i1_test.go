package cachehandler

import (
	"reflect"
	"testing"
)

func Test_NewHandler(t *testing.T) {
	res := NewHandler("127.0.0.1:6379", "secret", 3)

	h, ok := res.(*handler)
	if !ok {
		t.Fatalf("expected *handler, got %T", res)
	}
	t.Cleanup(func() { _ = h.Cache.Close() })

	if h.Addr != "127.0.0.1:6379" || h.Password != "secret" || h.DB != 3 {
		t.Errorf("wrong fields. addr: %s, password: %s, db: %d", h.Addr, h.Password, h.DB)
	}
	if h.Cache == nil {
		t.Fatalf("expected redis client to be set")
	}
	opts := h.Cache.Options()
	if !reflect.DeepEqual([]interface{}{opts.Addr, opts.Password, opts.DB}, []interface{}{"127.0.0.1:6379", "secret", 3}) {
		t.Errorf("wrong redis options. addr: %s, db: %d", opts.Addr, opts.DB)
	}
}

func Test_Connect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h, _ := newTestHandler(t)
		if err := h.Connect(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("server closed returns error", func(t *testing.T) {
		h, mr := newTestHandler(t)
		mr.Close()
		if err := h.Connect(); err == nil {
			t.Errorf("expected connect error, got nil")
		}
	})
}
