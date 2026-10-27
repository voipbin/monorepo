package cachehandler

import (
	"context"
	"testing"
)

// Test_setSerialize_unencodableValue checks that a value that cannot be encoded
// as JSON is rejected before anything is sent to the cache.
func Test_setSerialize_unencodableValue(t *testing.T) {
	f := newFakeRedis(t)
	h := newTestHandler(t, f)

	err := h.setSerialize(context.Background(), "agent:test", make(chan int))
	if err == nil {
		t.Fatalf("Wrong match. expect an error, got nil")
	}
	if got := f.commandsNamed("SET"); len(got) != 0 {
		t.Errorf("Wrong match. expect no SET command, got: %v", got)
	}
	if _, ok := f.get("agent:test"); ok {
		t.Errorf("Wrong match. expect nothing to be stored")
	}
}
