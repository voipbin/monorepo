package healthcheckhandler

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"

	"monorepo/bin-route-manager/models/provider"
)

const (
	healthCheckPageSize uint64 = 100
	timeLayout                 = "2006-01-02T15:04:05.000000Z"
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
func (h *healthCheckHandler) runOnce(ctx context.Context) error {
	log := logrus.WithField("func", "runOnce")
	log.Debug("Running provider health check cycle.")

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
