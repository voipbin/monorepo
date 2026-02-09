package failedeventhandler

//go:generate mockgen -package failedeventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"monorepo/bin-billing-manager/models/failedevent"
	"monorepo/bin-billing-manager/pkg/dbhandler"
)

// EventProcessor is a callback function that processes a sock event.
type EventProcessor func(m *sock.Event) error

// FailedEventHandler manages persistence and retry of failed billing events.
type FailedEventHandler interface {
	Save(ctx context.Context, event *sock.Event, processingErr error) error
	RetryPending(ctx context.Context) error
}

type failedEventHandler struct {
	utilHandler    utilhandler.UtilHandler
	db             dbhandler.DBHandler
	eventProcessor EventProcessor
}

const (
	maxRetries = 5
)

var (
	metricsNamespace = "billing_manager"

	promFailedEventSaveTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "failed_event_save_total",
			Help:      "Total number of failed events saved for retry.",
		},
		[]string{"event_type", "publisher"},
	)

	promFailedEventRetryTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "failed_event_retry_total",
			Help:      "Total number of failed event retry attempts.",
		},
		[]string{"result"},
	)

	promFailedEventExhaustedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "failed_event_exhausted_total",
			Help:      "Total number of failed events that exhausted all retries.",
		},
		[]string{"event_type"},
	)
)

func init() {
	prometheus.MustRegister(
		promFailedEventSaveTotal,
		promFailedEventRetryTotal,
		promFailedEventExhaustedTotal,
	)
}

// NewFailedEventHandler creates a new FailedEventHandler.
func NewFailedEventHandler(db dbhandler.DBHandler, processor EventProcessor) FailedEventHandler {
	return &failedEventHandler{
		utilHandler:    utilhandler.NewUtilHandler(),
		db:             db,
		eventProcessor: processor,
	}
}

// Save persists a failed event for later retry with exponential backoff.
func (h *failedEventHandler) Save(ctx context.Context, event *sock.Event, processingErr error) error {
	log := logrus.WithFields(logrus.Fields{
		"func":      "Save",
		"type":      event.Type,
		"publisher": event.Publisher,
	})

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("could not marshal event. err: %v", err)
	}

	id := h.utilHandler.UUIDCreate()
	now := time.Now().UTC()
	nextRetry := now.Add(1 * time.Minute) // first retry in 1 minute

	fe := &failedevent.FailedEvent{
		ID:             id,
		EventType:      string(event.Type),
		EventPublisher: event.Publisher,
		EventData:      string(eventData),
		ErrorMessage:   processingErr.Error(),
		RetryCount:     0,
		MaxRetries:     maxRetries,
		NextRetryAt:    &nextRetry,
		Status:         failedevent.StatusPending,
	}

	if err := h.db.FailedEventCreate(ctx, fe); err != nil {
		log.Errorf("Could not save failed event. err: %v", err)
		return fmt.Errorf("could not save failed event. err: %v", err)
	}

	promFailedEventSaveTotal.WithLabelValues(string(event.Type), event.Publisher).Inc()
	log.Infof("Saved failed event for retry. event_id: %s, next_retry_at: %s", id, nextRetry)

	return nil
}

// RetryPending processes all pending failed events that are due for retry.
func (h *failedEventHandler) RetryPending(ctx context.Context) error {
	log := logrus.WithField("func", "RetryPending")

	now := time.Now().UTC()

	events, err := h.db.FailedEventListPendingRetry(ctx, now)
	if err != nil {
		return fmt.Errorf("could not query failed events. err: %v", err)
	}

	for _, fe := range events {
		// unmarshal the event
		var event sock.Event
		if err := json.Unmarshal([]byte(fe.EventData), &event); err != nil {
			log.Errorf("Could not unmarshal failed event. err: %v", err)
			h.markExhausted(ctx, fe)
			continue
		}

		// attempt retry
		if err := h.eventProcessor(&event); err != nil {
			promFailedEventRetryTotal.WithLabelValues("failure").Inc()
			newRetryCount := fe.RetryCount + 1

			if newRetryCount >= fe.MaxRetries {
				log.Errorf("Failed event exhausted all retries. event_type: %s, publisher: %s", fe.EventType, fe.EventPublisher)
				promFailedEventExhaustedTotal.WithLabelValues(fe.EventType).Inc()
				h.markExhausted(ctx, fe)
				continue
			}

			// exponential backoff: 1m, 5m, 25m, 125m, 625m
			backoff := time.Duration(math.Pow(5, float64(newRetryCount))) * time.Minute
			nextRetry := now.Add(backoff)

			fields := map[failedevent.Field]any{
				failedevent.FieldRetryCount:  newRetryCount,
				failedevent.FieldNextRetryAt: nextRetry,
				failedevent.FieldStatus:      failedevent.StatusRetrying,
			}
			if errUpdate := h.db.FailedEventUpdate(ctx, fe.ID, fields); errUpdate != nil {
				log.Errorf("Could not update failed event retry. err: %v", errUpdate)
			}
			log.Infof("Retrying failed event. retry_count: %d, next_retry_at: %s", newRetryCount, nextRetry)
			continue
		}

		// success - delete the record
		promFailedEventRetryTotal.WithLabelValues("success").Inc()
		if errDelete := h.db.FailedEventDelete(ctx, fe.ID); errDelete != nil {
			log.Errorf("Could not delete retried event. err: %v", errDelete)
		}
		log.Infof("Successfully retried failed event. event_type: %s, publisher: %s", fe.EventType, fe.EventPublisher)
	}

	return nil
}

// markExhausted marks a failed event as exhausted.
func (h *failedEventHandler) markExhausted(ctx context.Context, fe *failedevent.FailedEvent) {
	fields := map[failedevent.Field]any{
		failedevent.FieldStatus: failedevent.StatusExhausted,
	}
	if err := h.db.FailedEventUpdate(ctx, fe.ID, fields); err != nil {
		logrus.Errorf("Could not mark failed event as exhausted. err: %v", err)
	}
}
