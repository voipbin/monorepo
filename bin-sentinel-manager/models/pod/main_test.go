package pod

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodStruct(t *testing.T) {
	tests := []struct {
		name string

		pod Pod
	}{
		{
			name: "creates_pod_with_corev1_pod",

			pod: Pod{
				Pod: corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.pod.Name != "test-pod" {
				t.Errorf("Wrong pod name. expect: test-pod, got: %s", tt.pod.Name)
			}
			if tt.pod.Namespace != "default" {
				t.Errorf("Wrong pod namespace. expect: default, got: %s", tt.pod.Namespace)
			}
		})
	}
}
