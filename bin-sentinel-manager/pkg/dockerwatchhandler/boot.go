package dockerwatchhandler

import (
	"context"
	"time"

	dockercontainer "github.com/docker/docker/api/types/container"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// bootReconcile seeds the state table from the containers already running when sentinel starts
// (design §3.3 step 0).
//
// Without this, the table would only ever be populated by `start` EVENTS, which sentinel observes
// only for containers that start after it is already watching. Every sentinel restart -- its own
// crash, or a routine redeploy -- would leave the already-running Asterisk containers with no
// entry, an INDEFINITE blind spot lasting until their next recreation rather than a bounded one,
// since sentinel runs single-replica and has no sibling to cover for it while down.
//
// A failure here is returned, not swallowed: sentinel must fail loud if the docker-socket-proxy is
// unreachable. A sentinel that LOOKS up but watches nothing is worse than one that is visibly
// down.
func (h *dockerWatchHandler) bootReconcile(ctx context.Context) error {
	log := logrus.WithField("func", "bootReconcile")

	summaries, err := h.dockerClient.ContainerList(ctx, dockercontainer.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "could not list the running containers")
	}

	seeded := 0
	for _, summary := range summaries {
		containerName := containerNameOf(summary)
		if containerName == "" {
			continue
		}

		service, ok := matchWatchedContainer(containerName)
		if !ok {
			continue
		}

		inspect, errInspect := h.dockerClient.ContainerInspect(ctx, summary.ID)
		if errInspect != nil {
			// one unreadable container must not abort the whole reconciliation -- the remaining
			// containers are still worth seeding, and this one gets its entry on its next start.
			log.Warnf("Could not inspect the running container. Skipping it. container_name: %s, err: %v", containerName, errInspect)
			continue
		}

		ip := resolveContainerIP(inspect)
		if ip == "" {
			log.Warnf("Could not resolve the ip address of the running container. Its asterisk id will stay unresolved. container_name: %s", containerName)
		}

		h.state.Create(containerName, service, ip, time.Now())
		seeded++
		log.Infof("Seeded the state table from a running container. container_name: %s, service: %s, ip: %s", containerName, service, ip)
	}

	log.Infof("Completed the boot-time reconciliation. seeded: %d", seeded)

	return nil
}
