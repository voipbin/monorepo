package healthcheckhandler

import (
	"context"
	"errors"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/provider"
)

const (
	healthCheckPageSize uint64 = 100
	timeLayout                 = "2006-01-02T15:04:05.000000Z"

	// healthCheckLockName is the fixed cross-replica lock name for the
	// whole health check cycle (not per-provider) - only one replica
	// runs runOnce's cycle at a time.
	healthCheckLockName = "route-manager:healthcheck-lock"
	// healthCheckLockExpiry is the initial TTL, sized to comfortably
	// cover one page (100 providers) of probing. It is refreshed via
	// ExtendContext after every page, so this value only bounds the
	// worst case before the first extension - it does not need to cover
	// the whole cycle.
	healthCheckLockExpiry = 30 * time.Second
)

// Run starts the provider health check loop. Blocks until ctx is cancelled.
func (h *healthCheckHandler) Run(ctx context.Context, interval time.Duration) {
	log := logrus.WithField("func", "Run")
	log.Infof("Starting provider health check loop. interval: %s", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping provider health check loop.")
			return
		case <-ticker.C:
			if err := h.runOnce(ctx); err != nil {
				log.Errorf("Provider health check cycle failed. err: %v", err)
			}
		}
	}
}

// runOnce iterates all active providers and checks each one sequentially.
// A cross-replica redsync lock ensures only one replica runs a cycle at a
// time; see acquireLock for the fail-open/skip decision on lock errors.
func (h *healthCheckHandler) runOnce(ctx context.Context) error {
	log := logrus.WithField("func", "runOnce")
	log.Debug("Running provider health check cycle.")

	mutex, locked, skip := h.acquireLock(ctx, log)
	if skip {
		return nil
	}
	if locked {
		defer func() {
			if _, errUnlock := mutex.UnlockContext(ctx); errUnlock != nil {
				log.Warnf("Could not release the healthcheck lock. err: %v", errUnlock)
			}
		}()
	}

	start := time.Now()
	token := ""
	checked := 0

	for {
		providers, err := h.db.ProviderList(ctx, token, healthCheckPageSize, map[provider.Field]any{})
		if err != nil {
			return err
		}

		for _, p := range providers {
			h.checkProvider(ctx, p)
			checked++
		}

		// Refresh the lock every page. redsync does not auto-renew, so
		// without this an oversized provider list would let the lock
		// expire mid-cycle and a second replica could start probing
		// concurrently - exactly what this lock exists to prevent. An
		// extend failure is treated the same as the initial-acquire
		// Redis-error case (fail-open, log and keep going): aborting
		// would not stop a peer from taking over once the TTL truly
		// lapses, so it would only leave the remaining providers in
		// this cycle unchecked for no benefit.
		if locked {
			if _, errExtend := mutex.ExtendContext(ctx); errExtend != nil {
				log.Warnf("Could not extend the healthcheck lock; continuing this cycle unlocked (fail-open, may double-probe remaining providers). err: %v", errExtend)
			}
		}

		if uint64(len(providers)) < healthCheckPageSize {
			break
		}

		last := providers[len(providers)-1]
		if last.TMCreate == nil {
			break
		}
		token = last.TMCreate.UTC().Format(timeLayout)
	}

	log.Debugf("Provider health check cycle complete. checked: %d, elapsed: %s", checked, time.Since(start))
	return nil
}

// acquireLock attempts to claim the healthcheck lock for this cycle.
// Returns skip=true when a peer already holds it (caller must return
// immediately without probing). Returns locked=false (with skip=false) when
// Redis itself is unreachable or any other unexpected error occurs - the
// caller proceeds to probe unlocked (fail-open): ProviderUpdateHealthStatus
// is a blind UPDATE with no CAS, so a concurrent unlocked write from another
// replica is harmless (last-write-wins on the same real-world status); the
// only cost is transiently doubled SIP OPTIONS probe traffic, the same
// tradeoff already accepted during a Komodo cutover overlap (see
// docs/operations.md).
func (h *healthCheckHandler) acquireLock(ctx context.Context, log *logrus.Entry) (mutex redsyncMutex, locked bool, skip bool) {
	mutex = h.locker.NewMutex(healthCheckLockName, redsync.WithExpiry(healthCheckLockExpiry))

	err := mutex.TryLockContext(ctx)
	if err == nil {
		return mutex, true, false
	}

	var errTaken *redsync.ErrTaken
	if errors.As(err, &errTaken) {
		log.Debug("Another replica already holds the healthcheck lock. Skipping this cycle.")
		return nil, false, true
	}

	var redisErr *redsync.RedisError
	if errors.As(err, &redisErr) {
		log.Warnf("Could not reach Redis to acquire the healthcheck lock; proceeding without a lock (fail-open, may double-probe providers). err: %v", err)
		return nil, false, false
	}

	log.Warnf("Unexpected error acquiring the healthcheck lock; proceeding without a lock (fail-open). err: %v", err)
	return nil, false, false
}

// checkProvider sends a SIP OPTIONS probe and updates the provider's health status.
func (h *healthCheckHandler) checkProvider(ctx context.Context, p *provider.Provider) {
	log := logrus.WithFields(logrus.Fields{
		"func":        "checkProvider",
		"provider_id": p.ID,
		"hostname":    p.Hostname,
	})

	result, err := h.reqHandler.KamailioV1ProviderHealthCheck(ctx, p.Hostname)
	if err != nil {
		log.Errorf("Could not check provider health. err: %v", err)
		return
	}
	log.WithField("result", result).Debugf("Received health check result.")

	now := time.Now()
	if errUpdate := h.db.ProviderUpdateHealthStatus(ctx, p.ID, result.Status, &now); errUpdate != nil {
		log.Errorf("Could not update provider health status. err: %v", errUpdate)
	}
}
