package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
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
				Data:       []byte(`{"id":"05a4617c-bb41-11ed-8591-d72108ff17fd","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"call_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
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

func Test_processV1GroupcallsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		responseGroupcalls []*groupcall.Groupcall

		expectCusomterID uuid.UUID
		expectPageSize   uint64
		expectPageToken  string
		expectRes        *rabbitmqhandler.Response
	}{
		{
			name: "normal",
			request: &rabbitmqhandler.Request{
				URI:    "/v1/groupcalls?page_size=10&page_token=2023-05-03%2021:35:02.809&customer_id=256d8080-bd7e-11ed-b083-93a9d3f167e7",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			responseGroupcalls: []*groupcall.Groupcall{
				{
					ID: uuid.FromStringOrNil("d8896324-bd7d-11ed-bdea-5b96b47c0bf4"),
				},
				{
					ID: uuid.FromStringOrNil("d8ae01d4-bd7d-11ed-b570-236fa9212eba"),
				},
			},

			expectCusomterID: uuid.FromStringOrNil("256d8080-bd7e-11ed-b083-93a9d3f167e7"),
			expectPageSize:   10,
			expectPageToken:  "2023-05-03 21:35:02.809",
			expectRes: &rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d8896324-bd7d-11ed-bdea-5b96b47c0bf4","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"call_count":0,"tm_create":"","tm_update":"","tm_delete":""},{"id":"d8ae01d4-bd7d-11ed-b570-236fa9212eba","customer_id":"00000000-0000-0000-0000-000000000000","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"call_count":0,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().Gets(gomock.Any(), tt.expectCusomterID, tt.expectPageSize, tt.expectPageToken).Return(tt.responseGroupcalls, nil)

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

func Test_processV1GroupcallsIDGet(t *testing.T) {

	tests := []struct {
		name              string
		request           *rabbitmqhandler.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/6b59c9a6-bd7d-11ed-98cc-536b0b571118",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("6b59c9a6-bd7d-11ed-98cc-536b0b571118"),
				CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
			},

			uuid.FromStringOrNil("6b59c9a6-bd7d-11ed-98cc-536b0b571118"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6b59c9a6-bd7d-11ed-98cc-536b0b571118","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"call_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().Get(gomock.Any(), tt.responseGroupcall.ID).Return(tt.responseGroupcall, nil)

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

func Test_processV1GroupcallsIDDelete(t *testing.T) {

	tests := []struct {
		name              string
		request           *rabbitmqhandler.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/922b2b46-bd7e-11ed-8754-3772984da05b",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("922b2b46-bd7e-11ed-8754-3772984da05b"),
				CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
			},

			uuid.FromStringOrNil("922b2b46-bd7e-11ed-8754-3772984da05b"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"922b2b46-bd7e-11ed-8754-3772984da05b","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"call_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseGroupcall, nil)

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

func Test_processV1GroupcallsIDHangupPost(t *testing.T) {

	tests := []struct {
		name              string
		request           *rabbitmqhandler.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/groupcalls/b055775c-bd7e-11ed-a2b8-1f2c8369029a/hangup",
				Method: rabbitmqhandler.RequestMethodPost,
			},
			&groupcall.Groupcall{
				ID:         uuid.FromStringOrNil("b055775c-bd7e-11ed-a2b8-1f2c8369029a"),
				CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
			},

			uuid.FromStringOrNil("b055775c-bd7e-11ed-a2b8-1f2c8369029a"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b055775c-bd7e-11ed-a2b8-1f2c8369029a","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","source":null,"destinations":null,"ring_method":"","answer_method":"","answer_call_id":"00000000-0000-0000-0000-000000000000","call_ids":null,"call_count":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().Hangingup(gomock.Any(), tt.expectID).Return(tt.responseGroupcall, nil)

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
