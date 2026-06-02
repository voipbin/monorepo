package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/confbridgehandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/outboundconfighandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
)

// Test_processV1CallsIDDelete_notFound verifies that a handler returning
// dbhandler.ErrNotFound is translated to HTTP 404 (not 500).
func Test_processV1CallsIDDelete_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "DELETE non-existent call returns 404",
			request: &sock.Request{
				URI:    "/v1/calls/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodDelete,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().Delete(gomock.Any(), tt.id).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1ConfbridgesIDGet_notFound verifies that confbridgeHandler.Get
// returning dbhandler.ErrNotFound produces HTTP 404.
func Test_processV1ConfbridgesIDGet_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "GET non-existent confbridge returns 404",
			request: &sock.Request{
				URI:    "/v1/confbridges/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().Get(gomock.Any(), tt.id).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1ConfbridgesIDDelete_notFound verifies that confbridgeHandler.Delete
// returning dbhandler.ErrNotFound produces HTTP 404.
func Test_processV1ConfbridgesIDDelete_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "DELETE non-existent confbridge returns 404",
			request: &sock.Request{
				URI:    "/v1/confbridges/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodDelete,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockConfbridge := confbridgehandler.NewMockConfbridgeHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				callHandler:       mockCall,
				confbridgeHandler: mockConfbridge,
			}

			mockConfbridge.EXPECT().Delete(gomock.Any(), tt.id).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1RecordingsGet_notFound verifies that recordingHandler.List
// returning dbhandler.ErrNotFound produces HTTP 404 (not 500).
func Test_processV1RecordingsGet_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "GET recordings list with ErrNotFound returns 404",
			request: &sock.Request{
				URI:    "/v1/recordings?page_size=10",
				Method: sock.RequestMethodGet,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockRecording.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1ExternalMediasGet_notFound verifies that externalMediaHandler.List
// returning dbhandler.ErrNotFound produces HTTP 404 (not 500).
func Test_processV1ExternalMediasGet_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "GET external-medias list with ErrNotFound returns 404",
			request: &sock.Request{
				URI:    "/v1/external-medias?page_size=10",
				Method: sock.RequestMethodGet,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockExternalMedia := externalmediahandler.NewMockExternalMediaHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				callHandler:          mockCall,
				externalMediaHandler: mockExternalMedia,
			}

			mockExternalMedia.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}

// Test_processV1OutboundConfigsGet_notFound verifies that outboundConfigHandler.List
// returning dbhandler.ErrNotFound produces HTTP 404 (not 500).
func Test_processV1OutboundConfigsGet_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
	}{
		{
			name: "GET outbound configs list with ErrNotFound returns 404",
			request: &sock.Request{
				URI:    "/v1/outbound_configs?customer_id=00000000-0000-0000-0000-000000000001",
				Method: sock.RequestMethodGet,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockOutboundConfig := outboundconfighandler.NewMockOutboundConfigHandler(mc)

			h := &listenHandler{
				sockHandler:           mockSock,
				callHandler:           mockCall,
				outboundConfigHandler: mockOutboundConfig,
			}

			mockOutboundConfig.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("expected nil error, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}
