package cachehandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func Test_setSerialize_unmarshalable_value(t *testing.T) {
	// the marshal step fails before the redis client is touched, so no server is needed
	h := &handler{}

	err := h.setSerialize(context.Background(), "test-key", make(chan int))

	var typeErr *json.UnsupportedTypeError
	if !errors.As(err, &typeErr) {
		t.Errorf("Wrong match. expect: json.UnsupportedTypeError, got: %v", err)
	}
}
