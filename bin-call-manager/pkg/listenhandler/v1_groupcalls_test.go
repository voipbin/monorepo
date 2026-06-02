package listenhandler

import (
	reflect "reflect"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/externalmediahandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

func Test_processV1GroupcallsPost(t *testing.T) {

	type test struct {
		name    string
		request *sock.Request

		responseGroupcall *groupcall.Groupcall

		expectID                uuid.UUID
		expectCustomerID        uuid.UUID
		expectSource            *commonaddress.Address
		expectDestinations      []commonaddress.Address
		expectFlowID            uuid.UUID
		expectMasterCallID      uuid.UUID
		expectMasterGroupcallID uuid.UUID
		expectRingMethod        groupcall.RingMethod
		expectAnswerMethod      groupcall.AnswerMethod

		expectRes *sock.Response
	}

	tests := []test{
		{
			name: "normal type connect",
			request: &sock.Request{
				URI:      "/v1/groupcalls",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"id":"114c466e-e46a-11ed-9034-5fd743781da6","customer_id":"dabd81b0-bb3f-11ed-8542-3bb36342932e","source":{"type":"tel","target":"+821100000001"},"destinations":[{"type":"tel","target":"+821100000002"},{"type":"tel","target":"+821100000003"}],"flow_id":"db049be0-bb3f-11ed-901a-eff2e3b25b21","master_call_id":"db3ccfc4-bb3f-11ed-bb95-238737bb066d","master_groupcall_id":"1184419a-e46a-11ed-971d-5b09d31146cf","ring_method":"ring_all","answer_method":"hangup_others"}`),
			},

			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("05a4617c-bb41-11ed-8591-d72108ff17fd"),
				},
			},

			expectID:         uuid.FromStringOrNil("114c466e-e46a-11ed-9034-5fd743781da6"),
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
			expectFlowID:            uuid.FromStringOrNil("db049be0-bb3f-11ed-901a-eff2e3b25b21"),
			expectMasterCallID:      uuid.FromStringOrNil("db3ccfc4-bb3f-11ed-bb95-238737bb066d"),
			expectMasterGroupcallID: uuid.FromStringOrNil("1184419a-e46a-11ed-971d-5b09d31146cf"),
			expectRingMethod:        groupcall.RingMethodRingAll,
			expectAnswerMethod:      groupcall.AnswerMethodHangupOthers,

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"05a4617c-bb41-11ed-8591-d72108ff17fd","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockExternal := externalmediahandler.NewMockExternalMediaHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				externalMediaHandler: mockExternal,
				groupcallHandler:     mockGroupcall,
			}

			mockGroupcall.EXPECT().Start(
				gomock.Any(),
				tt.expectID,
				tt.expectCustomerID,
				tt.expectFlowID,
				tt.expectSource,
				tt.expectDestinations,
				tt.expectMasterCallID,
				tt.expectMasterGroupcallID,
				tt.expectRingMethod,
				tt.expectAnswerMethod,
				"",
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
		request *sock.Request

		responseGroupcalls []*groupcall.Groupcall

		expectPageSize  uint64
		expectPageToken string
		expectRes       *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/groupcalls?page_size=10&page_token=2023-05-03T21:35:02.809Z",
				Method: sock.RequestMethodGet,
				Data:   []byte(`{"customer_id":"256d8080-bd7e-11ed-b083-93a9d3f167e7","deleted":false}`),
			},
			responseGroupcalls: []*groupcall.Groupcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d8896324-bd7d-11ed-bdea-5b96b47c0bf4"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("d8ae01d4-bd7d-11ed-b570-236fa9212eba"),
					},
				},
			},

			expectPageSize:  10,
			expectPageToken: "2023-05-03T21:35:02.809Z",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"d8896324-bd7d-11ed-bdea-5b96b47c0bf4","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"},{"id":"d8ae01d4-bd7d-11ed-b570-236fa9212eba","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().List(gomock.Any(), tt.expectPageSize, tt.expectPageToken, gomock.Any()).Return(tt.responseGroupcalls, nil)

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
		request           *sock.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:    "/v1/groupcalls/6b59c9a6-bd7d-11ed-98cc-536b0b571118",
				Method: sock.RequestMethodGet,
			},
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6b59c9a6-bd7d-11ed-98cc-536b0b571118"),
					CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
				},
			},

			expectID: uuid.FromStringOrNil("6b59c9a6-bd7d-11ed-98cc-536b0b571118"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6b59c9a6-bd7d-11ed-98cc-536b0b571118","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
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
		request           *sock.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/groupcalls/922b2b46-bd7e-11ed-8754-3772984da05b",
				Method: sock.RequestMethodDelete,
			},
			responseGroupcall: &groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("922b2b46-bd7e-11ed-8754-3772984da05b"),
					CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
				},
			},

			expectID: uuid.FromStringOrNil("922b2b46-bd7e-11ed-8754-3772984da05b"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"922b2b46-bd7e-11ed-8754-3772984da05b","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
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
		request           *sock.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/groupcalls/b055775c-bd7e-11ed-a2b8-1f2c8369029a/hangup",
				Method: sock.RequestMethodPost,
			},
			&groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b055775c-bd7e-11ed-a2b8-1f2c8369029a"),
					CustomerID: uuid.FromStringOrNil("ab0fb69e-7f50-11ec-b0d3-2b4311e649e0"),
				},
			},

			uuid.FromStringOrNil("b055775c-bd7e-11ed-a2b8-1f2c8369029a"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b055775c-bd7e-11ed-a2b8-1f2c8369029a","customer_id":"ab0fb69e-7f50-11ec-b0d3-2b4311e649e0","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
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

func Test_processV1GroupcallsIDAnswerGroupcallIDPost(t *testing.T) {

	tests := []struct {
		name              string
		request           *sock.Request
		responseGroupcall *groupcall.Groupcall

		expectID               uuid.UUID
		expectAnwerGroupcallID uuid.UUID
		expectRes              *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/groupcalls/c5292994-e443-11ed-9d25-f79431094c08/answer_groupcall_id",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"answer_groupcall_id":"c5a12f84-e443-11ed-82ef-47e49bddaa68"}`),
			},
			&groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c5292994-e443-11ed-9d25-f79431094c08"),
				},
			},

			uuid.FromStringOrNil("c5292994-e443-11ed-9d25-f79431094c08"),
			uuid.FromStringOrNil("c5a12f84-e443-11ed-82ef-47e49bddaa68"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c5292994-e443-11ed-9d25-f79431094c08","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().AnswerGroupcall(gomock.Any(), tt.expectID, tt.expectAnwerGroupcallID).Return(tt.responseGroupcall, nil)

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

func Test_processV1GroupcallsIDHangupGroupcallPost(t *testing.T) {

	tests := []struct {
		name              string
		request           *sock.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/groupcalls/dd850fba-e445-11ed-a841-9bf7ed18abe2/hangup_groupcall",
				Method: sock.RequestMethodPost,
			},
			&groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("dd850fba-e445-11ed-a841-9bf7ed18abe2"),
				},
			},

			uuid.FromStringOrNil("dd850fba-e445-11ed-a841-9bf7ed18abe2"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"dd850fba-e445-11ed-a841-9bf7ed18abe2","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().HangupGroupcall(gomock.Any(), tt.expectID).Return(tt.responseGroupcall, nil)

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

func Test_processV1GroupcallsIDHangupCallPost(t *testing.T) {

	tests := []struct {
		name              string
		request           *sock.Request
		responseGroupcall *groupcall.Groupcall

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/groupcalls/0b4d5a38-e446-11ed-9b91-bb8a66b3fb46/hangup_call",
				Method: sock.RequestMethodPost,
			},
			&groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0b4d5a38-e446-11ed-9b91-bb8a66b3fb46"),
				},
			},

			uuid.FromStringOrNil("0b4d5a38-e446-11ed-9b91-bb8a66b3fb46"),
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"0b4d5a38-e446-11ed-9b91-bb8a66b3fb46","customer_id":"00000000-0000-0000-0000-000000000000","owner_type":"","owner_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","master_call_id":"00000000-0000-0000-0000-000000000000","master_groupcall_id":"00000000-0000-0000-0000-000000000000","answer_call_id":"00000000-0000-0000-0000-000000000000","answer_groupcall_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().HangupCall(gomock.Any(), tt.expectID).Return(tt.responseGroupcall, nil)

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

// Test_processV1GroupcallsID_notFoundTyped verifies that the typed
// *cerrors.VoipbinError (NotFound) returned by the real groupcallHandler.Get
// is correctly translated to HTTP 404 via the errors.As → cerrors.ToResponse branch
// in errorResponse.
func Test_processV1GroupcallsID_notFoundTyped(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
	}{
		{
			name: "GET non-existent groupcall returns 404 via typed cerrors.NotFound",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)
			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}

			mockGroupcall.EXPECT().Get(gomock.Any(), tt.id).Return(nil, cerrors.NotFound(
				commonoutline.ServiceNameCallManager,
				"GROUPCALL_NOT_FOUND",
				"The group call was not found.",
			))

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
			if res.DataType != cerrors.DataTypeVoipbinError {
				t.Errorf("DataType mismatch. expected: %s, got: %s", cerrors.DataTypeVoipbinError, res.DataType)
			}
		})
	}
}

func Test_processV1GroupcallsID_notFound(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request
		id      uuid.UUID
		setup   func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID)
	}{
		{
			name: "GET non-existent groupcall returns 404",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodGet,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID) {
				m.EXPECT().Get(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "DELETE non-existent groupcall returns 404",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099",
				Method: sock.RequestMethodDelete,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID) {
				m.EXPECT().Delete(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
		{
			name: "POST hangup non-existent groupcall returns 404",
			request: &sock.Request{
				URI:    "/v1/groupcalls/00000000-0000-0000-0000-000000000099/hangup",
				Method: sock.RequestMethodPost,
			},
			id: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000099"),
			setup: func(m *groupcallhandler.MockGroupcallHandler, id uuid.UUID) {
				m.EXPECT().Hangingup(gomock.Any(), id).Return(nil, dbhandler.ErrNotFound)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)
			h := &listenHandler{
				sockHandler:      mockSock,
				callHandler:      mockCall,
				groupcallHandler: mockGroupcall,
			}
			tt.setup(mockGroupcall, tt.id)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode mismatch. expected: 404, got: %d", res.StatusCode)
			}
		})
	}
}
