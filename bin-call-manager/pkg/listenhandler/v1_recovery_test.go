package listenhandler

import (
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	reflect "reflect"
	"testing"

	gomock "go.uber.org/mock/gomock"
)

func Test_processV1RecoveryPost(t *testing.T) {
	tests := []struct {
		name    string
		request *sock.Request

		expectedAsteriskID string
		expectedRes        *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:      "/v1/recovery",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"asterisk_id": "42:01:0a:a4:00:03"}`),
			},

			expectedAsteriskID: "42:01:0a:a4:00:03",
			expectedRes: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockSock := sockhandler.NewMockSockHandler(mc)
			mockCall := callhandler.NewMockCallHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)

			h := &listenHandler{
				utilHandler:      mockUtil,
				sockHandler:      mockSock,
				callHandler:      mockCall,
				recordingHandler: mockRecording,
			}

			mockCall.EXPECT().Recovery(gomock.Any(), "42:01:0a:a4:00:03").Return(nil).Times(1)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}
