package monitoringhandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-sentinel-manager/models/pod"
	"testing"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_runPodAdded(t *testing.T) {

	tests := []struct {
		name string

		pod *corev1.Pod
	}{
		{
			name: "asterisk-call pod added",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "asterisk-call-12345",
					Namespace: namespaceVOIP,
					Labels: map[string]string{
						"app": lableAppAsteriskCall,
					},
					Annotations: map[string]string{
						"asterisk-id": "00:11:22:33:44:55",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := monitoringHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockNotify.EXPECT().PublishEvent(ctx, pod.EventTypePodAdded, tt.pod)
			if errRun := h.runPodAdded(ctx, tt.pod); errRun != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRun)
			}
		})
	}
}

func Test_runPodUpdated(t *testing.T) {

	tests := []struct {
		name string

		pod *corev1.Pod
	}{
		{
			name: "asterisk-call pod added",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "asterisk-call-12345",
					Namespace: namespaceVOIP,
					Labels: map[string]string{
						"app": lableAppAsteriskCall,
					},
					Annotations: map[string]string{
						"asterisk-id": "00:11:22:33:44:55",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := monitoringHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockNotify.EXPECT().PublishEvent(ctx, pod.EventTypePodUpdated, tt.pod)
			if errRun := h.runPodUpdated(ctx, tt.pod); errRun != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRun)
			}
		})
	}
}

func Test_runPodDeleted(t *testing.T) {

	tests := []struct {
		name string

		pod *corev1.Pod

		expectedAsteriskID string
	}{
		{
			name: "asterisk-call pod deleted",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "asterisk-call-12345",
					Namespace: namespaceVOIP,
					Labels: map[string]string{
						"app": lableAppAsteriskCall,
					},
					Annotations: map[string]string{
						"asterisk-id": "00:11:22:33:44:55",
					},
				},
			},

			expectedAsteriskID: "00:11:22:33:44:55",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := monitoringHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockNotify.EXPECT().PublishEvent(ctx, pod.EventTypePodDeleted, tt.pod)
			mockReq.EXPECT().CallV1RecoveryStart(ctx, tt.expectedAsteriskID).Return(nil)
			if errRun := h.runPodDeleted(ctx, tt.pod); errRun != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRun)
			}
		})
	}
}
