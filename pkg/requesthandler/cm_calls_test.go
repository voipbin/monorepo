package requesthandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestTTSSpeechesPOST(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)

	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueCall:     "bin-manager.call-manager.request",
	}

	type test struct {
		name string

		callID         uuid.UUID
		externalHost   string
		encapsulation  string
		transport      string
		connectionType string
		format         string
		direction      string

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectIP      string
		expectPort    int
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("a099a2a4-0ac7-11ec-b8ae-438c5d2fe6fb"),
			"localhost:5060",
			"rtp",
			"udp",
			"client",
			"ulaw",
			"both",

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"media_addr_ip":"127.0.0.1","media_addr_port":9999}`),
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/calls/a099a2a4-0ac7-11ec-b8ae-438c5d2fe6fb/external-media",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"external_host":"localhost:5060","encapsulation":"rtp","transport":"udp","connection_type":"client","format":"ulaw","direction":"both"}`),
			},
			"127.0.0.1",
			9999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.call-manager.request", tt.expectRequest).Return(tt.response, nil)

			resIP, resPort, err := reqHandler.CMCallExternalMedia(tt.callID, tt.externalHost, tt.encapsulation, tt.transport, tt.connectionType, tt.format, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if resIP != tt.expectIP || resPort != tt.expectPort {
				t.Errorf("Wrong match. expect: %s:%v, got: %s:%v", tt.expectIP, tt.expectPort, resIP, resPort)
			}
		})
	}
}
