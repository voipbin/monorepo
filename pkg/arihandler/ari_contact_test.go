package arihandler

import (
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/callhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestEventHandlerContactStatusChange(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSvc := callhandler.NewMockCallHandler(mc)

	h := eventHandler{
		db:          mockDB,
		rabbitSock:  mockSock,
		reqHandler:  mockReq,
		callHandler: mockSvc,
	}

	type test struct {
		name     string
		event    *rabbitmqhandler.Event
		endpoint string
	}

	tests := []test{
		{
			"normal",
			&rabbitmqhandler.Event{
				Type:     "ari_event",
				DataType: "application/json",
				Data:     []byte(`{ "application": "voipbin", "contact_info": { "uri": "sip:jgo101ml@r5e5vuutihlr.invalid;transport=ws", "roundtrip_usec": "0", "aor": "test11@test.sip.voipbin.net", "contact_status": "NonQualified" }, "type": "ContactStatusChange", "endpoint": { "channel_ids": [], "resource": "test11@test.sip.voipbin.net", "state": "online", "technology": "PJSIP" }, "timestamp": "2021-02-19T06:32:14.621+0000", "asterisk_id": "8e:86:e2:2c:a7:51"}`),
			},
			"test11@test.sip.voipbin.net",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().RMV1ContactUpdate(gomock.Any(), tt.endpoint).Return(nil)
			if err := h.processEvent(tt.event); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
