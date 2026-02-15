package listenhandler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/pkg/extensionhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestProcessV1ExtensionDirectsGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExtHandler := extensionhandler.NewMockExtensionHandler(ctrl)

	handler := &listenHandler{
		extensionHandler: mockExtHandler,
	}

	ctx := context.Background()
	extensionID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	hash := "abc123def456"

	tests := []struct {
		name           string
		request        *sock.Request
		setup          func()
		expectedStatus int
		wantErr        bool
	}{
		{
			name: "get_extension_direct_success",
			request: &sock.Request{
				URI: "/v1/extension-directs?hash=" + hash,
			},
			setup: func() {
				ed := &extensiondirect.ExtensionDirect{
					Identity: identity.Identity{
						ID:         uuid.Must(uuid.NewV4()),
						CustomerID: customerID,
					},
					ExtensionID: extensionID,
					Hash:        hash,
				}
				mockExtHandler.EXPECT().
					GetDirectByHash(ctx, hash).
					Return(ed, nil)
			},
			expectedStatus: 200,
			wantErr:        false,
		},
		{
			name: "missing_hash_parameter",
			request: &sock.Request{
				URI: "/v1/extension-directs",
			},
			setup:          func() {},
			expectedStatus: 400,
			wantErr:        false,
		},
		{
			name: "extension_direct_not_found",
			request: &sock.Request{
				URI: "/v1/extension-directs?hash=" + hash,
			},
			setup: func() {
				mockExtHandler.EXPECT().
					GetDirectByHash(ctx, hash).
					Return(nil, errors.New("not found"))
			},
			expectedStatus: 0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			res, err := handler.processV1ExtensionDirectsGet(ctx, tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("processV1ExtensionDirectsGet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if res == nil {
					t.Error("processV1ExtensionDirectsGet() returned nil response")
					return
				}
				if res.StatusCode != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, res.StatusCode)
				}
				if tt.expectedStatus == 200 {
					var result extensiondirect.ExtensionDirect
					if err := json.Unmarshal(res.Data, &result); err != nil {
						t.Errorf("Failed to unmarshal response: %v", err)
					}
				}
			}
		})
	}
}
