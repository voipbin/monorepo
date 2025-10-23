package pipecatcallhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func Test_runKeepAlive(t *testing.T) {
	tests := []struct {
		name        string
		interval    time.Duration
		streamingID uuid.UUID
	}{
		{
			name:        "send keep-alive once",
			interval:    10 * time.Millisecond,
			streamingID: uuid.FromStringOrNil("10c6616e-af26-11f0-9407-e352eaba2dd0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			h := &pipecatcallHandler{}

			conn := &DummyConn{}
			h.runKeepAlive(ctx, conn, tt.interval, tt.streamingID)

			expectMessage := []byte{0x10, 0x00, 0x01, 0x00}
			if !reflect.DeepEqual(conn.Written[0], expectMessage) {
				t.Errorf("KeepAlive message mismatch.\nexpect: %v\ngot:    %v", expectMessage, conn.Written)
			}

			if len(conn.Written) == 0 {
				t.Errorf("No keep-alive messages were written to DummyConn")
			}
		})
	}
}
