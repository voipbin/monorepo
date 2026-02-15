package servicehandler

import (
	"context"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func TestMockServiceHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	// Test NewMockServiceHandler and EXPECT
	mockSvc := NewMockServiceHandler(mc)
	if mockSvc == nil {
		t.Error("NewMockServiceHandler returned nil")
	}

	// Verify EXPECT returns MockServiceHandlerMockRecorder
	recorder := mockSvc.EXPECT()
	if recorder == nil {
		t.Error("EXPECT() returned nil")
	}

	// Test mock Conversation method
	ctx := context.Background()
	uri := "test.uri"
	data := []byte("test data")

	mockSvc.EXPECT().Conversation(ctx, uri, data).Return(nil)
	if err := mockSvc.Conversation(ctx, uri, data); err != nil {
		t.Errorf("Mock Conversation failed: %v", err)
	}

	// Test mock Email method
	mockSvc.EXPECT().Email(ctx, uri, data).Return(nil)
	if err := mockSvc.Email(ctx, uri, data); err != nil {
		t.Errorf("Mock Email failed: %v", err)
	}

	// Test mock Message method
	mockSvc.EXPECT().Message(ctx, uri, data).Return(nil)
	if err := mockSvc.Message(ctx, uri, data); err != nil {
		t.Errorf("Mock Message failed: %v", err)
	}
}

func TestMockServiceHandlerRecorder(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := NewMockServiceHandler(mc)
	ctx := context.Background()
	uri := "test.uri"
	data := []byte("test data")

	// Test recorder methods by chaining
	recorder := mockSvc.EXPECT()

	// Set expectations using recorder methods
	recorder.Conversation(ctx, uri, data).Return(nil)
	recorder.Email(ctx, uri, data).Return(nil)
	recorder.Message(ctx, uri, data).Return(nil)

	// Execute the mocked calls
	_ = mockSvc.Conversation(ctx, uri, data)
	_ = mockSvc.Email(ctx, uri, data)
	_ = mockSvc.Message(ctx, uri, data)
}
