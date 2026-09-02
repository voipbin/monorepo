package main

import (
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	testNamespace = "voip"
	testPodName   = "voip-asterisk-call-docker-1"
)

// Test_patchPodAnnotation_noPreExistingAnnotations pins a real bug found while adding this
// coverage, and its fix. patchPodAnnotation used to build an RFC 6902 JSON Patch with a single
// "add" operation targeting /metadata/annotations/<key>. That operation requires the
// /metadata/annotations parent object to already exist -- a pod that has never had any
// annotation applied to it (Annotations is omitempty and genuinely absent) failed every one of
// defaultMaxRetries attempts, since retrying an inherently-malformed patch never helps. This was
// not a client-go-version issue -- the same failure occurred against a real apiserver. Fixed by
// switching to an RFC 7386 JSON Merge Patch, which creates the missing annotations object rather
// than requiring it to pre-exist. See VOIP-1446 design doc.
func Test_patchPodAnnotation_noPreExistingAnnotations(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
			// Annotations deliberately omitted -- the normal state for a freshly created pod
			// that nothing else has annotated yet.
		},
	}

	clientset := fake.NewClientset(pod)

	if err := patchPodAnnotation(clientset, testNamespace, testPodName, defaultAnnotationKeyAsteriskID, "aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatalf("patchPodAnnotation() error = %v, want nil against a pod with no pre-existing annotations", err)
	}

	got, err := clientset.CoreV1().Pods(testNamespace).Get(t.Context(), testPodName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if want := "aa:bb:cc:dd:ee:ff"; got.Annotations[defaultAnnotationKeyAsteriskID] != want {
		t.Errorf("annotation %q = %q, want %q", defaultAnnotationKeyAsteriskID, got.Annotations[defaultAnnotationKeyAsteriskID], want)
	}
}

// Test_patchPodAnnotation_preExistingAnnotations confirms the merge patch also correctly merges
// into an already-populated annotations map rather than only handling the absent-map case above.
func Test_patchPodAnnotation_preExistingAnnotations(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				"some-other-annotation": "unrelated-value",
			},
		},
	}

	clientset := fake.NewClientset(pod)

	if err := patchPodAnnotation(clientset, testNamespace, testPodName, defaultAnnotationKeyAsteriskID, "11:22:33:44:55:66"); err != nil {
		t.Fatalf("patchPodAnnotation() error = %v, want nil against a pod with pre-existing annotations", err)
	}

	got, err := clientset.CoreV1().Pods(testNamespace).Get(t.Context(), testPodName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if want := "11:22:33:44:55:66"; got.Annotations[defaultAnnotationKeyAsteriskID] != want {
		t.Errorf("annotation %q = %q, want %q", defaultAnnotationKeyAsteriskID, got.Annotations[defaultAnnotationKeyAsteriskID], want)
	}
	// the merge patch must not clobber the pre-existing, unrelated annotation.
	if want := "unrelated-value"; got.Annotations["some-other-annotation"] != want {
		t.Errorf("pre-existing annotation %q = %q, want %q (must survive the merge patch)", "some-other-annotation", got.Annotations["some-other-annotation"], want)
	}
}

// Test_patchPodAnnotation_retryThenSucceed exercises the retry loop: the first two patch
// attempts fail with a transient error, the third succeeds.
func Test_patchPodAnnotation_retryThenSucceed(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
			Annotations: map[string]string{
				"placeholder": "value",
			},
		},
	}

	clientset := fake.NewClientset(pod)

	attempts := 0
	const failuresBeforeSuccess = 2
	clientset.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		attempts++
		if attempts <= failuresBeforeSuccess {
			return true, nil, fmt.Errorf("transient apiserver error (attempt %d)", attempts)
		}
		// Let the fake tracker's default patch reactor actually apply the patch.
		return false, nil, nil
	})

	if err := patchPodAnnotation(clientset, testNamespace, testPodName, defaultAnnotationKeyAsteriskID, "aa:aa:aa:aa:aa:aa"); err != nil {
		t.Fatalf("patchPodAnnotation() error = %v, want nil after transient failures followed by success", err)
	}

	if attempts != failuresBeforeSuccess+1 {
		t.Errorf("attempts = %d, want %d (retry then succeed on the next call)", attempts, failuresBeforeSuccess+1)
	}

	got, err := clientset.CoreV1().Pods(testNamespace).Get(t.Context(), testPodName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if want := "aa:aa:aa:aa:aa:aa"; got.Annotations[defaultAnnotationKeyAsteriskID] != want {
		t.Errorf("annotation %q = %q, want %q", defaultAnnotationKeyAsteriskID, got.Annotations[defaultAnnotationKeyAsteriskID], want)
	}
}

// Test_patchPodAnnotation_exhaustedRetries confirms the function gives up after exactly
// defaultMaxRetries attempts (not fewer, not more) when every attempt fails, and that the
// underlying error is surfaced rather than swallowed. This case necessarily costs
// defaultMaxRetries * ~500ms of real wall-clock time (patchPodAnnotation has no injectable
// clock or retry count -- see VOIP-1446 design doc's explicit decision not to add one).
func Test_patchPodAnnotation_exhaustedRetries(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testPodName,
			Namespace: testNamespace,
		},
	}

	clientset := fake.NewClientset(pod)

	attempts := 0
	wantErr := fmt.Errorf("persistent apiserver error")
	clientset.PrependReactor("patch", "pods", func(action k8stesting.Action) (bool, k8sruntime.Object, error) {
		attempts++
		return true, nil, wantErr
	})

	err := patchPodAnnotation(clientset, testNamespace, testPodName, defaultAnnotationKeyAsteriskID, "bb:bb:bb:bb:bb:bb")
	if err == nil {
		t.Fatal("patchPodAnnotation() error = nil, want an error after exhausting all retries")
	}
	if attempts != defaultMaxRetries {
		t.Errorf("attempts = %d, want %d (defaultMaxRetries)", attempts, defaultMaxRetries)
	}
	if !containsErr(err, wantErr.Error()) {
		t.Errorf("error = %v, want it to surface the underlying cause %q verbatim, not swallow it", err, wantErr.Error())
	}
}

// Test_patchPodAnnotation_nonexistentPod exercises the naturally-occurring NotFound error path
// (as opposed to an injected reactor error above) -- a pod that was never seeded into the fake
// clientset.
func Test_patchPodAnnotation_nonexistentPod(t *testing.T) {
	clientset := fake.NewClientset() // no pods seeded

	err := patchPodAnnotation(clientset, testNamespace, testPodName, defaultAnnotationKeyAsteriskID, "cc:cc:cc:cc:cc:cc")
	if err == nil {
		t.Fatal("patchPodAnnotation() error = nil, want a NotFound error for a pod that was never created")
	}
	if !containsErr(err, "not found") {
		t.Errorf("error = %v, want it to surface the underlying NotFound cause, not swallow it", err)
	}
}

// containsErr reports whether err's message contains substr, walking the %w-wrapped chain via
// err.Error() (patchPodAnnotation wraps with fmt.Errorf("...: %w", err), so the substring is
// always present in the top-level message).
func containsErr(err error, substr string) bool {
	return err != nil && strings.Contains(err.Error(), substr)
}
