package zmqsubhandler

import (
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"github.com/pebbe/zmq4"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/zmq"
)

func Test_initSock(t *testing.T) {

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

			mockSock.EXPECT().Connect(zmq4.SUB, sockAddress).Return(nil)

			if err := h.initSock(); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_Terminate(t *testing.T) {

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

			mockSock.EXPECT().Terminate()

			h.Terminate()

		})
	}
}

func Test_Subscribe(t *testing.T) {

	tests := []struct {
		name string

		topics []string

		topic string

		duplicated bool

		expectResTopics []string
	}{
		{
			"normal",

			[]string{},

			"test",

			false,

			[]string{
				"test",
			},
		},
		{
			"subscribe the same topic",

			[]string{"hello"},

			"hello",

			true,

			[]string{
				"hello",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := zmq.NewMockZMQ(mc)

			h := &zmqSubHandler{
				sock:   mockSock,
				topics: tt.topics,
			}

			if !tt.duplicated {
				mockSock.EXPECT().Subscribe(tt.topic).Return(nil)
			}

			if err := h.Subscribe(tt.topic); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(h.topics, tt.expectResTopics) {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectResTopics, h.topics)
			}

		})
	}
}

func Test_Unsubscribe(t *testing.T) {

	tests := []struct {
		name string

		topics []string

		topic string

		expectResTopics []string
	}{
		{
			"normal",

			[]string{
				"test",
			},

			"test",

			[]string{},
		},
		{
			"have 2 items",

			[]string{
				"hello",
				"world",
			},

			"hello",

			[]string{
				"world",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := zmq.NewMockZMQ(mc)

			h := &zmqSubHandler{
				sock:   mockSock,
				topics: tt.topics,
			}

			mockSock.EXPECT().Unsubscribe(tt.topic).Return(nil)

			if err := h.Unsubscribe(tt.topic); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(h.topics, tt.expectResTopics) {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectResTopics, h.topics)
			}

		})
	}
}
