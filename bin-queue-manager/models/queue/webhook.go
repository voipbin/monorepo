package queue

import (
	"encoding/json"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`          // queue id
	CustomerID uuid.UUID `json:"customer_id"` // owner id

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
	WaitQueuecallIDs    []uuid.UUID `json:"wait_queuecall_ids"`    // waiting queue call ids.
	ServiceQueuecallIDs []uuid.UUID `json:"service_queuecall_ids"` // service queue call ids(ms).

	TotalIncomingCount  int `json:"total_incoming_count"`  // total incoming call count
	TotalServicedCount  int `json:"total_serviced_count"`  // total serviced call count
	TotalAbandonedCount int `json:"total_abandoned_count"` // total abandoned call count

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage Convert to the publishable message.
func (h *Queue) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Name:          h.Name,
		Detail:        h.Detail,
		RoutingMethod: h.RoutingMethod,
		TagIDs:        h.TagIDs,

		WaitActions:    h.WaitActions,
		WaitTimeout:    h.WaitTimeout,
		ServiceTimeout: h.ServiceTimeout,

		WaitQueuecallIDs:    h.WaitQueuecallIDs,
		ServiceQueuecallIDs: h.ServiceQueuecallIDs,

		TotalIncomingCount:  h.TotalIncomingCount,
		TotalServicedCount:  h.TotalServicedCount,
		TotalAbandonedCount: h.TotalAbandonedCount,

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
