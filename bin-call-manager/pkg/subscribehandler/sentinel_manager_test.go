package subscribehandler

import (
	"monorepo/bin-call-manager/pkg/arieventhandler"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	smpod "monorepo/bin-sentinel-manager/models/pod"
	"testing"

	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_processEvent_processEventSMPodDeleted(t *testing.T) {

	tests := []struct {
		name  string
		event *sock.Event

		expectedPod *smpod.Pod
	}{
		{
			name: "normal",

			event: &sock.Event{
				Publisher: "sentinel-manager",
				Type:      smpod.EventTypePodDeleted,
				DataType:  "application/json",
				Data:      []byte(`{"metadata":{"annotations":{"asterisk-id":"3e:50:6b:43:bb:32"}}}`),
			},

			expectedPod: &smpod.Pod{
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"asterisk-id": "3e:50:6b:43:bb:32",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockARIEvent := arieventhandler.NewMockARIEventHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := subscribeHandler{
				sockHandler:     mockSock,
				ariEventHandler: mockARIEvent,
				callHandler:     mockCall,
			}

			mockCall.EXPECT().EventSMPodDeleted(gomock.Any(), tt.expectedPod.Return(nil)

			if errProcess := h.processEvent(tt.event); errProcess != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errProcess)
			}
		})
	}
}
