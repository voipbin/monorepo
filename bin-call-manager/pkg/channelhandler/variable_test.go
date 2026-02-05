package channelhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_VariableSet(t *testing.T) {

	type test struct {
		name string

		id    string
		key   string
		value string

		responseChannel *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			id:    "10dcefd8-f2db-11ed-8e26-af08a1c6a5e3",
			key:   "key1",
			value: "value1",

			responseChannel: &channel.Channel{
				ID:         "10dcefd8-f2db-11ed-8e26-af08a1c6a5e3",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   nil,
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
			ctx := context.Background()

			mockDB.EXPECT().ChannelGet(gomock.Any(), tt.id).Return(tt.responseChannel, nil)
			mockReq.EXPECT().AstChannelVariableSet(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, tt.key, tt.value).Return(nil)

			if errSet := h.VariableSet(ctx, tt.id, tt.key, tt.value); errSet != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errSet)
			}
		})
	}
}

func Test_variableGet(t *testing.T) {

	type test struct {
		name string

		id      string
		key     string
		channel *channel.Channel

		responseValue string
	}

	tests := []test{
		{
			name: "normal",

			id:  "477849fc-f2db-11ed-9edb-434f4687b972",
			key: "key1",
			channel: &channel.Channel{
				ID:         "477849fc-f2db-11ed-9edb-434f4687b972",
				AsteriskID: "3e:50:6b:43:bb:30",
				TMDelete:   nil,
			},

			responseValue: "value1",
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
			ctx := context.Background()

			mockReq.EXPECT().AstChannelVariableGet(ctx, tt.channel.AsteriskID, tt.channel.ID, tt.key).Return(tt.responseValue, nil)

			res, err := h.variableGet(ctx, tt.channel, tt.key)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.responseValue {
				t.Errorf("Wrong match. expect: %s, got: %v", tt.responseValue, res)
			}
		})
	}
}
