package requesthandler

import (
	"context"
	cmrecording "monorepo/bin-call-manager/models/recording"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_CallV1RecoveryStart(t *testing.T) {

	tests := []struct {
		name string

		asteriskID string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRe      *cmrecording.Recording
	}{
		{
			name: "normal",

			asteriskID: "00:11:22:33:44:55",

			response: &sock.Response{
				StatusCode: 200,
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/recovery",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"asterisk_id":"00:11:22:33:44:55"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			if errRecovery := reqHandler.CallV1RecoveryStart(ctx, tt.asteriskID); errRecovery != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRecovery)
			}

		})
	}
}
