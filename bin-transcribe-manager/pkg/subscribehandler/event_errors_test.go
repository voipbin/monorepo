package subscribehandler

import (
	"fmt"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-transcribe-manager/pkg/transcribehandler"

	gomock "go.uber.org/mock/gomock"
)

// Test_processEvent_malformedData checks a malformed payload is reported to the caller and the
// business handler is never reached (no expectation is set on the mock, so a call fails).
func Test_processEvent_malformedData(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
	}{
		{
			"call hangup",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCallManager), Type: cmcall.EventTypeCallHangup, DataType: "application/json", Data: []byte(`{"id":`)},
		},
		{
			"confbridge terminated",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCallManager), Type: cmconfbridge.EventTypeConfbridgeTerminated, DataType: "application/json", Data: []byte(`not json`)},
		},
		{
			"customer deleted",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: cucustomer.EventTypeCustomerDeleted, DataType: "application/json", Data: []byte(`[`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := subscribeHandler{
				sockHandler:       sockhandler.NewMockSockHandler(mc),
				transcribeHandler: transcribehandler.NewMockTranscribeHandler(mc),
			}

			if err := h.processEvent(tt.event); err == nil {
				t.Errorf("processEvent() expected an error for malformed data")
			}
		})
	}
}

// Test_processEvent_businessHandlerError checks an error from the business handler is logged and
// not returned, so the event is acknowledged instead of redelivered forever.
func Test_processEvent_businessHandlerError(t *testing.T) {
	tests := []struct {
		name  string
		event *sock.Event
		set   func(m *transcribehandler.MockTranscribeHandler)
	}{
		{
			"call hangup",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCallManager), Type: cmcall.EventTypeCallHangup, DataType: "application/json", Data: []byte(`{"id":"961e297a-f2f1-11ee-a261-57833f5b870a"}`)},
			func(m *transcribehandler.MockTranscribeHandler) {
				m.EXPECT().EventCMCallHangup(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed"))
			},
		},
		{
			"confbridge terminated",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCallManager), Type: cmconfbridge.EventTypeConfbridgeTerminated, DataType: "application/json", Data: []byte(`{"id":"d0f35cb4-f2f1-11ee-891e-376de80e03da"}`)},
			func(m *transcribehandler.MockTranscribeHandler) {
				m.EXPECT().EventCMConfbridgeTerminated(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed"))
			},
		},
		{
			"customer deleted",
			&sock.Event{Publisher: string(commonoutline.ServiceNameCustomerManager), Type: cucustomer.EventTypeCustomerDeleted, DataType: "application/json", Data: []byte(`{"id":"b2f0a2c6-0000-11ee-9c11-000000000001"}`)},
			func(m *transcribehandler.MockTranscribeHandler) {
				m.EXPECT().EventCUCustomerDeleted(gomock.Any(), gomock.Any()).Return(fmt.Errorf("failed"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
			tt.set(mockTranscribe)

			h := subscribeHandler{
				sockHandler:       sockhandler.NewMockSockHandler(mc),
				transcribeHandler: mockTranscribe,
			}

			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("processEvent() expected the business error to be swallowed. got: %v", err)
			}
		})
	}
}
