package cachehandler

import (
	"context"
	"testing"
)

func Test_setSerialize_unencodableValue(t *testing.T) {
	h, mr := newTestHandler(t)
	ctx := context.Background()

	// a channel cannot be encoded, so nothing may be written to the cache
	if err := h.setSerialize(ctx, "queue:test:unencodable", make(chan int)); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}
	if mr.Exists("queue:test:unencodable") {
		t.Errorf("Wrong match. the key must not be written")
	}
}
