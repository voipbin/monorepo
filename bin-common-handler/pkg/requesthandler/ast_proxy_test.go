package requesthandler

import (
	"context"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_AstProxyRecordingFileMove(t *testing.T) {

	tests := []struct {
		name string

		asteriskID string
		filenames  []string

		expectTarget  string
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			name: "normal",

			asteriskID: "00:11:22:33:44:55",
			filenames: []string{
				"cfbb212a-087f-11f0-82ec-0304f568db95_in.wav",
				"cfbb212a-087f-11f0-82ec-0304f568db95_out.wav",
			},

			expectTarget: "asterisk.00:11:22:33:44:55.request",
			expectRequest: &sock.Request{
				URI:      "/proxy/recording_file_move",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"filenames":["cfbb212a-087f-11f0-82ec-0304f568db95_in.wav","cfbb212a-087f-11f0-82ec-0304f568db95_out.wav"]}`),
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstProxyRecordingFileMove(context.Background(), tt.asteriskID, tt.filenames, requestTimeoutDefault)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
