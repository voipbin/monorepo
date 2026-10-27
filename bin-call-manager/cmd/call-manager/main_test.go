package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

func Test_runSubscribe(t *testing.T) {
	tests := []struct {
		name string

		queueCreateErr error

		expectErr string
	}{
		{
			name:           "queue create failure is reported",
			queueCreateErr: fmt.Errorf("broker unavailable"),
			expectErr:      "could not run the ari event listen handler correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			// the subscribe queue must be declared first, and nothing else may be attempted after a failure
			mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameCallSubscribe), "normal").Return(tt.queueCreateErr)

			err := runSubscribe(
				mockSock,
				arieventhandler.NewMockARIEventHandler(mc),
				callhandler.NewMockCallHandler(mc),
				groupcallhandler.NewMockGroupcallHandler(mc),
				confbridgehandler.NewMockConfbridgeHandler(mc),
			)
			if err == nil || !strings.Contains(err.Error(), tt.expectErr) || !strings.Contains(err.Error(), "broker unavailable") {
				t.Errorf("Wrong match. expect error containing: %q and the cause, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_runRequestListen(t *testing.T) {
	newHandlers := func(mc *gomock.Controller) (callhandler.CallHandler, confbridgehandler.ConfbridgeHandler, channelhandler.ChannelHandler, recordinghandler.RecordingHandler, externalmediahandler.ExternalMediaHandler, groupcallhandler.GroupcallHandler, outboundconfighandler.OutboundConfigHandler) {
		return callhandler.NewMockCallHandler(mc),
			confbridgehandler.NewMockConfbridgeHandler(mc),
			channelhandler.NewMockChannelHandler(mc),
			recordinghandler.NewMockRecordingHandler(mc),
			externalmediahandler.NewMockExternalMediaHandler(mc),
			groupcallhandler.NewMockGroupcallHandler(mc),
			outboundconfighandler.NewMockOutboundConfigHandler(mc)
	}

	t.Run("queue create failure is reported", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameCallRequest), "normal").Return(fmt.Errorf("broker unavailable"))

		call, conf, channel, rec, ext, group, outbound := newHandlers(mc)
		err := runRequestListen(mockSock, call, conf, channel, rec, ext, group, outbound)
		if err == nil || !strings.Contains(err.Error(), "could not run the listenhandler correctly") || !strings.Contains(err.Error(), "broker unavailable") {
			t.Errorf("Wrong match. expect: listen handler error with the cause, got: %v", err)
		}
	})

	t.Run("consumer is started on the request queue", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		consumed := make(chan string, 1)

		mockSock.EXPECT().QueueCreate(string(commonoutline.QueueNameCallRequest), "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), string(commonoutline.QueueNameCallRequest), gomock.Any(), false, false, false, 10, gomock.Any()).
			DoAndReturn(func(_ context.Context, queue, _ string, _, _, _ bool, _ int, _ any) error {
				consumed <- queue
				return nil
			})

		call, conf, channel, rec, ext, group, outbound := newHandlers(mc)
		if err := runRequestListen(mockSock, call, conf, channel, rec, ext, group, outbound); err != nil {
			t.Fatalf("Wrong match. expect: ok, got: %v", err)
		}

		select {
		case queue := <-consumed:
			if queue != string(commonoutline.QueueNameCallRequest) {
				t.Errorf("Wrong match. expect: %s, got: %s", commonoutline.QueueNameCallRequest, queue)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("Wrong match. expect: consumer started, got: timeout")
		}
	})
}

func Test_signalHandler(t *testing.T) {
	// drain leftovers so the test does not depend on other tests
	for len(chDone) > 0 {
		<-chDone
	}

	chSigs <- syscallSignalTerm{}

	done := make(chan struct{})
	go func() {
		signalHandler()
		close(done)
	}()

	select {
	case v := <-chDone:
		if !v {
			t.Errorf("Wrong match. expect: true, got: false")
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect: done notification, got: timeout")
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatalf("Wrong match. expect: signalHandler returned, got: timeout")
	}
}

// syscallSignalTerm is a signal value that is sent through the signal channel without a real OS signal.
type syscallSignalTerm struct{}

func (syscallSignalTerm) String() string { return "terminated" }
func (syscallSignalTerm) Signal()        {}
