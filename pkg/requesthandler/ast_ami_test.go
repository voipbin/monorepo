package requesthandler

import (
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestAstAMIRedirect(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := NewRequestHandler(mockSock, "bin-manager.delay", "bin-manager.call-manager.request", "bin-manager.flow-manager.request", "bin-manager.tts-manager.request")

	type test struct {
		name       string
		asteriskID string
		channelID  string

		context  string
		exten    string
		priority string

		expectQueue  string
		expectURI    string
		expectMethod rabbitmqhandler.RequestMethod
		expectData   []byte
	}

	tests := []test{
		{
			"normal",
			"00:11:22:33:44:55",
			"6b79ae28-e3f1-11ea-bf62-7f539c6300fc",

			"svc-echo",
			"s",
			"1",

			"asterisk.00:11:22:33:44:55.request",
			"/ami",
			rabbitmqhandler.RequestMethodPost,
			[]byte(`{"Action":"Redirect","Channel":"6b79ae28-e3f1-11ea-bf62-7f539c6300fc","Context":"svc-echo","Exten":"s","Priority":"1"}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(
				gomock.Any(),
				tt.expectQueue,
				&rabbitmqhandler.Request{
					URI:      tt.expectURI,
					Method:   tt.expectMethod,
					DataType: "application/json",
					Data:     tt.expectData,
				},
			).Return(&rabbitmqhandler.Response{StatusCode: 200, Data: []byte(`{"Response":"Success","Message":"Redirect successful"}`)}, nil)

			err := reqHandler.AstAMIRedirect(tt.asteriskID, tt.channelID, tt.context, tt.exten, tt.priority)
			if err != nil {
				t.Errorf("Wrong match. expact: ok, got: %v", err)
			}
		})
	}
}
