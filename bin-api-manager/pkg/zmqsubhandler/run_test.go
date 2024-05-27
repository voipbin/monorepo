package zmqsubhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	"github.com/pebbe/zmq4"

	"monorepo/bin-api-manager/pkg/zmq"
)

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
		{
			"500 times",

			500,
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
			ctx, cancel := context.WithCancel(context.Background())

			for i := 0; i < tt.count; i++ {
				topic := fmt.Sprintf("topic: %d", i)
				message := fmt.Sprintf("message: %d", i)
				mockSock.EXPECT().ReceiveNoBlock().Return([]string{topic, message}, nil).AnyTimes()
			}

			go func() {
				topic, message, err := h.receiveMessage(ctx)
				if err != nil {
					if err.Error() != "" {
						t.Errorf("Wrong match. expect: ok, got: %v", err)
					}

					return
				}
				t.Logf("Received message. topic: %s, message: %v", topic, message)
			}()

			time.Sleep(time.Millisecond * 300)

			cancel()
		})
	}
}

func Test_recevieMessage_socket_closed(t *testing.T) {

	tests := []struct {
		name string

		count int
	}{
		{
			"socket close while receiving the message",

			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// init socket publish
			sockPub := zmq.NewZMQ()
			if errBind := sockPub.Bind(zmq4.PUB, sockAddress); errBind != nil {
				t.Errorf("Could not bind the zmq socket. err: %v", errBind)
			}
			defer sockPub.Terminate()
			ctx := context.Background()

			// init socket subscribe
			sockSub := zmq.NewZMQ()
			if errConnect := sockSub.Connect(zmq4.SUB, sockAddress); errConnect != nil {
				t.Errorf("Could not connect the zmq socket. err: %v", errConnect)
			}

			h := &zmqSubHandler{
				sock: sockSub,
			}
			if errSubscribe := h.Subscribe("topic"); errSubscribe != nil {
				t.Errorf("Could not subscribe the zmq socket. err: %v", errSubscribe)
			}

			t.Logf("Receiving the message.")
			go func() {
				t.Logf("Receiving message")
				time.Sleep(time.Millisecond * 1000)

				_, _, err := h.receiveMessage(ctx)
				if err != nil {
					if !strings.Contains(err.Error(), "Socket is closed") {
						t.Errorf("Wrong match. expect: %s, got: %v", "Socket is closed", err)
					}
					return
				}
			}()

			time.Sleep(time.Millisecond * 1000)

			t.Logf("Publishing the message.")
			go func() {
				for i := 0; i < tt.count; i++ {
					topic := fmt.Sprintf("topic: %d", i)
					message := fmt.Sprintf("message: %d", i)
					if errPub := sockPub.Publish(topic, message); errPub != nil {
						t.Errorf("Could not publish the zmq socket. err: %v", errPub)
					}
					time.Sleep(time.Millisecond * 300)
				}

			}()

			time.Sleep(time.Millisecond * 5000)

			// close the socket
			t.Logf("Closing the socket.")
			sockSub.Terminate()

			time.Sleep(time.Millisecond * 1000)
		})
	}
}

func Test_recevieMessage_context_canceled(t *testing.T) {

	tests := []struct {
		name string

		count int
	}{
		{
			"context has canceled while receiving the message",

			5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// init socket publish
			sockPub := zmq.NewZMQ()
			if errBind := sockPub.Bind(zmq4.PUB, sockAddress); errBind != nil {
				t.Errorf("Could not bind the zmq socket. err: %v", errBind)
			}
			ctx, cancel := context.WithCancel(context.Background())

			// init socket subscribe
			sockSub := zmq.NewZMQ()
			if errConnect := sockSub.Connect(zmq4.SUB, sockAddress); errConnect != nil {
				t.Errorf("Could not connect the zmq socket. err: %v", errConnect)
			}

			h := &zmqSubHandler{
				sock: sockSub,
			}
			if errSubscribe := h.Subscribe("topic"); errSubscribe != nil {
				t.Errorf("Could not subscribe the zmq socket. err: %v", errSubscribe)
			}

			t.Logf("Receiving the message.")
			go func() {
				t.Logf("Receiving message")
				time.Sleep(time.Millisecond * 1000)

				_, _, err := h.receiveMessage(ctx)
				if err != nil {
					if !strings.Contains(err.Error(), "context canceled") {
						t.Errorf("Wrong match. expect: %s, got: %v", "context canceled", err)
					}
					return
				}
			}()

			time.Sleep(time.Millisecond * 1000)

			t.Logf("Publishing the message.")
			go func() {
				for i := 0; i < tt.count; i++ {
					topic := fmt.Sprintf("topic: %d", i)
					message := fmt.Sprintf("message: %d", i)
					if errPub := sockPub.Publish(topic, message); errPub != nil {
						t.Errorf("Could not publish the zmq socket. err: %v", errPub)
					}
					time.Sleep(time.Millisecond * 300)
				}

			}()

			time.Sleep(time.Millisecond * 5000)

			// cancel the context
			t.Logf("Closing the socket.")
			cancel()

			time.Sleep(time.Millisecond * 1000)
		})
	}
}
