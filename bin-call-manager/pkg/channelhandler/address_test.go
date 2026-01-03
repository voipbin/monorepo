package channelhandler

import (
	reflect "reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_AddressGetSource(t *testing.T) {

	type test struct {
		name string

		channel     *channel.Channel
		addressType commonaddress.Type

		expectRes *commonaddress.Address
	}

	tests := []test{
		{
			name: "address type is endpoint",

			channel: &channel.Channel{
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeDomain: "test.trunk.voipbin.net",
				},
				SourceNumber: "2000",
			},
			addressType: commonaddress.TypeSIP,

			expectRes: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "2000@test.trunk.voipbin.net",
			},
		},
		{
			name: "address type is tel",

			channel: &channel.Channel{
				SourceNumber: "+821100000001",
				SourceName:   "test number",
			},
			addressType: commonaddress.TypeTel,

			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "test number",
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
			res := h.AddressGetSource(tt.channel, tt.addressType)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_AddressGetDestination(t *testing.T) {

	type test struct {
		name string

		channel     *channel.Channel
		addressType commonaddress.Type

		expectRes *commonaddress.Address
	}

	tests := []test{
		{
			name: "normal",

			channel: &channel.Channel{
				DestinationNumber: "+821100000001",
				DestinationName:   "test number",
			},
			addressType: commonaddress.TypeTel,

			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "test number",
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
			res := h.AddressGetDestination(tt.channel, tt.addressType)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

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
			"type extension",

			&channel.Channel{
				DestinationNumber: "2000",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeDomain: "test.trunk.voipbin.net",
				},
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "2000",
			},
		},
		{
			"type extension",

			&channel.Channel{
				// extension:2000
				DestinationNumber: "extension%3A2000",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeDomain: "60a6292a-e8de-11f0-b37a-07eea6c7482e.registrar.voipbin.net",
				},
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeExtension,
				Target: "2000",
			},
		},
		{
			"type sip",

			&channel.Channel{
				// sip:user@example.com
				DestinationNumber: "sip%3Auser%40example.com",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "user@example.com",
			},
		},
		{
			"type tel explicit",

			&channel.Channel{
				// tel:+821100000001
				DestinationNumber: "tel%3A%2B821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
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
