package channelhandler

import (
	reflect "reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
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
				DestinationNumber: "conference-34613ee5-5456-40fe-bb3b-395254270a9d",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeConference,
				Target: "34613ee5-5456-40fe-bb3b-395254270a9d",
			},
		},
		{
			"type agent",

			&channel.Channel{
				DestinationNumber: "agent-a04a1f51-2495-48a5-9012-8081aa90b902",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "a04a1f51-2495-48a5-9012-8081aa90b902",
			},
		},
		{
			"type line",

			&channel.Channel{
				DestinationNumber: "line-07d16b0a-302f-4db8-ae4a-a2c9a65f88b7",
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
				Target: "2000@test.sip.voipbin.net",
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
