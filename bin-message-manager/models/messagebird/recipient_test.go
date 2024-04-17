package messagebird

import (
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-message-manager/models/target"
)

func Test_ConvertTartget(t *testing.T) {

	type test struct {
		name string

		recipient *Recipient
		expectRes *target.Target
	}

	tests := []test{
		{
			"normal",

			&Recipient{
				Recipient:        31616818985,
				Status:           "sent",
				StatusDatetime:   "2022-03-09T05:21:45+00:00",
				MessagePartCount: 1,
			},
			&target.Target{
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+31616818985",
				},
				Status: target.StatusSent,
				Parts:  1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := tt.recipient.ConvertTartget()

			tt.expectRes.TMUpdate = res.TMUpdate
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
