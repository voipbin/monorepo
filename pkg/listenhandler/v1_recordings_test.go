package listenhandler

import (
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/storagehandler"
)

func Test_v1RecordingsIDGet(t *testing.T) {

	type test struct {
		name        string
		recordingID uuid.UUID
		request     *rabbitmqhandler.Request
		expectRes   *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("0c920df8-9821-11eb-91f1-976b4da76663"),
			&rabbitmqhandler.Request{
				URI:    "/v1/recordings/0c920df8-9821-11eb-91f1-976b4da76663",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().RecordingGet(gomock.Any(), tt.recordingID)
			_, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_v1RecordingsIDDelete(t *testing.T) {

	type test struct {
		name    string
		request *rabbitmqhandler.Request

		expectRecordingID uuid.UUID
		expectRes         *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/recordings/e1f3eb20-8eaa-11ed-8013-a7cd667479cb",
				Method: rabbitmqhandler.RequestMethodDelete,
			},

			uuid.FromStringOrNil("e1f3eb20-8eaa-11ed-8013-a7cd667479cb"),
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockStorage := storagehandler.NewMockStorageHandler(mc)

			h := &listenHandler{
				rabbitSock:     mockSock,
				storageHandler: mockStorage,
			}

			mockStorage.EXPECT().RecordingDelete(gomock.Any(), tt.expectRecordingID)
			_, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
