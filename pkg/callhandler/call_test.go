package callhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_create(t *testing.T) {
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

	type test struct {
		name          string
		call          *call.Call
		expectReqCall *call.Call
		expectRes     *call.Call
	}

	tests := []test{
		{
			"normal",
			&call.Call{
				ID:     uuid.FromStringOrNil("0a9b21ca-992d-11ec-b0ad-f3426b2148d6"),
				Status: call.StatusProgressing,
			},
			&call.Call{
				ID:            uuid.FromStringOrNil("0a9b21ca-992d-11ec-b0ad-f3426b2148d6"),
				Status:        call.StatusProgressing,
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},

			&call.Call{
				ID:            uuid.FromStringOrNil("0a9b21ca-992d-11ec-b0ad-f3426b2148d6"),
				Status:        call.StatusProgressing,
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().CallCreate(ctx, tt.expectReqCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.expectReqCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectReqCall.CustomerID, call.EventTypeCallCreated, tt.expectReqCall)

			res, err := h.create(ctx, tt.call)
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

	type test struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string

		responseGets []*call.Call
		expectRes    []*call.Call
	}

	tests := []test{
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
