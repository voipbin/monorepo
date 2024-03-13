package webhookhandler

import (
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/pkg/dbhandler"
)

func Test_sendMessage(t *testing.T) {

	type test struct {
		name     string
		uri      string
		method   string
		dataType string
		data     []byte
	}

	tests := []test{
		{
			"normal",
			"http://example.com",
			"POST",
			"application/json",
			[]byte(`{"test key":"test value"}`),
		},
		{
			"empty data",
			"http://example.com",
			"POST",
			"",
			nil,
		},
		{
			"real destination ",
			"https://eoxdgfr5e7087sf.m.pipedream.net",
			"POST",
			"application/json",
			[]byte(`{"type":"call_create","data":{"master_call_id":"00000000-0000-0000-0000-000000000000","direction":"outgoing","hangup_by":"","user_id":1,"id":"cdbe20b4-4f49-4a27-96d5-4dda0ec2134b","action":{"id":"00000000-0000-0000-0000-000000000001","tm_execute":"","type":""},"flow_id":"883f40c7-7f16-48fe-a404-772d9b038808","type":"flow","source":{"target":"+821028286521","name":"","type":"tel"},"tm_update":"","tm_ringing":"","data":null,"recording_id":"00000000-0000-0000-0000-000000000000","chained_call_ids":[],"hangup_reason":"","webhook_uri":"https://endxhr87aa0bkge.m.pipedream.net","tm_progressing":"","conf_id":"00000000-0000-0000-0000-000000000000","tm_hangup":"","asterisk_id":"","status":"dialing","recording_ids":[],"destination":{"name":"","target":"+821021656521","type":"tel"},"channel_id":"8edc9ed7-2ec0-4027-a7b9-3d6eced18afd","tm_create":"2021-03-13 18:18:02.489462"}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &webhookHandler{
				db: mockDB,
			}

			resp, err := h.sendMessage(tt.uri, tt.method, tt.dataType, tt.data)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if resp.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %d", resp.StatusCode)
			}
		})
	}
}
