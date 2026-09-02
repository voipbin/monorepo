package k8swatchhandler

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
	"monorepo/bin-sentinel-manager/pkg/monitoringbackend"
)

// Run starts one informer per watched (namespace, label-selector) pair and blocks until the
// context is cancelled or any informer fails.
//
// Fail-loud fan-in: the pre-VOIP-1418 implementation spawned informers in bare goroutines with no
// error channel back to the caller, blocked on <-ctx.Done(), and returned nil unconditionally —
// so an informer that never synced, or died outright, left the process running and apparently
// healthy while watching nothing. errgroup gives every informer a path to surface its own failure
// out of Run, which cmd/sentinel-manager turns into a non-zero exit.
func (h *k8sWatchHandler) Run(ctx context.Context) error {
	log := logrus.WithField("func", "Run")

	g, gctx := errgroup.WithContext(ctx)
	for _, target := range watchTargets {
		g.Go(func() error {
			return h.runInformer(gctx, target)
		})
	}

	if err := g.Wait(); err != nil {
		return errors.Wrap(err, "the pod watch stopped with an error")
	}

	log.Info("Context cancelled. Shutting down the pod watchers.")

	return nil
}

// runInformer runs a single (namespace, selector) informer to completion.
//
// It returns nil for a graceful shutdown and non-nil for anything else. The distinction matters
// more than it looks: errgroup.WithContext cancels the derived context as soon as ANY sibling
// fails, so every other goroutine also observes a cancelled context at that moment. If a
// cancelled context produced a synthesized error here, one real failure would be reported as
// several, and — worse — an ordinary SIGTERM shutdown would be reported as a failure and exit
// non-zero. Cancellation is therefore always nil; only this goroutine's own genuine fault returns
// an error, and errgroup surfaces the first one.
func (h *k8sWatchHandler) runInformer(ctx context.Context, target watchTarget) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "runInformer",
		"namespace": target.Namespace,
		"selector":  target.LabelSelector,
	})

	budget := newWatchFailureBudget(h.maxWatchFailures, func(outcome string) {
		promWatchHealthCounter.WithLabelValues(target.Namespace, target.LabelSelector, outcome).Inc()
	})

	informer := cache.NewSharedIndexInformer(
		// ListWithContextFunc/WatchFuncWithContext (rather than the deprecated
		// ListFunc/WatchFunc) match what client-go's Reflector actually calls
		// (tools/cache/reflector.go calls only the WithContext variants). listCtx/watchCtx are
		// named distinctly from this function's own ctx to avoid shadowing it -- ctx stays live
		// below for handleUpdate/handleDelete/waitForCacheSync.
		&cache.ListWatch{
			ListWithContextFunc: func(listCtx context.Context, options metav1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = target.LabelSelector
				return h.clientset.CoreV1().Pods(target.Namespace).List(listCtx, options)
			},
			WatchFuncWithContext: func(watchCtx context.Context, options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = target.LabelSelector
				return h.clientset.CoreV1().Pods(target.Namespace).Watch(watchCtx, options)
			},
		},
		&corev1.Pod{},
		0, // no resync: rely on the watch, not periodic full reconciliation
		cache.Indexers{},
	)

	if _, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// AddFunc stays a no-op, unconditionally — including after the initial sync.
		//
		// Informers replay every already-existing pod as a synthetic Add on the initial list, so
		// publishing "started" here would misrepresent long-lived pods as freshly started on every
		// sentinel restart. A genuinely new pod is still covered: its first post-creation status
		// transition fires UpdateFunc, which publishes started (design §8.4 item 1).
		//
		// Do NOT add "helpful" logic here for the new-pod case. The resulting late-and-possibly-
		// repeated `started` timing is an accepted, documented asymmetry with the Docker backend,
		// not a gap to close.
		AddFunc: func(obj any) {},

		UpdateFunc: func(oldObj, newObj any) {
			h.handleUpdate(ctx, budget, oldObj, newObj)
		},

		DeleteFunc: func(obj any) {
			h.handleDelete(ctx, budget, obj)
		},
	}); err != nil {
		return errors.Wrapf(err, "could not add the pod event handler. namespace: %s, selector: %s", target.Namespace, target.LabelSelector)
	}

	if err := informer.SetWatchErrorHandler(func(_ *cache.Reflector, err error) {
		// both values come from the SAME in-lock snapshot: a separately-read count would describe
		// a different moment than the decision it accompanies.
		consecutive, exhausted := budget.RecordFailure()

		log.Warnf("The pod watch reported an error. consecutive: %d/%d, err: %v", consecutive, h.maxWatchFailures, err)
		if exhausted {
			log.Errorf("The pod watch failed %d consecutive times without recovering. Giving up.", h.maxWatchFailures)
		}
	}); err != nil {
		return errors.Wrapf(err, "could not set the pod watch error handler. namespace: %s, selector: %s", target.Namespace, target.LabelSelector)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		informer.RunWithContext(ctx)
	}()

	log.Infof("Started the pod informer.")

	if err := h.waitForCacheSync(ctx, informer); err != nil {
		return err
	}

	log.Infof("The pod informer completed its initial sync.")

	return h.watchUntilDone(ctx, informer, budget, done, log)
}

// waitForCacheSync bounds the initial sync with an explicit deadline.
//
// A bare cache.WaitForCacheSync returns false only when its stop channel closes. Under the exact
// failure this backend must fail loud on — missing `pod-reader` RBAC — the reflector's initial
// List is denied and client-go retries with backoff forever, so the bare call blocks indefinitely
// and never returns false. The timeout is what converts that hang into a startup failure
// (design §8.4 item 3).
func (h *k8sWatchHandler) waitForCacheSync(ctx context.Context, informer cache.SharedIndexInformer) error {
	syncCtx, cancel := context.WithTimeout(ctx, h.cacheSyncTimeout)
	defer cancel()

	if cache.WaitForCacheSync(syncCtx.Done(), informer.HasSynced) {
		return nil
	}

	if ctx.Err() != nil {
		// the PARENT context was cancelled during startup: a shutdown, not a failure.
		return nil
	}

	return errors.Errorf(
		"the pod informer did not complete its initial sync within %v. the pod-reader RBAC role is the usual cause: without it the initial List is denied and client-go retries forever",
		h.cacheSyncTimeout,
	)
}

// resourceVersionReporter is the one method watchUntilDone needs from an informer.
//
// Narrowed deliberately: cache.SharedIndexInformer is a large interface, and depending on all of
// it here would make this loop -- which carries the fail-loud decision and the budget reset --
// effectively untestable without a live informer and a live apiserver.
type resourceVersionReporter interface {
	LastSyncResourceVersion() string
}

// watchUntilDone blocks until shutdown, budget exhaustion, or the informer stopping on its own.
func (h *k8sWatchHandler) watchUntilDone(
	ctx context.Context,
	informer resourceVersionReporter,
	budget *watchFailureBudget,
	done <-chan struct{},
	log *logrus.Entry,
) error {
	ticker := time.NewTicker(h.watchHealthInterval)
	defer ticker.Stop()

	lastResourceVersion := informer.LastSyncResourceVersion()

	for {
		select {
		case <-ctx.Done():
			<-done
			return nil

		case <-budget.Fatal():
			// select picks uniformly at random among ready cases, so a shutdown landing at the
			// same moment as budget exhaustion could otherwise be reported as a failure and exit
			// non-zero. Re-check the context first: a cancelled context always means graceful.
			if ctx.Err() != nil {
				<-done
				return nil
			}

			return errors.Errorf(
				"the pod watch failed %d consecutive times without recovering. refusing to keep running blind",
				h.maxWatchFailures,
			)

		case <-done:
			if ctx.Err() != nil {
				return nil
			}
			return errors.New("the pod informer stopped unexpectedly")

		case <-ticker.C:
			// A changed resource version proves the reflector completed a list/watch. This is the
			// only recovery signal available when the selector matches zero pods: no events are
			// delivered then, however healthy the watch is, so relying on deliveries alone would
			// drain the budget and restart a working process.
			if current := informer.LastSyncResourceVersion(); current != lastResourceVersion {
				lastResourceVersion = current
				budget.RecordHealthy()
			}
		}
	}
}

// handleUpdate publishes a started event, and — first — a died event for a replaced pod.
//
// THE UID CHECK IS THE POINT OF THIS FUNCTION. client-go's reflector only synthesizes a Deleted
// callback for keys ABSENT from a relist's object set. If a pod is deleted and a same-name
// replacement created while the watch was interrupted — entirely plausible on a rolling update or
// a node eviction that lands before the watch recovers — the key is still present in the relist,
// client-go delivers it as a Replaced delta through UpdateFunc, and NO DELETE CALLBACK EVER FIRES
// for the dead generation. Without this check that death is dropped silently: no panic, no error,
// no counter, and the calls stranded on that pod are never recovered (design §8.4 item 2).
func (h *k8sWatchHandler) handleUpdate(ctx context.Context, budget *watchFailureBudget, oldObj, newObj any) {
	log := logrus.WithField("func", "handleUpdate")

	// a delivered event is direct evidence the watch is working.
	budget.RecordHealthy()

	newPod, ok := newObj.(*corev1.Pod)
	if !ok || newPod == nil {
		log.Errorf("Received a pod update whose new object is not a pod. Skipping. type: %T", newObj)
		return
	}

	oldPod, ok := oldObj.(*corev1.Pod)
	if ok && oldPod != nil && oldPod.UID != newPod.UID {
		log.Warnf(
			"Detected a same-name pod replacement with no delete callback. Publishing the death of the previous generation. pod_name: %s, old_uid: %s, new_uid: %s",
			oldPod.Name, oldPod.UID, newPod.UID,
		)
		// publish the death BEFORE the replacement's started, so a consumer sees the two in the
		// order they actually happened.
		h.publishDied(ctx, oldPod, diedSourceUIDMismatch)
	}

	h.publishStarted(ctx, newPod)
}

// handleDelete publishes a died event, unwrapping a tombstone when the deletion was only
// discovered on relist.
//
// NEVER bare-assert the argument. client-go delivers cache.DeletedFinalStateUnknown instead of a
// pod whenever a deletion was missed during a watch interruption, and a bare assertion panics on
// that shape. The reflexive fix — assert with `ok` and return on mismatch — is WORSE than the
// panic, not better: it silently drops exactly the death event this service exists to detect. The
// failure budget makes this path reachable BY DESIGN, since its whole purpose is keeping the
// process alive across the transient interruptions after which a tombstone shows up
// (design §8.4 item 3).
func (h *k8sWatchHandler) handleDelete(ctx context.Context, budget *watchFailureBudget, obj any) {
	log := logrus.WithField("func", "handleDelete")

	budget.RecordHealthy()

	source := diedSourceLive

	p, ok := obj.(*corev1.Pod)
	if !ok {
		tombstone, okTombstone := obj.(cache.DeletedFinalStateUnknown)
		if !okTombstone {
			// nothing publishable, so this counter is the only trace the death was seen at all.
			promDiedDetectionCounter.WithLabelValues(diedSourceUnrecoverable).Inc()
			log.Errorf("Received a pod delete of an unrecognized shape. A death may have gone unrecovered. type: %T", obj)
			return
		}

		p, ok = tombstone.Obj.(*corev1.Pod)
		if !ok || p == nil {
			promDiedDetectionCounter.WithLabelValues(diedSourceUnrecoverable).Inc()
			log.Errorf("Received a pod delete tombstone wrapping a non-pod. A death may have gone unrecovered. key: %s, type: %T", tombstone.Key, tombstone.Obj)
			return
		}

		source = diedSourceTombstone
		log.Warnf("Recovered a missed pod deletion from a relist tombstone. pod_name: %s", p.Name)
	}

	h.publishDied(ctx, p, source)
}

// publishStarted publishes a container_started event for a watched pod.
func (h *k8sWatchHandler) publishStarted(ctx context.Context, p *corev1.Pod) {
	log := logrus.WithField("func", "publishStarted")

	service, ok := mapService(p.Labels[podLabelApp])
	if !ok {
		log.Debugf("Ignoring a pod whose app label is not a watched service. pod_name: %s, app: %s", p.Name, p.Labels[podLabelApp])
		return
	}

	asteriskID := p.Annotations[podAnnotationAsteriskID]

	log.Infof("Pod started. pod_name: %s, service: %s, asterisk_id: %s", p.Name, service, asteriskID)
	h.notifyHandler.PublishEvent(ctx, smcontainer.EventTypeContainerStarted, &smcontainer.Event{
		ContainerName: p.Name,
		Service:       service,
		AsteriskID:    asteriskID,
	})
	monitoringbackend.PromContainerStateChangeCounter.WithLabelValues(p.Name, service, monitoringbackend.StateStarted).Inc()
}

// publishDied publishes a container_died event for a watched pod.
//
// The asterisk-id is read straight off the annotation — no resolution step, no state table. An
// absent annotation publishes an empty id, which is the same degrade path the Docker backend's
// unresolved case already established and which bin-call-manager's empty-id guard already handles.
// That is expected rather than exceptional here: voip-asterisk-proxy self-patches the annotation
// shortly AFTER its pod starts, so a pod dying inside that window genuinely never had one
// (design §8.2).
func (h *k8sWatchHandler) publishDied(ctx context.Context, p *corev1.Pod, source string) {
	log := logrus.WithField("func", "publishDied")

	service, ok := mapService(p.Labels[podLabelApp])
	if !ok {
		log.Debugf("Ignoring a pod whose app label is not a watched service. pod_name: %s, app: %s", p.Name, p.Labels[podLabelApp])
		return
	}

	asteriskID := p.Annotations[podAnnotationAsteriskID]
	if asteriskID == "" {
		monitoringbackend.PromContainerUnresolvedAsteriskIDCounter.WithLabelValues(p.Name).Inc()
		log.Warnf("Publishing a died event without a resolved asterisk id. No recovery will be triggered for this pod. pod_name: %s, service: %s", p.Name, service)
	}

	log.Infof("Pod died. pod_name: %s, service: %s, asterisk_id: %s, source: %s", p.Name, service, asteriskID, source)
	h.notifyHandler.PublishEvent(ctx, smcontainer.EventTypeContainerDied, &smcontainer.Event{
		ContainerName: p.Name,
		Service:       service,
		AsteriskID:    asteriskID,
	})
	monitoringbackend.PromContainerStateChangeCounter.WithLabelValues(p.Name, service, monitoringbackend.StateDied).Inc()
	promDiedDetectionCounter.WithLabelValues(source).Inc()
}
