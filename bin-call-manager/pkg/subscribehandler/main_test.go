package subscribehandler

import (
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run verifies the full Run() sequencing (VOIP-1406): queue create -> sentinel
// defensive declare + fanout subscribes -> global topic exchange declare -> all pattern
// binds -> fanout unbinds. It also verifies that Run() declares the sentinel-manager
// event exchange before binding to it: sentinel-manager is Kubernetes-only, so in
// non-Kubernetes deployments nothing else declares that exchange and the bind would
// otherwise fail with an AMQP 404, taking this service down at boot.
func Test_Run(t *testing.T) {

	tests := []struct {
		name string

		subscribeQueue   commonoutline.QueueName
		subscribeTargets []string
	}{
		{
			name: "normal",

			subscribeQueue: commonoutline.QueueNameCallSubscribe,
			subscribeTargets: []string{
				string(commonoutline.QueueNameAsteriskEventAll),
				string(commonoutline.QueueNameCustomerEvent),
				string(commonoutline.QueueNameFlowEvent),
				string(commonoutline.QueueNameSentinelEvent),
			},
		},
		{
			name: "no sentinel target",

			subscribeQueue: commonoutline.QueueNameCallSubscribe,
			subscribeTargets: []string{
				string(commonoutline.QueueNameAsteriskEventAll),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := subscribeHandler{
				sockHandler:      mockSock,
				subscribeQueue:   tt.subscribeQueue,
				subscribeTargets: tt.subscribeTargets,
			}

			calls := []any{
				mockSock.EXPECT().QueueCreate(string(tt.subscribeQueue), "normal").Return(nil),
			}
			for _, target := range tt.subscribeTargets {
				if target == string(commonoutline.QueueNameSentinelEvent) {
					calls = append(calls, mockSock.EXPECT().TopicCreate(target).Return(nil))
				}
				calls = append(calls, mockSock.EXPECT().QueueSubscribe(string(tt.subscribeQueue), target).Return(nil))
			}

			// VOIP-1406 migration block: declare -> bind all patterns -> unbind all fanouts
			calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
			for _, pattern := range topicPatterns {
				calls = append(calls, mockSock.EXPECT().QueueBind(string(tt.subscribeQueue), pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
			}
			for _, target := range fanoutUnbindTargets {
				calls = append(calls, mockSock.EXPECT().QueueUnbind(string(tt.subscribeQueue), "", target, nil).Return(nil))
			}
			gomock.InOrder(calls...)

			mockSock.EXPECT().ConsumeMessage(gomock.Any(), string(tt.subscribeQueue), gomock.Any(), false, false, false, 20, gomock.Any()).Return(nil).AnyTimes()

			if err := h.Run(); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Run_error(t *testing.T) {

	tests := []struct {
		name string

		subscribeQueue   commonoutline.QueueName
		subscribeTargets []string

		responseTopicCreate error
		responseSubscribe   error
	}{
		{
			name: "topic create failed for the sentinel target",

			subscribeQueue: commonoutline.QueueNameCallSubscribe,
			subscribeTargets: []string{
				string(commonoutline.QueueNameSentinelEvent),
			},

			responseTopicCreate: fmt.Errorf("could not create the topic"),
		},
		{
			name: "subscribe failed for a non-sentinel target",

			subscribeQueue: commonoutline.QueueNameCallSubscribe,
			subscribeTargets: []string{
				string(commonoutline.QueueNameAsteriskEventAll),
			},

			responseSubscribe: fmt.Errorf("could not subscribe the target"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			h := subscribeHandler{
				sockHandler:      mockSock,
				subscribeQueue:   tt.subscribeQueue,
				subscribeTargets: tt.subscribeTargets,
			}

			mockSock.EXPECT().QueueCreate(string(tt.subscribeQueue), "normal").Return(nil)
			for _, target := range tt.subscribeTargets {
				if target == string(commonoutline.QueueNameSentinelEvent) {
					mockSock.EXPECT().TopicCreate(target).Return(tt.responseTopicCreate)
					if tt.responseTopicCreate != nil {
						continue
					}
				}
				mockSock.EXPECT().QueueSubscribe(string(tt.subscribeQueue), target).Return(tt.responseSubscribe)
				if tt.responseSubscribe != nil {
					break
				}
			}

			if err := h.Run(); err == nil {
				t.Error("Wrong match. expect: error, got: ok")
			}
		})
	}
}

// Test_Run_topicDeclareFails verifies failure case (a) of the VOIP-1406 template: when
// the global topic exchange declare fails, the service stays fully on fanout -- ZERO
// QueueBind and ZERO QueueUnbind calls (enforced by the strict mock) -- and Run() still
// succeeds.
func Test_Run_topicDeclareFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	subscribeQueue := commonoutline.QueueNameCallSubscribe
	subscribeTargets := []string{
		string(commonoutline.QueueNameAsteriskEventAll),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameFlowEvent),
		string(commonoutline.QueueNameSentinelEvent),
	}

	h := subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		if target == string(commonoutline.QueueNameSentinelEvent) {
			calls = append(calls, mockSock.EXPECT().TopicCreate(target).Return(nil))
		}
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(fmt.Errorf("could not declare the topic exchange")))
	gomock.InOrder(calls...)

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), string(subscribeQueue), gomock.Any(), false, false, false, 20, gomock.Any()).Return(nil).AnyTimes()

	if err := h.Run(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}

// Test_Run_topicBindFails verifies failure case (b) of the VOIP-1406 template: when
// pattern bind i fails, the already-made topic binds 0..i-1 are rolled back (unbound)
// in order, ZERO fanout unbinds happen (strict mock), and Run() still succeeds.
func Test_Run_topicBindFails(t *testing.T) {

	tests := []struct {
		name string

		failIndex int
	}{
		{
			name: "first bind fails - nothing to roll back",

			failIndex: 0,
		},
		{
			name: "third bind fails - first two rolled back in order",

			failIndex: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)

			subscribeQueue := commonoutline.QueueNameCallSubscribe
			subscribeTargets := []string{
				string(commonoutline.QueueNameAsteriskEventAll),
				string(commonoutline.QueueNameCustomerEvent),
				string(commonoutline.QueueNameFlowEvent),
				string(commonoutline.QueueNameSentinelEvent),
			}

			h := subscribeHandler{
				sockHandler:      mockSock,
				subscribeQueue:   subscribeQueue,
				subscribeTargets: subscribeTargets,
			}

			calls := []any{
				mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
			}
			for _, target := range subscribeTargets {
				if target == string(commonoutline.QueueNameSentinelEvent) {
					calls = append(calls, mockSock.EXPECT().TopicCreate(target).Return(nil))
				}
				calls = append(calls, mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), target).Return(nil))
			}
			calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))

			// binds 0..failIndex-1 succeed, bind failIndex fails
			for i := 0; i < tt.failIndex; i++ {
				calls = append(calls, mockSock.EXPECT().QueueBind(string(subscribeQueue), topicPatterns[i], string(commonoutline.QueueNameEvent), false, nil).Return(nil))
			}
			calls = append(calls, mockSock.EXPECT().QueueBind(string(subscribeQueue), topicPatterns[tt.failIndex], string(commonoutline.QueueNameEvent), false, nil).Return(fmt.Errorf("could not bind the pattern")))

			// best-effort rollback of the bound patterns, in bind order. NO fanout unbinds.
			for i := 0; i < tt.failIndex; i++ {
				calls = append(calls, mockSock.EXPECT().QueueUnbind(string(subscribeQueue), topicPatterns[i], string(commonoutline.QueueNameEvent), nil).Return(nil))
			}
			gomock.InOrder(calls...)

			mockSock.EXPECT().ConsumeMessage(gomock.Any(), string(subscribeQueue), gomock.Any(), false, false, false, 20, gomock.Any()).Return(nil).AnyTimes()

			if err := h.Run(); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_Run_fanoutUnbindFails verifies failure case (c) of the VOIP-1406 template: when
// one fanout unbind fails (double delivery, CRITICAL logged), the remaining fanout
// unbinds are still attempted and Run() still succeeds.
func Test_Run_fanoutUnbindFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)

	subscribeQueue := commonoutline.QueueNameCallSubscribe
	subscribeTargets := []string{
		string(commonoutline.QueueNameAsteriskEventAll),
		string(commonoutline.QueueNameCustomerEvent),
		string(commonoutline.QueueNameFlowEvent),
		string(commonoutline.QueueNameSentinelEvent),
	}

	h := subscribeHandler{
		sockHandler:      mockSock,
		subscribeQueue:   subscribeQueue,
		subscribeTargets: subscribeTargets,
	}

	calls := []any{
		mockSock.EXPECT().QueueCreate(string(subscribeQueue), "normal").Return(nil),
	}
	for _, target := range subscribeTargets {
		if target == string(commonoutline.QueueNameSentinelEvent) {
			calls = append(calls, mockSock.EXPECT().TopicCreate(target).Return(nil))
		}
		calls = append(calls, mockSock.EXPECT().QueueSubscribe(string(subscribeQueue), target).Return(nil))
	}
	calls = append(calls, mockSock.EXPECT().TopicCreateWithKind(string(commonoutline.QueueNameEvent), "topic").Return(nil))
	for _, pattern := range topicPatterns {
		calls = append(calls, mockSock.EXPECT().QueueBind(string(subscribeQueue), pattern, string(commonoutline.QueueNameEvent), false, nil).Return(nil))
	}

	// the first fanout unbind fails; the remaining unbinds are still attempted.
	for i, target := range fanoutUnbindTargets {
		var responseUnbind error
		if i == 0 {
			responseUnbind = fmt.Errorf("could not unbind the fanout exchange")
		}
		calls = append(calls, mockSock.EXPECT().QueueUnbind(string(subscribeQueue), "", target, nil).Return(responseUnbind))
	}
	gomock.InOrder(calls...)

	mockSock.EXPECT().ConsumeMessage(gomock.Any(), string(subscribeQueue), gomock.Any(), false, false, false, 20, gomock.Any()).Return(nil).AnyTimes()

	if err := h.Run(); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
