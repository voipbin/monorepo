package zmqpubhandler

import (
	"testing"

	"github.com/pebbe/zmq4"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/zmq"
)

func Test_init(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			"normal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockZMQ := zmq.NewMockZMQ(mc)
			h := zmqPubHandler{
				sock: mockZMQ,
			}

			mockZMQ.EXPECT().Bind(zmq4.PUB, sockAddress.Return(nil)

			if err := h.initSock(); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Publish(t *testing.T) {
	tests := []struct {
		name    string
		topic   string
		message string
	}{
		{
			"normal",
			"test_topic",
			"test message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockZMQ := zmq.NewMockZMQ(mc)
			h := zmqPubHandler{
				sock: mockZMQ,
			}

			mockZMQ.EXPECT().Publish(tt.topic, tt.message.Return(nil)

			if err := h.Publish(tt.topic, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
