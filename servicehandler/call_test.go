package servicehandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/api-manager/models/call"
	"gitlab.com/voipbin/bin-manager/api-manager/models/user"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmaction"
	"gitlab.com/voipbin/bin-manager/api-manager/pkg/requesthandler/models/cmcall"
)

func TestCallCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	type test struct {
		name        string
		user        *user.User
		flowID      uuid.UUID
		source      call.Address
		destination call.Address
		cmCall      *cmcall.Call
		expectCall  call.Call
	}

	tests := []test{
		{
			"normal",
			&user.User{
				ID: 1,
			},
			uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
			call.Address{
				Type:   call.AddressTypeSIP,
				Target: "testsource@test.com",
			},
			call.Address{
				Type:   call.AddressTypeSIP,
				Target: "testdestination@test.com",
			},
			&cmcall.Call{
				ID:         uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				AsteriskID: "02:42:5d:f3:a7:05",
				ChannelID:  "d66d7c02-efc5-11ea-9f77-6fe9fae57afd",
				UserID:     1,
				FlowID:     uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
				ConfID:     uuid.Nil,
				Type:       cmcall.TypeFlow,

				Source: cmcall.Address{
					Type:   cmcall.AddressTypeSIP,
					Target: "testsource@test.com",
				},
				Destination: cmcall.Address{
					Type:   cmcall.AddressTypeSIP,
					Target: "testdestination@test.com",
				},

				Status:       cmcall.StatusDialing,
				Data:         map[string]interface{}{},
				Action:       cmaction.Action{},
				Direction:    cmcall.DirectionIncoming,
				HangupBy:     "",
				HangupReason: "",
			},
			call.Call{
				ID:     uuid.FromStringOrNil("88d05668-efc5-11ea-940c-b39a697e7abe"),
				UserID: 1,
				FlowID: uuid.FromStringOrNil("2c45d0b8-efc4-11ea-9a45-4f30fc2e0b02"),
				ConfID: uuid.Nil,
				Type:   call.TypeFlow,

				Source: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "testsource@test.com",
				},
				Destination: call.Address{
					Type:   call.AddressTypeSIP,
					Target: "testdestination@test.com",
				},

				Status:       call.StatusDialing,
				Direction:    call.DirectionIncoming,
				HangupBy:     "",
				HangupReason: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			mockReq.EXPECT().CallCallCreate(tt.user.ID, tt.flowID, tt.cmCall.Source, tt.cmCall.Destination).Return(tt.cmCall, nil)

			res, err := h.CallCreate(tt.user, tt.flowID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, tt.expectCall) != true {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectCall, res)
			}
		})
	}

}
