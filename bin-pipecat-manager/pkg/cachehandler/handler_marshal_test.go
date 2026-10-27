package cachehandler

import (
	"context"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

func Test_PipecatcallSet_marshalFailure(t *testing.T) {
	h, mr := newTestHandler(t)

	// the llm messages can not be serialized, so nothing is written to the cache
	pc := &pipecatcall.Pipecatcall{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("9a0b9d52-4c63-4c5b-8f3a-0a7d2b1a0030")},
		LLMMessages: []map[string]any{{"content": make(chan int)}},
	}
	if err := h.PipecatcallSet(context.Background(), pc); err == nil {
		t.Errorf("Wrong match. expect: error, got: nil")
	}

	if len(mr.Keys()) != 0 {
		t.Errorf("Wrong keys. expect: none, got: %v", mr.Keys())
	}
}
