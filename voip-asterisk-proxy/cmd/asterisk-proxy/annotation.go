package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultAnnotationKeyAsteriskID = "asterisk-id"
	maxRetries                     = 3
	retryDelay                     = 3 * time.Second
	inClusterNamespacePath         = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
)

func getPodName() string {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		hn, err := os.Hostname()
		if err != nil {
			logrus.Warn("Could not get hostname as POD_NAME fallback")
			return ""
		}
		return hn
	}
	return podName
}

func getPodNamespace() string {
	nsBytes, err := os.ReadFile(inClusterNamespacePath)
	if err != nil {
		logrus.Warnf("Could not read namespace from %s: %v", inClusterNamespacePath, err)
		nsEnv := os.Getenv("POD_NAMESPACE")
		if nsEnv != "" {
			return nsEnv
		}
		return ""
	}
	return strings.TrimSpace(string(nsBytes))
}

func setProxyInfoAnnotation(asteriskID string) error {
	log := logrus.WithField("func", "setProxyInfoAnnotation")

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrapf(err, "failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to create Kubernetes clientset: %v", err)
	}

	// get namespace
	namespace := getPodNamespace()
	if namespace == "" {
		log.Errorf("Could not determine namespace for pod annotation")
		return fmt.Errorf("namespace is empty")
	}

	podName := getPodName()
	if podName == "" {
		log.Errorf("Could not determine pod name for annotation")
		return fmt.Errorf("pod name is empty")
	}

	if errPatch := patchPodAnnotation(clientset, namespace, podName, defaultAnnotationKeyAsteriskID, asteriskID); errPatch != nil {
		return fmt.Errorf("failed to patch pod annotation: %w", errPatch)
	}

	return nil
}

func patchPodAnnotation(clientset *kubernetes.Clientset, namespace, podName, annotationKey, annotationValue string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":       "patchPodAnnotation",
		"namespace":  namespace,
		"podName":    podName,
		"annotation": fmt.Sprintf("%s=%s", annotationKey, annotationValue),
	})

	log.Info("Attempting to patch pod annotation")

	// 어노테이션 키의 '/'를 JSON Patch 경로에 맞게 '~1'로 이스케이프
	escapedAnnotationKey := strings.ReplaceAll(annotationKey, "/", "~1")
	patchPayload := []map[string]string{
		{
			"op":    "add",
			"path":  fmt.Sprintf("/metadata/annotations/%s", escapedAnnotationKey),
			"value": annotationValue,
		},
	}
	patchBytes, err := json.Marshal(patchPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal patch payload: %w", err)
	}

	for i := 0; i < defaultMaxRetries; i++ {
		_, err = clientset.CoreV1().Pods(namespace).Patch(context.TODO(), podName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
		if err == nil {
			log.Info("Successfully patched pod annotation")
			return nil
		}

		log.WithError(err).Warnf("Failed to patch pod (attempt %d/%d), retrying in %v...", i+1, defaultMaxRetries, defaultRetryDelay)
		retryDelay := time.Millisecond * 500
		time.Sleep(retryDelay)
	}
	return fmt.Errorf("failed to patch pod annotation after %d retries: %w", defaultMaxRetries, err)
}
