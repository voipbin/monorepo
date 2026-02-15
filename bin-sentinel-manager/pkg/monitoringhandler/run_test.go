package monitoringhandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-sentinel-manager/models/pod"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
		{
			name: "pod_with_different_namespace",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: namespaceBIN,
					Labels: map[string]string{
						"app": "test-app",
					},
				},
			},
		},
		{
			name: "pod_without_labels",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-label-pod",
					Namespace: namespaceVOIP,
				},
			},
		},
		{
			name: "pod_with_empty_labels",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "empty-label-pod",
					Namespace: namespaceVOIP,
					Labels:    map[string]string{},
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
		{
			name: "pod_with_different_app_label",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod-67890",
					Namespace: namespaceBIN,
					Labels: map[string]string{
						"app": "different-app",
					},
				},
			},

			expectedAsteriskID: "",
		},
		{
			name: "pod_without_labels",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-label-pod",
					Namespace: namespaceVOIP,
				},
			},

			expectedAsteriskID: "",
		},
		{
			name: "pod_with_empty_annotations",

			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-annotation-pod",
					Namespace: namespaceVOIP,
					Labels: map[string]string{
						"app": lableAppAsteriskCall,
					},
					Annotations: map[string]string{},
				},
			},

			expectedAsteriskID: "",
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
			if errRun := h.runPodDeleted(ctx, tt.pod); errRun != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRun)
			}
		})
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name string

		selectors map[string][]string
		expectErr bool
	}{
		{
			name: "fails_outside_kubernetes_cluster",

			selectors: map[string][]string{
				"default": {"app=test"},
			},
			expectErr: true,
		},
		{
			name: "fails_with_empty_selectors",

			selectors: map[string][]string{},
			expectErr: true,
		},
		{
			name: "fails_with_nil_selectors",

			selectors: nil,
			expectErr: true,
		},
		{
			name: "fails_with_multiple_namespaces",

			selectors: map[string][]string{
				"voip": {"app=asterisk-call"},
				"bin":  {"app=test-app"},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := NewMonitoringHandler(mockReq, mockNotify, mockUtil)

			// Create a context with timeout to avoid hanging
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()

			err := h.Run(ctx, tt.selectors)

			// When not running in a Kubernetes cluster, Run should return an error
			// because rest.InClusterConfig() will fail
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error when running outside Kubernetes cluster, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRunContextCancellation(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "handles_context_cancellation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := NewMonitoringHandler(mockReq, mockNotify, mockUtil)

			// Create a context that's already cancelled
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			selectors := map[string][]string{
				"default": {"app=test"},
			}

			// This should fail quickly because we're not in a cluster
			// and context is already cancelled
			err := h.Run(ctx, selectors)

			// Should return an error (in-cluster config failure, not context cancellation)
			if err == nil {
				t.Errorf("Expected error when running outside Kubernetes cluster")
			}
		})
	}
}
