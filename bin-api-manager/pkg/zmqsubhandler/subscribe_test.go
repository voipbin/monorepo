package zmqsubhandler

import (
	reflect "reflect"
	"testing"

	"github.com/pebbe/zmq4"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/zmq"
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
		{
			// Regression test for a production panic (prod api-manager crash-loop,
			// 2026-07-18): slices.Delete(s, idx, 1) is only valid when idx <= 1, because
			// the second argument is an end-index, not a count. Unsubscribing a topic
			// held at index >= 2 previously panicked with
			// "slice bounds out of range [idx:1]". This case deletes the topic at
			// index 4 (5 topics total) to guard against that regression.
			"unsubscribe a topic beyond index 1 (regression: slices.Delete end-index bug)",

			[]string{
				"topic0",
				"topic1",
				"topic2",
				"topic3",
				"topic4",
			},

			"topic4",

			[]string{
				"topic0",
				"topic1",
				"topic2",
				"topic3",
			},
		},
		{
			"unsubscribe a topic in the middle of a longer list",

			[]string{
				"topic0",
				"topic1",
				"topic2",
				"topic3",
				"topic4",
			},

			"topic2",

			[]string{
				"topic0",
				"topic1",
				"topic3",
				"topic4",
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
