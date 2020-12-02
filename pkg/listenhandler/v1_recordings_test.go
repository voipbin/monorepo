package listenhandler

import (
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/storage-manager.git/pkg/buckethandler"
)

func TestV1RecordingsIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockBucket := buckethandler.NewMockBucketHandler(mc)
	// mockCall := callhandler.NewMockCallHandler(mc)

	h := &listenHandler{
		rabbitSock:    mockSock,
		bucketHandler: mockBucket,
	}

	type test struct {
		name        string
		recordingID string
		request     *rabbitmqhandler.Request
		expectRes   *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			"call_776c8a94-34bd-11eb-abef-0b279f3eabc1_2020-04-18T03:22:17.995000Z",
			&rabbitmqhandler.Request{
				URI:    "/v1/recordings/call_776c8a94-34bd-11eb-abef-0b279f3eabc1_2020-04-18T03:22:17.995000Z",
				Method: rabbitmqhandler.RequestMethodGet,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockBucket.EXPECT().RecordingGetDownloadURL(tt.recordingID, gomock.Any())

			_, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
