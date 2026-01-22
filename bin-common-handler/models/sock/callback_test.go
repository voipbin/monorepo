package sock

import (
	"errors"
	"testing"
)

func TestCbMsgConsumeType(t *testing.T) {
	// Test that we can create and call a CbMsgConsume function
	called := false
	var receivedEvent *Event

	callback := CbMsgConsume(func(e *Event) error {
		called = true
		receivedEvent = e
		return nil
	})

	testEvent := &Event{
		Type:      "test_event",
		Publisher: "test_publisher",
	}

	err := callback(testEvent)

	if err != nil {
		t.Errorf("Callback returned error: %v", err)
	}
	if !called {
		t.Error("Callback was not called")
	}
	if receivedEvent != testEvent {
		t.Errorf("Callback received wrong event: %v, expected %v", receivedEvent, testEvent)
	}
}

func TestCbMsgConsumeError(t *testing.T) {
	expectedErr := errors.New("consume error")

	callback := CbMsgConsume(func(e *Event) error {
		return expectedErr
	})

	err := callback(&Event{})

	if err != expectedErr {
		t.Errorf("Callback returned wrong error: %v, expected %v", err, expectedErr)
	}
}

func TestCbMsgRPCType(t *testing.T) {
	// Test that we can create and call a CbMsgRPC function
	called := false
	var receivedRequest *Request

	expectedResponse := &Response{
		StatusCode: 200,
		DataType:   "application/json",
	}

	callback := CbMsgRPC(func(r *Request) (*Response, error) {
		called = true
		receivedRequest = r
		return expectedResponse, nil
	})

	testRequest := &Request{
		URI:       "/v1/test",
		Method:    RequestMethodGet,
		Publisher: "test_publisher",
	}

	response, err := callback(testRequest)

	if err != nil {
		t.Errorf("Callback returned error: %v", err)
	}
	if !called {
		t.Error("Callback was not called")
	}
	if receivedRequest != testRequest {
		t.Errorf("Callback received wrong request: %v, expected %v", receivedRequest, testRequest)
	}
	if response != expectedResponse {
		t.Errorf("Callback returned wrong response: %v, expected %v", response, expectedResponse)
	}
}

func TestCbMsgRPCError(t *testing.T) {
	expectedErr := errors.New("rpc error")

	callback := CbMsgRPC(func(r *Request) (*Response, error) {
		return nil, expectedErr
	})

	response, err := callback(&Request{})

	if err != expectedErr {
		t.Errorf("Callback returned wrong error: %v, expected %v", err, expectedErr)
	}
	if response != nil {
		t.Errorf("Callback returned response when it should be nil: %v", response)
	}
}
