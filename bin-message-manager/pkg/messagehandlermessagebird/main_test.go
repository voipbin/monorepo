package messagehandlermessagebird

import (
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/pkg/dbhandler"
	"monorepo/bin-message-manager/pkg/requestexternal"
)

func TestNewMessageHandlerMessagebird(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockExternal := requestexternal.NewMockRequestExternal(mc)

	h := NewMessageHandlerMessagebird(mockReq, mockDB, mockExternal)

	if h == nil {
		t.Error("Expected non-nil handler")
	}

	// Verify the handler is properly initialized
	handler, ok := h.(*messageHandlerMessagebird)
	if !ok {
		t.Error("Handler is not of type *messageHandlerMessagebird")
		return
	}

	if handler.reqHandler == nil {
		t.Error("reqHandler is nil")
	}
	if handler.db == nil {
		t.Error("db is nil")
	}
	if handler.requestExternal == nil {
		t.Error("requestExternal is nil")
	}
}
