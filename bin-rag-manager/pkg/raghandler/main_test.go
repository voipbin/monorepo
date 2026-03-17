package raghandler

import (
	"testing"
)

func TestRagHandler_Interface(t *testing.T) {
	var _ RagHandler = &ragHandler{}
}

func TestNewRagHandler(t *testing.T) {
	h := NewRagHandler(nil, nil)
	if h == nil {
		t.Error("expected non-nil handler")
	}
}
