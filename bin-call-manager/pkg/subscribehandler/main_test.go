package subscribehandler

import (
	"fmt"
	"testing"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/sockhandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_Run verifies that Run() declares the sentinel-manager event exchange before binding to
// it. sentinel-manager is Kubernetes-only, so in non-Kubernetes deployments nothing else
// declares that exchange and the bind would otherwise fail with an AMQP 404, taking this
// service down at boot.
func Test_Run(t *testing.T) {

	tests := []struct {
		name string

		subscribeQueue   commonoutline.QueueName
		subscribeTargets []string

		expectedTopicCreates []string
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

			expectedTopicCreates: []string{
				string(commonoutline.QueueNameSentinelEvent),
			},
		},
		{
			name: "no sentinel target",

			subscribeQueue: commonoutline.QueueNameCallSubscribe,
			subscribeTargets: []string{
				string(commonoutline.QueueNameAsteriskEventAll),
			},

			expectedTopicCreates: []string{},
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
			for _, target := range tt.expectedTopicCreates {
				mockSock.EXPECT().TopicCreate(target).Return(nil)
			}
			for _, target := range tt.subscribeTargets {
				mockSock.EXPECT().QueueSubscribe(string(tt.subscribeQueue), target).Return(nil)
			}
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
