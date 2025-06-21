package monitoringhandler

import (
	"context"
	"monorepo/bin-sentinel-manager/models/pod"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

func (h *monitoringHandler) Run(ctx context.Context, selectors map[string][]string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Run",
		"selectors": selectors,
	})

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrapf(err, "failed to create in-cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrapf(err, "failed to create Kubernetes clientset")
	}

	for namespace, selectorList := range selectors {
		for _, selector := range selectorList {
			ns := namespace
			labelSelector := selector

			go func() {
				log.Infof("Starting pod informer. namespace: %s, selector: %s", ns, labelSelector)

				podInformer := cache.NewSharedIndexInformer(
					&cache.ListWatch{
						ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
							options.LabelSelector = labelSelector
							return clientset.CoreV1().Pods(ns).List(ctx, options)
						},
						WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
							options.LabelSelector = labelSelector
							return clientset.CoreV1().Pods(ns).Watch(ctx, options)
						},
					},
					&corev1.Pod{},
					0, // no resync
					cache.Indexers{},
				)

				regstrantion, err := podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
					AddFunc: func(obj any) {
						pod := obj.(*corev1.Pod)
						if errRun := h.runPodAdded(ctx, pod); errRun != nil {
							log.WithError(errRun).Errorf("Failed to run pod added handler for pod: %s/%s", pod.Namespace, pod.Name)
						}
					},
					UpdateFunc: func(oldObj, newObj any) {
						newPod := newObj.(*corev1.Pod)
						if errRun := h.runPodUpdated(ctx, newPod); errRun != nil {
							log.WithError(errRun).Errorf("Failed to run pod updated handler for pod: %s/%s", newPod.Namespace, newPod.Name)
						}
					},
					DeleteFunc: func(obj any) {
						pod := obj.(*corev1.Pod)
						if errRun := h.runPodDeleted(ctx, pod); errRun != nil {
							log.WithError(errRun).Errorf("Failed to run pod deleted handler for pod: %s/%s", pod.Namespace, pod.Name)
						}
					},
				})
				if err != nil {
					log.WithError(err).Errorf("Failed to add event handler for pod informer. namespace: %s, selector: %s", ns, labelSelector)
					return
				}
				log.WithField("registration", regstrantion).Infof("Event handler registered for pod informer. namespace: %s, selector: %s", ns, labelSelector)

				stopCh := make(chan struct{})

				go func() {
					<-ctx.Done()
					close(stopCh)
				}()

				podInformer.Run(stopCh)
			}()
		}
	}

	<-ctx.Done()
	log.Info("Context cancelled. Shutting down informers.")
	return nil
}

func (h *monitoringHandler) runPodAdded(ctx context.Context, p *corev1.Pod) error {
	log := logrus.WithField("func", "runPodAdded")

	log.WithField("pod", p).Infof("Pod added. namespace: %s, name: %s", p.Namespace, p.Name)
	h.notifyHandler.PublishEvent(ctx, pod.EventTypePodAdded, p)

	promPodStateChangeCounter.WithLabelValues(p.Namespace, p.Labels["app"], "added").Inc()

	return nil
}

func (h *monitoringHandler) runPodUpdated(ctx context.Context, p *corev1.Pod) error {
	log := logrus.WithField("func", "runPodUpdated")

	log.WithField("pod", p).Infof("Pod updated. namespace: %s, name: %s", p.Namespace, p.Name)
	h.notifyHandler.PublishEvent(ctx, pod.EventTypePodUpdated, p)

	promPodStateChangeCounter.WithLabelValues(p.Namespace, p.Labels["app"], "updated").Inc()

	return nil
}

func (h *monitoringHandler) runPodDeleted(ctx context.Context, p *corev1.Pod) error {
	log := logrus.WithField("func", "runPodDeleted")

	log.WithField("pod", p).Infof("Pod deleted. namespace: %s, name: %s", p.Namespace, p.Name)
	h.notifyHandler.PublishEvent(ctx, pod.EventTypePodDeleted, p)

	promPodStateChangeCounter.WithLabelValues(p.Namespace, p.Labels["app"], "deleted").Inc()

	return nil
}
