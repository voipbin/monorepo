package queue

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

func TestQueueStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	waitFlowID := uuid.Must(uuid.NewV4())

	q := Queue{
		Name:          "Support Queue",
		Detail:        "Customer support queue",
		RoutingMethod: RoutingMethodRandom,
		Execute:       ExecuteRun,
		WaitFlowID:    waitFlowID,
		WaitTimeout:   60000,
		ServiceTimeout: 300000,
		TotalIncomingCount:  100,
		TotalServicedCount:  80,
		TotalAbandonedCount: 20,
	}
	q.ID = id
	q.CustomerID = customerID

	if q.ID != id {
		t.Errorf("Queue.ID = %v, expected %v", q.ID, id)
	}
	if q.CustomerID != customerID {
		t.Errorf("Queue.CustomerID = %v, expected %v", q.CustomerID, customerID)
	}
	if q.Name != "Support Queue" {
		t.Errorf("Queue.Name = %v, expected %v", q.Name, "Support Queue")
	}
	if q.Detail != "Customer support queue" {
		t.Errorf("Queue.Detail = %v, expected %v", q.Detail, "Customer support queue")
	}
	if q.RoutingMethod != RoutingMethodRandom {
		t.Errorf("Queue.RoutingMethod = %v, expected %v", q.RoutingMethod, RoutingMethodRandom)
	}
	if q.Execute != ExecuteRun {
		t.Errorf("Queue.Execute = %v, expected %v", q.Execute, ExecuteRun)
	}
	if q.WaitFlowID != waitFlowID {
		t.Errorf("Queue.WaitFlowID = %v, expected %v", q.WaitFlowID, waitFlowID)
	}
	if q.WaitTimeout != 60000 {
		t.Errorf("Queue.WaitTimeout = %v, expected %v", q.WaitTimeout, 60000)
	}
	if q.ServiceTimeout != 300000 {
		t.Errorf("Queue.ServiceTimeout = %v, expected %v", q.ServiceTimeout, 300000)
	}
	if q.TotalIncomingCount != 100 {
		t.Errorf("Queue.TotalIncomingCount = %v, expected %v", q.TotalIncomingCount, 100)
	}
	if q.TotalServicedCount != 80 {
		t.Errorf("Queue.TotalServicedCount = %v, expected %v", q.TotalServicedCount, 80)
	}
	if q.TotalAbandonedCount != 20 {
		t.Errorf("Queue.TotalAbandonedCount = %v, expected %v", q.TotalAbandonedCount, 20)
	}
}

// Queue carries an explicit subscription address on the global topic exchange (VOIP-1419: every
// published event type implements the interface; no JSON fallback exists). The assertion pins the
// POINTER type: the event data reaches notifyhandler as a POINTER and the assertion matches the
// dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Queue)(nil)

func TestQueueEventSubscriptionID(t *testing.T) {
	queueID := uuid.Must(uuid.NewV4())

	tests := []struct {
		name   string
		queue  *Queue
		expect string
	}{
		{
			"normal",
			&Queue{
				Identity: commonidentity.Identity{
					ID:         queueID,
					CustomerID: uuid.Must(uuid.NewV4()),
				},
				RoutingMethod: RoutingMethodRandom,
			},
			queueID.String(),
		},
		{
			"empty id",
			&Queue{},
			uuid.Nil.String(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.queue.EventSubscriptionID()
			if res != tt.expect {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expect, res)
			}
		})
	}
}

// TestQueueEventSubscriptionIDIsOwnIDNotCustomerID pins the address choice mutation-sensitively:
// the queue's OWN id is the subscription address, never its customer id (or any other id field).
// A consumer following one queue binds `queue-manager.queue.<queue-id>.#`.
func TestQueueEventSubscriptionIDIsOwnIDNotCustomerID(t *testing.T) {
	queueID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	waitFlowID := uuid.Must(uuid.NewV4())

	data := &Queue{
		Identity: commonidentity.Identity{
			ID:         queueID,
			CustomerID: customerID,
		},
		WaitFlowID: waitFlowID,
	}

	res := data.EventSubscriptionID()
	if res != queueID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", queueID.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Queue must not be addressed by its customer id. got: %s", res)
	}
	if res == waitFlowID.String() {
		t.Errorf("Queue must not be addressed by its wait flow id. got: %s", res)
	}
}

func TestRoutingMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant RoutingMethod
		expected string
	}{
		{"routing_method_none", RoutingMethodNone, ""},
		{"routing_method_random", RoutingMethodRandom, "random"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestExecuteConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Execute
		expected string
	}{
		{"execute_run", ExecuteRun, "run"},
		{"execute_stop", ExecuteStop, "stop"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}
