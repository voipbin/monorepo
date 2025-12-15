package requesthandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AstPlaybackStop(t *testing.T) {

	tests := []struct {
		name string

		asteriskID string
		playbackID string

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
	}{
		{
			name: "normal",

			asteriskID: "00:11:22:33:44:55",
			playbackID: "5734c890-7f6e-11ea-9520-6f774800cd74",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},

			expectTarget: "asterisk.00:11:22:33:44:55.request",
			expectRequest: &sock.Request{
				URI:      "/ari/playbacks/5734c890-7f6e-11ea-9520-6f774800cd74",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
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

			err := reqHandler.AstPlaybackStop(context.Background(), tt.asteriskID, tt.playbackID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
