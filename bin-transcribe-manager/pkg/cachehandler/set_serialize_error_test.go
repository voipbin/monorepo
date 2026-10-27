package cachehandler

import (
	"context"
	"strings"
	"testing"
)

// Test_setSerialize_unencodableData checks that data that cannot be encoded is reported before
// anything is written to the cache.
func Test_setSerialize_unencodableData(t *testing.T) {
	h, mr := newTestCacheHandler(t)

	err := h.setSerialize(context.Background(), "test-key", make(chan int))
	if err == nil || !strings.Contains(err.Error(), "unsupported type") {
		t.Errorf("Wrong match. expect: encoding error, got: %v", err)
	}
	if mr.Exists("test-key") {
		t.Errorf("Wrong match. expect: nothing stored, but the key exists")
	}
}
