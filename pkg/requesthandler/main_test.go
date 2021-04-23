package requesthandler

import (
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestNewRequestHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)

	type test struct {
		name string

		sock rabbitmqhandler.Rabbit

		exchangeDelay  string
		queueCall      string
		queueFlow      string
		queueStorage   string
		queueRegistrar string
		queueNumber    string
		queueTranscode string
	}

	tests := []test{
		{
			"normal",
			mockSock,
			"bin-manager.delay",
			"bin-manager.call-manager.request",
			"bin-manager.flow-manager.request",
			"bin-manager.storage-manager.request",
			"bin-manager.registrar-manager.request",
			"bin-manager.number-manager.request",
			"bin-manager.transcode-manager.request",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			reqHandler := NewRequestHandler(tt.sock, tt.exchangeDelay, tt.queueCall, tt.queueFlow, tt.queueStorage, tt.queueRegistrar, tt.queueNumber, tt.queueTranscode)
			if reqHandler == nil {
				t.Errorf("Wrong match. expect: not nil, got: nil")
			}
		})
	}
}
