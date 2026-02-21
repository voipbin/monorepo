package queue

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	// basic info
	Name   string `json:"name,omitempty"`   // queue's name
	Detail string `json:"detail,omitempty"` // queue's detail

	// operation info
	RoutingMethod RoutingMethod `json:"routing_method,omitempty"` // queue's routing method
	TagIDs        []uuid.UUID   `json:"tag_ids,omitempty"`        // queue's tag ids

	// wait/service info
	WaitFlowID     uuid.UUID `json:"wait_flow_id,omitempty"`    // flow id for queue waiting
	WaitTimeout    int       `json:"wait_timeout,omitempty"`    // wait queue timeout.(ms)
	ServiceTimeout int       `json:"service_timeout,omitempty"` // service queue timeout(ms).

	// queuecall info
	WaitQueuecallIDs    []uuid.UUID `json:"wait_queuecall_ids,omitempty"`    // waiting queue call ids.
	ServiceQueuecallIDs []uuid.UUID `json:"service_queuecall_ids,omitempty"` // service queue call ids(ms).

	TotalIncomingCount  int `json:"total_incoming_count,omitempty"`  // total incoming call count
	TotalServicedCount  int `json:"total_serviced_count,omitempty"`  // total serviced call count
	TotalAbandonedCount int `json:"total_abandoned_count,omitempty"` // total abandoned call count

	TMCreate *time.Time `json:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage Convert to the publishable message.
func (h *Queue) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:          h.Name,
		Detail:        h.Detail,
		RoutingMethod: h.RoutingMethod,
		TagIDs:        h.TagIDs,

		WaitFlowID:     h.WaitFlowID,
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
