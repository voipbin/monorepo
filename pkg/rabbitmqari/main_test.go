package rabbitmqari

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager/pkg/rabbitmq"
)

func TestSendARIRequest(t *testing.T) {
	type test struct {
		name       string
		asteriskID string
		url        string
		method     string
		timeout    int64
		dataType   string
		data       string

		expectTarget  string
		expectPayload string
	}

	tests := []test{
		{
			"have all item",
			"00:11:22:33:44:55",
			"bae178e2-7f6f-11ea-809d-b3dec50dc8f3",
			"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/continue",
			3,
			"application/json",
			`{"context":"test-context","extension":"testcall","priority":1,"label":"testlabel"}`,

			"asterisk_ari_request-00:11:22:33:44:55",
			`{"url":"bae178e2-7f6f-11ea-809d-b3dec50dc8f3","method":"/ari/channels/bae178e2-7f6f-11ea-809d-b3dec50dc8f3/continue","data_type":"application/json","data":"{\"context\":\"test-context\",\"extension\":\"testcall\",\"priority\":1,\"label\":\"testlabel\"}"}`,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockRabbit := rabbitmq.NewMockRabbit(mc)
	requester := requester{
		sock: mockRabbit,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRabbit.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectPayload).Return([]byte(`{"status_code":200,"data":""}`), nil)

			_, err := requester.SendARIRequest(tt.asteriskID, tt.url, tt.method, tt.timeout, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
