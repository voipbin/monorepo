package subscribehandler

import (
	"context"
	"errors"
	"sync"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

func Test_processEvent_cov(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event

		expectErr bool
	}{
		{
			name:  "ignored publisher",
			event: &sock.Event{Publisher: "unknown-manager", Type: "whatever"},
		},
		{
			name:  "customer deleted with bad data returns error",
			event: &sock.Event{Publisher: "customer-manager", Type: "customer_deleted", Data: []byte(`{invalid`)},

			expectErr: true,
		},
		{
			name:  "customer frozen with bad data returns error",
			event: &sock.Event{Publisher: "customer-manager", Type: "customer_frozen", Data: []byte(`{invalid`)},

			expectErr: true,
		},
		{
			name:  "activeflow updated with bad data returns error",
			event: &sock.Event{Publisher: "flow-manager", Type: "activeflow_updated", Data: []byte(`{invalid`)},

			expectErr: true,
		},
		{
			name:  "container died with bad data returns error",
			event: &sock.Event{Publisher: "sentinel-manager", Type: "container_died", Data: []byte(`{invalid`)},

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &subscribeHandler{
				sockHandler:       sockhandler.NewMockSockHandler(mc),
				callHandler:       callhandler.NewMockCallHandler(mc),
				groupcallHandler:  groupcallhandler.NewMockGroupcallHandler(mc),
				confbridgeHandler: confbridgehandler.NewMockConfbridgeHandler(mc),
			}

			err := h.processEvent(tt.event)
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventRun_cov(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &subscribeHandler{callHandler: callhandler.NewMockCallHandler(mc)}

	// processEvent errors must be swallowed (message is acked).
	if err := h.processEventRun(&sock.Event{Publisher: "sentinel-manager", Type: "container_died", Data: []byte(`{invalid`)}); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}

	// ignored event.
	if err := h.processEventRun(&sock.Event{Publisher: "x", Type: "y"}); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

func Test_NewSubscribeHandler_cov(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	res := NewSubscribeHandler(sockhandler.NewMockSockHandler(mc), "queue", nil, nil, nil, nil)
	if res == nil {
		t.Errorf("Wrong match. expect: handler, got: nil")
	}
}

func Test_Run_cov(t *testing.T) {
	tests := []struct {
		name string

		createErr    error
		subscribeErr error
		topicErr     error
		bindErr      error

		expectSubscribe bool
		expectTopic     bool
		expectBinds     int
		expectConsume   bool
		expectErr       bool
	}{
		{name: "queue create failure", createErr: errors.New("create"), expectErr: true},
		{name: "queue subscribe failure", subscribeErr: errors.New("subscribe"), expectSubscribe: true, expectErr: true},
		{name: "topic create failure", topicErr: errors.New("topic"), expectSubscribe: true, expectTopic: true, expectErr: true},
		{name: "queue bind failure", bindErr: errors.New("bind"), expectSubscribe: true, expectTopic: true, expectBinds: 1, expectErr: true},
		{name: "all succeed", expectSubscribe: true, expectTopic: true, expectBinds: len(topicPatterns), expectConsume: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &subscribeHandler{sockHandler: mockSock, subscribeQueue: "queue-cov"}

			mockSock.EXPECT().QueueCreate("queue-cov", "normal").Return(tt.createErr)
			if tt.expectSubscribe {
				mockSock.EXPECT().QueueSubscribe("queue-cov", gomock.Any()).Return(tt.subscribeErr)
			}
			if tt.expectTopic {
				mockSock.EXPECT().TopicCreateWithKind(gomock.Any(), "topic").Return(tt.topicErr)
			}
			if tt.expectBinds > 0 {
				mockSock.EXPECT().QueueBind("queue-cov", gomock.Any(), gomock.Any(), false, nil).Return(tt.bindErr).Times(tt.expectBinds)
			}

			var wg sync.WaitGroup
			if tt.expectConsume {
				wg.Add(1)
				mockSock.EXPECT().ConsumeMessage(gomock.Any(), "queue-cov", gomock.Any(), false, false, false, 20, gomock.Any()).DoAndReturn(
					func(context.Context, string, string, bool, bool, bool, int, sock.CbMsgConsume) error {
						wg.Done()
						return nil
					}).Times(1)
			}

			err := h.Run()
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect err: %v, got: %v", tt.expectErr, err)
			}
			if err == nil {
				wg.Wait()
			}
		})
	}
}
