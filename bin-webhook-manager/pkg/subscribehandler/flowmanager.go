package subscribehandler

import (
	"context"
	"encoding/json"
	"time"

	"monorepo/bin-common-handler/models/sock"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/sirupsen/logrus"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
	"monorepo/bin-webhook-manager/models/webhook"
)

// default cache TTLs (design 5.3). Kept package-private; mirror activeflowhandler.
const (
	subTLive = 24 * time.Hour
	subTNeg  = 10 * time.Minute
)

// processEventFMActiveflowCreatedUpdated handles flow-manager's
// activeflow_created and activeflow_updated events.
//
// A positive entry is cached when webhook_uri is set, otherwise a negative
// entry. Writes are monotonic (cachehandler guards by timestamp) so a late
// event cannot overwrite a newer one (design 5.6).
func (h *subscribeHandler) processEventFMActiveflowCreatedUpdated(m *sock.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventFMActiveflowCreatedUpdated",
		"event": m,
	})
	log.Debugf("Received activeflow event. event: %s", m.Type)

	e := &fmactiveflow.WebhookMessage{}
	if err := json.Unmarshal([]byte(m.Data), e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	tm := activeflowEventTimestamp(e)

	if e.WebhookURI == "" {
		// no per-activeflow webhook: cache negative.
		if err := h.cacheHandler.ActiveflowWebhookSetNegative(ctx, e.ID, tm, nil, subTNeg); err != nil {
			log.Errorf("Could not set the negative activeflow webhook cache. err: %v", err)
			return err
		}
		return nil
	}

	// positive: cache the destination.
	entry := &mwactiveflow.Webhook{
		URI:    e.WebhookURI,
		Method: webhook.MethodType(string(e.WebhookMethod)),
		Tm:     tm,
	}
	if err := h.cacheHandler.ActiveflowWebhookSet(ctx, e.ID, entry, subTLive); err != nil {
		log.Errorf("Could not set the positive activeflow webhook cache. err: %v", err)
		return err
	}

	return nil
}

// processEventFMActiveflowDeleted handles flow-manager's activeflow_deleted
// event. It writes a NEGATIVE tombstone (carrying the delete timestamp), NOT a
// bare delete, so out-of-order writes self-correct (design 5.6).
func (h *subscribeHandler) processEventFMActiveflowDeleted(m *sock.Event) error {
	ctx := context.Background()
	log := logrus.WithFields(logrus.Fields{
		"func":  "processEventFMActiveflowDeleted",
		"event": m,
	})
	log.Debugf("Received activeflow deleted event. event: %s", m.Type)

	e := &fmactiveflow.WebhookMessage{}
	if err := json.Unmarshal([]byte(m.Data), e); err != nil {
		log.Errorf("Could not unmarshal the data. err: %v", err)
		return err
	}

	tm := activeflowEventTimestamp(e)
	if err := h.cacheHandler.ActiveflowWebhookSetNegative(ctx, e.ID, tm, e.TMDelete, subTNeg); err != nil {
		log.Errorf("Could not set the delete tombstone. err: %v", err)
		return err
	}

	return nil
}

// activeflowEventTimestamp returns the source timestamp used for monotonic
// ordering: delete > update > create.
func activeflowEventTimestamp(e *fmactiveflow.WebhookMessage) time.Time {
	if e.TMDelete != nil {
		return *e.TMDelete
	}
	if e.TMUpdate != nil {
		return *e.TMUpdate
	}
	if e.TMCreate != nil {
		return *e.TMCreate
	}
	return time.Now()
}
