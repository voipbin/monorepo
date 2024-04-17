package subscribehandler

import (
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
	"gitlab.com/voipbin/bin-manager/transfer-manager.git/pkg/transferhandler"
)

func Test_processEventCMGroupcallProgressing(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseTransfer *transfer.Transfer

		expectGroupcall *cmgroupcall.Groupcall
	}{
		{
			name: "type conference",
			event: &rabbitmqhandler.Event{
				Type:      cmgroupcall.EventTypeGroupcallProgressing,
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      []byte(`{"id":"0ea68fee-da26-11ed-ada5-2febe5011cb8","answer_call_id":"0ecb37e0-da26-11ed-a869-db2b71931ccd"}`),
			},

			responseTransfer: &transfer.Transfer{
				ID: uuid.FromStringOrNil("0d891352-da26-11ed-bc4a-0b4f86826133"),
			},

			expectGroupcall: &cmgroupcall.Groupcall{
				ID:           uuid.FromStringOrNil("0ea68fee-da26-11ed-ada5-2febe5011cb8"),
				AnswerCallID: uuid.FromStringOrNil("0ecb37e0-da26-11ed-a869-db2b71931ccd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTransfer := transferhandler.NewMockTransferHandler(mc)

			h := &subscribeHandler{
				rabbitSock: mockSock,

				transferHandler: mockTransfer,
			}

			mockTransfer.EXPECT().GetByGroupcallID(gomock.Any(), tt.expectGroupcall.ID).Return(tt.responseTransfer, nil)
			mockTransfer.EXPECT().TransfereeAnswer(gomock.Any(), tt.responseTransfer, tt.expectGroupcall).Return(nil)
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventCMGroupcallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseTransfer *transfer.Transfer
		expectGroupcall  *cmgroupcall.Groupcall
	}{
		{
			name: "normal",
			event: &rabbitmqhandler.Event{
				Type:      cmgroupcall.EventTypeGroupcallHangup,
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      []byte(`{"id":"e29736e6-da26-11ed-a609-b755ca01899f","answer_call_id":"e2c65a66-da26-11ed-9479-c7409ab92ef8"}`),
			},
			responseTransfer: &transfer.Transfer{
				ID: uuid.FromStringOrNil("e2f76cc8-da26-11ed-9c2f-7bd1ea71acf7"),
			},
			expectGroupcall: &cmgroupcall.Groupcall{
				ID:           uuid.FromStringOrNil("e29736e6-da26-11ed-a609-b755ca01899f"),
				AnswerCallID: uuid.FromStringOrNil("e2c65a66-da26-11ed-9479-c7409ab92ef8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTransfer := transferhandler.NewMockTransferHandler(mc)

			h := &subscribeHandler{
				rabbitSock: mockSock,

				transferHandler: mockTransfer,
			}

			mockTransfer.EXPECT().GetByGroupcallID(gomock.Any(), tt.expectGroupcall.ID).Return(tt.responseTransfer, nil)
			mockTransfer.EXPECT().TransfereeHangup(gomock.Any(), tt.responseTransfer, tt.expectGroupcall).Return(nil)
			h.processEvent(tt.event)
		})
	}
}

func Test_processEventCMCallHangup(t *testing.T) {

	tests := []struct {
		name  string
		event *rabbitmqhandler.Event

		responseTransfer *transfer.Transfer
		expectCall       *cmcall.Call
	}{
		{
			name: "normal",
			event: &rabbitmqhandler.Event{
				Type:      cmcall.EventTypeCallHangup,
				Publisher: "call-manager",
				DataType:  "application/json",
				Data:      []byte(`{"id":"a07bf864-dd19-11ed-a362-a792c5b0fd6d"}`),
			},
			responseTransfer: &transfer.Transfer{
				ID: uuid.FromStringOrNil("a0ae40ee-dd19-11ed-bc9e-63ab3d06e4c3"),
			},
			expectCall: &cmcall.Call{
				ID: uuid.FromStringOrNil("a07bf864-dd19-11ed-a362-a792c5b0fd6d"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockTransfer := transferhandler.NewMockTransferHandler(mc)

			h := &subscribeHandler{
				rabbitSock: mockSock,

				transferHandler: mockTransfer,
			}

			mockTransfer.EXPECT().GetByTransfererCallID(gomock.Any(), tt.expectCall.ID).Return(tt.responseTransfer, nil)
			mockTransfer.EXPECT().TransfererHangup(gomock.Any(), tt.responseTransfer, tt.expectCall).Return(nil)
			h.processEvent(tt.event)
		})
	}
}
