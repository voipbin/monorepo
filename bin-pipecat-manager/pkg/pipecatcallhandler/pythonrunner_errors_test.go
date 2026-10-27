package pipecatcallhandler

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
)

func Test_pythonRunner_requestBuildErrors(t *testing.T) {
	id := uuid.FromStringOrNil("a1b2c3d4-6401-4000-8000-000000000001")
	h := &pythonRunner{}

	t.Run("start with llm messages that json can not encode", func(t *testing.T) {
		messages := []map[string]any{{"content": make(chan int)}}

		err := h.Start(context.Background(), id, "openai.gpt-5", "", messages, "", "", "", "", "", nil, nil, nil, false, uuid.Nil, "", "")
		if err == nil || !strings.Contains(err.Error(), "could not marshal request body for python runner") {
			t.Errorf("Wrong error: %v", err)
		}
	})

	// a nil context is rejected while the request is built, before any connection is made.
	var noCtx context.Context

	t.Run("start without a context", func(t *testing.T) {
		err := h.Start(noCtx, id, "openai.gpt-5", "", nil, "", "", "", "", "", nil, nil, nil, false, uuid.Nil, "", "")
		if err == nil || !strings.Contains(err.Error(), "could not create request for python runner") {
			t.Errorf("Wrong error: %v", err)
		}
	})

	t.Run("stop without a context", func(t *testing.T) {
		err := h.Stop(noCtx, id)
		if err == nil || !strings.Contains(err.Error(), "could not create stop request for python runner") {
			t.Errorf("Wrong error: %v", err)
		}
	})
}
