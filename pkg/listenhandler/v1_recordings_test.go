package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
)

func TestProcessV1RecordingsGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		db:          mockDB,
		reqHandler:  mockReq,
		callHandler: mockCall,
	}

	type test struct {
		name       string
		request    *rabbitmqhandler.Request
		userID     uint64
		pageSize   uint64
		pageToken  string
		recordings []*recording.Recording
		expectRes  *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/recordings?page_size=10&page_token=2020-05-03%2021:35:02.809&user_id=0",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			0,
			10,
			"2020-05-03 21:35:02.809",
			[]*recording.Recording{
				{
					ID:          uuid.FromStringOrNil("cfa4d576-6128-11eb-b69b-9f7a738a1ad7"),
					UserID:      0,
					Type:        recording.TypeCall,
					ReferenceID: uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
					Status:      recording.StatusEnd,
					Filename:    "call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"cfa4d576-6128-11eb-b69b-9f7a738a1ad7","user_id":0,"type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","filename":"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockDB.EXPECT().RecordingGets(gomock.Any(), tt.userID, tt.pageSize, tt.pageToken).Return(tt.recordings, nil)

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

func TestProcessV1RecordingsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:  mockSock,
		db:          mockDB,
		reqHandler:  mockReq,
		callHandler: mockCall,
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
				URI:    "/v1/recordings/00c711be-6129-11eb-9404-b73dcf512957",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&recording.Recording{
				ID:          uuid.FromStringOrNil("00c711be-6129-11eb-9404-b73dcf512957"),
				UserID:      0,
				Type:        recording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:      recording.StatusEnd,
				Filename:    "call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00c711be-6129-11eb-9404-b73dcf512957","user_id":0,"type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","filename":"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
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
