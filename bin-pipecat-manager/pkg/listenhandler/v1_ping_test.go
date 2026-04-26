package listenhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/pipecatcallhandler"
)

func Test_processV1PingGet(t *testing.T) {
	tests := []struct {
		name           string
		request        *sock.Request
		mockPing       *pipecatcall.PingResult
		expectedStatus int
	}{
		{
			name: "GET /v1/ping returns 200 with PingResult body",
			request: &sock.Request{
				URI:    "/v1/ping",
				Method: sock.RequestMethodGet,
			},
			mockPing:       &pipecatcall.PingResult{HostID: "10.4.2.18", Timestamp: time.Now().UTC()},
			expectedStatus: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockPCH := pipecatcallhandler.NewMockPipecatcallHandler(mc)
			mockPCH.EXPECT().Ping(gomock.Any()).Return(tt.mockPing, nil)

			h := &listenHandler{
				sockHandler:        sockhandler.NewMockSockHandler(mc),
				pipecatcallHandler: mockPCH,
			}

			res, err := h.processV1PingGet(context.Background(), tt.request)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != tt.expectedStatus {
				t.Errorf("StatusCode = %d, want %d", res.StatusCode, tt.expectedStatus)
			}
			if res.DataType != "application/json" {
				t.Errorf("DataType = %q, want application/json", res.DataType)
			}
			var got pipecatcall.PingResult
			if err := json.Unmarshal(res.Data, &got); err != nil {
				t.Fatalf("response body did not unmarshal as PingResult: %v", err)
			}
			if !reflect.DeepEqual(&got, tt.mockPing) {
				t.Errorf("body = %+v, want %+v", got, tt.mockPing)
			}
		})
	}
}
