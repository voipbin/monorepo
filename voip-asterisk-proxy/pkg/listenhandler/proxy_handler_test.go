package listenhandler

import (
	"monorepo/bin-common-handler/models/sock"
	"monorepo/voip-asterisk-proxy/pkg/servicehandler"
	"reflect"
	"testing"

	"go.uber.org/mock/gomock"
)

func Test_processProxyRecordingFileMovePost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		expectFilenames []string
		expectRes       *sock.Response
	}{
		{
			name: "basic",
			request: &sock.Request{
				URI:      "/proxy/recording_file_move",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"filenames": ["test1.wav", "test2.wav"]}`),
			},

			expectFilenames: []string{
				"test1.wav",
				"test2.wav",
			},
			expectRes: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockService := servicehandler.NewMockServiceHandler(mc)
			h := &listenHandler{
				serviceHandler: mockService,
			}

			mockService.EXPECT().RecordingFileMove(gomock.Any(), tt.expectFilenames).Return(nil)

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
