package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencecallhandler"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/pkg/conferencehandler"
)

func Test_processV1ConferencecallsIDGet(t *testing.T) {

	tests := []struct {
		name             string
		request          *rabbitmqhandler.Request
		conferencecallID uuid.UUID

		responseConferencecall *conferencecall.Conferencecall

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",

			&rabbitmqhandler.Request{
				URI:    "/v1/conferencecalls/1015da76-14cc-11ed-b156-5b7904da0071",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			uuid.FromStringOrNil("1015da76-14cc-11ed-b156-5b7904da0071"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("1015da76-14cc-11ed-b156-5b7904da0071"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"1015da76-14cc-11ed-b156-5b7904da0071","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)
			mockConferencecall := conferencecallhandler.NewMockConferencecallHandler(mc)

			h := &listenHandler{
				rabbitSock:            mockSock,
				conferenceHandler:     mockConf,
				conferencecallHandler: mockConferencecall,
			}

			mockConferencecall.EXPECT().Get(gomock.Any(), tt.conferencecallID).Return(tt.responseConferencecall, nil)
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

func Test_processV1ConferencecallsIDDelete(t *testing.T) {

	tests := []struct {
		name             string
		request          *rabbitmqhandler.Request
		conferencecallID uuid.UUID

		responseConferencecall *conferencecall.Conferencecall

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",

			&rabbitmqhandler.Request{
				URI:    "/v1/conferencecalls/8a1fd900-3bf3-11ec-bd15-eb0c54c84612",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			uuid.FromStringOrNil("8a1fd900-3bf3-11ec-bd15-eb0c54c84612"),

			&conferencecall.Conferencecall{
				ID: uuid.FromStringOrNil("8a1fd900-3bf3-11ec-bd15-eb0c54c84612"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"8a1fd900-3bf3-11ec-bd15-eb0c54c84612","customer_id":"00000000-0000-0000-0000-000000000000","conference_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","status":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockConf := conferencehandler.NewMockConferenceHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				conferenceHandler: mockConf,
			}

			mockConf.EXPECT().Leave(gomock.Any(), tt.conferencecallID).Return(tt.responseConferencecall, nil)
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
