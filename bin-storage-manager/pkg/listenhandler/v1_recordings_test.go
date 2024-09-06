package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-storage-manager/pkg/storagehandler"
)

func Test_v1RecordingsIDGet(t *testing.T) {

	type test struct {
		name        string
		recordingID uuid.UUID
		request     *sock.Request
		expectRes   *sock.Response
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("0c920df8-9821-11eb-91f1-976b4da76663"),
			&sock.Request{
				URI:    "/v1/recordings/0c920df8-9821-11eb-91f1-976b4da76663",
				Method: sock.RequestMethodGet,
			},
			&sock.Response{
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
		request *sock.Request

		expectRecordingID uuid.UUID
		expectRes         *sock.Response
	}

	tests := []test{
		{
			"normal",
			&sock.Request{
				URI:    "/v1/recordings/e1f3eb20-8eaa-11ed-8013-a7cd667479cb",
				Method: sock.RequestMethodDelete,
			},

			uuid.FromStringOrNil("e1f3eb20-8eaa-11ed-8013-a7cd667479cb"),
			&sock.Response{
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
