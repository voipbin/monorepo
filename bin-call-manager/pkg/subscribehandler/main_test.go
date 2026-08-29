package subscribehandler

import (
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run verifies the full Run() sequencing (VOIP-1407): queue create -> the retained
// asterisk fanout subscribe -> global topic exchange declare -> all pattern binds ->
// ConsumeMessage. The asterisk.all.event QueueSubscribe is the one fanout leg that
// survives the VOIP-1407 cutover (design §3.2) -- it feeds from voip-asterisk-proxy,
// which is excluded from this ticket's scope.
func Test_Run(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	subscribeQueue := commonoutline.QueueNameCallSubscribe

	h := subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: subscribeQueue,
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
		mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), string(commonoutline.QueueNameAsteriskEventAll)).Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
	}
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(string(subscribeQueue), pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}
	gomock.InOrder(calls...)

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), string(subscribeQueue), gomock.Any(), false, false, false, 20, gomock.Any()).Return(nil).AnyTimes()

	if err := h.Run(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_Run_QueueCreateError verifies Run() returns the error immediately when the
// subscribe queue declare fails, without calling anything downstream.
func Test_Run_QueueCreateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	subscribeQueue := commonoutline.QueueNameCallSubscribe

	h := subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: subscribeQueue,
	}

	mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(fmt.Errorf("could not declare the queue"))

	if err := h.Run(); err == nil {
		t.Error("Wrong match. expect: error, got: ok")
	}
}

// Test_Run_AsteriskSubscribeError verifies Run() returns the error immediately when the
// retained asterisk fanout subscribe fails (VOIP-1407 design §3.3: fatal, no fallback).
func Test_Run_AsteriskSubscribeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	subscribeQueue := commonoutline.QueueNameCallSubscribe

	h := subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: subscribeQueue,
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
		mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), string(commonoutline.QueueNameAsteriskEventAll)).Return(fmt.Errorf("could not subscribe the asterisk fanout exchange")),
	}
	gomock.InOrder(calls...)

	if err := h.Run(); err == nil {
		t.Error("Wrong match. expect: error, got: ok")
	}
}

// Test_Run_topicDeclareFails verifies Run() returns the error immediately when the
// global topic exchange declare fails -- VOIP-1407 removed the "stay fully on fanout"
// degrade path, since there is no fanout fallback left once the generic fanout target
// loop is gone.
func Test_Run_topicDeclareFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	subscribeQueue := commonoutline.QueueNameCallSubscribe

	h := subscribeHandler{
		sockHandler:    mockSock,
		subscribeQueue: subscribeQueue,
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
		mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), string(commonoutline.QueueNameAsteriskEventAll)).Return(nil),
		mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("could not declare the topic exchange")),
	}
	gomock.InOrder(calls...)

	if err := h.Run(); err == nil {
		t.Error("Wrong match. expect: error, got: ok")
	}
}

// Test_Run_topicBindFails verifies Run() returns the error immediately when a
// topicPatterns bind fails -- VOIP-1407 removed the "roll back partial binds, stay on
// fanout" machinery entirely, since there is no fanout fallback left to protect.
func Test_Run_topicBindFails(t *testing.T) {

	tests := []struct {
		name string

		failIndex int
	}{
		{
			name: "first bind fails",

			failIndex: 0,
		},
		{
			name: "third bind fails",

			failIndex: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			subscribeQueue := commonoutline.QueueNameCallSubscribe

			h := subscribeHandler{
				sockHandler:    mockSock,
				subscribeQueue: subscribeQueue,
			}

			calls := []any{
				mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
				mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), string(commonoutline.QueueNameAsteriskEventAll)).Return(nil),
				mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil),
			}

			// binds 0..failIndex-1 succeed, bind failIndex fails, and NOTHING is rolled
			// back (no more roll-back-on-partial-failure machinery).
			for i := 0; i < tt.failIndex; i++ {
				calls = append(calls, mockSock.EXPECT().QueueBind(string(subscribeQueue), topicPatterns[i], string(commonoutline.QueueNameEvent), false, nil).Return(nil))
			}
			calls = append(calls, mockSock.EXPECT().QueueBind(string(subscribeQueue), topicPatterns[tt.failIndex], string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("could not bind the pattern")))
			gomock.InOrder(calls...)

			if err := h.Run(); err == nil {
				t.Error("Wrong match. expect: error, got: ok")
			}
		})
	}
}
