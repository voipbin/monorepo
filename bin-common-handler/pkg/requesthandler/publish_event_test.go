package requesthandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/golang/mock/gomock"
)

func Test_CallPublishEvent(t *testing.T) {

	tests := []struct {
		name string

		eventType string
		publisher string
		dataType  string
		data      []byte

		expectTarget string
		expectEvent  *sock.Event
	}{
		{
			name: "normal",

			eventType: "test_event",
			publisher: "test-manager",
			dataType:  "application/json",
			data:      []byte(`{"key":"value"}`),

			expectTarget: "bin-manager.call-manager.subscribe",
			expectEvent: &sock.Event{
				Type:      "test_event",
				Publisher: "test-manager",
				DataType:  "application/json",
				Data:      []byte(`{"key":"value"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().EventPublish("", tt.expectTarget, tt.expectEvent).Return(nil)

			if err := reqHandler.CallPublishEvent(ctx, tt.eventType, tt.publisher, tt.dataType, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
