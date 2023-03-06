package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/externalmediahandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/groupcallhandler"
)

func Test_processV1GroupcallsPost(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		responseGroupcall *groupcall.Groupcall

		expectCustomerID   uuid.UUID
		expectSource       *commonaddress.Address
		expectDestinations []commonaddress.Address
		expectFlowID       uuid.UUID
		expectMasterCallID uuid.UUID
		expectRingMethod   groupcall.RingMethod
		expectAnswerMethod groupcall.AnswerMethod

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			name: "normal type connect",
			request: &rabbitmqhandler.Request{
				URI:      "/v1/groupcalls",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"dabd81b0-bb3f-11ed-8542-3bb36342932e","source":{"type":"tel","target":"+821100000001"},"destinations":[{"type":"tel","target":"+821100000002"},{"type":"tel","target":"+821100000003"}],"flow_id":"db049be0-bb3f-11ed-901a-eff2e3b25b21","master_call_id":"db3ccfc4-bb3f-11ed-bb95-238737bb066d","ring_method":"ring_all","answer_method":"hangup_others"}`),
			},

			responseGroupcall: &groupcall.Groupcall{
				ID: uuid.FromStringOrNil("05a4617c-bb41-11ed-8591-d72108ff17fd"),
			},

			expectCustomerID: uuid.FromStringOrNil("dabd81b0-bb3f-11ed-8542-3bb36342932e"),
			expectSource: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			expectDestinations: []commonaddress.Address{
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				{
					Type:   commonaddress.TypeTel,
					Target: "+821100000003",
				},
			},
			expectFlowID:       uuid.FromStringOrNil("db049be0-bb3f-11ed-901a-eff2e3b25b21"),
			expectMasterCallID: uuid.FromStringOrNil("db3ccfc4-bb3f-11ed-bb95-238737bb066d"),
			expectRingMethod:   groupcall.RingMethodRingAll,
			expectAnswerMethod: groupcall.AnswerMethodHangupOthers,

			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"05a4617c-bb41-11ed-8591-d72108ff17fd","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				rabbitSock:           mockSock,
				externalMediaHandler: mockExternal,
				groupcallHandler:     mockGroupcall,
			}

			mockGroupcall.EXPECT().Start(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectSource,
				tt.expectDestinations,
				tt.expectFlowID,
				tt.expectMasterCallID,
				tt.expectRingMethod,
				tt.expectAnswerMethod,
			).Return(tt.responseGroupcall, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
