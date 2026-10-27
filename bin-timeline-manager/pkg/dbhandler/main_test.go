package dbhandler

import (
	"context"
	"testing"
	"time"
)

func Test_WaitForConnection_Connected(t *testing.T) {
	h := &dbHandler{conn: &fakeConn{}}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := h.WaitForConnection(ctx); err != nil {
		t.Errorf("WaitForConnection() = %v, want nil", err)
	}
}
