package dockerwatchhandler

import (
	"context"
	"fmt"
	"time"

	dockerevents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	smcontainer "monorepo/bin-sentinel-manager/models/container"
)

// list of the container state labels used on promContainerStateChangeCounter.
const (
	stateStarted = "started"
	stateDied    = "died"
)

// list of the result labels used on promContainerEventStreamReconnectCounter.
const (
	// streamResultDelivered marks a stream attempt that delivered at least one event before
	// ending. The stream was genuinely established; something merely interrupted it.
	streamResultDelivered = "delivered"
	// streamResultEmpty marks a SHORT-LIVED stream attempt that ended without delivering anything
	// -- the signature of a stream that could not be established at all. Only this result counts
	// toward the give-up budget.
	streamResultEmpty = "empty"
	// streamResultIdle marks a stream attempt that delivered nothing but SURVIVED long enough to
	// prove the proxy was reachable. That is what an idle fleet's stream looks like when the proxy
	// is restarted underneath it, so it resets the budget rather than counting against it.
	streamResultIdle = "idle"
)

// Run performs the boot-time reconciliation, starts the background asterisk-id refresh loop, and
// then consumes the Docker event stream until ctx is cancelled.
//
// The ordering matters and is design §3.3 step 0: seed the table, run ONE immediate refresh pass
// (not a wait-for-the-first-tick), and only then enter the event loop. A sentinel that started
// watching before seeding would treat its first observed death as unresolvable.
func (h *dockerWatchHandler) Run(ctx context.Context) error {
	log := logrus.WithField("func", "Run")

	if errBoot := h.bootReconcile(ctx); errBoot != nil {
		// fail loud: an unreachable docker-socket-proxy must crash the process (Komodo's health
		// monitoring surfaces the crash-loop), not leave sentinel silently watching nothing.
		return errors.Wrap(errBoot, "could not reconcile the running containers at boot")
	}

	if errRefresh := h.refreshOnce(ctx); errRefresh != nil {
		// Redis being briefly unavailable is not fatal -- the loop retries every tick, and the
		// event stream is independently useful.
		log.Warnf("Could not run the initial asterisk id refresh pass. err: %v", errRefresh)
	}

	go h.runRefreshLoop(ctx)

	if errLoop := h.runEventLoop(ctx); errLoop != nil {
		return errors.Wrap(errLoop, "the docker event loop stopped with an error")
	}

	log.Info("Context cancelled. Shutting down the docker watcher.")

	return nil
}

// runEventLoop consumes the Docker event stream, reconnecting on any disconnect.
//
// Reconnection resumes from the last processed event's timestamp so a proxy restart or network
// blip does not silently drop a `die`. The gap is bounded rather than fully replayed, which is
// acceptable because the recovery it feeds is already fire-and-forget and best-effort on the
// call-manager side -- sentinel's own delivery does not need a stronger guarantee than the
// mechanism it drives.
//
// Retrying is NOT unbounded. Post-boot stream loss used to be invisible: the loop would spin
// every reconnectDelay forever with only a log line, leaving sentinel "up" while watching
// nothing. Each attempt is now counted on promContainerEventStreamReconnectCounter, and
// maxConsecutiveEmptyStreams attempts that deliver nothing return an error, which propagates to a
// non-zero process exit and a visible Komodo crash-loop.
func (h *dockerWatchHandler) runEventLoop(ctx context.Context) error {
	log := logrus.WithField("func", "runEventLoop")

	since := ""
	consecutiveEmpty := 0

	for {
		if ctx.Err() != nil {
			return nil
		}

		log.Infof("Opening the docker event stream. since: %s", since)

		openedAt := time.Now()
		next, delivered := h.consumeEvents(ctx, since)
		lifetime := time.Since(openedAt)
		since = next

		if ctx.Err() != nil {
			return nil
		}

		switch {
		case delivered:
			promContainerEventStreamReconnectCounter.WithLabelValues(streamResultDelivered).Inc()
			consecutiveEmpty = 0
			log.Infof("The docker event stream ended after delivering events. Reconnecting. lifetime: %v, delay: %v, since: %s", lifetime, h.reconnectDelay, since)

		case h.healthyStreamLifetime > 0 && lifetime >= h.healthyStreamLifetime:
			// the stream carried no events, but it stayed open long enough to prove the proxy was
			// reachable. On a genuinely idle fleet this is the NORMAL shape of a proxy restart, and
			// counting it as a failure would let those endings accumulate across days until the
			// give-up threshold fired on a perfectly healthy system.
			promContainerEventStreamReconnectCounter.WithLabelValues(streamResultIdle).Inc()
			consecutiveEmpty = 0
			log.Infof("The docker event stream ended after a healthy lifetime without delivering events. Reconnecting. lifetime: %v, delay: %v, since: %s", lifetime, h.reconnectDelay, since)

		default:
			promContainerEventStreamReconnectCounter.WithLabelValues(streamResultEmpty).Inc()
			consecutiveEmpty++
			log.Warnf(
				"The docker event stream ended immediately without delivering any event. The socket proxy may be unreachable. consecutive: %d/%d, lifetime: %v, delay: %v, since: %s",
				consecutiveEmpty, maxConsecutiveEmptyStreams, lifetime, h.reconnectDelay, since,
			)
		}

		if consecutiveEmpty >= maxConsecutiveEmptyStreams {
			return errors.Errorf(
				"the docker event stream could not be established after %d consecutive attempts. refusing to keep running blind",
				consecutiveEmpty,
			)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(h.reconnectDelay):
		}
	}
}

// consumeEvents reads one Docker event stream to completion (or error).
//
// It returns the `since` cursor to resume from, and whether this attempt delivered at least one
// message. A healthy stream blocks rather than returning, so "ended without delivering anything"
// is the signature of a stream that could not be established -- see maxConsecutiveEmptyStreams.
func (h *dockerWatchHandler) consumeEvents(ctx context.Context, since string) (string, bool) {
	log := logrus.WithField("func", "consumeEvents")

	messages, errs := h.dockerClient.Events(ctx, dockerevents.ListOptions{
		Since:   since,
		Filters: watchedEventFilters(),
	})

	delivered := false
	for {
		select {
		case <-ctx.Done():
			return since, delivered

		case err := <-errs:
			if err != nil {
				log.Warnf("The docker event stream returned an error. err: %v", err)
			}
			return since, delivered

		case message, ok := <-messages:
			if !ok {
				return since, delivered
			}

			delivered = true

			// advance the cursor for EVERY message the daemon delivered, not only the ones this
			// service acts on: resuming from an older cursor would re-deliver events already
			// seen.
			since = sinceCursor(message)

			if errHandle := h.handleEvent(ctx, message); errHandle != nil {
				log.Errorf("Could not handle the docker event. action: %s, err: %v", message.Action, errHandle)
			}
		}
	}
}

// watchedEventFilters restricts the stream to container start/die events. Filtering server-side
// keeps the proxy's response small; the name match still happens locally because Docker's `name`
// filter has no prefix form.
func watchedEventFilters() filters.Args {
	res := filters.NewArgs()
	res.Add("type", string(dockerevents.ContainerEventType))
	res.Add("event", string(dockerevents.ActionStart))
	res.Add("event", string(dockerevents.ActionDie))

	return res
}

// sinceCursor renders a resume cursor one nanosecond past the given message.
//
// Docker's `since` is INCLUSIVE, so resuming at the last message's own timestamp would redeliver
// it. The `<seconds>.<nanoseconds>` form is the daemon's full-precision accepted format.
func sinceCursor(message dockerevents.Message) string {
	nano := message.TimeNano
	if nano == 0 {
		// very old daemons only populate second precision.
		nano = message.Time * int64(time.Second)
	}
	if nano == 0 {
		return ""
	}

	nano++

	return fmt.Sprintf("%d.%09d", nano/int64(time.Second), nano%int64(time.Second))
}

// handleEvent dispatches one container start/die event.
func (h *dockerWatchHandler) handleEvent(ctx context.Context, message dockerevents.Message) error {
	if message.Type != dockerevents.ContainerEventType {
		return nil
	}

	containerName := message.Actor.Attributes["name"]
	if containerName == "" {
		return nil
	}

	service, ok := matchWatchedContainer(containerName)
	if !ok {
		return nil
	}

	switch message.Action {
	case dockerevents.ActionStart:
		return h.handleContainerStarted(ctx, message, containerName, service)

	case dockerevents.ActionDie:
		return h.handleContainerDied(ctx, containerName, service)

	default:
		return nil
	}
}

// handleContainerStarted seeds a fresh table entry and publishes the started event.
//
// The inspect call here is safe: the container is freshly running, so its NetworkSettings carry a
// real address. This is the ONLY event-time inspect -- never at die time, where a dead container's
// inspect response has an empty IPAddress and a die-time fallback would silently resolve nothing.
func (h *dockerWatchHandler) handleContainerStarted(ctx context.Context, message dockerevents.Message, containerName string, service string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "handleContainerStarted",
		"container_name": containerName,
	})

	ip := ""
	inspect, err := h.dockerClient.ContainerInspect(ctx, message.Actor.ID)
	if err != nil {
		// the entry is still created, with an empty IP: a later refresh pass cannot resolve it,
		// but the entry's existence is what lets the eventual `die` publish at all.
		log.Warnf("Could not inspect the started container. Its asterisk id will stay unresolved. err: %v", err)
	} else {
		ip = resolveContainerIP(inspect)
		if ip == "" {
			log.Warnf("Could not resolve the ip address of the started container. Its asterisk id will stay unresolved.")
		}
	}

	h.state.Create(containerName, service, ip, time.Now())
	log.Infof("Container started. service: %s, ip: %s", service, ip)

	h.notifyHandler.PublishEvent(ctx, smcontainer.EventTypeContainerStarted, &smcontainer.Event{
		ContainerName: containerName,
		Service:       service,
		AsteriskID:    "",
	})
	promContainerStateChangeCounter.WithLabelValues(containerName, service, stateStarted).Inc()

	return nil
}

// handleContainerDied consumes the entry this generation left behind and publishes the died
// event.
//
// It NEVER re-scans and never inspects: the asterisk-id was resolved before the death or it was
// never resolvable. Deleting the entry is part of the same critical section as reading it, so a
// same-name replacement container's `start` cannot interleave and have its data answer this
// death.
func (h *dockerWatchHandler) handleContainerDied(ctx context.Context, containerName string, service string) error {
	log := logrus.WithFields(logrus.Fields{
		"func":           "handleContainerDied",
		"container_name": containerName,
	})

	asteriskID := ""
	entry, ok := h.state.Delete(containerName)
	if ok {
		asteriskID = entry.AsteriskID
	} else {
		// no entry: the container both started and died entirely within sentinel's own downtime,
		// or before it ever reached the table.
		log.Warnf("Received a die event for a container with no state table entry. Publishing with an unresolved asterisk id.")
	}

	if !h.flap.Record(containerName, time.Now()) {
		log.Warnf(
			"The container is flapping. Skipping the died event to avoid repeated recovery attempts. threshold: %d, window: %v, asterisk_id: %s",
			flapThreshold, flapWindow, asteriskID,
		)
		return nil
	}

	if asteriskID == "" {
		promContainerUnresolvedAsteriskIDCounter.WithLabelValues(containerName).Inc()
		log.Warnf("Publishing a died event without a resolved asterisk id. No recovery will be triggered for this container. service: %s", service)
	}

	log.Infof("Container died. service: %s, asterisk_id: %s", service, asteriskID)
	h.notifyHandler.PublishEvent(ctx, smcontainer.EventTypeContainerDied, &smcontainer.Event{
		ContainerName: containerName,
		Service:       service,
		AsteriskID:    asteriskID,
	})
	promContainerStateChangeCounter.WithLabelValues(containerName, service, stateDied).Inc()

	return nil
}
