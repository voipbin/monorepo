package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
)

func Test_processV1RecordingsGet(t *testing.T) {
	type test struct {
		name       string
		request    *rabbitmqhandler.Request
		customerID uuid.UUID
		pageSize   uint64
		pageToken  string
		recordings []*recording.Recording
		expectRes  *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"basic",
			&rabbitmqhandler.Request{
				URI:    "/v1/recordings?page_size=10&page_token=2020-05-03%2021:35:02.809&customer_id=c15af818-7f51-11ec-8eeb-f733ba8df393",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			uuid.FromStringOrNil("c15af818-7f51-11ec-8eeb-f733ba8df393"),
			10,
			"2020-05-03 21:35:02.809",
			[]*recording.Recording{
				{
					ID:            uuid.FromStringOrNil("cfa4d576-6128-11eb-b69b-9f7a738a1ad7"),
					CustomerID:    uuid.FromStringOrNil("c15af818-7f51-11ec-8eeb-f733ba8df393"),
					ReferenceType: recording.ReferenceTypeCall,
					ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
					Status:        recording.StatusEnd,
					Filenames: []string{
						"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
					},
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"cfa4d576-6128-11eb-b69b-9f7a738a1ad7","customer_id":"c15af818-7f51-11ec-8eeb-f733ba8df393","reference_type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","recording_name":"","filenames":["call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav"],"asterisk_id":"","channel_ids":null,"tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().RecordingGets(gomock.Any(), tt.customerID, tt.pageSize, tt.pageToken).Return(tt.recordings, nil)

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

func Test_processV1RecordingsIDGet(t *testing.T) {

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
				ID:            uuid.FromStringOrNil("00c711be-6129-11eb-9404-b73dcf512957"),
				CustomerID:    uuid.FromStringOrNil("d063099a-7f51-11ec-adbd-cf15a2e7ae7d"),
				ReferenceType: recording.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:        recording.StatusEnd,
				Filenames: []string{
					"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"00c711be-6129-11eb-9404-b73dcf512957","customer_id":"d063099a-7f51-11ec-adbd-cf15a2e7ae7d","reference_type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","recording_name":"","filenames":["call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav"],"asterisk_id":"","channel_ids":null,"tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockCall := callhandler.NewMockCallHandler(mc)

			h := &listenHandler{
				rabbitSock:  mockSock,
				callHandler: mockCall,
			}

			mockCall.EXPECT().RecordingGet(gomock.Any(), tt.recording.ID).Return(tt.recording, nil)

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
