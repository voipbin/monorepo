package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestProcessV1RecordingsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		db:                mockDB,
		reqHandler:        mockReq,
		callHandler:       mockCall,
		conferenceHandler: mockConf,
	}

	type test struct {
		name      string
		request   *rabbitmqhandler.Request
		recording *recording.Recording
		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/recordings/call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21%3A35%3A02.809Z",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&recording.Recording{
				ID:          "call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z",
				UserID:      0,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:      recording.StatusEnd,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z","user_id":0,"type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().RecordingGet(gomock.Any(), tt.recording.ID).Return(tt.recording, nil)

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
