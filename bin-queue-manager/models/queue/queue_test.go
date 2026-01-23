package queue

import (
	"testing"

	"github.com/gofrs/uuid"
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
