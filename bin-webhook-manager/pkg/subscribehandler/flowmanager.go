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
	// subTLive is the TTL for a positive entry (cache lifetime safety net).
	subTLive = 24 * time.Hour
	// subTNeg is the TTL for a negative entry (no webhook / deleted).
	subTNeg = 10 * time.Minute
)

// processEventFMActiveflowCreatedUpdated handles flow-manager's
// activeflow_created and activeflow_updated events.
//
// The lifecycle event payload carries webhook_uri / webhook_method (the
// activeflow's OWN data, returned only to the customer's OWN endpoint, design
// 5.2 / Option A). This handler therefore pre-populates the cache eagerly:
//   - webhook_uri set   → POSITIVE entry (ActiveflowWebhookSet, subTLive)
//   - webhook_uri empty → NEGATIVE entry (ActiveflowWebhookSetNegative, subTNeg)
//
// The monotonic Tm is the event timestamp so out-of-order writes self-correct
// (design 5.6). The activeflowhandler.Get fallback path remains the lazy/miss
// safety net for events that arrive before this lifecycle event is consumed.
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

	if e.WebhookURI != "" {
		// positive: pre-populate the deliverable destination.
		entry := &mwactiveflow.Webhook{
			URI:    e.WebhookURI,
			Method: webhook.MethodType(string(e.WebhookMethod)),
			Tm:     tm,
		}
		if err := h.cacheHandler.ActiveflowWebhookSet(ctx, e.ID, entry, subTLive); err != nil {
			log.Errorf("Could not set the positive cache. err: %v", err)
			return err
		}
		return nil
	}

	// negative: the activeflow has no per-activeflow webhook configured.
	if err := h.cacheHandler.ActiveflowWebhookSetNegative(ctx, e.ID, tm, nil, subTNeg); err != nil {
		log.Errorf("Could not set the negative cache. err: %v", err)
		return err
	}

	return nil
}

// processEventFMActiveflowDeleted handles flow-manager's activeflow_deleted
// event. It writes a NEGATIVE tombstone (carrying the delete timestamp), NOT a
// bare delete, so out-of-order writes self-correct (design 5.6). The deleted
// event carries id + tm_delete.
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
