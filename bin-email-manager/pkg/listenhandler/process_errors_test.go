package listenhandler

import (
	"fmt"
	"net/http"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-email-manager/pkg/dbhandler"
	"monorepo/bin-email-manager/pkg/emailhandler"

	"go.uber.org/mock/gomock"
)

func Test_processRequest_handlerErrors(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request
		// setup registers the expected email handler call, if the request reaches it
		setup func(m *emailhandler.MockEmailHandler)

		expectStatus   int
		expectDataType string
	}{
		{
			name:    "list_handler_error_maps_to_400",
			request: &sock.Request{URI: "/v1/emails?page_size=10", Method: sock.RequestMethodGet, Data: []byte(`{"deleted":false}`)},
			setup: func(m *emailhandler.MockEmailHandler) {
				m.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return(nil, fmt.Errorf("list failed"))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "create_malformed_body_maps_to_400",
			request:      &sock.Request{URI: "/v1/emails", Method: sock.RequestMethodPost, Data: []byte(`{not json`)},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "create_handler_error_maps_to_400",
			request: &sock.Request{URI: "/v1/emails", Method: sock.RequestMethodPost, Data: []byte(`{"subject":"s"}`)},
			setup: func(m *emailhandler.MockEmailHandler) {
				m.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), "s", "", gomock.Any()).Return(nil, fmt.Errorf("create failed"))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "get_not_found_maps_to_404",
			request: &sock.Request{URI: "/v1/emails/5e8f0f0a-5555-4555-8555-000000000001", Method: sock.RequestMethodGet},
			setup: func(m *emailhandler.MockEmailHandler) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, dbhandler.ErrNotFound)
			},
			expectStatus: http.StatusNotFound,
		},
		{
			name:    "get_typed_error_keeps_typed_response",
			request: &sock.Request{URI: "/v1/emails/5e8f0f0a-5555-4555-8555-000000000001", Method: sock.RequestMethodGet},
			setup: func(m *emailhandler.MockEmailHandler) {
				m.EXPECT().Get(gomock.Any(), gomock.Any()).Return(nil, cerrors.PermissionDenied(commonoutline.ServiceNameEmailManager, "EMAIL_DENIED", "denied"))
			},
			expectStatus:   http.StatusForbidden,
			expectDataType: cerrors.DataTypeVoipbinError,
		},
		{
			name:    "delete_handler_error_maps_to_400",
			request: &sock.Request{URI: "/v1/emails/5e8f0f0a-5555-4555-8555-000000000001", Method: sock.RequestMethodDelete},
			setup: func(m *emailhandler.MockEmailHandler) {
				m.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("delete failed"))
			},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:         "hook_malformed_body_maps_to_400",
			request:      &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: []byte(`{not json`)},
			expectStatus: http.StatusBadRequest,
		},
		{
			name:    "hook_handler_error_is_swallowed_with_200",
			request: &sock.Request{URI: "/v1/hooks", Method: sock.RequestMethodPost, Data: []byte(`{"received_uri":"https://example.com/v1.0/emails/sendgrid","received_data":"e30="}`)},
			setup: func(m *emailhandler.MockEmailHandler) {
				m.EXPECT().Hook(gomock.Any(), "https://example.com/v1.0/emails/sendgrid", []byte(`{}`)).Return(fmt.Errorf("hook failed"))
			},
			expectStatus:   http.StatusOK,
			expectDataType: "application/json",
		},
		{
			name:         "unknown_uri_maps_to_404",
			request:      &sock.Request{URI: "/v1/unknown", Method: sock.RequestMethodGet},
			expectStatus: http.StatusNotFound,
		},
		{
			name:         "known_uri_with_unsupported_method_maps_to_404",
			request:      &sock.Request{URI: "/v1/emails", Method: sock.RequestMethodDelete},
			expectStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockEmail := emailhandler.NewMockEmailHandler(mc)
			h := &listenHandler{
				sockHandler:  mockSock,
				emailHandler: mockEmail,
			}

			if tt.setup != nil {
				tt.setup(mockEmail)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong status. expect: %d, got: %d", tt.expectStatus, res.StatusCode)
			}
			if res.DataType != tt.expectDataType {
				t.Errorf("Wrong data type. expect: %q, got: %q", tt.expectDataType, res.DataType)
			}
		})
	}
}
