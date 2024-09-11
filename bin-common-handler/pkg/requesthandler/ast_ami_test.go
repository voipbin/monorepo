package requesthandler

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_AstAMIRedirect(t *testing.T) {

	tests := []struct {
		name       string
		asteriskID string
		channelID  string

		context  string
		exten    string
		priority string

		expectQueue  string
		expectURI    string
		expectMethod sock.RequestMethod
		expectData   []byte
	}{
		{
			"normal",
			"00:11:22:33:44:55",
			"6b79ae28-e3f1-11ea-bf62-7f539c6300fc",

			"svc-echo",
			"s",
			"1",

			"asterisk.00:11:22:33:44:55.request",
			"/ami",
			sock.RequestMethodPost,
			[]byte(`{"Action":"Redirect","Channel":"6b79ae28-e3f1-11ea-bf62-7f539c6300fc","Context":"svc-echo","Exten":"s","Priority":"1"}`),
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

			mockSock.EXPECT().RequestPublish(
				gomock.Any(),
				tt.expectQueue,
				&sock.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "application/json",
					Data:     tt.expectData,
				},
			).Return(&sock.Response{StatusCode: 200, Data: []byte(`{"Response":"Success","Message":"Redirect successful"}`)}, nil)

			err := reqHandler.AstAMIRedirect(context.Background(), tt.asteriskID, tt.channelID, tt.context, tt.exten, tt.priority)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
