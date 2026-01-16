package outplanhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID   uuid.UUID
		outplanName  string
		detail       string
		source       *commonaddress.Address
		DialTimeout  int
		tryInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int

		responseUUID uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("bccbd45a-b3d8-11ec-a518-b79a1b2fe501"),
			"test name",
			"test detail",
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},

			30000,
			300000,
			3,
			3,
			3,
			3,
			3,

			uuid.FromStringOrNil("2c3a6208-6d04-11ee-ae68-27b62bc40354"),
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
			h := outplanHandler{
				util:          mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().OutplanCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().OutplanGet(ctx, gomock.Any()).Return(&outplan.Outplan{}, nil)

			_, err := h.Create(
				ctx,
				tt.customerID,
				tt.outplanName,
				tt.detail,
				tt.source,
				tt.DialTimeout,
				tt.tryInterval,
				tt.maxTryCount0,
				tt.maxTryCount1,
				tt.maxTryCount2,
				tt.maxTryCount3,
				tt.maxTryCount4,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID
	}{
		{
			"normal",

			uuid.FromStringOrNil("25c10840-b483-11ec-adef-db7166e52bbe"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outplanHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutplanDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().OutplanGet(ctx, tt.id).Return(&outplan.Outplan{}, nil)
			_, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ListByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		token      string
		limit      uint64
	}{
		{
			"normal",

			uuid.FromStringOrNil("68d4b05a-b3d9-11ec-99be-d305ba7d7154"),
			"2020-10-10 03:30:17.000000",
			10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outplanHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutplanListByCustomerID(ctx, tt.customerID, tt.token, tt.limit).Return([]*outplan.Outplan{}, nil)
			_, err := h.ListByCustomerID(ctx, tt.customerID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateBasicInfo(t *testing.T) {
	tests := []struct {
		name string

		id          uuid.UUID
		outplanName string
		detail      string
	}{
		{
			"test normal",

			uuid.FromStringOrNil("ede467d0-b3da-11ec-bb49-e75e301e41f0"),
			"update name",
			"update detail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outplanHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutplanUpdateBasicInfo(ctx, tt.id, tt.outplanName, tt.detail).Return(nil)
			mockDB.EXPECT().OutplanGet(ctx, tt.id).Return(&outplan.Outplan{}, nil)

			_, err := h.UpdateBasicInfo(ctx, tt.id, tt.outplanName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_UpdateDialInfo(t *testing.T) {
	tests := []struct {
		name string

		id           uuid.UUID
		source       *commonaddress.Address
		dialTimeout  int
		tyrInterval  int
		maxTryCount0 int
		maxTryCount1 int
		maxTryCount2 int
		maxTryCount3 int
		maxTryCount4 int
	}{
		{
			"normal",

			uuid.FromStringOrNil("1f7003d0-b3dc-11ec-906b-33094783cdd2"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			30000,
			600000,
			3,
			3,
			3,
			3,
			3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &outplanHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutplanUpdateDialInfo(ctx, tt.id, tt.source, tt.dialTimeout, tt.tyrInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4).Return(nil)
			mockDB.EXPECT().OutplanGet(ctx, tt.id).Return(&outplan.Outplan{}, nil)

			_, err := h.UpdateDialInfo(ctx, tt.id, tt.source, tt.dialTimeout, tt.tyrInterval, tt.maxTryCount0, tt.maxTryCount1, tt.maxTryCount2, tt.maxTryCount3, tt.maxTryCount4)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
