package streaminghandler

import "testing"

func Test_Run_NoOp(t *testing.T) {
	h := &streamingHandler{}
	if err := h.Run(); err != nil {
		t.Errorf("Run() should be a no-op, got error: %v", err)
	}
}
