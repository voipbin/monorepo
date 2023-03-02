package channelhandler

import (
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_AddressGetDestinationWithoutSpecificType(t *testing.T) {

	type test struct {
		name string

		channel   *channel.Channel
		expectRes *commonaddress.Address
	}

	tests := []test{
		{
			"type tel",

			&channel.Channel{
				DestinationNumber: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
		{
			"type conference",

			&channel.Channel{
				// conference:34613ee5-5456-40fe-bb3b-395254270a9d
				DestinationNumber: "conference%3A34613ee5-5456-40fe-bb3b-395254270a9d",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeConference,
				Target: "34613ee5-5456-40fe-bb3b-395254270a9d",
			},
		},
		{
			"type agent",

			&channel.Channel{
				// agent:a04a1f51-2495-48a5-9012-8081aa90b902
				DestinationNumber: "agent%3Aa04a1f51-2495-48a5-9012-8081aa90b902",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "a04a1f51-2495-48a5-9012-8081aa90b902",
			},
		},
		{
			"type line",

			&channel.Channel{
				// line:07d16b0a-302f-4db8-ae4a-a2c9a65f88b7
				DestinationNumber: "line%3A07d16b0a-302f-4db8-ae4a-a2c9a65f88b7",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "07d16b0a-302f-4db8-ae4a-a2c9a65f88b7",
			},
		},
		{
			"type endpoint",

			&channel.Channel{
				DestinationNumber: "2000",
				StasisData: map[string]string{
					"domain": "test.sip.voipbin.net",
				},
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeEndpoint,
				Target: "2000@test",
			},
		},
		{
			"type endpoint in abolute reference",

			&channel.Channel{
				// 3000@test_domain
				DestinationNumber: "3000%40test_domain",
				StasisData: map[string]string{
					"domain": "test.sip.voipbin.net",
				},
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeEndpoint,
				Target: "3000@test_domain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			res := h.AddressGetDestinationWithoutSpecificType(tt.channel)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
