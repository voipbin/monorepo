package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"

	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_CallV1GroupcallCreate(t *testing.T) {

	tests := []struct {
		name string

		ctx          context.Context
		customerID   uuid.UUID
		source       commonaddress.Address
		destinations []commonaddress.Address
		flowID       uuid.UUID
		masterCallID uuid.UUID
		ringMethod   cmgroupcall.RingMethod
		answerMethod cmgroupcall.AnswerMethod
		connect      bool

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectRes     *cmgroupcall.Groupcall
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("2ac49ec8-bbae-11ed-b9cd-8f47fd0602b9"),
			source: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			destinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			flowID:       uuid.FromStringOrNil("2b1f1682-bbae-11ed-b06b-3be413b33b07"),
			masterCallID: uuid.FromStringOrNil("2b4f5b44-bbae-11ed-9629-dfffd3ac6a43"),
			ringMethod:   cmgroupcall.RingMethodRingAll,
			answerMethod: cmgroupcall.AnswerMethodHangupOthers,
			connect:      true,

			response: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"2b7dc4ac-bbae-11ed-b868-f762b6f7fd23"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &rabbitmqhandler.Request{
				URI:      "/v1/groupcalls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"2ac49ec8-bbae-11ed-b9cd-8f47fd0602b9","source":{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000002","target_name":"","name":"","detail":""},{"type":"tel","target":"+821100000003","target_name":"","name":"","detail":""}],"flow_id":"2b1f1682-bbae-11ed-b06b-3be413b33b07","master_call_id":"2b4f5b44-bbae-11ed-9629-dfffd3ac6a43","ring_method":"ring_all","answer_method":"hangup_others","connect":true}`),
			},
			expectRes: &cmgroupcall.Groupcall{
				ID: uuid.FromStringOrNil("2b7dc4ac-bbae-11ed-b868-f762b6f7fd23"),
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

			ctx := context.Background()

			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1GroupcallCreate(ctx, tt.customerID, tt.source, tt.destinations, tt.flowID, tt.masterCallID, tt.ringMethod, tt.answerMethod, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
