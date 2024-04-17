package outdialtargethandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
	"gitlab.com/voipbin/bin-manager/outdial-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		outdialID    uuid.UUID
		outdialName  string
		detail       string
		data         string
		destination0 *commonaddress.Address
		destination1 *commonaddress.Address
		destination2 *commonaddress.Address
		destination3 *commonaddress.Address
		destination4 *commonaddress.Address
	}{
		{
			"normal",

			uuid.FromStringOrNil("4b290cbe-b2c0-11ec-ae19-9773a0bdaf28"),
			"test name",
			"test detail",
			"test data",
			&commonaddress.Address{},
			nil,
			nil,
			nil,
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetCreate(ctx, gomock.Any()).Return(nil)
			mockDB.EXPECT().OutdialTargetGet(ctx, gomock.Any()).Return(&outdialtarget.OutdialTarget{}, nil)

			_, err := h.Create(
				ctx,
				tt.outdialID,
				tt.outdialName,
				tt.detail,
				tt.data,
				tt.destination0,
				tt.destination1,
				tt.destination2,
				tt.destination3,
				tt.destination4,
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

		outdialID uuid.UUID
	}{
		{
			"normal",
			uuid.FromStringOrNil("81597cf8-b561-11ec-bd35-fbc2e9bf73b8"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetDelete(ctx, tt.outdialID).Return(nil)
			mockDB.EXPECT().OutdialTargetGet(ctx, tt.outdialID).Return(&outdialtarget.OutdialTarget{}, nil)

			_, err := h.Delete(ctx, tt.outdialID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetsByOutdialID(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
		token     string
		limit     uint64
	}{
		{
			"normal",

			uuid.FromStringOrNil("05b5c738-b2c2-11ec-acd3-27fc70dc1b15"),
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
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetGetsByOutdialID(ctx, tt.outdialID, tt.token, tt.limit).Return([]*outdialtarget.OutdialTarget{}, nil)
			_, err := h.GetsByOutdialID(ctx, tt.outdialID, tt.token, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_GetAvailable(t *testing.T) {

	tests := []struct {
		name string

		outdialID uuid.UUID
		tryCount0 int
		tryCount1 int
		tryCount2 int
		tryCount3 int
		tryCount4 int
		limit     uint64
	}{
		{
			"normal",

			uuid.FromStringOrNil("e16c55f8-b2c2-11ec-8d2b-bf830eef4a7d"),
			1,
			2,
			3,
			4,
			5,
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
			h := outdialTargetHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().OutdialTargetGetAvailable(ctx, tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.limit).Return([]*outdialtarget.OutdialTarget{}, nil)
			_, err := h.GetAvailable(ctx, tt.outdialID, tt.tryCount0, tt.tryCount1, tt.tryCount2, tt.tryCount3, tt.tryCount4, tt.limit)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
