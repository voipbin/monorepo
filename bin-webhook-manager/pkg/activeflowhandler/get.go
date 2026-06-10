package activeflowhandler

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/requesthandler"

	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	mwactiveflow "monorepo/bin-webhook-manager/models/activeflow"
	"monorepo/bin-webhook-manager/models/webhook"
)

// Get resolves the per-activeflow webhook destination for the given
// activeflowID. See the interface doc for the return contract.
func (h *activeflowHandler) Get(ctx context.Context, activeflowID uuid.UUID) (*Destination, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "Get",
		"activeflow_id": activeflowID,
	})

	// 1. cache lookup
	entry, found, err := h.cache.ActiveflowWebhookGet(ctx, activeflowID)
	if err != nil {
		// Redis error: treat as a miss but do not poison the cache; fall back.
		log.Errorf("Could not get the activeflow webhook from cache. err: %v", err)
	}

	if err == nil && found {
		if entry.IsPositive() {
			promActiveflowCacheTotal.WithLabelValues("hit_positive").Inc()
			return &Destination{URI: entry.URI, Method: entry.Method}, nil
		}
		// negative hit: no webhook configured.
		promActiveflowCacheTotal.WithLabelValues("hit_negative").Inc()
		return nil, nil
	}

	// 2. miss -> singleflight-coalesced fallback.
	promActiveflowCacheTotal.WithLabelValues("miss").Inc()

	res, errFallback, _ := h.sfGroup.Do(activeflowID.String(), func() (interface{}, error) {
		return h.fallback(ctx, activeflowID)
	})
	if errFallback != nil {
		// rpc/transport error: do not cache, skip the extra delivery.
		log.Errorf("Could not resolve the activeflow webhook via fallback. err: %v", errFallback)
		return nil, nil
	}

	dest, _ := res.(*Destination)
	return dest, nil
}

// fallback performs the one-shot RPC and classifies the result per design 5.4,
// backfilling the cache accordingly. It returns a positive *Destination when a
// deliverable destination exists, otherwise nil.
func (h *activeflowHandler) fallback(ctx context.Context, activeflowID uuid.UUID) (*Destination, error) {
	log := logrus.WithFields(logrus.Fields{
		"func":          "fallback",
		"activeflow_id": activeflowID,
	})

	af, err := h.reqHandler.FlowV1ActiveflowGet(ctx, activeflowID)
	if err != nil {
		if errors.Cause(err) == requesthandler.ErrNotFound {
			// NotFound: treat as transient (flow-manager lag). Cache a very short
			// negative so the next event retries soon.
			promActiveflowResolveTotal.WithLabelValues("notfound").Inc()
			now := time.Now()
			if errSet := h.cache.ActiveflowWebhookSetNegative(ctx, activeflowID, now, nil, h.tTransient); errSet != nil {
				log.Errorf("Could not set the transient negative cache. err: %v", errSet)
			}
			return nil, nil
		}

		// rpc/transport error: do not cache; signal no delivery.
		promActiveflowResolveTotal.WithLabelValues("rpc_error").Inc()
		return nil, errors.Wrapf(err, "could not get the activeflow")
	}

	tm := fallbackTimestamp(af)

	// soft-deleted: cache NEGATIVE, never positive.
	if af.TMDelete != nil {
		promActiveflowResolveTotal.WithLabelValues("negative").Inc()
		if errSet := h.cache.ActiveflowWebhookSetNegative(ctx, activeflowID, tm, af.TMDelete, h.tNeg); errSet != nil {
			log.Errorf("Could not set the negative cache. err: %v", errSet)
		}
		return nil, nil
	}

	// normal + empty uri: cache NEGATIVE.
	if af.WebhookURI == "" {
		promActiveflowResolveTotal.WithLabelValues("negative").Inc()
		if errSet := h.cache.ActiveflowWebhookSetNegative(ctx, activeflowID, tm, nil, h.tNeg); errSet != nil {
			log.Errorf("Could not set the negative cache. err: %v", errSet)
		}
		return nil, nil
	}

	// normal + uri set: cache POSITIVE, deliver.
	promActiveflowResolveTotal.WithLabelValues("positive").Inc()
	method := webhook.MethodType(string(af.WebhookMethod))
	entry := &mwactiveflow.Webhook{
		URI:    af.WebhookURI,
		Method: method,
		Tm:     tm,
	}
	if errSet := h.cache.ActiveflowWebhookSet(ctx, activeflowID, entry, h.tLive); errSet != nil {
		log.Errorf("Could not set the positive cache. err: %v", errSet)
	}

	return &Destination{URI: af.WebhookURI, Method: method}, nil
}

// fallbackTimestamp returns the best source timestamp for monotonic ordering.
func fallbackTimestamp(af *fmactiveflow.Activeflow) time.Time {
	if af.TMUpdate != nil {
		return *af.TMUpdate
	}
	if af.TMCreate != nil {
		return *af.TMCreate
	}
	return time.Now()
}
