package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_AstPlaybackStop(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		playbackID string

		expectTarget  string
		expectURI     string
		expectMethod  sock.RequestMethod
		expectRequest *sock.Request

		response *sock.Response
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"5734c890-7f6e-11ea-9520-6f774800cd74",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/playbacks/5734c890-7f6e-11ea-9520-6f774800cd74",
			sock.RequestMethodDelete,
			&sock.Request{
				URI:      "/ari/playbacks/5734c890-7f6e-11ea-9520-6f774800cd74",
				Method:   sock.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},

			&sock.Response{
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

			err := reqHandler.AstPlaybackStop(context.Background(), tt.asteriskID, tt.playbackID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
