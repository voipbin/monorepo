package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_AstPlaybackStop(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		playbackID string

		expectTarget  string
		expectURI     string
		expectMethod  rabbitmqhandler.RequestMethod
		expectRequest *rabbitmqhandler.Request

		response *rabbitmqhandler.Response
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"5734c890-7f6e-11ea-9520-6f774800cd74",

			"asterisk.00:11:22:33:44:55.request",
			"/ari/playbacks/5734c890-7f6e-11ea-9520-6f774800cd74",
			rabbitmqhandler.RequestMethodDelete,
			&rabbitmqhandler.Request{
				URI:      "/ari/playbacks/5734c890-7f6e-11ea-9520-6f774800cd74",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: ContentTypeJSON,
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			err := reqHandler.AstPlaybackStop(context.Background(), tt.asteriskID, tt.playbackID)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
