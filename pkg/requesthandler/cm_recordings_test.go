package requesthandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestCMRecordingGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:          mockSock,
		exchangeDelay: "bin-manager.delay",
		queueCall:     "bin-manager.call-manager.request",
	}

	type test struct {
		name string

		recordingID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmrecording.Recording
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("147ff2c6-981e-11eb-90c2-d3780290d3d1"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"147ff2c6-981e-11eb-90c2-d3780290d3d1","user_id":0,"type":"call","reference_id":"e2951d7c-ac2d-11ea-8d4b-aff0e70476d6","status":"ended","format":"","filename":"call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav","webhook_uri":"","asterisk_id":"","channel_id":"","tm_start":"","tm_end":"","tm_create":"","tm_update":"","tm_delete":""}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/147ff2c6-981e-11eb-90c2-d3780290d3d1",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&cmrecording.Recording{
				ID:          uuid.FromStringOrNil("147ff2c6-981e-11eb-90c2-d3780290d3d1"),
				UserID:      0,
				Type:        cmrecording.TypeCall,
				ReferenceID: uuid.FromStringOrNil("e2951d7c-ac2d-11ea-8d4b-aff0e70476d6"),
				Status:      cmrecording.StatusEnd,
				Filename:    "call_e2951d7c-ac2d-11ea-8d4b-aff0e70476d6_2020-05-03T21:35:02.809Z.wav",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CMRecordingGet(tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
