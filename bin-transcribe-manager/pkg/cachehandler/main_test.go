package cachehandler

import (
	"testing"
)

func Test_NewHandler(t *testing.T) {
	tests := []struct {
		name string

		addr     string
		password string
		db       int
	}{
		{
			name:     "keeps the connection settings",
			addr:     "127.0.0.1:6390",
			password: "test-password",
			db:       3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := NewHandler(tt.addr, tt.password, tt.db)

			h, ok := res.(*handler)
			if !ok {
				t.Fatalf("Wrong match. expect: *handler, got: %T", res)
			}
			defer func() { _ = h.Cache.Close() }()

			if h.Addr != tt.addr || h.Password != tt.password || h.DB != tt.db {
				t.Errorf("Wrong match. expect: %s/%s/%d, got: %s/%s/%d", tt.addr, tt.password, tt.db, h.Addr, h.Password, h.DB)
			}

			opt := h.Cache.Options()
			if opt.Addr != tt.addr || opt.Password != tt.password || opt.DB != tt.db {
				t.Errorf("Wrong match. expect client options: %s/%s/%d, got: %s/%s/%d", tt.addr, tt.password, tt.db, opt.Addr, opt.Password, opt.DB)
			}
		})
	}
}

func Test_Connect(t *testing.T) {
	t.Run("server reachable", func(t *testing.T) {
		h, _ := newTestCacheHandler(t)

		if err := h.Connect(); err != nil {
			t.Errorf("Wrong match. expect: ok, got: %v", err)
		}
	})

	t.Run("server down", func(t *testing.T) {
		h, mr := newTestCacheHandler(t)
		mr.Close()

		if err := h.Connect(); err == nil {
			t.Errorf("Wrong match. expect: err, got: ok")
		}
	})
}
