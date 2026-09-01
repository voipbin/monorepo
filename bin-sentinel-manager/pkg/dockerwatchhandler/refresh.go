package dockerwatchhandler

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"monorepo/bin-sentinel-manager/models/asteriskaddress"
)

// runRefreshLoop re-resolves table entries' asterisk-ids on a fixed cadence until ctx is done.
//
// It is independent of the event stream on purpose: an id must already be known BEFORE the death
// that consumes it, so resolution cannot be event-driven.
func (h *dockerWatchHandler) runRefreshLoop(ctx context.Context) {
	log := logrus.WithField("func", "runRefreshLoop")

	ticker := time.NewTicker(h.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Context cancelled. Stopping the asterisk-id refresh loop.")
			return

		case <-ticker.C:
			if errRefresh := h.refreshOnce(ctx); errRefresh != nil {
				// a failed pass is not fatal and MUST NOT clear anything: it simply means this
				// pass learned nothing.
				log.Warnf("Could not refresh the asterisk ids. err: %v", errRefresh)
			}
		}
	}
}

// refreshOnce runs a single resolution pass over the whole state table.
//
// The algorithm, and specifically what it refuses to do, is design §3.3 step 2:
//
//   - Only a FRESH key (remaining ttl >= 24h - 12min) counts as evidence about the current
//     occupant of an IP. The 24h TTL is refreshed every 5 minutes, so a dead generation's key for
//     an IP can coexist with the live generation's key for the SAME IP for up to 24h; without the
//     filter, an unfiltered ip->id map silently picks whichever the scan yielded last.
//   - A pass that finds no fresh candidate for an entry LEAVES THAT ENTRY'S ID UNCHANGED. The
//     freshness filter gates LEARNING, never FORGETTING. Regressing a resolved id to "" would,
//     combined with call-manager's empty-id guard, silently skip the exact recovery this service
//     exists to trigger.
//   - An id already bound to a DIFFERENT container name currently alive in the table is excluded,
//     which narrows the pathological "two generations of the same IP both inside the freshness
//     window" case. It does not eliminate it: a same-second overlap remains a documented residual.
func (h *dockerWatchHandler) refreshOnce(ctx context.Context) error {
	log := logrus.WithField("func", "refreshOnce")

	entries := h.state.List()
	if len(entries) == 0 {
		return nil
	}

	addresses, err := h.cacheHandler.AsteriskAddressInternalScan(ctx)
	if err != nil {
		return errors.Wrap(err, "could not scan the asterisk addresses")
	}

	freshByIP := freshCandidatesByIP(addresses)
	boundIDs := boundAsteriskIDs(entries)

	for _, entry := range entries {
		candidates := selectCandidates(freshByIP[entry.IP], entry, boundIDs)

		switch len(candidates) {
		case 0:
			if entry.AsteriskID == "" {
				log.Debugf("No fresh asterisk address for the unresolved container yet. container_name: %s, ip: %s", entry.ContainerName, entry.IP)
				continue
			}

			// sticky-last-known: keep the resolved id. This is the leading indicator that the
			// NEXT death for this container may go unrecovered.
			promContainerRefreshMissCounter.WithLabelValues(entry.ContainerName).Inc()
			log.Warnf(
				"Found no fresh asterisk address for an already-resolved container. Keeping the last known asterisk id. container_name: %s, ip: %s, asterisk_id: %s",
				entry.ContainerName, entry.IP, entry.AsteriskID,
			)

		case 1:
			resolved := candidates[0]
			if resolved == entry.AsteriskID {
				continue
			}

			if entry.AsteriskID != "" {
				// A same-entry id CHANGE contradicts the invariant this whole design rests on:
				// the asterisk-id derives from the container's MAC, which is fixed for that
				// container object's entire lifetime, and one entry spans exactly one container
				// generation (created on start/boot, destroyed on die). One live container writes
				// one Redis key with one fixed id, so this branch structurally should not fire.
				//
				// If it does, we are looking at either a real anomaly (a second container claiming
				// this IP) or a latent bug in the resolution path, and the new value is no more
				// trustworthy than the old one. KEEP THE OLD ID -- the conservative choice: the
				// old id was resolved while this generation was demonstrably alive, whereas
				// adopting an unexplained new one risks firing recovery against a DIFFERENT,
				// still-live instance and redialing channels that never dropped. The WARN below
				// makes the anomaly observable either way, which is what makes "keep" safe rather
				// than merely silent.
				log.Warnf(
					"Resolved a DIFFERENT asterisk id for a container that already had one. This contradicts the fixed-MAC-per-generation invariant. Keeping the existing id. container_name: %s, ip: %s, existing_asterisk_id: %s, rejected_asterisk_id: %s",
					entry.ContainerName, entry.IP, entry.AsteriskID, resolved,
				)
				continue
			}

			if !h.state.Resolve(entry.ContainerName, resolved) {
				// the entry died between List and Resolve. Nothing to do -- the die handler
				// already consumed whatever it held.
				log.Debugf("The container entry disappeared during the refresh pass. container_name: %s", entry.ContainerName)
				continue
			}
			log.Infof("Resolved the asterisk id for the container. container_name: %s, ip: %s, asterisk_id: %s", entry.ContainerName, entry.IP, resolved)

		default:
			// two generations' keys for the same IP are both inside the freshness window. Sticky
			// again: an ambiguous pass has not learned anything trustworthy.
			log.Warnf(
				"Found multiple fresh asterisk addresses for one container ip. Keeping the last known asterisk id. container_name: %s, ip: %s, candidates: %v, asterisk_id: %s",
				entry.ContainerName, entry.IP, candidates, entry.AsteriskID,
			)
		}
	}

	return nil
}

// freshCandidatesByIP groups the FRESH addresses by IP. Non-fresh keys are dropped entirely --
// they are not evidence in either direction.
func freshCandidatesByIP(addresses []*asteriskaddress.AsteriskAddress) map[string][]string {
	res := map[string][]string{}

	for _, address := range addresses {
		if address == nil || address.Address == "" || address.ID == "" {
			continue
		}
		if !address.IsFresh() {
			continue
		}

		res[address.Address] = append(res[address.Address], address.ID)
	}

	return res
}

// boundAsteriskIDs maps every already-resolved asterisk-id to the container name holding it.
func boundAsteriskIDs(entries []*containerState) map[string]string {
	res := map[string]string{}

	for _, entry := range entries {
		if entry.AsteriskID == "" {
			continue
		}

		res[entry.AsteriskID] = entry.ContainerName
	}

	return res
}

// selectCandidates filters an IP's fresh ids down to those that could plausibly belong to this
// entry: an id already bound to a DIFFERENT live container name is somebody else's and is
// excluded. The entry's own current id is of course kept.
func selectCandidates(freshIDs []string, entry *containerState, boundIDs map[string]string) []string {
	res := make([]string, 0, len(freshIDs))

	for _, id := range freshIDs {
		owner, bound := boundIDs[id]
		if bound && owner != entry.ContainerName {
			continue
		}

		res = append(res, id)
	}

	return res
}
