package failedeventhandler

//go:generate mockgen -package failedeventhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
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
	db             *sql.DB
	eventProcessor EventProcessor
}

const (
	failedEventsTable = "billing_failed_events"
	maxRetries        = 5
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
func NewFailedEventHandler(db *sql.DB, processor EventProcessor) FailedEventHandler {
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

	q := `
	INSERT INTO ` + failedEventsTable + `
		(id, event_type, event_publisher, event_data, error_message, retry_count, max_retries, next_retry_at, status, tm_create, tm_update)
	VALUES
		(?, ?, ?, ?, ?, 0, ?, ?, 'pending', ?, ?)
	`

	_, err = h.db.ExecContext(ctx, q,
		id.Bytes(), event.Type, event.Publisher, eventData,
		processingErr.Error(), maxRetries, nextRetry, now, now,
	)
	if err != nil {
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

	rows, err := h.db.QueryContext(ctx,
		"SELECT id, event_type, event_publisher, event_data, retry_count, max_retries FROM "+failedEventsTable+" WHERE status IN ('pending', 'retrying') AND next_retry_at <= ?",
		now,
	)
	if err != nil {
		return fmt.Errorf("could not query failed events. err: %v", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var id []byte
		var eventType, publisher string
		var eventData []byte
		var retryCount, maxRetryCount int

		if err := rows.Scan(&id, &eventType, &publisher, &eventData, &retryCount, &maxRetryCount); err != nil {
			log.Errorf("Could not scan failed event row. err: %v", err)
			continue
		}

		// unmarshal the event
		var event sock.Event
		if err := json.Unmarshal(eventData, &event); err != nil {
			log.Errorf("Could not unmarshal failed event. err: %v", err)
			h.markExhausted(ctx, id, eventType)
			continue
		}

		// attempt retry
		if err := h.eventProcessor(&event); err != nil {
			promFailedEventRetryTotal.WithLabelValues("failure").Inc()
			newRetryCount := retryCount + 1

			if newRetryCount >= maxRetryCount {
				log.Errorf("Failed event exhausted all retries. event_type: %s, publisher: %s", eventType, publisher)
				promFailedEventExhaustedTotal.WithLabelValues(eventType).Inc()
				h.markExhausted(ctx, id, eventType)
				continue
			}

			// exponential backoff: 1m, 5m, 25m, 125m, 625m
			backoff := time.Duration(math.Pow(5, float64(newRetryCount))) * time.Minute
			nextRetry := now.Add(backoff)

			_, errUpdate := h.db.ExecContext(ctx,
				"UPDATE "+failedEventsTable+" SET retry_count = ?, next_retry_at = ?, status = 'retrying', tm_update = ? WHERE id = ?",
				newRetryCount, nextRetry, now, id,
			)
			if errUpdate != nil {
				log.Errorf("Could not update failed event retry. err: %v", errUpdate)
			}
			log.Infof("Retrying failed event. retry_count: %d, next_retry_at: %s", newRetryCount, nextRetry)
			continue
		}

		// success - delete the record
		promFailedEventRetryTotal.WithLabelValues("success").Inc()
		_, errDelete := h.db.ExecContext(ctx, "DELETE FROM "+failedEventsTable+" WHERE id = ?", id)
		if errDelete != nil {
			log.Errorf("Could not delete retried event. err: %v", errDelete)
		}
		log.Infof("Successfully retried failed event. event_type: %s, publisher: %s", eventType, publisher)
	}

	return nil
}

// markExhausted marks a failed event as exhausted.
func (h *failedEventHandler) markExhausted(ctx context.Context, id []byte, eventType string) {
	now := time.Now().UTC()
	_, err := h.db.ExecContext(ctx,
		"UPDATE "+failedEventsTable+" SET status = 'exhausted', tm_update = ? WHERE id = ?",
		now, id,
	)
	if err != nil {
		logrus.Errorf("Could not mark failed event as exhausted. err: %v", err)
	}
}
