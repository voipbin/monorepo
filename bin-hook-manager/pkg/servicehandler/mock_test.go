package servicehandler

import (
	"context"
	"net/http"
	"strings"
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

	ctx := context.Background()

	// Test mock Conversation method
	r1, _ := http.NewRequest("POST", "http://test.uri/conversation", strings.NewReader("test data"))
	mockSvc.EXPECT().Conversation(ctx, r1).Return("", nil)
	if _, err := mockSvc.Conversation(ctx, r1); err != nil {
		t.Errorf("Mock Conversation failed: %v", err)
	}

	// Test mock Email method
	r2, _ := http.NewRequest("POST", "http://test.uri/emails", strings.NewReader("test data"))
	mockSvc.EXPECT().Email(ctx, r2).Return(nil)
	if err := mockSvc.Email(ctx, r2); err != nil {
		t.Errorf("Mock Email failed: %v", err)
	}

	// Test mock Message method
	r3, _ := http.NewRequest("POST", "http://test.uri/messages", strings.NewReader("test data"))
	mockSvc.EXPECT().Message(ctx, r3).Return(nil)
	if err := mockSvc.Message(ctx, r3); err != nil {
		t.Errorf("Mock Message failed: %v", err)
	}

	// Test mock Billing method
	r4, _ := http.NewRequest("POST", "http://test.uri/billing/paddle", strings.NewReader("test data"))
	mockSvc.EXPECT().Billing(ctx, r4).Return(nil)
	if err := mockSvc.Billing(ctx, r4); err != nil {
		t.Errorf("Mock Billing failed: %v", err)
	}
}

func TestMockServiceHandlerRecorder(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSvc := NewMockServiceHandler(mc)
	ctx := context.Background()

	// Set expectations using recorder methods
	recorder := mockSvc.EXPECT()

	r1, _ := http.NewRequest("POST", "http://test.uri/conversation", strings.NewReader("data"))
	r2, _ := http.NewRequest("POST", "http://test.uri/emails", strings.NewReader("data"))
	r3, _ := http.NewRequest("POST", "http://test.uri/messages", strings.NewReader("data"))
	r4, _ := http.NewRequest("POST", "http://test.uri/billing/paddle", strings.NewReader("data"))

	recorder.Conversation(ctx, r1).Return("", nil)
	recorder.Email(ctx, r2).Return(nil)
	recorder.Message(ctx, r3).Return(nil)
	recorder.Billing(ctx, r4).Return(nil)

	// Execute the mocked calls
	_, _ = mockSvc.Conversation(ctx, r1)
	_ = mockSvc.Email(ctx, r2)
	_ = mockSvc.Message(ctx, r3)
	_ = mockSvc.Billing(ctx, r4)
}
