package queue

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"` // queue id

	// basic info
	Name   string `json:"name"`   // queue's name
	Detail string `json:"detail"` // queue's detail

	// operation info
	RoutingMethod RoutingMethod `json:"routing_method"` // queue's routing method
	TagIDs        []uuid.UUID   `json:"tag_ids"`        // queue's tag ids

	// wait/service info
	WaitActions    []fmaction.Action `json:"wait_actions"`    // actions for queue waiting
	WaitTimeout    int               `json:"wait_timeout"`    // wait queue timeout.(ms)
	ServiceTimeout int               `json:"service_timeout"` // service queue timeout(ms).

	// queuecall info
	WaitQueueCallIDs    []uuid.UUID `json:"wait_queue_call_ids"`    // waiting queue call ids.
	ServiceQueueCallIDs []uuid.UUID `json:"service_queue_call_ids"` // service queue call ids(ms).

	TotalIncomingCount   int `json:"total_incoming_count"`   // total incoming call count
	TotalServicedCount   int `json:"total_serviced_count"`   // total serviced call count
	TotalAbandonedCount  int `json:"total_abandoned_count"`  // total abandoned call count
	TotalWaitDuration    int `json:"total_waittime"`         // total wait time(ms)
	TotalServiceDuration int `json:"total_service_duration"` // total service duration(ms)

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage Convert to the publishable message.
func (h *Queue) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:            h.ID,
		Name:          h.Name,
		Detail:        h.Detail,
		RoutingMethod: h.RoutingMethod,
		TagIDs:        h.TagIDs,

		WaitActions:    h.WaitActions,
		WaitTimeout:    h.WaitTimeout,
		ServiceTimeout: h.ServiceTimeout,

		WaitQueueCallIDs:    h.WaitQueueCallIDs,
		ServiceQueueCallIDs: h.ServiceQueueCallIDs,

		TotalIncomingCount:   h.TotalIncomingCount,
		TotalServicedCount:   h.TotalServicedCount,
		TotalAbandonedCount:  h.TotalAbandonedCount,
		TotalWaitDuration:    h.TotalWaitDuration,
		TotalServiceDuration: h.TotalServiceDuration,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent byte
func (h *Queue) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}
