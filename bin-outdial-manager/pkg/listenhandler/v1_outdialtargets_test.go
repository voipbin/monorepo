package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

func Test_v1OutdialtargetsIDGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outdialtargetID       uuid.UUID
		responseOutdialtarget *outdialtarget.OutdialTarget

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdialtargets/50d5c500-c51a-11ec-9c67-eb2ec9b83a3b",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("50d5c500-c51a-11ec-9c67-eb2ec9b83a3b"),
			&outdialtarget.OutdialTarget{
				ID: uuid.FromStringOrNil("50d5c500-c51a-11ec-9c67-eb2ec9b83a3b"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"50d5c500-c51a-11ec-9c67-eb2ec9b83a3b","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdialTargetHandler := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialTargetHandler: mockOutdialTargetHandler,
			}

			mockOutdialTargetHandler.EXPECT().Get(gomock.Any(), tt.outdialtargetID).Return(tt.responseOutdialtarget, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialtargetsIDDelete(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outdialtargetID uuid.UUID

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdialtargets/0681ad52-b57a-11ec-824c-8353dafd28f1",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("0681ad52-b57a-11ec-824c-8353dafd28f1"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdialTargetHandler := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialTargetHandler: mockOutdialTargetHandler,
			}

			mockOutdialTargetHandler.EXPECT().Delete(gomock.Any(), tt.outdialtargetID).Return(&outdialtarget.OutdialTarget{}, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_v1OutdialtargetsIDProgressingPost(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outdialtargetID  uuid.UUID
		destinationIndex int

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdialtargets/58a99808-b57d-11ec-8d82-d75af383ea0d/progressing",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"destination_index": 0}`),
			},

			uuid.FromStringOrNil("58a99808-b57d-11ec-8d82-d75af383ea0d"),
			0,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdialTargetHandler := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialTargetHandler: mockOutdialTargetHandler,
			}

			mockOutdialTargetHandler.EXPECT().UpdateProgressing(gomock.Any(), tt.outdialtargetID, tt.destinationIndex).Return(&outdialtarget.OutdialTarget{}, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_v1OutdialtargetsIDStatusPut(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		outdialtargetID uuid.UUID
		status          outdialtarget.Status

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/outdialtargets/7a3b6b22-b62c-11ec-9ded-cb7b1f5f8878/status",
				Method:   sock.RequestMethodPut,
				DataType: "application/json",
				Data:     []byte(`{"status": "idle"}`),
			},

			uuid.FromStringOrNil("7a3b6b22-b62c-11ec-9ded-cb7b1f5f8878"),
			outdialtarget.StatusIdle,

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00000000-0000-0000-0000-000000000000","outdial_id":"00000000-0000-0000-0000-000000000000","name":"","detail":"","data":"","status":"","destination_0":null,"destination_1":null,"destination_2":null,"destination_3":null,"destination_4":null,"try_count_0":0,"try_count_1":0,"try_count_2":0,"try_count_3":0,"try_count_4":0,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockOutdialTargetHandler := outdialtargethandler.NewMockOutdialTargetHandler(mc)

			h := &listenHandler{
				sockHandler:          mockSock,
				outdialTargetHandler: mockOutdialTargetHandler,
			}

			mockOutdialTargetHandler.EXPECT().UpdateStatus(gomock.Any(), tt.outdialtargetID, tt.status).Return(&outdialtarget.OutdialTarget{}, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
