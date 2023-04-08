package zmqsubhandler

import (
	"context"
	"fmt"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmq"
)

func Test_Run(t *testing.T) {

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

			mockSock := zmq.NewMockZMQ(mc)

			h := &zmqSubHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().Receive().Return([]string{"topic", "message"}, nil).AnyTimes()

			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				if errListen := h.Run(ctx, cancel, func(topic, message string) error {
					return nil
				}); errListen != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", errListen)
				}
			}()

			time.Sleep(1 * time.Second)

			cancel()

			time.Sleep(1 * time.Second)
		})
	}
}

func Test_recevieMessage(t *testing.T) {

	tests := []struct {
		name string

		count int
	}{
		{
			"10 times",

			10,
		},
		{
			"50 times",

			50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := zmq.NewMockZMQ(mc)

			h := &zmqSubHandler{
				sock: mockSock,
			}

			chanMessage := make(chan subMessage)

			for i := 0; i < tt.count; i++ {
				topic := fmt.Sprintf("topic: %d", i)
				message := fmt.Sprintf("message: %d", i)
				mockSock.EXPECT().Receive().Return([]string{topic, message}, nil)
			}
			mockSock.EXPECT().Receive().Return(nil, fmt.Errorf(""))

			go func() {
				if errRecv := h.recevieMessage(chanMessage); errRecv != nil {
					if errRecv.Error() != "" {
						t.Errorf("Wrong match. expect: ok, got: %v", errRecv)
					}

					return
				}
			}()

			time.Sleep(2 * time.Second)

			for i := 0; i < tt.count; i++ {
				<-chanMessage
			}
		})
	}
}
