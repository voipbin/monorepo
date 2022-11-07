package callhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/util"
)

// func Test_create(t *testing.T) {

// 	tests := []struct {
// 		name          string
// 		call          *call.Call
// 		expectReqCall *call.Call
// 		expectRes     *call.Call
// 	}{
// 		{
// 			"normal",
// 			&call.Call{
// 				ID:     uuid.FromStringOrNil("0a9b21ca-992d-11ec-b0ad-f3426b2148d6"),
// 				Status: call.StatusProgressing,
// 			},
// 			&call.Call{
// 				ID:            uuid.FromStringOrNil("0a9b21ca-992d-11ec-b0ad-f3426b2148d6"),
// 				Status:        call.StatusProgressing,
// 				TMUpdate:      dbhandler.DefaultTimeStamp,
// 				TMRinging:     dbhandler.DefaultTimeStamp,
// 				TMProgressing: dbhandler.DefaultTimeStamp,
// 				TMHangup:      dbhandler.DefaultTimeStamp,
// 			},

// 			&call.Call{
// 				ID:            uuid.FromStringOrNil("0a9b21ca-992d-11ec-b0ad-f3426b2148d6"),
// 				Status:        call.StatusProgressing,
// 				TMUpdate:      dbhandler.DefaultTimeStamp,
// 				TMRinging:     dbhandler.DefaultTimeStamp,
// 				TMProgressing: dbhandler.DefaultTimeStamp,
// 				TMHangup:      dbhandler.DefaultTimeStamp,
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

// 			h := &callHandler{
// 				reqHandler:    mockReq,
// 				db:            mockDB,
// 				notifyHandler: mockNotify,
// 			}

// 			ctx := context.Background()

// 			mockDB.EXPECT().CallCreate(ctx, tt.expectReqCall).Return(nil)
// 			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.expectReqCall, nil)
// 			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectReqCall.CustomerID, call.EventTypeCallCreated, tt.expectReqCall)

// 			res, err := h.create(ctx, tt.call)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectRes) {
// 				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
// 			}

// 		})
// 	}
// }

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		id         uuid.UUID
		customerID uuid.UUID

		asteriskID   string
		channelID    string
		bridgeID     string
		flowID       uuid.UUID
		activeflowID uuid.UUID
		confbridgeID uuid.UUID
		callType     call.Type

		masterCallID   uuid.UUID
		chainedcallIDs []uuid.UUID

		recordingID  uuid.UUID
		recordingIDs []uuid.UUID

		source      *commonaddress.Address
		destination *commonaddress.Address

		status call.Status
		data   map[string]string

		action    fmaction.Action
		direction call.Direction

		dialrouteID uuid.UUID
		dialroutes  []rmroute.Route

		expectCall *call.Call

		responseTime string
		responseCall *call.Call

		expectRes *call.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("db8e0052-5d15-11ed-afd1-f3139883b1f4"),
			uuid.FromStringOrNil("dc375314-5d15-11ed-afd7-b3f36cf2d4a6"),

			"3e:50:6b:43:bb:30",
			"dc88e2e2-5d15-11ed-bd52-f3db2e20c793",
			"dcb241dc-5d15-11ed-9aa3-23e3fffa7037",

			uuid.FromStringOrNil("dce0c20a-5d15-11ed-87b5-d7bb7b446647"),
			uuid.FromStringOrNil("dd115b5e-5d15-11ed-a748-3f6e22ac12a4"),
			uuid.FromStringOrNil("dd43df70-5d15-11ed-9eb2-7f19e0311fa0"),
			call.TypeFlow,

			uuid.FromStringOrNil("28ded57a-5d16-11ed-894f-3b5a0d35af31"),
			[]uuid.UUID{
				uuid.FromStringOrNil("28ded57a-5d16-11ed-894f-3b5a0d35af31"),
			},

			uuid.FromStringOrNil("2909d2c0-5d16-11ed-9c47-139f3f088f8b"),
			[]uuid.UUID{
				uuid.FromStringOrNil("2909d2c0-5d16-11ed-9c47-139f3f088f8b"),
			},

			&commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "source target name",
				Name:       "source name",
				Detail:     "source detail",
			},
			&commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000002",
				TargetName: "destination target name",
				Name:       "destination name",
				Detail:     "destination detail",
			},

			call.StatusRinging,
			map[string]string{
				"context": "call-in",
				"domain":  "pstn.voipbin.net",
			},

			fmaction.Action{
				ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
			},
			call.DirectionIncoming,

			uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
				},
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("db8e0052-5d15-11ed-afd1-f3139883b1f4"),
				CustomerID: uuid.FromStringOrNil("dc375314-5d15-11ed-afd7-b3f36cf2d4a6"),

				AsteriskID: "3e:50:6b:43:bb:30",
				ChannelID:  "dc88e2e2-5d15-11ed-bd52-f3db2e20c793",
				BridgeID:   "dcb241dc-5d15-11ed-9aa3-23e3fffa7037",

				FlowID:       uuid.FromStringOrNil("dce0c20a-5d15-11ed-87b5-d7bb7b446647"),
				ActiveFlowID: uuid.FromStringOrNil("dd115b5e-5d15-11ed-a748-3f6e22ac12a4"),
				ConfbridgeID: uuid.FromStringOrNil("dd43df70-5d15-11ed-9eb2-7f19e0311fa0"),
				Type:         call.TypeFlow,

				MasterCallID: uuid.FromStringOrNil("28ded57a-5d16-11ed-894f-3b5a0d35af31"),
				ChainedCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("28ded57a-5d16-11ed-894f-3b5a0d35af31"),
				},

				RecordingID: uuid.FromStringOrNil("2909d2c0-5d16-11ed-9c47-139f3f088f8b"),
				RecordingIDs: []uuid.UUID{
					uuid.FromStringOrNil("2909d2c0-5d16-11ed-9c47-139f3f088f8b"),
				},

				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "source target name",
					Name:       "source name",
					Detail:     "source detail",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000002",
					TargetName: "destination target name",
					Name:       "destination name",
					Detail:     "destination detail",
				},

				Status: call.StatusRinging,
				Data: map[string]string{
					"context": "call-in",
					"domain":  "pstn.voipbin.net",
				},

				Action: fmaction.Action{
					ID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				Direction:    call.DirectionIncoming,
				HangupBy:     call.HangupByNone,
				HangupReason: call.HangupReasonNone,

				DialrouteID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
				Dialroutes: []rmroute.Route{
					{
						ID: uuid.FromStringOrNil("60679904-f101-4b8a-802f-14e563808376"),
					},
				},

				TMCreate:      "2020-04-18T03:22:17.995000",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},

			"2020-04-18T03:22:17.995000",
			&call.Call{
				ID:         uuid.FromStringOrNil("f2e8b62a-2824-11eb-ba7a-b7fd7464daa3"),
				CustomerID: uuid.FromStringOrNil("dc375314-5d15-11ed-afd7-b3f36cf2d4a6"),
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("f2e8b62a-2824-11eb-ba7a-b7fd7464daa3"),
				CustomerID: uuid.FromStringOrNil("dc375314-5d15-11ed-afd7-b3f36cf2d4a6"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := util.NewMockUtil(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				util:          mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(tt.responseTime)
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallCreated, tt.responseCall)

			res, err := h.Create(
				ctx,

				tt.id,
				tt.customerID,

				tt.asteriskID,
				tt.channelID,
				tt.bridgeID,

				tt.flowID,
				tt.activeflowID,
				tt.confbridgeID,

				tt.callType,

				tt.masterCallID,
				tt.chainedcallIDs,

				tt.recordingID,
				tt.recordingIDs,

				tt.source,
				tt.destination,

				tt.status,
				tt.data,

				tt.action,
				tt.direction,

				tt.dialrouteID,
				tt.dialroutes,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		responseGets []*call.Call
		expectRes    []*call.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("9880aedc-992e-11ec-aed2-bf63c2b64858"),
			10,
			"2020-05-03%2021:35:02.809",

			[]*call.Call{
				{
					ID: uuid.FromStringOrNil("394ab8e8-9930-11ec-ae47-b7d8e9093ff3"),
				},
			},
			[]*call.Call{
				{
					ID: uuid.FromStringOrNil("394ab8e8-9930-11ec-ae47-b7d8e9093ff3"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGets(ctx, tt.customerID, tt.size, tt.token).Return(tt.responseGets, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
